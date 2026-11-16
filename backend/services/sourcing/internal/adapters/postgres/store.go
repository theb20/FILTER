// Package postgres : adaptateur PostgreSQL du service sourcing. Le budget est réservé par des instructions conditionnelles atomiques.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/pgxtx"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/application"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const MigrationsTable = "goose_sourcing"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// Migrate applique les migrations de l'outbox puis celles de sourcing.
func Migrate(ctx context.Context, url string) error {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := outbox.MigrateOutbox(ctx, db); err != nil {
		return err
	}
	return outbox.Migrate(ctx, db, Migrations(), MigrationsTable)
}

type Store struct{ Pool *pgxpool.Pool }

var _ application.Store = (*Store)(nil)

func New(ctx context.Context, url string) (*Store, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Store{Pool: p}, nil
}

func (s *Store) Close()                           { s.Pool.Close() }
func (s *Store) q(ctx context.Context) pgxtx.DBTX { return pgxtx.Querier(ctx, s.Pool) }
func (s *Store) do(ctx context.Context, f func(context.Context) error) error {
	return pgxtx.Run(ctx, s.Pool, f)
}

func pgCode(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

// ── sources ─────────────────────────────────────────────────────────────────

const srcCols = `id::text, kind, name, base_url, enabled, suspended_until, priority, timeout_ms, max_retries, secret_ref, cost_per_call_micros, currency, allows_raw_storage, terms_checked_at, max_cache_ttl_seconds, created_at`

func scanSource(r pgx.Row) (*domain.Source, error) {
	var x domain.Source
	if err := r.Scan(&x.ID, &x.Kind, &x.Name, &x.BaseURL, &x.Enabled, &x.SuspendedUntil, &x.Priority, &x.TimeoutMs, &x.MaxRetries, &x.SecretRef, &x.CostPerCallMicros, &x.Currency, &x.AllowsRawStorage, &x.TermsCheckedAt, &x.MaxCacheTTLSeconds, &x.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &x, nil
}

func (s *Store) CreateSource(ctx context.Context, x *domain.Source) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO sources (id, kind, name, base_url, enabled, priority, timeout_ms, max_retries, secret_ref, cost_per_call_micros, currency, max_cache_ttl_seconds, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, x.ID, x.Kind, x.Name, x.BaseURL, x.Enabled, x.Priority, x.TimeoutMs, x.MaxRetries, x.SecretRef, x.CostPerCallMicros, x.Currency, x.MaxCacheTTLSeconds, x.CreatedAt)
	if pgCode(err) == "23505" {
		return application.ErrNameTaken
	}
	return err
}

func (s *Store) UpdateSource(ctx context.Context, x *domain.Source) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE sources SET kind=$2, name=$3, base_url=$4, priority=$5, timeout_ms=$6, max_retries=$7, secret_ref=$8, cost_per_call_micros=$9, currency=$10, max_cache_ttl_seconds=$11, updated_at=now() WHERE id=$1`,
		x.ID, x.Kind, x.Name, x.BaseURL, x.Priority, x.TimeoutMs, x.MaxRetries, x.SecretRef, x.CostPerCallMicros, x.Currency, x.MaxCacheTTLSeconds)
	if pgCode(err) == "23505" {
		return application.ErrNameTaken
	}
	if err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) SourceByID(ctx context.Context, id string) (*domain.Source, error) {
	return scanSource(s.q(ctx).QueryRow(ctx, `SELECT `+srcCols+` FROM sources WHERE id = $1`, id))
}

func (s *Store) ListSources(ctx context.Context, query string, limit, offset int) ([]*domain.Source, int, error) {
	where, args := "", []any{}
	if query != "" {
		where, args = ` WHERE name ILIKE $1`, []any{"%" + query + "%"}
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM sources`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	n := len(args)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+srcCols+` FROM sources`+where+` ORDER BY priority, lower(name) LIMIT $`+string(rune('0'+n-1))+` OFFSET $`+string(rune('0'+n)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Source, error) { return scanSource(r) })
	return items, total, err
}

func (s *Store) SetEnabled(ctx context.Context, id string, enabled bool) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE sources SET enabled=$2, updated_at=now() WHERE id=$1`, id, enabled)
	if err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) SetSuspendedUntil(ctx context.Context, id string, until *time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE sources SET suspended_until=$2, updated_at=now() WHERE id=$1`, id, until)
	return err
}

func (s *Store) SetRawStorage(ctx context.Context, id string, allowed bool, checkedAt time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE sources SET allows_raw_storage=$2, terms_checked_at=CASE WHEN $2 THEN $3 ELSE terms_checked_at END, updated_at=now() WHERE id=$1`, id, allowed, checkedAt)
	return err
}

func (s *Store) ListActive(ctx context.Context, now time.Time) ([]application.ActiveSource, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT s.id::text, s.name, s.priority, q.fallback_source_id::text, s.max_cache_ttl_seconds
		FROM sources s LEFT JOIN source_quotas q ON q.source_id = s.id
		WHERE s.enabled AND (s.suspended_until IS NULL OR s.suspended_until <= $1) ORDER BY s.priority, lower(s.name)`, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (application.ActiveSource, error) {
		var a application.ActiveSource
		err := r.Scan(&a.ID, &a.Name, &a.Priority, &a.FallbackSourceID, &a.MaxCacheTTLSeconds)
		return a, err
	})
}

// ── quota ───────────────────────────────────────────────────────────────────

func (s *Store) QuotaByID(ctx context.Context, id string, now time.Time) (*domain.Quota, error) {
	q := &domain.Quota{SourceID: id}
	err := s.q(ctx).QueryRow(ctx, `SELECT s.currency, COALESCE(q.monthly_budget_micros,0), COALESCE(q.daily_cap_micros,0), COALESCE(q.alert_threshold_pct,80), q.fallback_source_id::text FROM sources s LEFT JOIN source_quotas q ON q.source_id = s.id WHERE s.id = $1`, id).
		Scan(&q.Currency, &q.MonthlyBudgetMicros, &q.DailyCapMicros, &q.AlertThresholdPct, &q.FallbackSourceID)
	if err != nil {
		return nil, notFound(err)
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT period_kind, spent_micros, reserved_micros FROM source_usage WHERE source_id = $1 AND ((period_kind='day' AND period_start=$2) OR (period_kind='month' AND period_start=$3))`, id, domain.DayStart(now), domain.MonthStart(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var spent, reserved int64
		if err := rows.Scan(&kind, &spent, &reserved); err != nil {
			return nil, err
		}
		if kind == "day" {
			q.DaySpent, q.DayReserved = spent, reserved
		} else {
			q.MonthSpent, q.MonthReserved = spent, reserved
		}
	}
	return q, rows.Err()
}

func (s *Store) PutQuota(ctx context.Context, q *domain.Quota) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO source_quotas (source_id, monthly_budget_micros, daily_cap_micros, alert_threshold_pct, fallback_source_id) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (source_id) DO UPDATE SET monthly_budget_micros=EXCLUDED.monthly_budget_micros, daily_cap_micros=EXCLUDED.daily_cap_micros, alert_threshold_pct=EXCLUDED.alert_threshold_pct, fallback_source_id=EXCLUDED.fallback_source_id`,
		q.SourceID, q.MonthlyBudgetMicros, q.DailyCapMicros, q.AlertThresholdPct, q.FallbackSourceID)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23514" && strings.Contains(pe.Message, "secours") { // déclencheur « un seul saut »
		return application.ErrFallbackChain
	}
	return err
}

// ── budget : réservations atomiques ─────────────────────────────────────────

// Reserve : (1) ligne de réservation (UNIQUE (search_id, source_id) → ErrAlreadyCalled) ; (2) compteurs du jour ET du mois incrémentés par une
// instruction CONDITIONNELLE (spent + reserved + montant ≤ plafond). Si l'une des deux ne touche aucune ligne, tout est annulé (ErrBudget).
func (s *Store) Reserve(ctx context.Context, r application.Reservation) error {
	err := s.do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `INSERT INTO source_reservations (id, source_id, search_id, amount_micros, day, month, created_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			r.ID, r.SourceID, r.SearchID, r.AmountMicros, r.Day, r.Month, r.CreatedAt, r.ExpiresAt); err != nil {
			if pgCode(err) == "23505" {
				return application.ErrAlreadyCalled
			}
			return err
		}
		for _, p := range []struct {
			kind  string
			start time.Time
			cap   int64
		}{{"day", r.Day, r.DailyCap}, {"month", r.Month, r.MonthlyCap}} {
			if _, err := db.Exec(ctx, `INSERT INTO source_usage (source_id, period_kind, period_start) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, r.SourceID, p.kind, p.start); err != nil {
				return err
			}
			tag, err := db.Exec(ctx, `UPDATE source_usage SET reserved_micros = reserved_micros + $4 WHERE source_id=$1 AND period_kind=$2 AND period_start=$3 AND spent_micros + reserved_micros + $4 <= $5`,
				r.SourceID, p.kind, p.start, r.AmountMicros, p.cap)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return application.ErrBudget
			}
		}
		return nil
	})
	return err
}

type resRow struct {
	sourceID string
	amount   int64
	day, mon time.Time
	status   string
}

func (s *Store) lockReservation(ctx context.Context, id string) (resRow, error) {
	var r resRow
	err := s.q(ctx).QueryRow(ctx, `SELECT source_id::text, amount_micros, day, month, status FROM source_reservations WHERE id = $1 FOR UPDATE`, id).Scan(&r.sourceID, &r.amount, &r.day, &r.mon, &r.status)
	return r, notFound(err)
}

func (s *Store) adjust(ctx context.Context, r resRow, releaseReserved bool, spent int64) error {
	db := s.q(ctx)
	for _, p := range []struct {
		kind  string
		start time.Time
	}{{"day", r.day}, {"month", r.mon}} {
		rel := int64(0)
		if releaseReserved {
			rel = r.amount
		}
		if _, err := db.Exec(ctx, `UPDATE source_usage SET reserved_micros = GREATEST(reserved_micros - $4, 0), spent_micros = spent_micros + $5 WHERE source_id=$1 AND period_kind=$2 AND period_start=$3`, r.sourceID, p.kind, p.start, rel, spent); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Settle(ctx context.Context, id string, actualCost int64, now time.Time) error {
	return s.do(ctx, func(ctx context.Context) error {
		r, err := s.lockReservation(ctx, id)
		if err != nil {
			return err
		}
		switch r.status {
		case "held":
			if err := s.adjust(ctx, r, true, actualCost); err != nil {
				return err
			}
		case "expired": // la tâche de libération est passée avant le règlement : la réservation est déjà retirée, il reste la dépense
			if err := s.adjust(ctx, r, false, actualCost); err != nil {
				return err
			}
		default:
			return nil // déjà réglée ou libérée
		}
		_, err = s.q(ctx).Exec(ctx, `UPDATE source_reservations SET status='settled', settled_at=$2 WHERE id=$1`, id, now)
		return err
	})
}

func (s *Store) Release(ctx context.Context, id string, now time.Time) error {
	return s.do(ctx, func(ctx context.Context) error {
		r, err := s.lockReservation(ctx, id)
		if err != nil {
			return err
		}
		if r.status != "held" {
			return nil
		}
		if err := s.adjust(ctx, r, true, 0); err != nil {
			return err
		}
		_, err = s.q(ctx).Exec(ctx, `UPDATE source_reservations SET status='released', settled_at=$2 WHERE id=$1`, id, now)
		return err
	})
}

// ReleaseExpired : toute réservation « held » dont l'échéance est passée devient « expired » et son montant est retiré du compteur.
func (s *Store) ReleaseExpired(ctx context.Context, now time.Time) (int, error) {
	n := 0
	err := s.do(ctx, func(ctx context.Context) error {
		rows, err := s.q(ctx).Query(ctx, `SELECT id::text FROM source_reservations WHERE status='held' AND expires_at <= $1 ORDER BY expires_at FOR UPDATE SKIP LOCKED LIMIT 500`, now)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, id := range ids {
			r, err := s.lockReservation(ctx, id)
			if err != nil {
				return err
			}
			if r.status != "held" {
				continue
			}
			if err := s.adjust(ctx, r, true, 0); err != nil {
				return err
			}
			if _, err := s.q(ctx).Exec(ctx, `UPDATE source_reservations SET status='expired', settled_at=$2 WHERE id=$1`, id, now); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}

// Reconcile recalcule `reserved_micros` à partir des lignes « held » : corrige toute dérive des compteurs.
func (s *Store) Reconcile(ctx context.Context, now time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE source_usage u SET reserved_micros = COALESCE((
			SELECT sum(r.amount_micros) FROM source_reservations r WHERE r.source_id = u.source_id AND r.status = 'held' AND ((u.period_kind = 'day' AND r.day = u.period_start) OR (u.period_kind = 'month' AND r.month = u.period_start))), 0)
		WHERE u.period_start >= $1`, domain.MonthStart(now).AddDate(0, -1, 0))
	return err
}

// ── journal, santé, coûts ───────────────────────────────────────────────────

func (s *Store) LogCall(ctx context.Context, c application.CallLog) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO source_calls (id, source_id, search_id, started_at, duration_ms, status, http_status, attempts, cost_micros, currency, error_kind) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,0),$8,$9,$10,NULLIF($11,''))`,
		c.ID, c.SourceID, c.SearchID, c.StartedAt, c.DurationMs, c.Status, c.HTTPStatus, c.Attempts, c.CostMicros, c.Currency, c.ErrorKind)
	return err
}

func (s *Store) Stats(ctx context.Context, sourceID string, since time.Time) (application.CallStats, error) {
	var st application.CallStats
	var p50, p95 *float64
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE status NOT IN ('ok','cached')), percentile_cont(0.5) WITHIN GROUP (ORDER BY duration_ms), percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms),
		max(started_at) FILTER (WHERE status = 'ok'), max(started_at) FILTER (WHERE status NOT IN ('ok','cached'))
		FROM source_calls WHERE source_id = $1 AND started_at >= $2`, sourceID, since).Scan(&st.Calls, &st.Errors, &p50, &p95, &st.LastSuccessAt, &st.LastErrAt)
	if err != nil {
		return st, err
	}
	if p50 != nil {
		st.P50Ms, st.P95Ms = int(*p50), int(*p95)
	}
	if st.LastErrAt != nil {
		var k *string
		_ = s.q(ctx).QueryRow(ctx, `SELECT error_kind FROM source_calls WHERE source_id=$1 AND status NOT IN ('ok','cached') ORDER BY started_at DESC LIMIT 1`, sourceID).Scan(&k)
		if k != nil {
			st.LastErrorKind = *k
		}
	}
	return st, nil
}

func (s *Store) Costs(ctx context.Context, sourceID string, from, to time.Time) ([]application.CostLine, error) {
	var sid any
	if sourceID != "" {
		sid = sourceID
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT c.source_id::text, s.name, c.currency, count(*), count(*) FILTER (WHERE c.status = 'cached'), count(*) FILTER (WHERE c.status NOT IN ('ok','cached')), COALESCE(sum(c.cost_micros),0),
		COALESCE(sum(c.cost_micros) / NULLIF(count(DISTINCT c.search_id),0), 0)
		FROM source_calls c JOIN sources s ON s.id = c.source_id WHERE c.started_at >= $1 AND c.started_at < $2 AND ($3::uuid IS NULL OR c.source_id = $3::uuid) GROUP BY c.source_id, s.name, c.currency`, from, to, sid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (application.CostLine, error) {
		var l application.CostLine
		err := r.Scan(&l.SourceID, &l.SourceName, &l.Currency, &l.Calls, &l.CachedCalls, &l.FailedCalls, &l.CostMicros, &l.PerSearchMicros)
		return l, err
	})
}

func (s *Store) PurgeCalls(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM source_calls WHERE started_at < $1`, before)
	return tag.RowsAffected(), err
}

func (s *Store) CountTests(ctx context.Context, actorID string, since time.Time) (int, error) {
	var n int
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM source_tests WHERE actor_id=$1 AND at >= $2`, actorID, since).Scan(&n)
	return n, err
}

func (s *Store) RecordTest(ctx context.Context, actorID string, at time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO source_tests (actor_id, at) VALUES ($1,$2)`, actorID, at)
	return err
}

// Do : unité de travail (voir application.Store).
func (s *Store) Do(ctx context.Context, f func(context.Context) error) error {
	return pgxtx.Run(ctx, s.Pool, f)
}
