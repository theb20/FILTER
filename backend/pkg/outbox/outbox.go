// Package outbox implémente le patron « outbox transactionnelle » réutilisable par tous les services :
//
//   - Publisher écrit l'événement dans la table `outbox` DANS LA TRANSACTION EN COURS (refuse sans transaction) ;
//   - Relay relaie les lignes non publiées vers un bus (NATS JetStream) : plusieurs instances peuvent tourner en
//     parallèle sans doublon grâce à FOR UPDATE SKIP LOCKED, et le bus déduplique par identifiant d'événement ;
//   - Process rend un consommateur IDEMPOTENT (table processed_events, même transaction que l'effet).
package outbox

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/theb20/FILTER/backend/pkg/events"
	"github.com/theb20/FILTER/backend/pkg/pgxtx"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrations renvoie le système de fichiers des migrations goose de l'outbox (table de version dédiée : voir
// MigrationsTable), à exécuter par chaque service en plus des siennes.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// MigrationsTable est le nom de la table de version goose réservée à l'outbox.
const MigrationsTable = "goose_outbox"

// ErrNoTransaction : un événement ne peut être publié que dans une transaction (règle de l'outbox).
var ErrNoTransaction = errors.New("outbox: publication hors transaction interdite")

// Publisher implémente events.Publisher en écrivant dans l'outbox de la transaction du contexte.
type Publisher struct {
	NewID func() string // défaut : UUID v7
}

var _ events.Publisher = (*Publisher)(nil)

func (p *Publisher) Publish(ctx context.Context, e events.Event) error {
	db := pgxtx.From(ctx)
	if db == nil {
		return ErrNoTransaction
	}
	id := uuid.Must(uuid.NewV7()).String()
	if p.NewID != nil {
		id = p.NewID()
	}
	env, err := events.ToEnvelope(e, id)
	if err != nil {
		return err
	}
	raw, err := marshal(env)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO outbox (id, subject, aggregate_id, payload) VALUES ($1, $2, $3, $4)`, id, e.Type, e.AggregateID, raw)
	return err
}

// Process exécute fn UNE SEULE FOIS par (consumer, eventID), dans une transaction qui contient aussi la trace de
// traitement. Renvoie false sans exécuter fn si l'événement a déjà été traité (livraison en double).
func Process(ctx context.Context, pool *pgxpool.Pool, consumer, eventID string, fn func(ctx context.Context) error) (bool, error) {
	processed := false
	err := pgxtx.Run(ctx, pool, func(ctx context.Context) error {
		tag, err := pgxtx.From(ctx).Exec(ctx, `INSERT INTO processed_events (consumer, event_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, consumer, eventID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil // déjà traité
		}
		if err := fn(ctx); err != nil {
			return err // annulation : la trace de traitement disparaît aussi, l'événement sera relivré
		}
		processed = true
		return nil
	})
	return processed, err
}

// PurgePublished supprime les lignes publiées avant `before` (rétention configurable).
func PurgePublished(ctx context.Context, pool *pgxpool.Pool, before time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `DELETE FROM outbox WHERE published_at IS NOT NULL AND published_at < $1`, before)
	return tag.RowsAffected(), err
}
