// Package secretbox chiffre les données sensibles au repos (AES-256-GCM, nonce aléatoire en préfixe). La clé vient d'un FICHIER (jamais de la base,
// d'une variable lisible dans les journaux, ni d'un argument de commande). Il n'y a volontairement pas de méthode de déchiffrement dans l'application :
// les numéros de compte bancaire complets ne sont jamais relus.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Box struct{ aead cipher.AEAD }

// FromKey construit une boîte depuis une clé de 32 octets.
func FromKey(key []byte) (*Box, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("la clé doit faire 32 octets (%d fournis)", len(key))
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	g, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	return &Box{aead: g}, nil
}

// FromFile lit une clé base64 de 32 octets. Le fichier ne doit être lisible que par son propriétaire.
func FromFile(path string) (*Box, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("permissions trop larges sur %s (attendu 0600)", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, errors.New("clé illisible (base64 attendu)")
	}
	return FromKey(key)
}

func (b *Box) Seal(plain []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, plain, nil), nil
}

// Open n'est utilisé que par les tests et d'éventuels outils d'exploitation ; l'application ne l'appelle pas.
func (b *Box) Open(sealed []byte) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(sealed) < n {
		return nil, errors.New("données chiffrées invalides")
	}
	return b.aead.Open(nil, sealed[:n], sealed[n:], nil)
}
