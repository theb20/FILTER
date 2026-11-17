package application

import (
	"context"
	"sync"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// DBSettings est un SettingsProvider alimenté par la base : lecture mise en cache (`TTL`), rafraîchie à l'écriture sur
// cette instance et au plus tard après `TTL` sur les autres instances. En cas d'erreur de lecture, la dernière valeur connue
// (ou les défauts) est conservée : un incident de base ne doit pas changer les durées de session.
type DBSettings struct {
	Repo SettingsRepo
	Base domain.Settings // paramètres non modifiables par le contrat
	TTL  time.Duration
	Now  func() time.Time

	mu      sync.Mutex
	value   domain.Settings
	loaded  bool
	fetched time.Time
}

func (d *DBSettings) Get(ctx context.Context) domain.Settings {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now
	if d.Now != nil {
		now = d.Now
	}
	if d.loaded && d.TTL > 0 && now().Sub(d.fetched) < d.TTL {
		return d.value
	}
	if s, err := d.Repo.LoadIdentitySettings(ctx); err == nil {
		d.value, d.loaded, d.fetched = s.Apply(d.Base), true, now()
	} else if !d.loaded {
		d.value, d.loaded, d.fetched = d.Base, true, now().Add(-d.TTL+time.Second) // défauts, nouvel essai dans 1 s
	}
	return d.value
}

// Invalidate force la relecture (après une écriture).
func (d *DBSettings) Invalidate() {
	d.mu.Lock()
	d.loaded = false
	d.mu.Unlock()
}

// ── cas d'usage ─────────────────────────────────────────────────────────────

func (s *Service) GetIdentitySettings(ctx context.Context) (domain.IdentitySettings, error) {
	v, err := s.SettingsStore.LoadIdentitySettings(ctx)
	if err != nil {
		return domain.IdentitySettings{}, apperr.Internal(err)
	}
	return v, nil
}

// PutIdentitySettings remplace les réglages (validés), en une transaction avec l'audit avant/après.
func (s *Service) PutIdentitySettings(ctx context.Context, a Actor, in domain.IdentitySettings) (domain.IdentitySettings, error) {
	if err := in.Validate(); err != nil {
		return domain.IdentitySettings{}, err
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		before, err := s.SettingsStore.LoadIdentitySettings(ctx)
		if err != nil {
			return apperr.Internal(err)
		}
		if err := s.SettingsStore.SaveIdentitySettings(ctx, in, a.ID, s.Now()); err != nil {
			return apperr.Internal(err)
		}
		return s.audit(ctx, a, "settings.update", "identity_settings", "identity", before.Snapshot(), in.Snapshot())
	})
	if err != nil {
		return domain.IdentitySettings{}, err
	}
	if inv, ok := s.Settings.(interface{ Invalidate() }); ok {
		inv.Invalidate()
	}
	return in, nil
}
