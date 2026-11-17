// Package testsupport fournit PostgreSQL et NATS réels aux tests d'intégration.
//
// Ordre : TEST_DATABASE_URL / TEST_NATS_URL s'ils sont définis (serveurs déjà lancés), sinon conteneurs
// testcontainers-go (Docker requis ; c'est le chemin de la CI). Sans l'un ni l'autre, le test est IGNORÉ, sauf si
// REQUIRE_INTEGRATION=1 : il ÉCHOUE alors (CI : un test ignoré ne doit jamais passer pour un test réussi).
package testsupport

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcnats "github.com/testcontainers/testcontainers-go/modules/nats"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	pgOnce, natsOnce sync.Once
	pgAdmin, natsURL string
	pgErr, natsErr   error
	dbSeq            atomic.Int64
)

func unavailable(t testing.TB, what string, err error) {
	t.Helper()
	if os.Getenv("REQUIRE_INTEGRATION") == "1" {
		t.Fatalf("%s indisponible (REQUIRE_INTEGRATION=1) : %v", what, err)
	}
	t.Skipf("%s indisponible (définir TEST_*_URL ou lancer Docker) : %v", what, err)
}

// startContainer démarre un conteneur pour toute la durée du processus de test (arrêté par Ryuk à la sortie).
func postgresAdminURL() (string, error) {
	pgOnce.Do(func() {
		if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
			pgAdmin = u
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		c, err := tcpg.Run(ctx, "postgres:17-alpine", tcpg.WithDatabase("postgres"), tcpg.WithUsername("test"), tcpg.WithPassword("test"),
			testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90*time.Second)))
		if err != nil {
			pgErr = err
			return
		}
		pgAdmin, pgErr = c.ConnectionString(ctx, "sslmode=disable")
	})
	return pgAdmin, pgErr
}

// Postgres renvoie l'URL d'une base NEUVE et vide (une par test : isolation totale), supprimée en fin de test.
func Postgres(t testing.TB) string {
	t.Helper()
	admin, err := postgresAdminURL()
	if err != nil {
		unavailable(t, "PostgreSQL", err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("t_%d_%d", os.Getpid(), dbSeq.Add(1))
	if _, err := pool.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	cfg := pool.Config().ConnConfig.Copy()
	pool.Close()
	cfg.Database = name
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", cfg.User, cfg.Password, cfg.Host, cfg.Port, name)
	t.Cleanup(func() {
		p, err := pgxpool.New(ctx, admin)
		if err != nil {
			return
		}
		defer p.Close()
		_, _ = p.Exec(ctx, `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
	})
	return url
}

// NATS renvoie l'URL d'un serveur NATS avec JetStream.
func NATS(t testing.TB) string {
	t.Helper()
	natsOnce.Do(func() {
		if u := os.Getenv("TEST_NATS_URL"); u != "" {
			natsURL = u
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		c, err := tcnats.Run(ctx, "nats:2.10-alpine", testcontainers.WithCmd("-js"))
		if err != nil {
			natsErr = err
			return
		}
		natsURL, natsErr = c.ConnectionString(ctx)
	})
	if natsErr != nil {
		unavailable(t, "NATS", natsErr)
	}
	return natsURL
}
