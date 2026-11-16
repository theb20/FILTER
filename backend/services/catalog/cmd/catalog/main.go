// Commande catalog : service gRPC du catalogue (taxonomie : catégories, attributs, marques, secteurs ; produits ensuite).
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

	catalogv1 "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	"github.com/theb20/FILTER/backend/pkg/logx"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/services/catalog/internal/adapters/grpcserver"
	"github.com/theb20/FILTER/backend/services/catalog/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/catalog/internal/application"
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
	log := logx.New(os.Stdout, "catalog", slog.LevelInfo)
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
	svc := application.New(application.Config{Store: st, Events: &outbox.Publisher{}})

	lis, err := net.Listen("tcp", getenv("CATALOG_GRPC_ADDR", ":9003"))
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	catalogv1.RegisterCatalogAdminServiceServer(srv, &grpcserver.Server{Svc: svc})
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
	log.Info("catalog démarré", "addr", lis.Addr().String(), "env", env)
	return srv.Serve(lis)
}
