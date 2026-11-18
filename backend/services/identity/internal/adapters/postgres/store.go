// Package postgres est l'adaptateur de persistance PostgreSQL d'identity : mêmes ports que l'adaptateur mémoire,
// intégrité garantie par les CONTRAINTES de la base (migrations goose), atomicité par les instructions SQL
// conditionnelles (UPDATE … WHERE … RETURNING) et transactions partagées avec l'outbox (pgxtx).
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // pilote database/sql pour goose
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/pgxtx"
	"github.com/theb20/FILTER/backend/pkg/throttle"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationsTable : table de version goose d'identity (distincte de celle de l'outbox).
const MigrationsTable = "goose_identity"

// Migrations renvoie les migrations d'identity.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// Migrate applique les migrations de l'outbox PUIS celles d'identity.
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

// Store implémente UserRepo, SessionRepo, TokenRepo, FailureStore et UnitOfWork.
type Store struct{ Pool *pgxpool.Pool }

// New ouvre le pool.
func New(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() { s.Pool.Close() }

func (s *Store) q(ctx context.Context) pgxtx.DBTX { return pgxtx.Querier(ctx, s.Pool) }

// Do : unité de travail (voir application.UnitOfWork).
func (s *Store) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return pgxtx.Run(ctx, s.Pool, fn)
}

func isUnique(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

func isFK(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23503"
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

// ── UserRepo ────────────────────────────────────────────────────────────────

const userCols = `u.id::text, u.email, u.password_hash, u.email_verified, u.kind, u.first_name, u.last_name, u.locale,
	u.preferred_currency, u.status, u.created_at, u.phone, c.id::text, c.legal_name, c.tax_id, c.country, c.approved, c.status, c.rejection_reason, c.decided_at, c.job_title`
const userFrom = ` FROM users u LEFT JOIN companies c ON c.id = u.company_id `

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var cid, legal, tax, country, cstatus, creason, cjob *string
	var approved *bool
	var decided *time.Time
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.EmailVerified, &u.Kind, &u.FirstName, &u.LastName, &u.Locale,
		&u.PreferredCurrency, &u.Status, &u.CreatedAt, &u.Phone, &cid, &legal, &tax, &country, &approved, &cstatus, &creason, &decided, &cjob); err != nil {
		return nil, notFound(err)
	}
	if cid != nil {
		u.Company = &domain.Company{ID: *cid, LegalName: *legal, TaxID: *tax, Country: *country, Approved: *approved, Status: *cstatus, DecidedAt: decided}
		if creason != nil {
			u.Company.RejectionReason = *creason
		}
		if cjob != nil {
			u.Company.JobTitle = *cjob
		}
	}
	return &u, nil
}

// Create insère le compte (et son entreprise) en UNE transaction. L'unicité de l'adresse est garantie par l'index
// unique : ON CONFLICT DO NOTHING évite d'ABORTER la transaction en cours (qui doit encore publier l'événement
// « tentative d'inscription sur un compte existant »).
func (s *Store) Create(ctx context.Context, u *domain.User) error {
	var companyID *string
	if u.Company != nil {
		companyID = &u.Company.ID
	}
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		tag, err := db.Exec(ctx, `INSERT INTO users (id, email, password_hash, email_verified, kind, first_name, last_name, locale, preferred_currency, status, company_id, created_at, phone)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (email) DO NOTHING`,
			u.ID, u.Email, u.PasswordHash, u.EmailVerified, u.Kind, u.FirstName, u.LastName, u.Locale, u.PreferredCurrency, u.Status, companyID, u.CreatedAt, u.Phone)
		if err != nil {
			if isUnique(err) {
				return application.ErrEmailTaken
			}
			return err
		}
		if tag.RowsAffected() == 0 {
			return application.ErrEmailTaken
		}
		if u.Company != nil { // clé étrangère DIFFÉRÉE : vérifiée à la validation de la transaction
			_, err = db.Exec(ctx, `INSERT INTO companies (id, legal_name, tax_id, country, approved, status) VALUES ($1,$2,$3,$4,$5,$6)`,
				u.Company.ID, u.Company.LegalName, u.Company.TaxID, u.Company.Country, u.Company.Approved, u.Company.State())
		}
		return err
	})
}

func (s *Store) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(s.q(ctx).QueryRow(ctx, `SELECT `+userCols+userFrom+`WHERE u.email = $1`, email))
}

func (s *Store) ByID(ctx context.Context, id string) (*domain.User, error) {
	return scanUser(s.q(ctx).QueryRow(ctx, `SELECT `+userCols+userFrom+`WHERE u.id = $1`, id))
}

func (s *Store) Update(ctx context.Context, u *domain.User) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		tag, err := db.Exec(ctx, `UPDATE users SET email=$9, password_hash=$2, email_verified=$3, first_name=$4, last_name=$5, locale=$6, preferred_currency=$7, status=$8, phone=$10 WHERE id=$1`,
			u.ID, u.PasswordHash, u.EmailVerified, u.FirstName, u.LastName, u.Locale, u.PreferredCurrency, u.Status, u.Email, u.Phone)
		if err != nil {
			if isUnique(err) {
				return application.ErrEmailTaken
			}
			return err
		}
		if tag.RowsAffected() == 0 {
			return application.ErrNotFound
		}
		if u.Company != nil {
			_, err = db.Exec(ctx, `UPDATE companies SET legal_name=$2, tax_id=$3, country=$4, approved=$5, status=$6, rejection_reason=NULLIF($7,''), decided_at=$8, job_title=$9 WHERE id=$1`,
				u.Company.ID, u.Company.LegalName, u.Company.TaxID, u.Company.Country, u.Company.Approved, u.Company.State(), u.Company.RejectionReason, u.Company.DecidedAt, u.Company.JobTitle)
		}
		return err
	})
}

// ── SessionRepo ─────────────────────────────────────────────────────────────

func (s *Store) CreateSession(ctx context.Context, t *domain.RefreshToken) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `INSERT INTO token_families (id, user_id, created_at) VALUES ($1,$2,$3) ON CONFLICT (id) DO NOTHING`, t.FamilyID, t.UserID, t.CreatedAt); err != nil {
			return err
		}
		_, err := db.Exec(ctx, `INSERT INTO refresh_tokens (id, family_id, token_hash, created_at, expires_at, used_at, ip, user_agent) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			t.ID, t.FamilyID, t.Hash, t.CreatedAt, t.ExpiresAt, t.UsedAt, t.IP, t.UserAgent)
		return err
	})
}

func (s *Store) SessionByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := s.q(ctx).QueryRow(ctx, `SELECT r.id::text, r.family_id::text, f.user_id::text, r.token_hash, r.created_at, r.expires_at, r.used_at, f.revoked_at, r.ip, r.user_agent
		FROM refresh_tokens r JOIN token_families f ON f.id = r.family_id WHERE r.token_hash = $1`, hash).
		Scan(&t.ID, &t.FamilyID, &t.UserID, &t.Hash, &t.CreatedAt, &t.ExpiresAt, &t.UsedAt, &t.RevokedAt, &t.IP, &t.UserAgent)
	if err != nil {
		return nil, notFound(err)
	}
	return &t, nil
}

// MarkUsed est atomique : un seul appelant concurrent voit RowsAffected()==1.
func (s *Store) MarkUsed(ctx context.Context, id string, at time.Time) (bool, error) {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE refresh_tokens SET used_at = $2 WHERE id = $1 AND used_at IS NULL`, id, at)
	return tag.RowsAffected() == 1, err
}

func (s *Store) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE token_families SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, familyID, at)
	return err
}

func (s *Store) RevokeAllExcept(ctx context.Context, userID, keepFamilyID string, at time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE token_families SET revoked_at = $3 WHERE user_id = $1 AND revoked_at IS NULL AND ($2 = '' OR id::text <> $2)`, userID, keepFamilyID, at)
	return err
}

// ── TokenRepo ───────────────────────────────────────────────────────────────

func (s *Store) Put(ctx context.Context, t *domain.OneTimeToken) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO one_time_tokens (purpose, token_hash, user_id, expires_at, used_at, payload) VALUES ($1,$2,$3,$4,$5,$6)`,
		t.Purpose, t.Hash, t.UserID, t.ExpiresAt, t.UsedAt, t.Payload)
	return err
}

// Consume est atomique : un seul appelant concurrent obtient le compte.
func (s *Store) Consume(ctx context.Context, purpose, hash string, now time.Time) (string, bool, error) {
	id, _, ok, err := s.ConsumePayload(ctx, purpose, hash, now)
	return id, ok, err
}

func (s *Store) ConsumePayload(ctx context.Context, purpose, hash string, now time.Time) (string, string, bool, error) {
	var id, payload string
	err := s.q(ctx).QueryRow(ctx, `UPDATE one_time_tokens SET used_at = $3 WHERE purpose = $1 AND token_hash = $2 AND used_at IS NULL AND expires_at > $3 RETURNING user_id::text, payload`, purpose, hash, now).Scan(&id, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	return id, payload, err == nil, err
}

func (s *Store) InvalidateUserTokens(ctx context.Context, purpose, userID string, now time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE one_time_tokens SET used_at = $3 WHERE purpose = $1 AND user_id = $2 AND used_at IS NULL`, purpose, userID, now)
	return err
}

// ── FailureStore ────────────────────────────────────────────────────────────

func (s *Store) Pair(ctx context.Context, key string) (throttle.State, error) {
	var st throttle.State
	err := s.q(ctx).QueryRow(ctx, `SELECT failures, last_failure FROM login_throttle WHERE key = $1`, key).Scan(&st.Failures, &st.LastFailure)
	if errors.Is(err, pgx.ErrNoRows) {
		return throttle.State{}, nil
	}
	return st, err
}

func (s *Store) SetPair(ctx context.Context, key string, st throttle.State) error {
	if st.Failures <= 0 {
		return s.ResetPair(ctx, key)
	}
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO login_throttle (key, failures, last_failure) VALUES ($1,$2,$3)
		ON CONFLICT (key) DO UPDATE SET failures = EXCLUDED.failures, last_failure = EXCLUDED.last_failure`, key, st.Failures, st.LastFailure)
	return err
}

func (s *Store) ResetPair(ctx context.Context, key string) error {
	_, err := s.q(ctx).Exec(ctx, `DELETE FROM login_throttle WHERE key = $1`, key)
	return err
}

func (s *Store) AccountFailures(ctx context.Context, email string, at time.Time, window time.Duration) (int, error) {
	var n int
	err := s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		// purge des échecs hors fenêtre, ajout de celui-ci, comptage : la table ne grossit pas indéfiniment
		if _, err := db.Exec(ctx, `DELETE FROM login_failures WHERE email = $1 AND at < $2`, email, at.Add(-window)); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, `INSERT INTO login_failures (email, at) VALUES ($1,$2)`, email, at); err != nil {
			return err
		}
		return db.QueryRow(ctx, `SELECT count(*) FROM login_failures WHERE email = $1`, email).Scan(&n)
	})
	return n, err
}

// MarkAlerted est atomique : un seul appelant concurrent obtient true tant que la marque n'a pas expiré.
func (s *Store) MarkAlerted(ctx context.Context, email string, now, until time.Time) (bool, error) {
	tag, err := s.q(ctx).Exec(ctx, `INSERT INTO login_alerts (email, until) VALUES ($1,$2)
		ON CONFLICT (email) DO UPDATE SET until = EXCLUDED.until WHERE login_alerts.until <= $3`, email, until, now)
	return tag.RowsAffected() == 1, err
}

// ── OAuthRepo ───────────────────────────────────────────────────────────────

func (s *Store) OAuthUser(ctx context.Context, provider, subject string) (string, bool, error) {
	var id string
	err := s.q(ctx).QueryRow(ctx, `SELECT user_id::text FROM oauth_identities WHERE provider = $1 AND subject = $2`, provider, subject).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

func (s *Store) LinkOAuth(ctx context.Context, provider, subject, userID string, at time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO oauth_identities (provider, subject, user_id, created_at) VALUES ($1,$2,$3,$4) ON CONFLICT (provider, subject) DO NOTHING`, provider, subject, userID, at)
	return err
}
