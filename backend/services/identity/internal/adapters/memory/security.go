package memory

import (
	"context"
	"sort"
	"time"

	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// secState : appareils connus, historique des connexions et images (implémentation de test de application.SecurityRepo).
// Non inclus dans l'instantané des transactions : une transaction annulée ne retire pas ces lignes (suffisant pour les tests).
type secState struct {
	devices map[string]map[string]time.Time // compte → condensat d'appareil → dernière vue
	events  []domain.LoginEvent
	images  map[string]domain.UserImage // « compte|type »
}

func (s *Store) secInit() {
	if s.sec.devices == nil {
		s.sec.devices, s.sec.images = map[string]map[string]time.Time{}, map[string]domain.UserImage{}
	}
}

func (s *Store) ActiveSessions(_ context.Context, userID string, now time.Time) ([]domain.ActiveSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	byFamily := map[string]*domain.ActiveSession{}
	first := map[string]time.Time{}
	expires := map[string]time.Time{}
	for _, t := range s.byID {
		if t.UserID != userID {
			continue
		}
		if _, revoked := s.families[t.FamilyID]; revoked {
			continue
		}
		if f, ok := first[t.FamilyID]; !ok || t.CreatedAt.Before(f) {
			first[t.FamilyID] = t.CreatedAt
		}
		if a := byFamily[t.FamilyID]; a == nil || t.CreatedAt.After(a.LastSeenAt) {
			byFamily[t.FamilyID] = &domain.ActiveSession{FamilyID: t.FamilyID, LastSeenAt: t.CreatedAt, IP: t.IP, UserAgent: t.UserAgent}
			expires[t.FamilyID] = t.ExpiresAt
		}
	}
	out := []domain.ActiveSession{}
	for id, a := range byFamily {
		if now.Before(expires[id]) {
			a.CreatedAt = first[id]
			out = append(out, *a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeenAt.After(out[j].LastSeenAt) })
	return out, nil
}

func (s *Store) RevokeUserSession(_ context.Context, userID, familyID string, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, revoked := s.families[familyID]; revoked {
		return false, nil
	}
	for _, t := range s.byID {
		if t.FamilyID == familyID && t.UserID == userID {
			s.families[familyID] = at
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) RevokeUserSessionsExcept(_ context.Context, userID, keepFamilyID string, at time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, t := range s.byID {
		if _, done := s.families[t.FamilyID]; t.UserID == userID && t.FamilyID != keepFamilyID && !done {
			s.families[t.FamilyID] = at
			n++
		}
	}
	return n, nil
}

func (s *Store) TouchDevice(_ context.Context, userID, deviceKey string, at, cutoff time.Time) (bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secInit()
	d := s.sec.devices[userID]
	if d == nil {
		d = map[string]time.Time{}
		s.sec.devices[userID] = d
	}
	for k, seen := range d {
		if seen.Before(cutoff) {
			delete(d, k)
		}
	}
	_, existed := d[deviceKey]
	others := len(d) > 0
	if existed {
		others = len(d) > 1
	}
	d[deviceKey] = at
	return existed, others, nil
}

func (s *Store) InsertLoginEvent(_ context.Context, ev *domain.LoginEvent, cutoff time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.sec.events[:0]
	for _, e := range s.sec.events {
		if !e.At.Before(cutoff) {
			kept = append(kept, e)
		}
	}
	s.sec.events = append(kept, *ev)
	return nil
}

func (s *Store) LoginEvents(_ context.Context, userID string, since time.Time, offset, limit int) ([]domain.LoginEvent, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var mine []domain.LoginEvent
	for _, e := range s.sec.events {
		if e.UserID == userID && !e.At.Before(since) {
			mine = append(mine, e)
		}
	}
	sort.Slice(mine, func(i, j int) bool { // comme PostgreSQL : date décroissante, puis identifiant (UUIDv7) décroissant
		if !mine[i].At.Equal(mine[j].At) {
			return mine[i].At.After(mine[j].At)
		}
		return mine[i].ID > mine[j].ID
	})
	total := len(mine)
	if offset >= total {
		return []domain.LoginEvent{}, total, nil
	}
	end := min(offset+limit, total)
	return mine[offset:end], total, nil
}

func (s *Store) PutImage(_ context.Context, userID string, img domain.UserImage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secInit()
	img.Data = append([]byte(nil), img.Data...)
	s.sec.images[userID+"|"+img.Kind] = img
	return nil
}

func (s *Store) Image(_ context.Context, userID, kind string) (*domain.UserImage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	img, ok := s.sec.images[userID+"|"+kind]
	if !ok {
		return nil, application.ErrNotFound
	}
	return &img, nil
}

func (s *Store) DeleteImage(_ context.Context, userID, kind string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sec.images, userID+"|"+kind)
	return nil
}

func (s *Store) ImageTimes(_ context.Context, userID string) (*time.Time, *time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var a, c *time.Time
	if img, ok := s.sec.images[userID+"|"+domain.ImageAvatar]; ok {
		t := img.UpdatedAt
		a = &t
	}
	if img, ok := s.sec.images[userID+"|"+domain.ImageCover]; ok {
		t := img.UpdatedAt
		c = &t
	}
	return a, c, nil
}
