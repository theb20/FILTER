// Commande finance : service gRPC de la comptabilité (grand livre), des opérations et de la trésorerie.
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

	financev1 "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/pkg/logx"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/services/finance/internal/adapters/grpcserver"
	"github.com/theb20/FILTER/backend/services/finance/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/finance/internal/adapters/secretbox"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
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
	log := logx.New(os.Stdout, "finance", slog.LevelInfo)
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
	cfg := application.Config{Store: st, Events: &outbox.Publisher{}}
	// Chiffrement des numéros de compte bancaire : clé dans un FICHIER (0600). Sans clé, aucun numéro n'est enregistré (le reste fonctionne).
	if kf := os.Getenv("FINANCE_BANK_KEY_FILE"); kf != "" {
		box, err := secretbox.FromFile(kf)
		if err != nil {
			return err
		}
		cfg.Box = box
	} else {
		log.Warn("FINANCE_BANK_KEY_FILE absente : les numéros de compte bancaire ne seront pas enregistrés")
	}
	svc := application.New(cfg)

	lis, err := net.Listen("tcp", getenv("FINANCE_GRPC_ADDR", ":9005"))
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	financev1.RegisterFinanceAdminServiceServer(srv, &grpcserver.Server{Svc: svc})
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
	log.Info("finance démarré", "addr", lis.Addr().String(), "env", env)
	return srv.Serve(lis)
}
