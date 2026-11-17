package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/problem"
)

// Chemins qui LISENT le cookie de rafraîchissement : protégés contre le CSRF (G16).
var csrfPaths = map[string]bool{"/v1/auth/refresh": true, "/v1/auth/logout": true}

func (s *Server) chain(next http.Handler) http.Handler {
	h := next
	h = s.csrf(h)
	h = s.rateLimit(h)
	h = s.logging(h)
	h = s.cors(h)
	h = s.securityHeaders(h)
	h = s.requestInfo(h)
	h = s.recoverPanics(h)
	return h
}

// ── contexte de requête, trace, IP ──────────────────────────────────────────

func (s *Server) requestInfo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace := traceID(r.Header.Get("traceparent"))
		w.Header().Set("X-Request-Id", trace)
		r.Body = http.MaxBytesReader(w, r.Body, s.bodyLimit(r)) // G10 : corps JSON ≤ 1 Mio, sauf l'envoi d'une pièce justificative
		info := reqInfo{TraceID: trace, IP: s.clientIP(r), UA: truncate(r.UserAgent(), 300), Locale: locale(r, s.cfg.DefaultLocale)}
		next.ServeHTTP(w, r.WithContext(withInfo(r.Context(), info)))
	})
}

// traceID reprend l'identifiant du `traceparent` W3C valide, sinon en génère un.
func traceID(tp string) string {
	if parts := strings.Split(tp, "-"); len(parts) == 4 && len(parts[1]) == 32 {
		return parts[1]
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func locale(r *http.Request, def string) string {
	al := r.Header.Get("Accept-Language")
	if al == "" {
		return def
	}
	tag := strings.ToLower(strings.TrimSpace(strings.SplitN(strings.SplitN(al, ",", 2)[0], ";", 2)[0]))
	if len(tag) >= 2 {
		return tag[:2]
	}
	return def
}

func truncate(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}

// ── panics, journalisation, en-têtes ────────────────────────────────────────

func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "trace_id", infoFrom(r.Context()).TraceID, "panic", "récupérée")
				problem.Write(w, problem.New(http.StatusInternalServerError, apperr.CodeInternal, "Une erreur est survenue.", infoFrom(r.Context()).TraceID))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(c int) { r.status = c; r.ResponseWriter.WriteHeader(c) }

func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		// jamais de corps, de jeton ni de cookie : méthode, chemin (sans requête), statut, durée, trace
		s.log.LogAttrs(r.Context(), slog.LevelInfo, "requête", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)), slog.String("trace_id", infoFrom(r.Context()).TraceID), slog.String("ip", infoFrom(r.Context()).IP))
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		if s.cfg.HSTS {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		if strings.HasPrefix(r.URL.Path, "/v1/auth") || strings.HasPrefix(r.URL.Path, "/v1/me") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// ── CORS avec identifiants (G6) ─────────────────────────────────────────────

func (s *Server) originAllowed(origin string) bool {
	return origin != "" && slices.Contains(s.cfg.AllowedOrigins, origin)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.originAllowed(origin) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin) // l'origine exacte, JAMAIS "*" avec identifiants
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Expose-Headers", "ETag, RateLimit-Limit, RateLimit-Remaining, RateLimit-Reset, Retry-After, X-Request-Id")
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if s.originAllowed(origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept-Language, Idempotency-Key, If-Match, X-Requested-With")
				h.Set("Access-Control-Max-Age", "600")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── CSRF (G16) : défense en couches, sans jeton lisible par JavaScript ───────

func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && csrfPaths[r.URL.Path] {
			if reason := s.csrfViolation(r); reason != "" {
				s.log.Warn("CSRF refusé", "reason", reason, "path", r.URL.Path, "trace_id", infoFrom(r.Context()).TraceID)
				problem.Write(w, problem.New(http.StatusForbidden, apperr.CodeCSRFFailed, "Requête refusée.", infoFrom(r.Context()).TraceID))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// csrfViolation renvoie la raison du refus, ou "" si la requête est légitime. Aucune couche ne suffit seule.
func (s *Server) csrfViolation(r *http.Request) string {
	// (2) Origin OBLIGATOIRE et dans la liste blanche
	if !s.originAllowed(r.Header.Get("Origin")) {
		return "origin"
	}
	// Sec-Fetch-Site, quand le navigateur l'envoie : même origine ou même site seulement
	if sfs := r.Header.Get("Sec-Fetch-Site"); sfs != "" && sfs != "same-origin" && sfs != "same-site" {
		return "sec-fetch-site"
	}
	// (3) en-tête personnalisé : impose un contrôle CORS préalable qu'un site tiers ne peut pas obtenir
	if r.Header.Get("X-Requested-With") != "ff-web" {
		return "x-requested-with"
	}
	// (4) corps vide ou JSON uniquement (pas de formulaire HTML classique)
	if r.ContentLength > 0 && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return "content-type"
	}
	return ""
}

// ── limitation de débit (G5) ────────────────────────────────────────────────

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/v1/healthz" || r.URL.Path == "/v1/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		ip := infoFrom(r.Context()).IP
		now := s.now()
		type rule struct {
			key    string
			limit  int
			window time.Duration
		}
		rules := []rule{{"ip:" + ip, s.cfg.AnonPerMinute, time.Minute}}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/login":
			rules = append(rules, rule{"login:" + ip, s.cfg.LoginPerMinute, time.Minute})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/register":
			rules = append(rules, rule{"register:" + ip, s.cfg.RegisterPerHour, time.Hour})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/password/forgot":
			rules = append(rules, rule{"forgot:" + ip, 5, time.Hour}) // 5 demandes de code par heure et par adresse IP
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/password/verify-code":
			rules = append(rules, rule{"verifycode:" + ip, 20, time.Hour})
		case r.Method == http.MethodPost && (r.URL.Path == "/v1/contact" || r.URL.Path == "/v1/newsletter/subscribe" || r.URL.Path == "/v1/quote-requests"):
			rules = append(rules, rule{"forms:" + ip, 10, time.Hour}) // formulaires publics : 10 envois par heure et par adresse
		}
		for _, ru := range rules {
			ok, remaining, retry := s.limiter.Allow(ru.key, ru.limit, ru.window, now)
			w.Header().Set("RateLimit-Limit", strconv.Itoa(ru.limit))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
			if !ok {
				secs := int((retry + time.Second - 1) / time.Second)
				w.Header().Set("RateLimit-Reset", strconv.Itoa(secs))
				p := problem.New(http.StatusTooManyRequests, apperr.CodeRateLimited, "Trop de requêtes. Réessayez plus tard.", infoFrom(r.Context()).TraceID).WithRetryAfter(secs)
				problem.Write(w, p)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// attachmentBodyLimit : une pièce de 5 Mo en base64 (≈ 6,7 Mo) ; réservé à cette seule route d'envoi, réservée au personnel.
const attachmentBodyLimit = 8 << 20

func (s *Server) bodyLimit(r *http.Request) int64 {
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/admin/finance/operations/") && strings.HasSuffix(r.URL.Path, "/attachments") && attachmentBodyLimit > s.cfg.MaxBodyBytes {
		return attachmentBodyLimit
	}
	return s.cfg.MaxBodyBytes
}
