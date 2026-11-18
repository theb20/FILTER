package memory

import (
	"context"

	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var _ application.ConsentRepo = (*Store)(nil)

func (s *Store) AppendConsent(_ context.Context, c *domain.ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.UserID != "" {
		if _, ok := s.users[c.UserID]; !ok {
			return application.ErrNotFound
		}
	}
	s.consents = append(s.consents, *c)
	return nil
}

func (s *Store) LatestUserConsent(_ context.Context, userID string) (*domain.ConsentRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.consents) - 1; i >= 0; i-- {
		if s.consents[i].UserID == userID {
			c := s.consents[i]
			return &c, nil
		}
	}
	return nil, application.ErrNotFound
}
