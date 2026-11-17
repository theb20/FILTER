package memory

import (
	"context"
	"time"

	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var _ application.SettingsRepo = (*Store)(nil)

func (s *Store) LoadIdentitySettings(context.Context) (domain.IdentitySettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settings == nil {
		return domain.DefaultIdentitySettings(), nil
	}
	return *s.settings, nil
}

func (s *Store) SaveIdentitySettings(_ context.Context, v domain.IdentitySettings, _ string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := v
	s.settings = &c
	return nil
}
