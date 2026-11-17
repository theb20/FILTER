package domain

import (
	"strings"
	"testing"
)

func lightHasher() Hasher {
	return Hasher{P: Argon2idParams{MemoryKiB: 64, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}}
}

func TestArgon2idHashEtVerification(t *testing.T) {
	h := lightHasher()
	enc, err := h.Hash("un mot de passe")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("format PHC inattendu : %s", enc)
	}
	if ok, _ := h.Verify("un mot de passe", enc); !ok {
		t.Fatal("le bon mot de passe doit être accepté")
	}
	if ok, _ := h.Verify("un autre", enc); ok {
		t.Fatal("un mauvais mot de passe doit être refusé")
	}
	enc2, _ := h.Hash("un mot de passe")
	if enc == enc2 {
		t.Fatal("deux hachages du même mot de passe doivent différer (sel aléatoire)")
	}
}

func TestVerifyRefuseUnCondensatIllisible(t *testing.T) {
	h := lightHasher()
	for _, bad := range []string{"", "plaintext", "$bcrypt$x$y", "$argon2id$v=19$m=x$a$b", "$argon2id$v=18$m=64,t=1,p=1$YQ$YQ"} {
		if _, err := h.Verify("x", bad); err == nil {
			t.Errorf("%q devrait être refusé", bad)
		}
	}
}

func TestParametresDeProductionOWASP(t *testing.T) {
	p := DefaultArgon2id()
	if p.MemoryKiB < 19*1024 || p.Time < 2 {
		t.Fatalf("paramètres trop faibles pour la production : %+v", p)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got, err := NormalizeEmail("  Awa.Kone@Exemple.CI "); err != nil || got != "awa.kone@exemple.ci" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "sans-arobase", "a b@c.ci", "Nom <a@b.ci>", "a@"} {
		if _, err := NormalizeEmail(bad); err == nil {
			t.Errorf("%q devrait être refusée", bad)
		}
	}
}

func TestJetonOpaqueEtCondensat(t *testing.T) {
	p1, h1, _ := NewOpaqueToken()
	p2, _, _ := NewOpaqueToken()
	if p1 == p2 || len(p1) < 40 {
		t.Fatal("jetons faibles ou identiques")
	}
	if h1 != HashToken(p1) || h1 == p1 {
		t.Fatal("seul le condensat est stocké et il doit être reproductible")
	}
}
