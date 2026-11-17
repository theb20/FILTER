package server

import (
	"sync"
	"time"
)

// Limiter limite le débit par clé. L'implémentation Redis (partagée entre instances) suit ; l'interface est stable.
type Limiter interface {
	// Allow enregistre une requête et indique si elle est autorisée, combien il en reste dans la fenêtre,
	// et le délai avant la prochaine autorisation si elle est refusée.
	Allow(key string, limit int, window time.Duration, now time.Time) (ok bool, remaining int, retryAfter time.Duration)
}

// MemoryLimiter : fenêtre glissante en mémoire (une seule instance : développement et tests).
type MemoryLimiter struct {
	mu    sync.Mutex
	hits  map[string][]time.Time
	calls int
}

func NewMemoryLimiter() *MemoryLimiter { return &MemoryLimiter{hits: map[string][]time.Time{}} }

func (m *MemoryLimiter) Allow(key string, limit int, window time.Duration, now time.Time) (bool, int, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.calls%4096 == 0 { // balayage périodique : borne la mémoire
		for k, v := range m.hits {
			if len(v) == 0 || now.Sub(v[len(v)-1]) > 2*time.Hour {
				delete(m.hits, k)
			}
		}
	}
	kept := m.hits[key][:0:0]
	for _, t := range m.hits[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		m.hits[key] = kept
		return false, 0, kept[0].Add(window).Sub(now)
	}
	kept = append(kept, now)
	m.hits[key] = kept
	return true, limit - len(kept), 0
}
