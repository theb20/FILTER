package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	catalogadminv1 "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Server est le gateway HTTP.
type Server struct {
	cfg     Config
	log     *slog.Logger
	limiter Limiter
	now     func() time.Time
	ready   func(context.Context) error
	probes  []HealthProbe
}

// Deps regroupe les dépendances (ports) du gateway.
type Deps struct {
	Config   Config
	Log      *slog.Logger
	Identity identityv1.IdentityServiceClient
	Sourcing sourcingv1.SourcingServiceClient         // facultatif
	Catalog  catalogadminv1.CatalogAdminServiceClient // facultatif
	Probes   []HealthProbe                            // sondes de « Santé des services »
	Keys     *KeyCache
	Limiter  Limiter
	Ready    func(context.Context) error
	Now      func() time.Time
}

// New assemble le gateway : serveur ogen GÉNÉRÉ + middlewares.
func New(d Deps) (http.Handler, error) {
	s := &Server{cfg: d.Config, log: d.Log, limiter: d.Limiter, now: d.Now, ready: d.Ready, probes: d.Probes}
	if s.now == nil {
		s.now = time.Now
	}
	if s.limiter == nil {
		s.limiter = NewMemoryLimiter()
	}
	if s.ready == nil {
		s.ready = func(context.Context) error { return nil }
	}
	h := &handler{srv: s, id: d.Identity, src: d.Sourcing, cat: d.Catalog}
	gen, err := api.NewServer(h, &security{keys: d.Keys},
		api.WithPathPrefix("/v1"),
		api.WithErrorHandler(s.errorHandler),
		api.WithNotFound(func(w http.ResponseWriter, r *http.Request) {
			s.errorHandler(r.Context(), w, r, apperr.NotFound("ressource introuvable"))
		}),
		api.WithMethodNotAllowed(func(w http.ResponseWriter, r *http.Request, _ string) {
			s.errorHandler(r.Context(), w, r, apperr.NotFound("ressource introuvable"))
		}),
	)
	if err != nil {
		return nil, err
	}
	return s.chain(aliasProbes(gen)), nil
}

// aliasProbes expose /healthz et /readyz à la racine (orchestrateurs) en plus de /v1/healthz et /v1/readyz.
func aliasProbes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/v1" + r.URL.Path
			r = r2
		}
		next.ServeHTTP(w, r)
	})
}
