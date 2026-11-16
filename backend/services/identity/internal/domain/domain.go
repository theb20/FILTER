// Package domain contient les règles métier d'identity, sans dépendance technique.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/throttle"
)

// Statuts et genres.
const (
	KindIndividual = "individual"
	KindCompany    = "company"

	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// Statuts d'une entreprise : la validation est MANUELLE (personnel habilité).
const (
	CompanyPending  = "pending"
	CompanyApproved = "approved"
	CompanyRejected = "rejected"
)

// Company est le titulaire d'un compte entreprise. `Approved` reste le miroir de `Status == approved`.
type Company struct {
	ID, LegalName, TaxID, Country string
	Approved                      bool
	Status                        string // vide = en attente (comptes créés avant le statut)
	RejectionReason               string
	DecidedAt                     *time.Time
}

// State renvoie le statut effectif (jamais vide).
func (c *Company) State() string {
	switch {
	case c.Status != "":
		return c.Status
	case c.Approved:
		return CompanyApproved
	}
	return CompanyPending
}

// User est un compte client.
type User struct {
	ID                string
	Email             string
	PasswordHash      string // vide pour un compte créé via un fournisseur OAuth
	EmailVerified     bool
	Kind              string
	FirstName         string
	LastName          string
	Locale            string
	PreferredCurrency string
	Status            string
	Company           *Company
	CreatedAt         time.Time
}

// ProActive : les conditions pro s'ouvrent seulement après validation de l'entreprise.
func (u *User) ProActive() bool { return u.Company != nil && u.Company.Approved }

// RefreshToken est un jeton de rafraîchissement à ROTATION. Seul son condensat est stocké.
type RefreshToken struct {
	ID        string
	FamilyID  string // une famille = une session ; un rejeu révoque toute la famille
	UserID    string
	Hash      string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
	IP        string
	UserAgent string
}

// OneTimeToken est un jeton à usage unique (vérification d'e-mail, changement d'e-mail…).
type OneTimeToken struct {
	Hash      string
	Purpose   string
	UserID    string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// Purposes des jetons à usage unique.
const (
	PurposeEmailVerification = "email_verification"
)

// Settings : valeurs par défaut VALIDÉES, toutes réglables (identity-settings).
type Settings struct {
	AccessTokenTTL         time.Duration
	CustomerRefreshTTL     time.Duration
	StaffRefreshTTL        time.Duration // 8 h (contrat : staffRefreshHours)
	EmailVerificationTTL   time.Duration // 48 h
	EmailChangeTTL         time.Duration // 24 h
	InvitationTTL          time.Duration // 72 h (personnel)
	StaffChallengeTTL      time.Duration // 5 min (jeton 2FA temporaire)
	RecentLoginWindow      time.Duration // 10 min (« connexion récente »)
	Throttle               throttle.Params
	LoginAlertAfterFailure int           // proposition : 5 échecs
	LoginAlertWindow       time.Duration // 15 min
	PasswordMin            int           // 8 (clients)
	StaffPasswordMin       int           // 12 : valeur PROPOSÉE par le contrat, à valider
	PasswordMax            int           // 128
}

// DefaultSettings renvoie les défauts.
func DefaultSettings() Settings {
	return Settings{
		AccessTokenTTL:         15 * time.Minute,
		CustomerRefreshTTL:     30 * 24 * time.Hour,
		StaffRefreshTTL:        8 * time.Hour,
		EmailVerificationTTL:   48 * time.Hour,
		EmailChangeTTL:         24 * time.Hour,
		InvitationTTL:          72 * time.Hour,
		StaffChallengeTTL:      5 * time.Minute,
		RecentLoginWindow:      10 * time.Minute,
		Throttle:               throttle.Defaults(),
		LoginAlertAfterFailure: 5,
		LoginAlertWindow:       15 * time.Minute,
		PasswordMin:            8,
		StaffPasswordMin:       12,
		PasswordMax:            128,
	}
}

// NormalizeEmail met l'adresse en forme canonique (insensible à la casse) ou refuse une adresse invalide.
func NormalizeEmail(s string) (string, error) {
	s = strings.TrimSpace(s)
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || strings.ContainsAny(s, " <>") || !strings.Contains(s, "@") {
		return "", apperr.Validation("adresse e-mail invalide", apperr.Field{Pointer: "/email", Code: apperr.CodeValidation, Message: "adresse e-mail invalide"})
	}
	return strings.ToLower(s), nil
}

// ValidatePasswordShape applique la politique VALIDÉE : 8 à 128 caractères, AUCUNE règle de complexité.
// Le refus des mots de passe compromis est une étape séparée (BreachChecker).
func ValidatePasswordShape(pw string, s Settings) error {
	n := utf8.RuneCountInString(pw)
	if n < s.PasswordMin || n > s.PasswordMax {
		return apperr.Validation("mot de passe invalide", apperr.Field{Pointer: "/password", Code: apperr.CodeValidation, Message: fmt.Sprintf("le mot de passe doit contenir de %d à %d caractères", s.PasswordMin, s.PasswordMax)})
	}
	return nil
}

// NewOpaqueToken génère un jeton aléatoire (256 bits) et son condensat SHA-256 : seul le condensat est stocké.
func NewOpaqueToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain), nil
}

// HashToken condense un jeton opaque.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
