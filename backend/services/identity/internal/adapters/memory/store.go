// Package memory est l'adaptateur de persistance EN MÉMOIRE (tests et développement sans base).
// L'adaptateur PostgreSQL (migrations versionnées, table outbox) suit ; il implémente les mêmes ports.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/theb20/FILTER/backend/pkg/throttle"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// Store implémente UserRepo, SessionRepo, TokenRepo et FailureStore.
type Store struct {
	mu       sync.Mutex
	users    map[string]*domain.User // par id
	byEmail  map[string]string
	tokens   map[string]*domain.RefreshToken // par condensat
	byID     map[string]*domain.RefreshToken
	oneTime  map[string]*domain.OneTimeToken
	pairs    map[string]throttle.State
	failures map[string][]time.Time
	alerted  map[string]time.Time
}

// New crée un Store vide.
func New() *Store {
	return &Store{users: map[string]*domain.User{}, byEmail: map[string]string{}, tokens: map[string]*domain.RefreshToken{}, byID: map[string]*domain.RefreshToken{},
		oneTime: map[string]*domain.OneTimeToken{}, pairs: map[string]throttle.State{}, failures: map[string][]time.Time{}, alerted: map[string]time.Time{}}
}

func cp[T any](v T) *T { c := v; return &c }

// ── UserRepo ────────────────────────────────────────────────────────────────

func (s *Store) Create(_ context.Context, u *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byEmail[u.Email]; ok {
		return application.ErrEmailTaken
	}
	s.users[u.ID] = cp(*u)
	s.byEmail[u.Email] = u.ID
	return nil
}

func (s *Store) ByEmail(_ context.Context, email string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byEmail[email]
	if !ok {
		return nil, application.ErrNotFound
	}
	return cp(*s.users[id]), nil
}

func (s *Store) ByID(_ context.Context, id string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return cp(*u), nil
}

func (s *Store) Update(_ context.Context, u *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[u.ID]; !ok {
		return application.ErrNotFound
	}
	s.users[u.ID] = cp(*u)
	return nil
}

// ── SessionRepo ─────────────────────────────────────────────────────────────

func (s *Store) CreateSession(_ context.Context, t *domain.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := cp(*t)
	s.tokens[t.Hash] = c
	s.byID[t.ID] = c
	return nil
}

func (s *Store) SessionByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[hash]
	if !ok {
		return nil, application.ErrNotFound
	}
	return cp(*t), nil
}

func (s *Store) MarkUsed(_ context.Context, id string, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.byID[id]
	if !ok || t.UsedAt != nil {
		return false, nil
	}
	t.UsedAt = &at
	return true, nil
}

func (s *Store) RevokeFamily(_ context.Context, familyID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.byID {
		if t.FamilyID == familyID && t.RevokedAt == nil {
			t.RevokedAt = &at
		}
	}
	return nil
}

func (s *Store) RevokeAllExcept(_ context.Context, userID, keepFamilyID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.byID {
		if t.UserID == userID && t.FamilyID != keepFamilyID && t.RevokedAt == nil {
			t.RevokedAt = &at
		}
	}
	return nil
}

// ── TokenRepo ───────────────────────────────────────────────────────────────

func (s *Store) Put(_ context.Context, t *domain.OneTimeToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.oneTime[t.Purpose+"|"+t.Hash] = cp(*t)
	return nil
}

func (s *Store) Consume(_ context.Context, purpose, hash string, now time.Time) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.oneTime[purpose+"|"+hash]
	if !ok || t.UsedAt != nil || !now.Before(t.ExpiresAt) {
		return "", false, nil
	}
	t.UsedAt = &now
	return t.UserID, true, nil
}

// ── FailureStore ────────────────────────────────────────────────────────────

func (s *Store) Pair(_ context.Context, key string) (throttle.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pairs[key], nil
}

func (s *Store) SetPair(_ context.Context, key string, st throttle.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pairs[key] = st
	return nil
}

func (s *Store) ResetPair(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pairs, key)
	return nil
}

func (s *Store) AccountFailures(_ context.Context, email string, at time.Time, window time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.failures[email][:0:0]
	for _, t := range s.failures[email] {
		if at.Sub(t) <= window {
			kept = append(kept, t)
		}
	}
	kept = append(kept, at)
	s.failures[email] = kept
	return len(kept), nil
}

func (s *Store) MarkAlerted(_ context.Context, email string, now, until time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.alerted[email]; ok && now.Before(u) {
		return false, nil
	}
	s.alerted[email] = until
	return true, nil
}
