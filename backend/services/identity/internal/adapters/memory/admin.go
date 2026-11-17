package memory

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var (
	_ application.StaffAdminRepo = (*Store)(nil)
	_ application.AuditRepo      = (*Store)(nil)
)

func containsFold(h string, q string) bool {
	return strings.Contains(strings.ToLower(h), strings.ToLower(q))
}

func window[T any](all []T, limit, offset int) []T {
	if offset >= len(all) {
		return nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}

func (s *Store) ListStaff(_ context.Context, query string, limit, offset int) ([]*domain.StaffUser, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []*domain.StaffUser
	for _, u := range s.staff.users {
		if query == "" || containsFold(u.Email, query) || containsFold(u.Name, query) {
			all = append(all, cloneStaff(u))
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		}
		return all[i].ID < all[j].ID
	})
	return window(all, limit, offset), len(all), nil
}

func (s *Store) SetStaffRoles(_ context.Context, staffID string, roleIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.staff.users[staffID]
	if !ok {
		return application.ErrNotFound
	}
	u.RoleIDs = append([]string(nil), roleIDs...)
	return nil
}

func (s *Store) ClearTwoFactor(_ context.Context, staffID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.staff.users[staffID]
	if !ok {
		return application.ErrNotFound
	}
	u.TOTPSecret, u.TwoFactorEnabled, u.TOTPLastStep = nil, false, 0
	delete(s.staff.recovery, staffID)
	return nil
}

func (s *Store) PutStaffInvitation(_ context.Context, hash, staffID string, expires time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for h, inv := range s.staff.invitations {
		if inv.staffID == staffID && inv.usedAt == nil {
			delete(s.staff.invitations, h)
		}
	}
	s.staff.invitations[hash] = &invitation{staffID: staffID, expires: expires}
	return nil
}

func (s *Store) ConsumeStaffInvitation(_ context.Context, hash string, now time.Time) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.staff.invitations[hash]
	if !ok || inv.usedAt != nil || !now.Before(inv.expires) {
		return "", false, nil
	}
	inv.usedAt = &now
	return inv.staffID, true, nil
}

func cloneRole(r *domain.Role) domain.Role {
	c := *r
	c.Permissions = append([]string(nil), r.Permissions...)
	return c
}

func (s *Store) ListRoles(_ context.Context, query string, limit, offset int) ([]domain.Role, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []domain.Role
	for _, r := range s.staff.roles {
		if query == "" || containsFold(r.Name, query) || containsFold(r.Description, query) {
			all = append(all, cloneRole(r))
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].System != all[j].System {
			return all[i].System
		}
		return strings.ToLower(all[i].Name) < strings.ToLower(all[j].Name)
	})
	return window(all, limit, offset), len(all), nil
}

func (s *Store) RoleByID(_ context.Context, id string) (*domain.Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.staff.roles[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	c := cloneRole(r)
	return &c, nil
}

func (s *Store) roleNameTaken(name, exceptID string) bool {
	for id, r := range s.staff.roles {
		if id != exceptID && strings.EqualFold(r.Name, name) {
			return true
		}
	}
	return false
}

func (s *Store) CreateRole(_ context.Context, r *domain.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roleNameTaken(r.Name, "") {
		return application.ErrRoleNameTaken
	}
	c := cloneRole(r)
	c.System = false
	s.staff.roles[r.ID] = &c
	return nil
}

func (s *Store) UpdateRole(_ context.Context, r *domain.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.staff.roles[r.ID]
	if !ok {
		return application.ErrNotFound
	}
	if s.roleNameTaken(r.Name, r.ID) {
		return application.ErrRoleNameTaken
	}
	cur.Name, cur.Description, cur.Permissions = r.Name, r.Description, append([]string(nil), r.Permissions...)
	return nil
}

func (s *Store) DeleteRole(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.staff.roles[id]; !ok {
		return application.ErrNotFound
	}
	for _, u := range s.staff.users {
		for _, rid := range u.RoleIDs {
			if rid == id {
				return application.ErrRoleInUse
			}
		}
	}
	delete(s.staff.roles, id)
	return nil
}

func (s *Store) RolesByIDsExist(_ context.Context, ids []string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, id := range ids {
		if _, ok := s.staff.roles[id]; ok {
			n++
		}
	}
	return n, nil
}

// ── audit ───────────────────────────────────────────────────────────────────

func (s *Store) AppendAudit(_ context.Context, e *domain.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := *e
	s.staff.audit = append(s.staff.audit, &c)
	return nil
}

type memCursor struct {
	At time.Time `json:"a"`
	ID string    `json:"i"`
}

func (s *Store) ListAudit(_ context.Context, f domain.AuditFilter) ([]*domain.AuditEntry, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []*domain.AuditEntry
	for _, e := range s.staff.audit {
		if (f.ActorID != "" && e.ActorID != f.ActorID) || (f.Entity != "" && e.Entity != f.Entity) || (f.EntityID != "" && e.EntityID != f.EntityID) || (f.Action != "" && e.Action != f.Action) ||
			(f.From != nil && e.At.Before(*f.From)) || (f.To != nil && e.At.After(*f.To)) {
			continue
		}
		c := *e
		all = append(all, &c)
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].At.Equal(all[j].At) {
			return all[i].At.After(all[j].At)
		}
		return all[i].ID > all[j].ID
	})
	if f.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(f.Cursor)
		var c memCursor
		if err != nil || json.Unmarshal(raw, &c) != nil {
			return nil, "", errors.New("curseur invalide")
		}
		i := sort.Search(len(all), func(i int) bool {
			return all[i].At.Before(c.At) || (all[i].At.Equal(c.At) && all[i].ID < c.ID)
		})
		all = all[i:]
	}
	next := ""
	if len(all) > f.Limit {
		all = all[:f.Limit]
		last := all[len(all)-1]
		raw, _ := json.Marshal(memCursor{At: last.At, ID: last.ID})
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return all, next, nil
}

// ── clients et entreprises ──────────────────────────────────────────────────

var _ application.CustomerAdminRepo = (*Store)(nil)

func (s *Store) ListCustomers(_ context.Context, query, kind, status string, limit, offset int) ([]*domain.User, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []*domain.User
	for _, u := range s.users {
		if kind != "" && u.Kind != kind || status != "" && u.Status != status {
			continue
		}
		if query != "" && !containsFold(u.Email, query) && !containsFold(u.FirstName+" "+u.LastName, query) &&
			!(u.Company != nil && containsFold(u.Company.LegalName, query)) {
			continue
		}
		all = append(all, cloneUser(u))
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		}
		return all[i].ID < all[j].ID
	})
	return window(all, limit, offset), len(all), nil
}

func (s *Store) UserByCompanyID(_ context.Context, companyID string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Company != nil && u.Company.ID == companyID {
			return cloneUser(u), nil
		}
	}
	return nil, application.ErrNotFound
}

type companyRow struct {
	c       domain.Company
	created time.Time
}

func (s *Store) companyRows(query, status string) []companyRow {
	var rows []companyRow
	for _, u := range s.users {
		c := u.Company
		if c == nil || status != "" && c.State() != status {
			continue
		}
		if query != "" && !containsFold(c.LegalName, query) && !containsFold(c.TaxID, query) {
			continue
		}
		rows = append(rows, companyRow{*c, u.CreatedAt})
	}
	return rows
}

func (s *Store) ListCompanies(_ context.Context, query, status string, limit, offset int) ([]*domain.Company, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.companyRows(query, status)
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].created.Equal(rows[j].created) {
			if status == domain.CompanyPending {
				return rows[i].created.Before(rows[j].created)
			}
			return rows[i].created.After(rows[j].created)
		}
		return rows[i].c.ID < rows[j].c.ID
	})
	var all []*domain.Company
	for i := range rows {
		c := rows[i].c
		all = append(all, &c)
	}
	return window(all, limit, offset), len(all), nil
}

func (s *Store) CountCompanies(_ context.Context, status string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.companyRows("", status)), nil
}
