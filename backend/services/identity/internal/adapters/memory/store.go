// Package memory est l'adaptateur de persistance EN MÉMOIRE : tests unitaires et développement sans base.
// Il implémente les mêmes ports que l'adaptateur PostgreSQL (package postgres), y compris l'unité de travail
// (annulation complète des écritures ET des événements en cas d'erreur). Jamais utilisé en production.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/theb20/FILTER/backend/pkg/events"
	"github.com/theb20/FILTER/backend/pkg/throttle"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// Store implémente UserRepo, SessionRepo, TokenRepo et FailureStore.
type Store struct {
	mu        sync.Mutex
	users     map[string]*domain.User // par id
	byEmail   map[string]string
	tokens    map[string]*domain.RefreshToken // par condensat
	byID      map[string]*domain.RefreshToken
	oneTime   map[string]*domain.OneTimeToken
	pairs     map[string]throttle.State
	failures  map[string][]time.Time
	alerted   map[string]time.Time
	families  map[string]time.Time // familles révoquées
	staff     *staffData
	settings  *domain.IdentitySettings
	staffSess *Store // sessions du personnel : mêmes mécanismes, données séparées

	// Events : si renseigné, les événements publiés dans une transaction annulée sont retirés.
	Events *events.Memory
	txMu   sync.Mutex // une transaction à la fois (comme un verrou global : suffisant pour des tests)
}

type txKey struct{}

// New crée un Store vide.
func New() *Store {
	s := newBare()
	s.staff = newStaffData()
	s.staffSess = newBare()
	return s
}

func newBare() *Store {
	return &Store{users: map[string]*domain.User{}, byEmail: map[string]string{}, tokens: map[string]*domain.RefreshToken{}, byID: map[string]*domain.RefreshToken{},
		oneTime: map[string]*domain.OneTimeToken{}, pairs: map[string]throttle.State{}, failures: map[string][]time.Time{}, alerted: map[string]time.Time{}, families: map[string]time.Time{}}
}

// Do exécute fn de façon atomique : en cas d'erreur (ou de panique), l'état et les événements reviennent à ce qu'ils
// étaient. Un appel imbriqué rejoint la transaction en cours.
func (s *Store) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	s.txMu.Lock()
	defer s.txMu.Unlock()
	s.mu.Lock()
	snap := s.snapshot()
	s.mu.Unlock()
	evN := 0
	if s.Events != nil {
		evN = s.Events.Len()
	}
	done := false
	defer func() {
		if !done || err != nil {
			s.mu.Lock()
			s.restore(snap)
			s.mu.Unlock()
			if s.Events != nil {
				s.Events.Truncate(evN)
			}
		}
	}()
	err = fn(context.WithValue(ctx, txKey{}, true))
	done = true
	return err
}

func (s *Store) snapshot() *Store {
	c := newBare()
	for k, v := range s.users {
		c.users[k] = cloneUser(v)
	}
	for k, v := range s.byEmail {
		c.byEmail[k] = v
	}
	for k, v := range s.byID {
		t := cp(*v)
		c.byID[k] = t
		c.tokens[t.Hash] = t
	}
	for k, v := range s.oneTime {
		c.oneTime[k] = cp(*v)
	}
	for k, v := range s.pairs {
		c.pairs[k] = v
	}
	for k, v := range s.failures {
		c.failures[k] = append([]time.Time(nil), v...)
	}
	for k, v := range s.alerted {
		c.alerted[k] = v
	}
	for k, v := range s.families {
		c.families[k] = v
	}
	if s.settings != nil {
		v := *s.settings
		c.settings = &v
	}
	if s.staff != nil {
		c.staff = s.staff.clone()
	}
	if s.staffSess != nil {
		c.staffSess = s.staffSess.snapshot()
	}
	return c
}

func (s *Store) restore(c *Store) {
	s.users, s.byEmail, s.tokens, s.byID, s.oneTime = c.users, c.byEmail, c.tokens, c.byID, c.oneTime
	s.pairs, s.failures, s.alerted, s.families = c.pairs, c.failures, c.alerted, c.families
	s.settings = c.settings
	if c.staff != nil {
		s.staff = c.staff
	}
	if c.staffSess != nil {
		s.staffSess.restore(c.staffSess)
	}
}

func cp[T any](v T) *T { c := v; return &c }

// cloneUser copie aussi l'entreprise (sinon un appelant modifierait le magasin sans passer par Update).
func cloneUser(u *domain.User) *domain.User {
	c := *u
	if u.Company != nil {
		co := *u.Company
		c.Company = &co
	}
	return &c
}

// ── UserRepo ────────────────────────────────────────────────────────────────

func (s *Store) Create(_ context.Context, u *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byEmail[u.Email]; ok {
		return application.ErrEmailTaken
	}
	s.users[u.ID] = cloneUser(u)
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
	return cloneUser(s.users[id]), nil
}

func (s *Store) ByID(_ context.Context, id string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, application.ErrNotFound
	}
	return cloneUser(u), nil
}

func (s *Store) Update(_ context.Context, u *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[u.ID]; !ok {
		return application.ErrNotFound
	}
	s.users[u.ID] = cloneUser(u)
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
	c := cp(*t)
	if at, revoked := s.families[t.FamilyID]; revoked && c.RevokedAt == nil {
		c.RevokedAt = &at
	}
	return c, nil
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
	if _, ok := s.families[familyID]; !ok {
		s.families[familyID] = at
	}
	return nil
}

func (s *Store) RevokeAllExcept(_ context.Context, userID, keepFamilyID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.byID {
		if _, done := s.families[t.FamilyID]; t.UserID == userID && t.FamilyID != keepFamilyID && !done {
			s.families[t.FamilyID] = at
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
