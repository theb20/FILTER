package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
)

type idemSrc struct{ ks map[string]*jwtx.KeySet }

func (s idemSrc) Fetch(_ context.Context, aud string) ([]byte, error) { return s.ks[aud].JWKS(), nil }

type idemRig struct {
	s     *Server
	h     http.Handler
	calls atomic.Int32
	ks    map[string]*jwtx.KeySet
	code  atomic.Int32 // statut renvoyé par le faux handler (défaut 201)
}

func newIdemRig(t *testing.T, store IdempotencyStore) *idemRig {
	t.Helper()
	r := &idemRig{ks: map[string]*jwtx.KeySet{}}
	for _, aud := range []string{jwtx.AudCustomer, jwtx.AudStaff} {
		k, err := jwtx.NewKeySet()
		if err != nil {
			t.Fatal(err)
		}
		r.ks[aud] = k
	}
	r.code.Store(201)
	r.s = &Server{log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: time.Now, idem: store, keys: NewKeyCache(idemSrc{r.ks}, 30*time.Second, 0, nil)}
	r.h = r.s.idempotency(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.calls.Add(1)
		b, _ := io.ReadAll(req.Body) // le corps doit rester lisible après l'empreinte
		time.Sleep(20 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(r.code.Load()))
		_, _ = w.Write([]byte(`{"echo":` + string(b) + `,"n":` + string(rune('0'+r.calls.Load())) + `}`))
	}))
	return r
}

func (r *idemRig) token(t *testing.T, aud, sub string) string {
	t.Helper()
	tok, err := r.ks[aud].Sign(aud, sub, time.Minute, time.Now(), jwtx.Claims{})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func (r *idemRig) post(path, key, tok, body string) *httptest.ResponseRecorder {
	q := httptest.NewRequest("POST", path, strings.NewReader(body))
	if key != "" {
		q.Header.Set("Idempotency-Key", key)
	}
	if tok != "" {
		q.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, q)
	return w
}

func stores(t *testing.T) map[string]IdempotencyStore {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return map[string]IdempotencyStore{"memoire": NewMemoryIdempotency(nil), "redis": NewRedisIdempotency(rdb)}
}

func TestIdempotence_RejeuMemeCorps(t *testing.T) {
	for name, st := range stores(t) {
		t.Run(name, func(t *testing.T) {
			r := newIdemRig(t, st)
			tok := r.token(t, jwtx.AudCustomer, uuid.NewString())
			a := r.post("/v1/orders", "cle-commande-1", tok, `{"x":1}`)
			b := r.post("/v1/orders", "cle-commande-1", tok, `{"x":1}`)
			if a.Code != 201 || b.Code != 201 || a.Body.String() != b.Body.String() {
				t.Fatalf("rejeu différent : %d %q / %d %q", a.Code, a.Body, b.Code, b.Body)
			}
			if r.calls.Load() != 1 {
				t.Fatalf("le handler a été appelé %d fois (attendu 1)", r.calls.Load())
			}
			if b.Header().Get("Idempotent-Replayed") != "true" || a.Header().Get("Idempotent-Replayed") != "" {
				t.Fatal("en-tête Idempotent-Replayed incorrect")
			}
			if b.Header().Get("Content-Type") != "application/json" {
				t.Fatal("Content-Type non rejoué")
			}
		})
	}
}

func TestIdempotence_CorpsDifferentEtPortee(t *testing.T) {
	for name, st := range stores(t) {
		t.Run(name, func(t *testing.T) {
			r := newIdemRig(t, st)
			u1, u2 := r.token(t, jwtx.AudCustomer, uuid.NewString()), r.token(t, jwtx.AudCustomer, uuid.NewString())
			r.post("/v1/orders", "cle-commande-1", u1, `{"x":1}`)
			c := r.post("/v1/orders", "cle-commande-1", u1, `{"x":2}`)
			if c.Code != 409 || !strings.Contains(c.Body.String(), "IDEMPOTENCY_KEY_REUSED") {
				t.Fatalf("corps différent : %d %s", c.Code, c.Body)
			}
			// un autre utilisateur, un autre chemin : clés indépendantes (aucune fuite de réponse entre acteurs)
			if d := r.post("/v1/orders", "cle-commande-1", u2, `{"x":1}`); d.Header().Get("Idempotent-Replayed") != "" {
				t.Fatal("la réponse d'un autre acteur a été rejouée")
			}
			if e := r.post("/v1/payments", "cle-commande-1", u1, `{"x":1}`); e.Header().Get("Idempotent-Replayed") != "" {
				t.Fatal("la réponse d'une autre route a été rejouée")
			}
			if r.calls.Load() != 3 {
				t.Fatalf("appels = %d, attendu 3", r.calls.Load())
			}
		})
	}
}

func TestIdempotence_ErreurLibereLaCle(t *testing.T) {
	for name, st := range stores(t) {
		t.Run(name, func(t *testing.T) {
			r := newIdemRig(t, st)
			tok := r.token(t, jwtx.AudStaff, uuid.NewString())
			r.code.Store(500)
			if w := r.post("/v1/admin/orders/abc/refunds", "cle-refund-01", tok, `{}`); w.Code != 500 {
				t.Fatal(w.Code)
			}
			r.code.Store(201)
			w := r.post("/v1/admin/orders/abc/refunds", "cle-refund-01", tok, `{}`)
			if w.Code != 201 || w.Header().Get("Idempotent-Replayed") != "" || r.calls.Load() != 2 {
				t.Fatalf("après un échec, la clé doit être réutilisable : %d replay=%q appels=%d", w.Code, w.Header().Get("Idempotent-Replayed"), r.calls.Load())
			}
		})
	}
}

// 20 requêtes simultanées avec la même clé : le handler (qui « débite ») ne s'exécute qu'UNE fois.
func TestIdempotence_ConcurrenceUneSeuleExecution(t *testing.T) {
	for name, st := range stores(t) {
		t.Run(name, func(t *testing.T) {
			r := newIdemRig(t, st)
			tok := r.token(t, jwtx.AudCustomer, uuid.NewString())
			var wg sync.WaitGroup
			var ok, conflict atomic.Int32
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					switch w := r.post("/v1/payments", "cle-paiement-1", tok, `{"amount":100}`); w.Code {
					case 201:
						ok.Add(1)
					case 409:
						conflict.Add(1)
					default:
						t.Errorf("statut inattendu %d", w.Code)
					}
				}()
			}
			wg.Wait()
			if r.calls.Load() != 1 {
				t.Fatalf("handler exécuté %d fois, attendu 1", r.calls.Load())
			}
			if ok.Load()+conflict.Load() != 20 || ok.Load() < 1 {
				t.Fatalf("ok=%d conflit=%d", ok.Load(), conflict.Load())
			}
		})
	}
}

func TestIdempotence_PasDeCleOuPasDeJetonOuHorsPerimetre(t *testing.T) {
	r := newIdemRig(t, NewMemoryIdempotency(nil))
	tok := r.token(t, jwtx.AudCustomer, uuid.NewString())
	r.post("/v1/orders", "", tok, `{}`)              // sans clé : le serveur généré refuse (400)
	r.post("/v1/orders", "cle-commande-1", "", `{}`) // sans jeton : refus (401) par la sécurité
	r.post("/v1/orders", "court", tok, `{}`)         // clé < 8 car. : refus par le contrat
	r.post("/v1/orders", "cle-commande-1", "pas.un.jeton", `{}`)
	r.post("/v1/cart", "cle-commande-1", tok, `{}`) // route non déclarée : intacte
	if r.calls.Load() != 5 {
		t.Fatalf("ces cas doivent traverser sans mémorisation : %d appels", r.calls.Load())
	}
	// rien n'a été mémorisé : la même clé valide s'exécute ensuite normalement
	if w := r.post("/v1/orders", "cle-commande-1", tok, `{}`); w.Header().Get("Idempotent-Replayed") != "" {
		t.Fatal("rien ne devait être mémorisé")
	}
}

type brokenStore struct{ IdempotencyStore }

func (brokenStore) Begin(context.Context, string, string, string) (bool, IdemRecord, error) {
	return false, IdemRecord{}, io.ErrClosedPipe
}

// Magasin HS : on refuse (503), on n'exécute JAMAIS un paiement sans garantie d'idempotence.
func TestIdempotence_MagasinIndisponibleRefuse(t *testing.T) {
	r := newIdemRig(t, brokenStore{})
	w := r.post("/v1/payments", "cle-paiement-1", r.token(t, jwtx.AudCustomer, uuid.NewString()), `{}`)
	if w.Code != 503 || r.calls.Load() != 0 {
		t.Fatalf("code=%d appels=%d", w.Code, r.calls.Load())
	}
}

// Anti-dérive : la table des routes idempotentes doit être EXACTEMENT celle que le contrat déclare.
func TestIdempotence_TableConformeAuContrat(t *testing.T) {
	raw, err := os.ReadFile("../../../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cur string
	var got []string
	pathRe, methRe := regexp.MustCompile(`^  (/\S+):\s*$`), regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
	var meth string
	for _, l := range strings.Split(string(raw), "\n") {
		if m := pathRe.FindStringSubmatch(l); m != nil {
			cur, meth = m[1], ""
		} else if m := methRe.FindStringSubmatch(l); m != nil {
			meth = strings.ToUpper(m[1])
		} else if strings.Contains(l, "parameters/IdempotencyKey'") {
			got = append(got, meth+" /v1"+cur)
		}
	}
	var want []string
	for _, r := range idempotentRoutes {
		p := strings.TrimSuffix(strings.TrimPrefix(r.Re.String(), "^"), "$")
		p = strings.ReplaceAll(p, "[^/]+", "{id}")
		want = append(want, r.Method+" "+p)
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("table ≠ contrat\ncontrat : %v\ntable   : %v", got, want)
	}
}
