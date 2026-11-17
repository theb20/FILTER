// Commande sourcing : service gRPC des sources externes (Brave Search API au lancement), du budget et de la santé.
//
// Garde-fous : le connecteur de test (test://) est refusé hors ENV=dev ; les clés d'API ne sont lues que dans l'environnement
// (SECRET_REF de la source = nom de la variable), jamais en base.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/pkg/logx"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/connectors"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/grpcserver"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	log := logx.New(os.Stdout, "sourcing", slog.LevelInfo)
	if err := run(log); err != nil {
		log.Error("arrêt", "error", err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	env := getenv("ENV", "dev")
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL est obligatoire")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	mctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := postgres.Migrate(mctx, url); err != nil {
		return err
	}
	st, err := postgres.New(mctx, url)
	if err != nil {
		return err
	}
	defer st.Close()

	fac := &connectors.Factory{Env: os.Getenv, AllowTest: env == "dev" && os.Getenv("ALLOW_TEST_SOURCES") == "1"}
	// Audit : événements `audit.recorded.v1` écrits dans l'outbox (même transaction que l'action) puis relayés vers NATS ;
	// identity les inscrit dans son journal d'audit append-only.
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		return errors.New("NATS_URL est obligatoire (audit des écritures)")
	}
	bus, err := natsbus.Connect(mctx, natsURL, getenv("NATS_AUDIT_STREAM", "AUDIT"), "audit.>")
	if err != nil {
		return err
	}
	defer bus.Close()
	go (&outbox.Relay{Pool: st.Pool, Bus: bus, Log: log}).Run(ctx)
	svc := application.New(application.Config{Store: st, Connectors: fac, Events: &outbox.Publisher{}})
	go svc.RunMaintenance(ctx, func(msg string, kv ...any) { log.Warn(msg, kv...) })

	lis, err := net.Listen("tcp", getenv("SOURCING_GRPC_ADDR", ":9002"))
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	sourcingv1.RegisterSourcingServiceServer(srv, &grpcserver.Server{Svc: svc})
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	go func() {
		<-ctx.Done()
		done := make(chan struct{})
		go func() { srv.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			srv.Stop()
		}
	}()
	log.Info("sourcing démarré", "addr", lis.Addr().String(), "env", env)
	return srv.Serve(lis)
}
