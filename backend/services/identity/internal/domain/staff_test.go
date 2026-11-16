package domain

import (
	"testing"
	"time"
)

// Vecteurs de test officiels de la RFC 6238 (annexe B, SHA-1, secret « 12345678901234567890 », 8 chiffres → on compare les 6 derniers).
func TestTOTPVecteursRFC6238(t *testing.T) {
	secret := []byte("12345678901234567890")
	for _, c := range []struct {
		unix int64
		want string
	}{{59, "287082"}, {1111111109, "081804"}, {1111111111, "050471"}, {1234567890, "005924"}, {2000000000, "279037"}} {
		if got := TOTPCode(secret, c.unix/30); got != c.want {
			t.Errorf("t=%d : %s, attendu %s", c.unix, got, c.want)
		}
	}
}

func TestTOTPFenetreEtUsageUnique(t *testing.T) {
	s, _ := NewTOTPSecret()
	now := time.Unix(1_800_000_000, 0)
	step := now.Unix() / 30
	if _, ok := VerifyTOTP(s, TOTPCode(s, step), now, 0); !ok {
		t.Fatal("code courant refusé")
	}
	if _, ok := VerifyTOTP(s, TOTPCode(s, step-1), now, 0); !ok {
		t.Fatal("code du pas précédent refusé (dérive tolérée)")
	}
	if _, ok := VerifyTOTP(s, TOTPCode(s, step-2), now, 0); ok {
		t.Fatal("code vieux de 2 pas accepté")
	}
	if _, ok := VerifyTOTP(s, TOTPCode(s, step), now, step); ok {
		t.Fatal("un code déjà utilisé ne doit pas resservir")
	}
	if _, ok := VerifyTOTP(s, "000000", now, 0); ok && TOTPCode(s, step) != "000000" {
		t.Fatal("code faux accepté")
	}
}

func TestSecretBoxLieAuProprietaire(t *testing.T) {
	b, err := NewSecretBox(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	sealed, _ := b.Seal([]byte("secret"), "staff-1")
	if got, err := b.Open(sealed, "staff-1"); err != nil || string(got) != "secret" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := b.Open(sealed, "staff-2"); err == nil {
		t.Fatal("un secret recopié vers un autre compte doit être refusé")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := b.Open(sealed, "staff-1"); err == nil {
		t.Fatal("un chiffré altéré doit être refusé")
	}
	if _, err := NewSecretBox(make([]byte, 16)); err == nil {
		t.Fatal("clé trop courte acceptée")
	}
}

func TestCodesDeSecours(t *testing.T) {
	plain, hashes, err := NewRecoveryCodes(8)
	if err != nil || len(plain) != 8 || len(hashes) != 8 {
		t.Fatal(err)
	}
	for i, p := range plain {
		if !LooksLikeRecoveryCode(p) || hashes[i] != HashToken(NormalizeRecoveryCode(p)) || plain[i] == hashes[i] {
			t.Fatalf("code %d incohérent : %s", i, p)
		}
	}
	if LooksLikeRecoveryCode("123456") {
		t.Fatal("un code TOTP n'est pas un code de secours")
	}
}
