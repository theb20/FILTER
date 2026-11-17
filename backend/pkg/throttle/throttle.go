// Package throttle implémente le RALENTISSEMENT PROGRESSIF des échecs de connexion par couple (compte, IP).
//
// Règle validée : AUCUN verrouillage de compte (un tiers pourrait bloquer un client). Après `FreeAttempts`
// échecs, chaque nouvel échec double le délai imposé avant la tentative suivante, jusqu'à `MaxDelay`.
// L'état est remis à zéro après un succès ou après `Reset` sans échec.
package throttle

import "time"

// Params sont réglables (identity-settings.loginThrottle). Défauts proposés : 3, 1 s, 900 s, 1 h.
type Params struct {
	FreeAttempts int
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Reset        time.Duration
}

// Defaults renvoie les valeurs proposées.
func Defaults() Params {
	return Params{FreeAttempts: 3, BaseDelay: time.Second, MaxDelay: 900 * time.Second, Reset: time.Hour}
}

// State est l'état d'un couple (compte, IP).
type State struct {
	Failures    int
	LastFailure time.Time
}

// Delay est le délai imposé après `failures` échecs consécutifs.
func (p Params) Delay(failures int) time.Duration {
	if failures <= p.FreeAttempts {
		return 0
	}
	exp := failures - p.FreeAttempts - 1
	d := p.BaseDelay
	for i := 0; i < exp; i++ {
		d *= 2
		if d >= p.MaxDelay {
			return p.MaxDelay
		}
	}
	if d > p.MaxDelay {
		return p.MaxDelay
	}
	return d
}

// effective ramène l'état à zéro s'il est inactif depuis plus de `Reset`.
func (p Params) effective(s State, now time.Time) State {
	if s.Failures > 0 && now.Sub(s.LastFailure) > p.Reset {
		return State{}
	}
	return s
}

// Wait renvoie le temps restant avant la prochaine tentative autorisée (0 = autorisée).
func (p Params) Wait(s State, now time.Time) time.Duration {
	s = p.effective(s, now)
	d := p.Delay(s.Failures)
	if d == 0 {
		return 0
	}
	if remaining := s.LastFailure.Add(d).Sub(now); remaining > 0 {
		return remaining
	}
	return 0
}

// RecordFailure renvoie le nouvel état après un échec.
func (p Params) RecordFailure(s State, now time.Time) State {
	s = p.effective(s, now)
	return State{Failures: s.Failures + 1, LastFailure: now}
}
