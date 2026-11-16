package throttle

import (
	"testing"
	"time"
)

func TestDelayProgressif(t *testing.T) {
	p := Defaults()
	want := map[int]time.Duration{0: 0, 1: 0, 3: 0, 4: time.Second, 5: 2 * time.Second, 6: 4 * time.Second, 7: 8 * time.Second}
	for n, d := range want {
		if got := p.Delay(n); got != d {
			t.Errorf("Delay(%d) = %v, attendu %v", n, got, d)
		}
	}
}

func TestDelayPlafonne(t *testing.T) {
	p := Defaults()
	if got := p.Delay(100); got != p.MaxDelay {
		t.Fatalf("Delay(100) = %v, attendu le plafond %v", got, p.MaxDelay)
	}
}

func TestAucunVerrouillageDefinitif(t *testing.T) {
	// Même après des milliers d'échecs, l'attente reste bornée par MaxDelay : le compte n'est jamais bloqué.
	p := Defaults()
	now := time.Unix(1_700_000_000, 0)
	s := State{}
	for i := 0; i < 5000; i++ {
		s = p.RecordFailure(s, now)
	}
	if w := p.Wait(s, now); w > p.MaxDelay {
		t.Fatalf("attente %v supérieure au plafond %v", w, p.MaxDelay)
	}
	if w := p.Wait(s, now.Add(p.MaxDelay+time.Second)); w != 0 {
		t.Fatalf("après le plafond, la tentative doit être autorisée (attente %v)", w)
	}
}

func TestWaitEtReinitialisationParInactivite(t *testing.T) {
	p := Defaults()
	t0 := time.Unix(1_700_000_000, 0)
	s := State{}
	for i := 0; i < 4; i++ { // 4 échecs : 1 s imposée
		s = p.RecordFailure(s, t0)
	}
	if w := p.Wait(s, t0); w != time.Second {
		t.Fatalf("attente = %v, attendu 1s", w)
	}
	if w := p.Wait(s, t0.Add(1500*time.Millisecond)); w != 0 {
		t.Fatalf("après le délai, attente = %v, attendu 0", w)
	}
	// Une heure sans échec : l'historique est oublié.
	s = p.RecordFailure(s, t0.Add(2*time.Hour))
	if s.Failures != 1 {
		t.Fatalf("Failures = %d, attendu 1 après inactivité", s.Failures)
	}
}
