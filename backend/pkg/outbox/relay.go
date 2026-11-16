package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bus publie un message sur le bus. msgID sert à la déduplication côté bus (NATS : en-tête Nats-Msg-Id).
type Bus interface {
	Publish(ctx context.Context, subject, msgID string, payload []byte) error
}

// Relay relaie les lignes non publiées de l'outbox vers le bus.
type Relay struct {
	Pool      *pgxpool.Pool
	Bus       Bus
	Batch     int           // défaut 100
	Interval  time.Duration // défaut 500 ms
	BaseDelay time.Duration // reprise après échec, doublée à chaque tentative (défaut 1 s)
	MaxDelay  time.Duration // plafond (défaut 5 min)
	Log       *slog.Logger
}

func (r *Relay) defaults() {
	if r.Batch <= 0 {
		r.Batch = 100
	}
	if r.Interval <= 0 {
		r.Interval = 500 * time.Millisecond
	}
	if r.BaseDelay <= 0 {
		r.BaseDelay = time.Second
	}
	if r.MaxDelay <= 0 {
		r.MaxDelay = 5 * time.Minute
	}
	if r.Log == nil {
		r.Log = slog.Default()
	}
}

// Run relaie en boucle jusqu'à l'annulation du contexte.
func (r *Relay) Run(ctx context.Context) {
	r.defaults()
	t := time.NewTicker(r.Interval)
	defer t.Stop()
	for {
		for {
			n, err := r.RunOnce(ctx)
			if err != nil {
				r.Log.Error("outbox: relais", "err", err)
				break
			}
			if n < r.Batch {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce traite un lot et renvoie le nombre de lignes examinées. Le lot est verrouillé (FOR UPDATE SKIP LOCKED) :
// des relais concurrents prennent des lots différents. Une ligne publiée est marquée dans la même transaction ; si le
// processus s'arrête entre publication et validation, la ligne est republiée avec le MÊME identifiant, que le bus
// déduplique (au moins une fois côté relais, une seule fois côté effet grâce aux consommateurs idempotents).
func (r *Relay) RunOnce(ctx context.Context) (int, error) {
	r.defaults()
	n := 0
	err := func() (err error) {
		tx, err := r.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
		rows, err := tx.Query(ctx, `SELECT id::text, subject, payload, attempts FROM outbox
			WHERE published_at IS NULL AND next_attempt_at <= now()
			ORDER BY created_at, id LIMIT $1 FOR UPDATE SKIP LOCKED`, r.Batch)
		if err != nil {
			return err
		}
		type row struct {
			id, subject string
			payload     []byte
			attempts    int
		}
		var batch []row
		for rows.Next() {
			var x row
			if err := rows.Scan(&x.id, &x.subject, &x.payload, &x.attempts); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, x)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		n = len(batch)
		b := &pgx.Batch{}
		for _, x := range batch {
			if perr := r.Bus.Publish(ctx, x.subject, x.id, x.payload); perr != nil {
				delay := r.BaseDelay << min(x.attempts, 20)
				if delay > r.MaxDelay || delay <= 0 {
					delay = r.MaxDelay
				}
				b.Queue(`UPDATE outbox SET attempts = attempts + 1, last_error = $2, next_attempt_at = now() + $3::interval WHERE id = $1`,
					x.id, perr.Error(), fmt.Sprintf("%d milliseconds", delay.Milliseconds()))
				continue
			}
			b.Queue(`UPDATE outbox SET published_at = now(), last_error = NULL WHERE id = $1`, x.id)
		}
		if n > 0 {
			if err := tx.SendBatch(ctx, b).Close(); err != nil {
				return err
			}
		}
		return tx.Commit(ctx)
	}()
	return n, err
}
