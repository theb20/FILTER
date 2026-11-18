// Commande gateway : point d'entrée HTTP unique (REST /v1). Le serveur est GÉNÉRÉ depuis api/openapi.yaml (ogen).
//
// Garde-fous : en production (ENV=prod), le démarrage est refusé si la liste blanche d'origines est vide ou contient
// "*", ou si les cookies ne sont pas `Secure`. La liaison au service identity est en clair (réseau privé Docker) :
// le mTLS entre services est prévu pour la mise en production.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	catalogadminv1 "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	contentadminv1 "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	financeadminv1 "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/pkg/logx"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	log := logx.New(os.Stdout, "gateway", slog.LevelInfo)
	if err := run(log); err != nil {
		log.Error("arrêt", "error", err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	// Reconnexion rapide : au démarrage conjoint des services, identity peut arriver quelques secondes après le gateway.
	conn, err := grpc.NewClient(getenv("IDENTITY_ADDR", "127.0.0.1:9001"), grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 200 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 3 * time.Second}))
	if err != nil {
		return err
	}
	defer conn.Close()
	id := identityv1.NewIdentityServiceClient(conn)
	health := healthpb.NewHealthClient(conn)

	// Service sourcing (sources de recherche) : facultatif. Sans SOURCING_ADDR, ses routes répondent 501.
	var sourcing sourcingv1.SourcingServiceClient
	probes := []server.HealthProbe{{Name: "identity", Kind: "service", Check: func(ctx context.Context) error {
		r, err := health.Check(ctx, &healthpb.HealthCheckRequest{})
		if err == nil && r.GetStatus() != healthpb.HealthCheckResponse_SERVING {
			err = errors.New("ne sert pas")
		}
		return err
	}}}
	if addr := os.Getenv("SOURCING_ADDR"); addr != "" {
		sconn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 200 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 3 * time.Second}))
		if err != nil {
			return err
		}
		defer sconn.Close()
		sourcing = sourcingv1.NewSourcingServiceClient(sconn)
		sh := healthpb.NewHealthClient(sconn)
		probes = append(probes, server.HealthProbe{Name: "sourcing", Kind: "service", Check: func(ctx context.Context) error {
			r, err := sh.Check(ctx, &healthpb.HealthCheckRequest{})
			if err == nil && r.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				err = errors.New("ne sert pas")
			}
			return err
		}})
	}

	// Service catalog (taxonomie) : facultatif. Sans CATALOG_ADDR, ses routes répondent 501.
	var catalog catalogadminv1.CatalogAdminServiceClient
	if addr := os.Getenv("CATALOG_ADDR"); addr != "" {
		cconn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 200 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 3 * time.Second}))
		if err != nil {
			return err
		}
		defer cconn.Close()
		catalog = catalogadminv1.NewCatalogAdminServiceClient(cconn)
		ch := healthpb.NewHealthClient(cconn)
		probes = append(probes, server.HealthProbe{Name: "catalog", Kind: "service", Check: func(ctx context.Context) error {
			r, err := ch.Check(ctx, &healthpb.HealthCheckRequest{})
			if err == nil && r.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				err = errors.New("ne sert pas")
			}
			return err
		}})
	}

	// Service content (contenu du site) : facultatif. Sans CONTENT_ADDR, ses routes répondent 501.
	var content contentadminv1.ContentAdminServiceClient
	if addr := os.Getenv("CONTENT_ADDR"); addr != "" {
		kconn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 200 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 3 * time.Second}))
		if err != nil {
			return err
		}
		defer kconn.Close()
		content = contentadminv1.NewContentAdminServiceClient(kconn)
		kh := healthpb.NewHealthClient(kconn)
		probes = append(probes, server.HealthProbe{Name: "content", Kind: "service", Check: func(ctx context.Context) error {
			r, err := kh.Check(ctx, &healthpb.HealthCheckRequest{})
			if err == nil && r.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				err = errors.New("ne sert pas")
			}
			return err
		}})
	}

	// Service finance (comptabilité) : facultatif. Sans FINANCE_ADDR, ses routes répondent 501.
	var financeCli financeadminv1.FinanceAdminServiceClient
	if addr := os.Getenv("FINANCE_ADDR"); addr != "" {
		fconn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 200 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 3 * time.Second}))
		if err != nil {
			return err
		}
		defer fconn.Close()
		financeCli = financeadminv1.NewFinanceAdminServiceClient(fconn)
		fh := healthpb.NewHealthClient(fconn)
		probes = append(probes, server.HealthProbe{Name: "finance", Kind: "service", Check: func(ctx context.Context) error {
			r, err := fh.Check(ctx, &healthpb.HealthCheckRequest{})
			if err == nil && r.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				err = errors.New("ne sert pas")
			}
			return err
		}})
	}

	// Limitation de débit : Redis (partagé entre instances) ; obligatoire hors développement.
	limiter, idem, redisProbe, err := buildLimiter(cfg, log)
	if err != nil {
		return err
	}
	if redisProbe != nil {
		probes = append(probes, *redisProbe)
	}

	keys := server.NewKeyCache(jwksFromIdentity{id}, cfg.ClockLeeway, cfg.JWKSMinRefresh, nil)
	h, err := server.New(server.Deps{
		Config: cfg, Log: log, Identity: id, Sourcing: sourcing, Catalog: catalog, Finance: financeCli, Content: content, Probes: probes, Keys: keys, Limiter: limiter, Idempotency: idem,
		Ready: func(ctx context.Context) error {
			r, err := health.Check(ctx, &healthpb.HealthCheckRequest{})
			if err != nil {
				return err
			}
			if r.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				return errors.New("identity ne sert pas")
			}
			if redisProbe != nil { // G14 : /readyz = Redis + canal gRPC vers identity
				if err := redisProbe.Check(ctx); err != nil {
					return fmt.Errorf("redis : %w", err)
				}
			}
			return nil
		},
	})
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr: getenv("GATEWAY_ADDR", ":8080"), Handler: h,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		log.Info("arrêt gracieux")
		sctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	log.Info("gateway démarré", "addr", srv.Addr, "env", cfg.Env, "origins", cfg.AllowedOrigins)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func loadConfig() (server.Config, error) {
	cfg := server.DefaultConfig()
	cfg.Env = getenv("ENV", "dev")
	cfg.TrustProxy = getenv("TRUST_PROXY", "false") == "true"
	cfg.UploadDir = getenv("UPLOAD_DIR", cfg.UploadDir)
	for _, o := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}
	if d := strings.TrimPrefix(strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")), "."); d != "" {
		if strings.ContainsAny(d, "/:@ ") || !strings.Contains(d, ".") {
			return cfg, fmt.Errorf("COOKIE_DOMAIN %q invalide : domaine seul, sans schéma ni port (ex. exemple.ci)", d)
		}
		cfg.CookieDomain = d
	}
	for env, dst := range map[string]*int{"ANON_PER_MINUTE": &cfg.AnonPerMinute, "USER_PER_MINUTE": &cfg.UserPerMinute, "LOGIN_PER_MINUTE": &cfg.LoginPerMinute, "REGISTER_PER_HOUR": &cfg.RegisterPerHour} {
		if v := os.Getenv(env); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return cfg, fmt.Errorf("%s doit être un entier ≥ 1", env)
			}
			*dst = n
		}
	}
	if cfg.Env == "dev" {
		cfg.CookieSecure = getenv("COOKIE_SECURE", "false") == "true" // http://localhost : Secure refusé par certains navigateurs
		cfg.HSTS = false
		return cfg, nil
	}
	if len(cfg.AllowedOrigins) == 0 {
		return cfg, errors.New("ALLOWED_ORIGINS est obligatoire hors développement")
	}
	for _, o := range cfg.AllowedOrigins {
		if o == "*" || !strings.HasPrefix(o, "https://") {
			return cfg, fmt.Errorf("origine %q refusée : HTTPS exigé, jamais de joker", o)
		}
	}
	return cfg, nil
}

type jwksFromIdentity struct {
	c identityv1.IdentityServiceClient
}

func (j jwksFromIdentity) Fetch(ctx context.Context, aud string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	r, err := j.c.GetJwks(ctx, &identityv1.GetJwksRequest{Audience: aud})
	if err != nil {
		return nil, err
	}
	return []byte(r.GetJwksJson()), nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// buildLimiter choisit la limitation de débit. REDIS_URL (ex. redis://:mdp@redis:6379/0) active la version partagée.
// Sans REDIS_URL : refusé hors développement (la limite par instance se contourne en répartissant les requêtes sur
// N instances) ; en développement, repli en mémoire avec avertissement.
func buildLimiter(cfg server.Config, log *slog.Logger) (server.Limiter, server.IdempotencyStore, *server.HealthProbe, error) {
	url := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if url == "" {
		if cfg.Env != "dev" {
			return nil, nil, nil, errors.New("REDIS_URL est obligatoire hors développement (limitation de débit partagée entre instances)")
		}
		log.Warn("REDIS_URL absent : limitation de débit EN MÉMOIRE (une seule instance, développement uniquement)")
		return server.NewMemoryLimiter(), server.NewMemoryIdempotency(nil), nil, nil
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("REDIS_URL invalide : %w", err)
	}
	l := server.NewRedisLimiter(redis.NewClient(opt), log)
	pctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := l.Ping(pctx); err != nil {
		if cfg.Env != "dev" {
			return nil, nil, nil, fmt.Errorf("Redis injoignable au démarrage : %w", err)
		}
		log.Warn("Redis injoignable au démarrage (développement : repli local actif)", "err", err.Error())
	}
	return l, server.NewRedisIdempotency(l.Client()), &server.HealthProbe{Name: "redis", Kind: "broker", Check: l.Ping}, nil
}
