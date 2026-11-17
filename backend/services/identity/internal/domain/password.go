package domain

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idParams sont les paramètres du hachage (réglables ; défauts OWASP : 64 Mio, 3 passes, 2 voies).
type Argon2idParams struct {
	MemoryKiB uint32
	Time      uint32
	Threads   uint8
	SaltLen   uint32
	KeyLen    uint32
}

// DefaultArgon2id : défauts de production.
func DefaultArgon2id() Argon2idParams {
	return Argon2idParams{MemoryKiB: 64 * 1024, Time: 3, Threads: 2, SaltLen: 16, KeyLen: 32}
}

// Hasher hache et vérifie les mots de passe (Argon2id, format PHC).
type Hasher struct{ P Argon2idParams }

// Hash renvoie `$argon2id$v=19$m=…,t=…,p=…$sel$condensat`.
func (h Hasher) Hash(password string) (string, error) {
	salt := make([]byte, h.P.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, h.P.Time, h.P.MemoryKiB, h.P.Threads, h.P.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, h.P.MemoryKiB, h.P.Time, h.P.Threads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// ErrBadHash signale un condensat illisible.
var ErrBadHash = errors.New("domain: condensat de mot de passe invalide")

// Verify compare en temps constant.
func (h Hasher) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrBadHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrBadHash
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, ErrBadHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrBadHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrBadHash
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// Dummy renvoie un condensat factice pour ÉGALISER le temps de réponse quand le compte n'existe pas.
func (h Hasher) Dummy() string {
	d, _ := h.Hash("dummy-password-for-timing-equalization")
	return d
}
