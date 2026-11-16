package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// ── StaffRepo ───────────────────────────────────────────────────────────────

const staffCols = `s.id::text, s.email, s.name, s.password_hash, s.status, s.totp_secret, s.two_factor_enabled, s.totp_last_step, s.last_login_at, s.created_at,
	COALESCE((SELECT array_agg(role_id::text ORDER BY role_id) FROM staff_user_roles r WHERE r.staff_id = s.id), '{}')`

func scanStaff(row pgx.Row) (*domain.StaffUser, error) {
	var u domain.StaffUser
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Status, &u.TOTPSecret, &u.TwoFactorEnabled, &u.TOTPLastStep, &u.LastLoginAt, &u.CreatedAt, &u.RoleIDs); err != nil {
		return nil, notFound(err)
	}
	return &u, nil
}

func (s *Store) StaffByEmail(ctx context.Context, email string) (*domain.StaffUser, error) {
	return scanStaff(s.q(ctx).QueryRow(ctx, `SELECT `+staffCols+` FROM staff_users s WHERE s.email = $1`, email))
}

func (s *Store) StaffByID(ctx context.Context, id string) (*domain.StaffUser, error) {
	return scanStaff(s.q(ctx).QueryRow(ctx, `SELECT `+staffCols+` FROM staff_users s WHERE s.id = $1`, id))
}

func (s *Store) CreateStaff(ctx context.Context, u *domain.StaffUser) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		tag, err := db.Exec(ctx, `INSERT INTO staff_users (id, email, name, password_hash, status, totp_secret, two_factor_enabled, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (email) DO NOTHING`, u.ID, u.Email, u.Name, u.PasswordHash, u.Status, u.TOTPSecret, u.TwoFactorEnabled, u.CreatedAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return application.ErrEmailTaken
		}
		for _, r := range u.RoleIDs {
			if _, err := db.Exec(ctx, `INSERT INTO staff_user_roles (staff_id, role_id) VALUES ($1,$2)`, u.ID, r); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateStaff ne modifie PAS le dernier pas TOTP (AdvanceTOTPStep, atomique, en est seul responsable).
func (s *Store) UpdateStaff(ctx context.Context, u *domain.StaffUser) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE staff_users SET name=$2, password_hash=$3, status=$4, totp_secret=$5, two_factor_enabled=$6, last_login_at=$7, updated_at=now() WHERE id=$1`,
		u.ID, u.Name, u.PasswordHash, u.Status, u.TOTPSecret, u.TwoFactorEnabled, u.LastLoginAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (s *Store) AdvanceTOTPStep(ctx context.Context, id string, step int64) (bool, error) {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE staff_users SET totp_last_step = $2 WHERE id = $1 AND totp_last_step < $2`, id, step)
	return tag.RowsAffected() == 1, err
}

func (s *Store) SetRecoveryCodes(ctx context.Context, id string, hashes []string) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `DELETE FROM staff_recovery_codes WHERE staff_id = $1`, id); err != nil {
			return err
		}
		for _, h := range hashes {
			if _, err := db.Exec(ctx, `INSERT INTO staff_recovery_codes (staff_id, code_hash) VALUES ($1,$2)`, id, h); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ConsumeRecoveryCode(ctx context.Context, id, hash string, at time.Time) (bool, error) {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE staff_recovery_codes SET used_at = $3 WHERE staff_id = $1 AND code_hash = $2 AND used_at IS NULL`, id, hash, at)
	return tag.RowsAffected() == 1, err
}

func (s *Store) RolesByIDs(ctx context.Context, ids []string) ([]domain.Role, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT id::text, name, description, permissions, system FROM staff_roles WHERE id::text = ANY($1) ORDER BY name`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Role, error) {
		var x domain.Role
		err := r.Scan(&x.ID, &x.Name, &x.Description, &x.Permissions, &x.System)
		return x, err
	})
}

func (s *Store) RoleByName(ctx context.Context, name string) (*domain.Role, error) {
	var x domain.Role
	err := s.q(ctx).QueryRow(ctx, `SELECT id::text, name, description, permissions, system FROM staff_roles WHERE lower(name) = lower($1)`, name).Scan(&x.ID, &x.Name, &x.Description, &x.Permissions, &x.System)
	if err != nil {
		return nil, notFound(err)
	}
	return &x, nil
}

func (s *Store) CountActiveWithRole(ctx context.Context, roleID string) (int, error) {
	var n int
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM staff_users u JOIN staff_user_roles r ON r.staff_id = u.id WHERE r.role_id = $1 AND u.status = 'active'`, roleID).Scan(&n)
	return n, err
}

// ── ChallengeRepo ───────────────────────────────────────────────────────────

func (s *Store) PutChallenge(ctx context.Context, c *domain.Challenge) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO staff_challenges (id, staff_id, scope, ip, expires_at) VALUES ($1,$2,$3,$4,$5)`, c.ID, c.StaffID, c.Scope, c.IP, c.ExpiresAt)
	return err
}

func (s *Store) ChallengeByID(ctx context.Context, id string) (*domain.Challenge, error) {
	var c domain.Challenge
	err := s.q(ctx).QueryRow(ctx, `SELECT id, staff_id::text, scope, ip, expires_at, used_at, failures FROM staff_challenges WHERE id = $1`, id).
		Scan(&c.ID, &c.StaffID, &c.Scope, &c.IP, &c.ExpiresAt, &c.UsedAt, &c.Failures)
	if err != nil {
		return nil, notFound(err)
	}
	return &c, nil
}

func (s *Store) ConsumeChallenge(ctx context.Context, id string, now time.Time) (bool, error) {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE staff_challenges SET used_at = $2 WHERE id = $1 AND used_at IS NULL AND expires_at > $2`, id, now)
	return tag.RowsAffected() == 1, err
}

func (s *Store) FailChallenge(ctx context.Context, id string) (int, error) {
	var n int
	err := s.q(ctx).QueryRow(ctx, `UPDATE staff_challenges SET failures = failures + 1 WHERE id = $1 RETURNING failures`, id).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, application.ErrNotFound
	}
	return n, err
}

// ── sessions du personnel : mêmes opérations que les clients, tables séparées ─

// StaffSessions renvoie le dépôt des sessions du personnel.
func (s *Store) StaffSessions() application.SessionRepo { return &staffSessions{s} }

type staffSessions struct{ s *Store }

func (r *staffSessions) CreateSession(ctx context.Context, t *domain.RefreshToken) error {
	return r.s.Do(ctx, func(ctx context.Context) error {
		db := r.s.q(ctx)
		if _, err := db.Exec(ctx, `INSERT INTO staff_token_families (id, staff_id, created_at) VALUES ($1,$2,$3) ON CONFLICT (id) DO NOTHING`, t.FamilyID, t.UserID, t.CreatedAt); err != nil {
			return err
		}
		_, err := db.Exec(ctx, `INSERT INTO staff_refresh_tokens (id, family_id, token_hash, created_at, expires_at, used_at, ip, user_agent) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			t.ID, t.FamilyID, t.Hash, t.CreatedAt, t.ExpiresAt, t.UsedAt, t.IP, t.UserAgent)
		return err
	})
}

func (r *staffSessions) SessionByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.s.q(ctx).QueryRow(ctx, `SELECT r.id::text, r.family_id::text, f.staff_id::text, r.token_hash, r.created_at, r.expires_at, r.used_at, f.revoked_at, r.ip, r.user_agent
		FROM staff_refresh_tokens r JOIN staff_token_families f ON f.id = r.family_id WHERE r.token_hash = $1`, hash).
		Scan(&t.ID, &t.FamilyID, &t.UserID, &t.Hash, &t.CreatedAt, &t.ExpiresAt, &t.UsedAt, &t.RevokedAt, &t.IP, &t.UserAgent)
	if err != nil {
		return nil, notFound(err)
	}
	return &t, nil
}

func (r *staffSessions) MarkUsed(ctx context.Context, id string, at time.Time) (bool, error) {
	tag, err := r.s.q(ctx).Exec(ctx, `UPDATE staff_refresh_tokens SET used_at = $2 WHERE id = $1 AND used_at IS NULL`, id, at)
	return tag.RowsAffected() == 1, err
}

func (r *staffSessions) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	_, err := r.s.q(ctx).Exec(ctx, `UPDATE staff_token_families SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, familyID, at)
	return err
}

func (r *staffSessions) RevokeAllExcept(ctx context.Context, staffID, keepFamilyID string, at time.Time) error {
	_, err := r.s.q(ctx).Exec(ctx, `UPDATE staff_token_families SET revoked_at = $3 WHERE staff_id = $1 AND revoked_at IS NULL AND ($2 = '' OR id::text <> $2)`, staffID, keepFamilyID, at)
	return err
}
