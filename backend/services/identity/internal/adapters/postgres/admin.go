package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var (
	_ application.StaffAdminRepo = (*Store)(nil)
	_ application.AuditRepo      = (*Store)(nil)
)

func like(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// ── personnel ───────────────────────────────────────────────────────────────

func (s *Store) ListStaff(ctx context.Context, query string, limit, offset int) ([]*domain.StaffUser, int, error) {
	var total int
	where, args := "", []any{}
	if query != "" {
		where, args = ` WHERE s.email ILIKE $1 OR s.name ILIKE $1`, []any{like(query)}
	}
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM staff_users s`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+staffCols+` FROM staff_users s`+where+` ORDER BY s.created_at DESC, s.id LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.StaffUser, error) { return scanStaff(r) })
	return items, total, err
}

func itoa(n int) string { return string(rune('0' + n)) } // n < 10 (paramètres SQL de ce fichier)

func (s *Store) SetStaffRoles(ctx context.Context, staffID string, roleIDs []string) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `DELETE FROM staff_user_roles WHERE staff_id = $1`, staffID); err != nil {
			return err
		}
		for _, r := range roleIDs {
			if _, err := db.Exec(ctx, `INSERT INTO staff_user_roles (staff_id, role_id) VALUES ($1,$2)`, staffID, r); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ClearTwoFactor(ctx context.Context, staffID string) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `UPDATE staff_users SET totp_secret = NULL, two_factor_enabled = false, totp_last_step = 0, updated_at = now() WHERE id = $1`, staffID); err != nil {
			return err
		}
		_, err := db.Exec(ctx, `DELETE FROM staff_recovery_codes WHERE staff_id = $1`, staffID)
		return err
	})
}

func (s *Store) PutStaffInvitation(ctx context.Context, hash, staffID string, expires time.Time) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `DELETE FROM staff_invitations WHERE staff_id = $1 AND used_at IS NULL`, staffID); err != nil {
			return err
		}
		_, err := db.Exec(ctx, `INSERT INTO staff_invitations (token_hash, staff_id, expires_at) VALUES ($1,$2,$3)`, hash, staffID, expires)
		return err
	})
}

func (s *Store) ConsumeStaffInvitation(ctx context.Context, hash string, now time.Time) (string, bool, error) {
	var id string
	err := s.q(ctx).QueryRow(ctx, `UPDATE staff_invitations SET used_at = $2 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2 RETURNING staff_id::text`, hash, now).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

// ── rôles ───────────────────────────────────────────────────────────────────

func (s *Store) ListRoles(ctx context.Context, query string, limit, offset int) ([]domain.Role, int, error) {
	var total int
	where, args := "", []any{}
	if query != "" {
		where, args = ` WHERE name ILIKE $1 OR description ILIKE $1`, []any{like(query)}
	}
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM staff_roles`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT id::text, name, description, permissions, system FROM staff_roles`+where+` ORDER BY system DESC, lower(name) LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.Role, error) {
		var x domain.Role
		err := r.Scan(&x.ID, &x.Name, &x.Description, &x.Permissions, &x.System)
		return x, err
	})
	return items, total, err
}

func (s *Store) RoleByID(ctx context.Context, id string) (*domain.Role, error) {
	var x domain.Role
	err := s.q(ctx).QueryRow(ctx, `SELECT id::text, name, description, permissions, system FROM staff_roles WHERE id = $1`, id).Scan(&x.ID, &x.Name, &x.Description, &x.Permissions, &x.System)
	if err != nil {
		return nil, notFound(err)
	}
	return &x, nil
}

func (s *Store) CreateRole(ctx context.Context, r *domain.Role) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO staff_roles (id, name, description, permissions, system) VALUES ($1,$2,$3,$4,false)`, r.ID, r.Name, r.Description, r.Permissions)
	if isUnique(err) {
		return application.ErrRoleNameTaken
	}
	return err
}

func (s *Store) UpdateRole(ctx context.Context, r *domain.Role) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE staff_roles SET name=$2, description=$3, permissions=$4 WHERE id=$1`, r.ID, r.Name, r.Description, r.Permissions)
	if isUnique(err) {
		return application.ErrRoleNameTaken
	}
	if err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) DeleteRole(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM staff_roles WHERE id = $1`, id)
	if isFK(err) {
		return application.ErrRoleInUse
	}
	if err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) RolesByIDsExist(ctx context.Context, ids []string) (int, error) {
	var n int
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM staff_roles WHERE id::text = ANY($1)`, ids).Scan(&n)
	return n, err
}

// ── audit (append-only) ─────────────────────────────────────────────────────

func nullJSON(m map[string]any) any {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func nullStr(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Store) AppendAudit(ctx context.Context, e *domain.AuditEntry) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO audit_log (id, at, actor_type, actor_id, action, entity, entity_id, before, after, ip, trace_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10,$11)`,
		e.ID, e.At, e.ActorType, nullStr(e.ActorID), e.Action, e.Entity, nullStr(e.EntityID), nullJSON(e.Before), nullJSON(e.After), nullStr(e.IP), nullStr(e.TraceID))
	return err
}

type auditCursor struct {
	At time.Time `json:"a"`
	ID string    `json:"i"`
}

func (s *Store) ListAudit(ctx context.Context, f domain.AuditFilter) ([]*domain.AuditEntry, string, error) {
	conds, args := []string{"true"}, []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.Replace(cond, "?", "$"+itoa(len(args)), 1))
	}
	if f.ActorID != "" {
		add("actor_id = ?", f.ActorID)
	}
	if f.Entity != "" {
		add("entity = ?", f.Entity)
	}
	if f.EntityID != "" {
		add("entity_id = ?", f.EntityID)
	}
	if f.Action != "" {
		add("action = ?", f.Action)
	}
	if f.From != nil {
		add("at >= ?", *f.From)
	}
	if f.To != nil {
		add("at <= ?", *f.To)
	}
	if f.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(f.Cursor)
		var c auditCursor
		if err != nil || json.Unmarshal(raw, &c) != nil {
			return nil, "", errors.New("curseur invalide")
		}
		args = append(args, c.At, c.ID)
		conds = append(conds, "(at, id::text) < ($"+itoa(len(args)-1)+", $"+itoa(len(args))+")")
	}
	args = append(args, f.Limit+1)
	rows, err := s.q(ctx).Query(ctx, `SELECT id::text, at, actor_type, COALESCE(actor_id,''), action, entity, COALESCE(entity_id,''), before, after, COALESCE(ip,''), COALESCE(trace_id,'')
		FROM audit_log WHERE `+strings.Join(conds, " AND ")+` ORDER BY at DESC, id DESC LIMIT $`+itoa(len(args)), args...)
	if err != nil {
		return nil, "", err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.AuditEntry, error) {
		var e domain.AuditEntry
		var before, after []byte
		if err := r.Scan(&e.ID, &e.At, &e.ActorType, &e.ActorID, &e.Action, &e.Entity, &e.EntityID, &before, &after, &e.IP, &e.TraceID); err != nil {
			return nil, err
		}
		if before != nil {
			_ = json.Unmarshal(before, &e.Before)
		}
		if after != nil {
			_ = json.Unmarshal(after, &e.After)
		}
		return &e, nil
	})
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > f.Limit {
		items = items[:f.Limit]
		last := items[len(items)-1]
		raw, _ := json.Marshal(auditCursor{At: last.At, ID: last.ID})
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return items, next, nil
}

// ── clients et entreprises ──────────────────────────────────────────────────

var _ application.CustomerAdminRepo = (*Store)(nil)

func (s *Store) ListCustomers(ctx context.Context, query, kind, status string, limit, offset int) ([]*domain.User, int, error) {
	where, args := ` WHERE true`, []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		where += ` AND ` + strings.ReplaceAll(cond, "?", "$"+itoa(len(args)))
	}
	if query != "" {
		add(`(u.email ILIKE ? OR (u.first_name || ' ' || u.last_name) ILIKE ? OR c.legal_name ILIKE ?)`, like(query))
	}
	if kind != "" {
		add(`u.kind = ?`, kind)
	}
	if status != "" {
		add(`u.status = ?`, status)
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*)`+userFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+userCols+userFrom+where+` ORDER BY u.created_at DESC, u.id LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.User, error) { return scanUser(r) })
	return items, total, err
}

func (s *Store) UserByCompanyID(ctx context.Context, companyID string) (*domain.User, error) {
	return scanUser(s.q(ctx).QueryRow(ctx, `SELECT `+userCols+userFrom+`WHERE u.company_id = $1`, companyID))
}

func (s *Store) ListCompanies(ctx context.Context, query, status string, limit, offset int) ([]*domain.Company, int, error) {
	where, args := ` WHERE true`, []any{}
	if query != "" {
		args = append(args, like(query))
		where += ` AND (c.legal_name ILIKE $1 OR c.tax_id ILIKE $1)`
	}
	if status != "" {
		args = append(args, status)
		where += ` AND c.status = $` + itoa(len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM companies c`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := `c.created_at DESC`
	if status == domain.CompanyPending {
		order = `c.created_at ASC`
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT c.id::text, c.legal_name, c.tax_id, c.country, c.approved, c.status, c.rejection_reason, c.decided_at FROM companies c`+where+
		` ORDER BY `+order+`, c.id LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Company, error) {
		var c domain.Company
		var reason *string
		if err := r.Scan(&c.ID, &c.LegalName, &c.TaxID, &c.Country, &c.Approved, &c.Status, &reason, &c.DecidedAt); err != nil {
			return nil, err
		}
		if reason != nil {
			c.RejectionReason = *reason
		}
		return &c, nil
	})
	return items, total, err
}

func (s *Store) CountCompanies(ctx context.Context, status string) (int, error) {
	var n int
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM companies WHERE ($1 = '' OR status = $1)`, status).Scan(&n)
	return n, err
}
