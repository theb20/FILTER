package jwtx

import (
	"errors"
	"testing"
	"time"
)

var now = time.Unix(1_800_000_000, 0)

func setup(t *testing.T, aud string) (*KeySet, *Verifier) {
	t.Helper()
	k, err := NewKeySet()
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewVerifierFromJWKS(k.JWKS(), aud, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return k, v
}

func TestSignVerify(t *testing.T) {
	k, v := setup(t, AudCustomer)
	tok, err := k.Sign(AudCustomer, "user-1", 15*time.Minute, now, Claims{SID: "fam-1", Kind: "company", Pro: true, EmailVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	c, err := v.Verify(tok, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if c.Subject != "user-1" || c.SID != "fam-1" || !c.Pro || !c.EmailVerified || c.ID == "" {
		t.Fatalf("claims inattendues : %+v", c)
	}
}

func TestRefuseMauvaiseAudience(t *testing.T) {
	k, _ := setup(t, AudCustomer)
	_, vStaff := setup(t, AudStaff) // autre clé ET autre audience
	tok, _ := k.Sign(AudCustomer, "u", time.Minute, now, Claims{})
	if _, err := vStaff.Verify(tok, now); err == nil {
		t.Fatal("un jeton customer ne doit pas être accepté comme jeton staff")
	}
	// même clé mais mauvaise audience attendue
	vWrongAud, _ := NewVerifierFromJWKS(k.JWKS(), AudStaff, 0)
	if _, err := vWrongAud.Verify(tok, now); err == nil {
		t.Fatal("l'audience doit être contrôlée")
	}
}

func TestRefuseExpire(t *testing.T) {
	k, v := setup(t, AudCustomer)
	tok, _ := k.Sign(AudCustomer, "u", time.Minute, now, Claims{})
	if _, err := v.Verify(tok, now.Add(time.Hour)); err == nil {
		t.Fatal("un jeton expiré doit être refusé")
	}
	// la tolérance d'horloge (30 s) accepte un léger dépassement
	if _, err := v.Verify(tok, now.Add(time.Minute+10*time.Second)); err != nil {
		t.Fatalf("la tolérance de 30 s doit s'appliquer : %v", err)
	}
}

func TestKidInconnuSignaleLeRafraichissement(t *testing.T) {
	k1, _ := setup(t, AudCustomer)
	_, v2 := setup(t, AudCustomer)
	tok, _ := k1.Sign(AudCustomer, "u", time.Minute, now, Claims{})
	_, err := v2.Verify(tok, now)
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("attendu ErrUnknownKey, obtenu %v", err)
	}
}

func TestRefuseAlgorithmeNone(t *testing.T) {
	_, v := setup(t, AudCustomer)
	// jeton non signé (alg none) : doit être refusé
	if _, err := v.Verify("eyJhbGciOiJub25lIiwidHlwIjoiSldUIiwia2lkIjoieCJ9.eyJzdWIiOiJ1In0.", now); err == nil {
		t.Fatal("alg=none doit être refusé")
	}
}
