package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/pkg/problem"
	"google.golang.org/grpc/codes"
)

// Idempotence (G7). `Idempotency-Key` est obligatoire sur les routes que le contrat déclare (voir idempotentRoutes ;
// un test compare cette table à openapi.yaml). Le résultat d'un appel réussi est mémorisé 24 h sous
// (acteur, méthode, chemin, clé) avec l'empreinte du corps :
//   - même clé, même corps  → la réponse mémorisée est rejouée (`Idempotent-Replayed: true`), le handler n'est PAS rappelé ;
//   - même clé, corps différent → 409 IDEMPOTENCY_KEY_REUSED ;
//   - même clé pendant que le premier appel s'exécute encore → 409 CONFLICT (+ Retry-After).
//
// Seules les réponses 2xx sont mémorisées : une erreur (4xx/5xx) libère la clé pour que le client puisse réessayer.
// Si le magasin est indisponible (Redis HS), on REFUSE l'appel (503) : ces routes touchent à l'argent, un doublon
// est pire qu'une indisponibilité.

var idempotentRoutes = []struct {
	Method string
	Re     *regexp.Regexp
}{
	{"POST", regexp.MustCompile(`^/v1/orders$`)},
	{"POST", regexp.MustCompile(`^/v1/payments$`)},
	{"POST", regexp.MustCompile(`^/v1/quotes/[^/]+/accept$`)},
	{"POST", regexp.MustCompile(`^/v1/admin/orders/[^/]+/refunds$`)},
	{"POST", regexp.MustCompile(`^/v1/admin/payments/[^/]+/mark-paid$`)},
	{"POST", regexp.MustCompile(`^/v1/admin/quotes/[^/]+/convert$`)},
	{"POST", regexp.MustCompile(`^/v1/admin/refunds/[^/]+/retry$`)},
}

const (
	idemTTL      = 24 * time.Hour
	idemLock     = 60 * time.Second // durée max d'un appel « en cours » : un gateway planté ne bloque pas la clé 24 h
	idemMaxStore = 1 << 20          // réponse > 1 Mio : non mémorisée (la clé est libérée)
)

// IdemRecord est l'état mémorisé pour une clé.
type IdemRecord struct {
	Done        bool
	Fingerprint string
	Status      int
	ContentType string
	Body        []byte
}

// IdempotencyStore : stockage partagé entre instances (Redis) ; mémoire pour le développement.
type IdempotencyStore interface {
	// Begin réserve la clé. created=true : l'appelant exécute la requête (et doit appeler Complete ou Release avec
	// `owner`). Sinon `existing` décrit l'état déjà présent.
	Begin(ctx context.Context, key, fingerprint, owner string) (created bool, existing IdemRecord, err error)
	Complete(ctx context.Context, key, owner string, status int, contentType string, body []byte) error
	Release(ctx context.Context, key, owner string) error
}

// ── Redis ───────────────────────────────────────────────────────────────────

var idemBegin = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then
  local v = redis.call('HMGET', KEYS[1], 's', 'fp', 'st', 'ct', 'b')
  return {0, v[1] or '', v[2] or '', v[3] or '', v[4] or '', v[5] or ''}
end
redis.call('HSET', KEYS[1], 's', 'p', 'fp', ARGV[1], 'o', ARGV[2])
redis.call('PEXPIRE', KEYS[1], ARGV[3])
return {1, '', '', '', '', ''}
`)

var idemComplete = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'o') ~= ARGV[1] then return 0 end
redis.call('HSET', KEYS[1], 's', 'd', 'st', ARGV[2], 'ct', ARGV[3], 'b', ARGV[4])
redis.call('PEXPIRE', KEYS[1], ARGV[5])
return 1
`)

var idemRelease = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'o') == ARGV[1] then return redis.call('DEL', KEYS[1]) end
return 0
`)

type RedisIdempotency struct {
	rdb    redis.UniversalClient
	prefix string
}

func NewRedisIdempotency(rdb redis.UniversalClient) *RedisIdempotency {
	return &RedisIdempotency{rdb: rdb, prefix: "ff:idem:"}
}

func (s *RedisIdempotency) Begin(ctx context.Context, key, fp, owner string) (bool, IdemRecord, error) {
	r, err := idemBegin.Run(ctx, s.rdb, []string{s.prefix + key}, fp, owner, idemLock.Milliseconds()).Slice()
	if err != nil || len(r) != 6 {
		return false, IdemRecord{}, firstErr(err)
	}
	if n, _ := r[0].(int64); n == 1 {
		return true, IdemRecord{}, nil
	}
	str := func(i int) string { v, _ := r[i].(string); return v }
	st, _ := strconv.Atoi(str(3))
	return false, IdemRecord{Done: str(1) == "d", Fingerprint: str(2), Status: st, ContentType: str(4), Body: []byte(str(5))}, nil
}

func (s *RedisIdempotency) Complete(ctx context.Context, key, owner string, status int, ct string, body []byte) error {
	return idemComplete.Run(ctx, s.rdb, []string{s.prefix + key}, owner, status, ct, string(body), idemTTL.Milliseconds()).Err()
}

func (s *RedisIdempotency) Release(ctx context.Context, key, owner string) error {
	return idemRelease.Run(ctx, s.rdb, []string{s.prefix + key}, owner).Err()
}

func firstErr(err error) error {
	if err != nil {
		return err
	}
	return io.ErrUnexpectedEOF
}

// ── mémoire (développement, tests ; une seule instance) ─────────────────────

type memIdem struct {
	mu  sync.Mutex
	now func() time.Time
	m   map[string]*memIdemEntry
}
type memIdemEntry struct {
	rec     IdemRecord
	owner   string
	expires time.Time
}

func NewMemoryIdempotency(now func() time.Time) IdempotencyStore {
	if now == nil {
		now = time.Now
	}
	return &memIdem{now: now, m: map[string]*memIdemEntry{}}
}

func (s *memIdem) Begin(_ context.Context, key, fp, owner string) (bool, IdemRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if e, ok := s.m[key]; ok && now.Before(e.expires) {
		return false, e.rec, nil
	}
	s.m[key] = &memIdemEntry{rec: IdemRecord{Fingerprint: fp}, owner: owner, expires: now.Add(idemLock)}
	return true, IdemRecord{}, nil
}

func (s *memIdem) Complete(_ context.Context, key, owner string, status int, ct string, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.m[key]; ok && e.owner == owner {
		e.rec = IdemRecord{Done: true, Fingerprint: e.rec.Fingerprint, Status: status, ContentType: ct, Body: body}
		e.expires = s.now().Add(idemTTL)
	}
	return nil
}

func (s *memIdem) Release(_ context.Context, key, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.m[key]; ok && e.owner == owner {
		delete(s.m, key)
	}
	return nil
}

// ── middleware ──────────────────────────────────────────────────────────────

func isIdempotentRoute(method, path string) bool {
	for _, r := range idempotentRoutes {
		if r.Method == method && r.Re.MatchString(path) {
			return true
		}
	}
	return false
}

// capture copie la réponse pour pouvoir la mémoriser, sans retarder son envoi.
type capture struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
	over   bool
}

func (c *capture) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *capture) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	if c.buf.Len()+len(b) > idemMaxStore {
		c.over = true
	} else {
		c.buf.Write(b)
	}
	return c.ResponseWriter.Write(b)
}

func (s *Server) idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isIdempotentRoute(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		trace := infoFrom(r.Context()).TraceID
		idemKey := r.Header.Get("Idempotency-Key")
		actor := s.actorOf(r)
		if idemKey == "" || len(idemKey) < 8 || len(idemKey) > 128 || actor == "" {
			// clé absente/invalide ou appelant non authentifié : le serveur généré répond (400 / 401), rien n'est mémorisé
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			problem.Write(w, problem.New(http.StatusRequestEntityTooLarge, apperr.CodeValidation, "Corps trop volumineux.", trace))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		fpSum := sha256.Sum256(body)
		fp := hex.EncodeToString(fpSum[:])
		ksum := sha256.Sum256([]byte(actor + "\x00" + r.Method + "\x00" + r.URL.Path + "\x00" + idemKey))
		key := hex.EncodeToString(ksum[:])
		var ob [12]byte
		_, _ = rand.Read(ob[:])
		owner := hex.EncodeToString(ob[:])

		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		created, ex, err := s.idem.Begin(ctx, key, fp, owner)
		cancel()
		if err != nil {
			s.log.Error("idempotence : magasin indisponible, appel refusé", "trace_id", trace, "err", err.Error())
			problem.Write(w, problem.From(apperr.New(codes.Unavailable, apperr.CodeInternal, "service momentanément indisponible, réessayez"), trace).WithRetryAfter(2))
			return
		}
		if !created {
			switch {
			case ex.Fingerprint != fp:
				problem.Write(w, problem.New(http.StatusConflict, apperr.CodeIdempotencyReused, "Cette clé d'idempotence a déjà servi avec une requête différente.", trace))
			case !ex.Done:
				problem.Write(w, problem.New(http.StatusConflict, apperr.CodeConflict, "Une requête identique est déjà en cours de traitement.", trace).WithRetryAfter(1))
			default:
				if ex.ContentType != "" {
					w.Header().Set("Content-Type", ex.ContentType)
				}
				w.Header().Set("Idempotent-Replayed", "true")
				w.WriteHeader(ex.Status)
				_, _ = w.Write(ex.Body)
			}
			return
		}

		cw := &capture{ResponseWriter: w}
		stored := false
		defer func() {
			// jamais bloquer la réponse déjà partie ; on libère ou on mémorise, quoi qu'il arrive (panic comprise)
			bg, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if cw.status >= 200 && cw.status < 300 && !cw.over {
				if err := s.idem.Complete(bg, key, owner, cw.status, cw.Header().Get("Content-Type"), cw.buf.Bytes()); err == nil {
					stored = true
				} else {
					s.log.Error("idempotence : mémorisation impossible", "trace_id", trace, "err", err.Error())
				}
			}
			if !stored {
				_ = s.idem.Release(bg, key, owner)
			}
		}()
		next.ServeHTTP(cw, r)
	})
}

// actorOf identifie l'appelant à partir du jeton (vérifié localement, comme le fait la couche de sécurité) : « aud:sub ».
// Vide si le jeton est absent ou invalide : le serveur généré refusera la requête.
func (s *Server) actorOf(r *http.Request) string {
	const p = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(p) || h[:len(p)] != p || s.keys == nil {
		return ""
	}
	tok := h[len(p):]
	for _, aud := range []string{jwtx.AudCustomer, jwtx.AudStaff} {
		if c, err := s.keys.Verify(r.Context(), aud, tok); err == nil && c.Subject != "" {
			return aud + ":" + c.Subject
		}
	}
	return ""
}
