package domain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // exigé par RFC 6238 (TOTP) : HMAC-SHA1 est le défaut universel des applications d'authentification
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Statuts du personnel (voir StaffStatus du contrat). Désactivation, jamais de suppression.
const (
	StaffInvited     = "invited"
	StaffActive      = "active"
	StaffDeactivated = "deactivated"
)

// Rôle système créé à l'installation. « Suggéré, à confirmer » dans le contrat.
const SuperAdminRoleName = "super-admin"

// AllPermissions : permission joker du super-admin.
const AllPermissions = "*"

// Role est un rôle du personnel avec ses permissions (`catalog:read`, `content:write`…).
type Role struct {
	ID          string
	Name        string
	Description string // vide = aucune
	Permissions []string
	System      bool
}

// StaffUser est un compte du personnel (pool SÉPARÉ des clients).
type StaffUser struct {
	ID               string
	Email            string
	Name             string
	PasswordHash     string
	Status           string
	RoleIDs          []string
	TOTPSecret       []byte // chiffré au repos (SecretBox) ; jamais renvoyé par une API
	TwoFactorEnabled bool
	TOTPLastStep     int64 // dernier pas TOTP accepté : un code ne sert qu'une fois
	LastLoginAt      *time.Time
	CreatedAt        time.Time
}

// Portées d'un jeton de défi 2FA.
const (
	ScopeVerify  = "verify"
	ScopeEnroll  = "enroll"
	ScopeConfirm = "confirm"
)

// MaxChallengeFailures : codes erronés tolérés par jeton de défi (contrat : 5).
const MaxChallengeFailures = 5

// ── TOTP (RFC 6238) ─────────────────────────────────────────────────────────

const (
	totpPeriod = 30
	totpDigits = 6
)

// NewTOTPSecret génère un secret de 160 bits.
func NewTOTPSecret() ([]byte, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	return b, err
}

// TOTPCode calcule le code du pas `step`.
func TOTPCode(secret []byte, step int64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(step))
	m := hmac.New(sha1.New, secret)
	m.Write(buf[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	n := (uint32(sum[off]&0x7f) << 24) | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	return fmt.Sprintf("%0*d", totpDigits, n%1_000_000)
}

// VerifyTOTP accepte le pas courant et ses voisins immédiats (±1 : dérive d'horloge), strictement APRÈS le dernier pas
// déjà accepté (un code ne sert qu'une fois). Renvoie le pas accepté.
func VerifyTOTP(secret []byte, code string, now time.Time, lastStep int64) (int64, bool) {
	cur := now.Unix() / totpPeriod
	ok, matched := false, int64(0)
	for _, st := range []int64{cur - 1, cur, cur + 1} { // pas d'arrêt anticipé : temps constant
		if subtle.ConstantTimeCompare([]byte(TOTPCode(secret, st)), []byte(code)) == 1 && st > lastStep {
			ok, matched = true, st
		}
	}
	return matched, ok
}

// OTPAuthURI est l'URI à encoder en QR code par l'application d'authentification.
func OTPAuthURI(issuer, account string, secret []byte) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{"secret": {base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)}, "issuer": {issuer}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// NewRecoveryCodes génère n codes de secours (chacun à usage unique) et leurs condensats.
func NewRecoveryCodes(n int) (plain, hashes []string, err error) {
	for i := 0; i < n; i++ {
		b := make([]byte, 10)
		if _, err = rand.Read(b); err != nil {
			return nil, nil, err
		}
		s := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)) // 16 caractères
		code := s[:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:]
		plain = append(plain, code)
		hashes = append(hashes, HashToken(NormalizeRecoveryCode(code)))
	}
	return plain, hashes, nil
}

// NormalizeRecoveryCode ignore la casse, les espaces et les tirets saisis.
func NormalizeRecoveryCode(c string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(c)))
}

// LooksLikeRecoveryCode distingue un code de secours (16 caractères) d'un code TOTP (6 chiffres).
func LooksLikeRecoveryCode(c string) bool { return len(NormalizeRecoveryCode(c)) == 16 }

// ── chiffrement du secret TOTP au repos ─────────────────────────────────────

// SecretBox chiffre les secrets TOTP (AES-256-GCM). La clé vient de la configuration, jamais de la base.
type SecretBox struct{ aead cipher.AEAD }

// NewSecretBox crée une SecretBox à partir d'une clé de 32 octets.
func NewSecretBox(key []byte) (*SecretBox, error) {
	if len(key) != 32 {
		return nil, errors.New("clé de chiffrement : 32 octets exigés")
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	g, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}
	return &SecretBox{aead: g}, nil
}

// Seal chiffre ; `aad` lie le chiffré à son propriétaire (identifiant du compte) : un secret recopié vers un autre compte est refusé.
func (s *SecretBox) Seal(plain []byte, aad string) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, plain, []byte(aad)), nil
}

// Open déchiffre.
func (s *SecretBox) Open(sealed []byte, aad string) ([]byte, error) {
	n := s.aead.NonceSize()
	if len(sealed) < n {
		return nil, errors.New("secret chiffré invalide")
	}
	return s.aead.Open(nil, sealed[:n], sealed[n:], []byte(aad))
}

// Challenge est l'enregistrement serveur d'un jeton de défi 2FA (le jeton lui-même est un JWT signé dont `jti` = ID).
// Usage unique, courte durée, lié à l'adresse IP, 5 codes erronés maximum.
type Challenge struct {
	ID        string
	StaffID   string
	Scope     string
	IP        string
	ExpiresAt time.Time
	UsedAt    *time.Time
	Failures  int
}

// AuditEntry est une ligne du journal d'audit (append-only : jamais modifiée ni supprimée).
type AuditEntry struct {
	ID        string
	At        time.Time
	ActorType string // staff | customer | system
	ActorID   string
	Action    string // ex. staff.invite
	Entity    string // ex. staff_user
	EntityID  string
	Before    map[string]any
	After     map[string]any
	IP        string
	TraceID   string
}

// Types d'acteur de l'audit.
const (
	ActorStaff    = "staff"
	ActorCustomer = "customer"
	ActorSystem   = "system"
)

// AuditFilter filtre la lecture du journal (curseur : keyset sur (at, id) décroissant).
type AuditFilter struct {
	ActorID, Entity, EntityID, Action string
	From, To                          *time.Time
	Cursor                            string
	Limit                             int
}
