// Commande identity : service gRPC des comptes, sessions et jetons.
//
// Stockage (STORE) : "postgres" (défaut hors développement) applique les migrations au démarrage, écrit les événements
// dans l'outbox (même transaction que les données) et les relaie vers NATS JetStream ; "memory" (ENV=dev seulement)
// ne sert qu'au développement sans base.
//
// Garde-fous : en production (ENV=prod) le service REFUSE de démarrer avec le stockage en mémoire ou avec des clés
// de signature éphémères.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/theb20/FILTER/backend/pkg/events"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/pkg/logx"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/breach"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/grpcserver"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/memory"
	"github.com/theb20/FILTER/backend/services/identity/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	log := logx.New(os.Stdout, "identity", slog.LevelInfo)
	if err := run(log); err != nil {
		log.Error("arrêt", "error", err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	env := getenv("ENV", "dev")
	addr := getenv("IDENTITY_GRPC_ADDR", ":9001")

	keys, err := loadKeys(env, log)
	if err != nil {
		return err
	}
	breachChecker, err := loadBreach(env, log)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc := application.Service{
		Hasher: domain.Hasher{P: domain.DefaultArgon2id()}, Breach: breachChecker,
		Keys: keys, Settings: application.StaticSettings(domain.DefaultSettings()),
	}
	switch mode := getenv("STORE", defaultStore(env)); mode {
	case "memory":
		if env != "dev" {
			return errors.New("STORE=memory est interdit hors développement (ENV=dev)")
		}
		log.Warn("stockage EN MÉMOIRE : les comptes disparaissent à l'arrêt (développement uniquement)")
		st, ev := memory.New(), &events.Memory{}
		st.Events = ev
		svc.Users, svc.Sessions, svc.Tokens, svc.Failures, svc.UoW, svc.Events = st, st, st, st, st, ev
	case "postgres":
		url := os.Getenv("DATABASE_URL")
		if url == "" {
			return errors.New("DATABASE_URL est obligatoire avec STORE=postgres")
		}
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			return errors.New("NATS_URL est obligatoire avec STORE=postgres (relais de l'outbox)")
		}
		mctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		if err := postgres.Migrate(mctx, url); err != nil {
			cancel()
			return fmt.Errorf("migrations : %w", err)
		}
		st, err := postgres.New(mctx, url)
		if err != nil {
			cancel()
			return err
		}
		defer st.Close()
		bus, err := natsbus.Connect(mctx, natsURL, getenv("NATS_STREAM", "IDENTITY"), "identity.>")
		cancel()
		if err != nil {
			return fmt.Errorf("nats : %w", err)
		}
		defer bus.Close()
		go (&outbox.Relay{Pool: st.Pool, Bus: bus, Log: log}).Run(ctx)
		svc.Users, svc.Sessions, svc.Tokens, svc.Failures, svc.UoW, svc.Events = st, st, st, st, st, &outbox.Publisher{}
		log.Info("PostgreSQL et NATS prêts (outbox active)")
	default:
		return fmt.Errorf("STORE=%q inconnu (postgres, memory)", mode)
	}
	service := application.New(svc)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(srv, &grpcserver.Server{Svc: service})
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	go func() {
		<-ctx.Done()
		log.Info("arrêt gracieux")
		done := make(chan struct{})
		go func() { srv.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			srv.Stop()
		}
	}()
	log.Info("identity démarré", "addr", addr, "env", env)
	return srv.Serve(lis)
}

// loadKeys charge une clé Ed25519 par audience (JWT_KEY_<AUD>_FILE) ; en dev seulement, une clé éphémère.
func loadKeys(env string, log *slog.Logger) (map[string]*jwtx.KeySet, error) {
	keys := map[string]*jwtx.KeySet{}
	for aud, envVar := range map[string]string{jwtx.AudCustomer: "JWT_KEY_CUSTOMER_FILE", jwtx.AudStaff: "JWT_KEY_STAFF_FILE", jwtx.AudStaffChallenge: "JWT_KEY_STAFF_CHALLENGE_FILE"} {
		if path := os.Getenv(envVar); path != "" {
			pem, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", envVar, err)
			}
			k, err := jwtx.LoadKeySetPEM(pem)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", envVar, err)
			}
			keys[aud] = k
			continue
		}
		if env != "dev" {
			return nil, fmt.Errorf("%s est obligatoire hors développement (clés éphémères interdites)", envVar)
		}
		k, err := jwtx.NewKeySet()
		if err != nil {
			return nil, err
		}
		log.Warn("clé de signature ÉPHÉMÈRE (les jetons deviennent invalides au redémarrage)", "audience", aud)
		keys[aud] = k
	}
	return keys, nil
}

func defaultStore(env string) string {
	if env == "dev" {
		return "memory"
	}
	return "postgres"
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// loadBreach choisit la source du contrôle des mots de passe compromis (BREACH_CHECK) :
//
//	pwned    (défaut) service par préfixe de condensat (k-anonymat), délai court, repli sur la liste locale
//	local    liste locale seulement (aucun appel externe)
//	disabled aucun contrôle — REFUSÉ hors développement
func loadBreach(env string, log *slog.Logger) (application.BreachChecker, error) {
	mode := getenv("BREACH_CHECK", "pwned")
	timeout := 1500 * time.Millisecond
	if v := os.Getenv("BREACH_TIMEOUT_MS"); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil || ms < 50 {
			return nil, errors.New("BREACH_TIMEOUT_MS doit être un entier ≥ 50")
		}
		timeout = time.Duration(ms) * time.Millisecond
	}
	switch mode {
	case "pwned":
		return breach.NewChecker(os.Getenv("BREACH_BASE_URL"), timeout, breach.NewLocal(), log), nil
	case "local":
		return localOnly{breach.NewLocal()}, nil
	case "disabled":
		if env != "dev" {
			return nil, errors.New("BREACH_CHECK=disabled est interdit hors développement")
		}
		log.Warn("refus des mots de passe compromis DÉSACTIVÉ (développement)")
		return application.DisabledBreachChecker{}, nil
	default:
		return nil, fmt.Errorf("BREACH_CHECK=%q inconnu (pwned, local, disabled)", mode)
	}
}

type localOnly struct{ l *breach.Local }

func (c localOnly) Compromised(_ context.Context, p string) (bool, error) {
	return c.l.Contains(p), nil
}
