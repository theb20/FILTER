package memory

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// Données du personnel (même unité de travail que le reste du Store).
type staffData struct {
	users       map[string]*domain.StaffUser
	byEmail     map[string]string
	roles       map[string]*domain.Role
	challenges  map[string]*domain.Challenge
	recovery    map[string]map[string]*time.Time // compte → condensat → date d'usage
	invitations map[string]*invitation
	audit       []*domain.AuditEntry
}

type invitation struct {
	staffID string
	expires time.Time
	usedAt  *time.Time
}

func newStaffData() *staffData {
	d := &staffData{users: map[string]*domain.StaffUser{}, byEmail: map[string]string{}, roles: map[string]*domain.Role{}, challenges: map[string]*domain.Challenge{}, recovery: map[string]map[string]*time.Time{}, invitations: map[string]*invitation{}}
	id := uuid.NewString()
	d.roles[id] = &domain.Role{ID: id, Name: domain.SuperAdminRoleName, Permissions: []string{domain.AllPermissions}, System: true}
	return d
}

func (d *staffData) clone() *staffData {
	c := &staffData{users: map[string]*domain.StaffUser{}, byEmail: map[string]string{}, roles: map[string]*domain.Role{}, challenges: map[string]*domain.Challenge{}, recovery: map[string]map[string]*time.Time{}, invitations: map[string]*invitation{}}
	for k, v := range d.users {
		u := *v
		u.RoleIDs = slices.Clone(v.RoleIDs)
		u.TOTPSecret = slices.Clone(v.TOTPSecret)
		c.users[k] = &u
	}
	for k, v := range d.byEmail {
		c.byEmail[k] = v
	}
	for k, v := range d.roles {
		r := *v
		r.Permissions = append([]string(nil), v.Permissions...)
		c.roles[k] = &r
	}
	for k, v := range d.challenges {
		ch := *v
		c.challenges[k] = &ch
	}
	for k, v := range d.invitations {
		iv := *v
		c.invitations[k] = &iv
	}
	c.audit = append([]*domain.AuditEntry(nil), d.audit...)
	for k, m := range d.recovery {
		c.recovery[k] = map[string]*time.Time{}
		for h, t := range m {
			c.recovery[k][h] = t
		}
	}
	return c
}

// StaffSessions renvoie le dépôt des sessions du personnel (jetons séparés de ceux des clients).
func (s *Store) StaffSessions() application.SessionRepo { return s.staffSess }

func cloneStaff(u *domain.StaffUser) *domain.StaffUser {
	c := *u
	c.RoleIDs = slices.Clone(u.RoleIDs)
	c.TOTPSecret = slices.Clone(u.TOTPSecret)
	return &c
}

func (s *Store) CreateStaff(_ context.Context, u *domain.StaffUser) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.staff.byEmail[u.Email]; ok {
		return application.ErrEmailTaken
	}
	s.staff.users[u.ID] = cloneStaff(u)
	s.staff.byEmail[u.Email] = u.ID
	return nil
}

func (s *Store) StaffByEmail(_ context.Context, email string) (*domain.StaffUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.staff.byEmail[email]
	if !ok {
		return nil, application.ErrNotFound
	}
	return cloneStaff(s.staff.users[id]), nil
}

func (s *Store) StaffByID(_ context.Context, id string) (*domain.StaffUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.staff.users[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return cloneStaff(u), nil
}

// UpdateStaff ne touche JAMAIS à TOTPLastStep (réservé à AdvanceTOTPStep, atomique), comme l'adaptateur PostgreSQL.
func (s *Store) UpdateStaff(_ context.Context, u *domain.StaffUser) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.staff.users[u.ID]
	if !ok {
		return application.ErrNotFound
	}
	last := cur.TOTPLastStep
	n := cloneStaff(u)
	n.TOTPLastStep = last
	s.staff.users[u.ID] = n
	return nil
}

func (s *Store) AdvanceTOTPStep(_ context.Context, id string, step int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.staff.users[id]
	if !ok || step <= u.TOTPLastStep {
		return false, nil
	}
	u.TOTPLastStep = step
	return true, nil
}

func (s *Store) SetRecoveryCodes(_ context.Context, id string, hashes []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := map[string]*time.Time{}
	for _, h := range hashes {
		m[h] = nil
	}
	s.staff.recovery[id] = m
	return nil
}

func (s *Store) ConsumeRecoveryCode(_ context.Context, id, hash string, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	used, ok := s.staff.recovery[id][hash]
	if !ok || used != nil {
		return false, nil
	}
	s.staff.recovery[id][hash] = &at
	return true, nil
}

func (s *Store) RolesByIDs(_ context.Context, ids []string) ([]domain.Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.Role
	for _, id := range ids {
		if r, ok := s.staff.roles[id]; ok {
			out = append(out, cloneRole(r))
		}
	}
	return out, nil
}

func (s *Store) RoleByName(_ context.Context, name string) (*domain.Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.staff.roles {
		if r.Name == name {
			c := *r
			return &c, nil
		}
	}
	return nil, application.ErrNotFound
}

func (s *Store) CountActiveWithRole(_ context.Context, roleID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, u := range s.staff.users {
		if u.Status == domain.StaffActive && slices.Contains(u.RoleIDs, roleID) {
			n++
		}
	}
	return n, nil
}

// ── ChallengeRepo ───────────────────────────────────────────────────────────

func (s *Store) PutChallenge(_ context.Context, c *domain.Challenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cc := *c
	s.staff.challenges[c.ID] = &cc
	return nil
}

func (s *Store) ChallengeByID(_ context.Context, id string) (*domain.Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.staff.challenges[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	cc := *c
	return &cc, nil
}

func (s *Store) ConsumeChallenge(_ context.Context, id string, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.staff.challenges[id]
	if !ok || c.UsedAt != nil || !now.Before(c.ExpiresAt) {
		return false, nil
	}
	c.UsedAt = &now
	return true, nil
}

func (s *Store) FailChallenge(_ context.Context, id string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.staff.challenges[id]
	if !ok {
		return 0, application.ErrNotFound
	}
	c.Failures++
	return c.Failures, nil
}
