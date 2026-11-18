package memory

import (
	"context"
	"time"

	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var _ application.PreferencesRepo = (*Store)(nil)

func prefKey(kind domain.PrincipalKind, id string) string { return string(kind) + ":" + id }

func (s *Store) LoadPreferences(_ context.Context, kind domain.PrincipalKind, id string) (*domain.Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.prefs[prefKey(kind, id)]
	if !ok {
		return nil, nil
	}
	c := p.Clone()
	return &c, nil
}

func (s *Store) SavePreferences(_ context.Context, kind domain.PrincipalKind, id string, p domain.Preferences, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch kind {
	case domain.PrincipalStaff:
		if _, ok := s.staff.users[id]; !ok {
			return application.ErrNotFound
		}
	case domain.PrincipalCustomer:
		if _, ok := s.users[id]; !ok {
			return application.ErrNotFound
		}
	default:
		return application.ErrNotFound
	}
	s.prefs[prefKey(kind, id)] = p.Clone()
	return nil
}
