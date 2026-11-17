// Package jwtx émet et vérifie les JWT (EdDSA / Ed25519) et publie les clés au format JWKS.
//
// Une clé par AUDIENCE (`customer`, `staff`, `staff-challenge`) : un jeton émis pour une audience ne peut
// pas être accepté pour une autre. Le gateway vérifie localement avec les clés de `identity.GetJwks`.
package jwtx

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Audiences du contrat.
const (
	AudCustomer       = "customer"
	AudStaff          = "staff"
	AudStaffChallenge = "staff-challenge"
)

// Claims regroupe les revendications du contrat (G3).
type Claims struct {
	jwt.RegisteredClaims
	SID           string   `json:"sid,omitempty"`            // famille de rafraîchissement (session)
	Kind          string   `json:"kind,omitempty"`           // individual | company
	Pro           bool     `json:"pro,omitempty"`            // compte entreprise VALIDÉ (affichage seulement)
	EmailVerified bool     `json:"email_verified,omitempty"` // filtrage grossier seulement
	Roles         []string `json:"roles,omitempty"`
	Perms         []string `json:"perms,omitempty"`
	AMR           []string `json:"amr,omitempty"`
	Scope         string   `json:"scope,omitempty"` // jeton 2FA : verify | enroll | confirm
}

// KeySet est une paire de clés Ed25519 identifiée par `kid`.
type KeySet struct {
	kid  string
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

// NewKeySet génère une paire éphémère (développement et tests).
func NewKeySet() (*KeySet, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return fromKeys(pub, priv), nil
}

// LoadKeySetPEM charge une clé privée Ed25519 au format PKCS#8 PEM.
func LoadKeySetPEM(pemBytes []byte) (*KeySet, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("jwtx: PEM invalide")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("jwtx: %w", err)
	}
	priv, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("jwtx: la clé n'est pas Ed25519")
	}
	return fromKeys(priv.Public().(ed25519.PublicKey), priv), nil
}

func fromKeys(pub ed25519.PublicKey, priv ed25519.PrivateKey) *KeySet {
	sum := sha256.Sum256(pub)
	return &KeySet{kid: base64.RawURLEncoding.EncodeToString(sum[:8]), priv: priv, pub: pub}
}

// KID renvoie l'identifiant de clé.
func (k *KeySet) KID() string { return k.kid }

// Sign émet un jeton pour l'audience donnée. `jti` et `iat`/`exp` sont renseignés ici.
func (k *KeySet) Sign(aud, subject string, ttl time.Duration, now time.Time, c Claims) (string, error) {
	jti := make([]byte, 12)
	if _, err := rand.Read(jti); err != nil {
		return "", err
	}
	c.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   subject,
		Audience:  jwt.ClaimStrings{aud},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		ID:        base64.RawURLEncoding.EncodeToString(jti),
		Issuer:    "filterfinder-identity",
	}
	t := jwt.NewWithClaims(jwt.SigningMethodEdDSA, c)
	t.Header["kid"] = k.kid
	return t.SignedString(k.priv)
}

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
}

// JWKS publie la clé publique au format JWKS (RFC 7517).
func (k *KeySet) JWKS() []byte {
	b, _ := json.Marshal(map[string][]jwk{"keys": {{
		Kty: "OKP", Crv: "Ed25519", X: base64.RawURLEncoding.EncodeToString(k.pub), Kid: k.kid, Use: "sig", Alg: "EdDSA",
	}}})
	return b
}

// Verifier vérifie les jetons d'UNE audience avec un jeu de clés publiques.
type Verifier struct {
	keys     map[string]ed25519.PublicKey
	audience string
	leeway   time.Duration
}

// NewVerifierFromJWKS construit un vérificateur depuis un document JWKS.
func NewVerifierFromJWKS(jwks []byte, audience string, leeway time.Duration) (*Verifier, error) {
	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(jwks, &doc); err != nil {
		return nil, fmt.Errorf("jwtx: JWKS invalide: %w", err)
	}
	v := &Verifier{keys: map[string]ed25519.PublicKey{}, audience: audience, leeway: leeway}
	for _, k := range doc.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, errors.New("jwtx: clé publique invalide")
		}
		v.keys[k.Kid] = ed25519.PublicKey(raw)
	}
	if len(v.keys) == 0 {
		return nil, errors.New("jwtx: aucune clé exploitable")
	}
	return v, nil
}

// ErrUnknownKey signale un `kid` absent : le gateway rafraîchit alors son JWKS.
var ErrUnknownKey = errors.New("jwtx: kid inconnu")

// Verify contrôle signature, algorithme (EdDSA seul), audience et dates.
func (v *Verifier) Verify(token string, now time.Time) (*Claims, error) {
	var claims Claims
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"EdDSA"}), jwt.WithAudience(v.audience), jwt.WithLeeway(v.leeway),
		jwt.WithTimeFunc(func() time.Time { return now }), jwt.WithExpirationRequired())
	_, err := parser.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		pub, ok := v.keys[kid]
		if !ok {
			return nil, ErrUnknownKey
		}
		return pub, nil
	})
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

// HasPermission : le joker `*` donne toutes les permissions ; sinon la permission doit figurer dans le jeton.
func (c *Claims) HasPermission(required string) bool {
	for _, p := range c.Perms {
		if p == "*" || p == required {
			return true
		}
	}
	return false
}
