// Package application contient les cas d'usage d'identity. Il ne dépend que des PORTS ci-dessous.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/theb20/FILTER/backend/pkg/throttle"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// Erreurs de dépôt.
var (
	ErrNotFound   = errors.New("introuvable")
	ErrEmailTaken = errors.New("adresse déjà utilisée")
)

// UserRepo persiste les comptes. Les adresses sont déjà normalisées.
type UserRepo interface {
	Create(ctx context.Context, u *domain.User) error // ErrEmailTaken si l'adresse existe (atomique)
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	ByID(ctx context.Context, id string) (*domain.User, error)
	// Update enregistre le compte (adresse comprise : ErrEmailTaken si elle appartient à un autre compte).
	Update(ctx context.Context, u *domain.User) error
}

// SessionRepo persiste les jetons de rafraîchissement (rotation, détection de rejeu).
type SessionRepo interface {
	CreateSession(ctx context.Context, t *domain.RefreshToken) error
	SessionByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	// MarkUsed est ATOMIQUE : renvoie false si le jeton était déjà utilisé (rejeu).
	MarkUsed(ctx context.Context, id string, at time.Time) (bool, error)
	RevokeFamily(ctx context.Context, familyID string, at time.Time) error
	// RevokeAllExcept révoque toutes les familles du compte sauf `keepFamilyID` (vide = aucune exception).
	RevokeAllExcept(ctx context.Context, userID, keepFamilyID string, at time.Time) error
}

// TokenRepo persiste les jetons à usage unique.
type TokenRepo interface {
	Put(ctx context.Context, t *domain.OneTimeToken) error
	// Consume est ATOMIQUE : renvoie l'identifiant du compte si le jeton est valide, non expiré et non utilisé.
	Consume(ctx context.Context, purpose, hash string, now time.Time) (userID string, ok bool, err error)
	// ConsumePayload : comme Consume, et renvoie la donnée liée au jeton (ex. la nouvelle adresse e-mail).
	ConsumePayload(ctx context.Context, purpose, hash string, now time.Time) (userID, payload string, ok bool, err error)
	// InvalidateUserTokens rend inutilisables les jetons non consommés de ce compte pour ce but (un seul lien valable à la fois).
	InvalidateUserTokens(ctx context.Context, purpose, userID string, now time.Time) error
}

// FailureStore mémorise les échecs de connexion (ralentissement progressif + alerte au titulaire).
type FailureStore interface {
	Pair(ctx context.Context, key string) (throttle.State, error)
	SetPair(ctx context.Context, key string, s throttle.State) error
	ResetPair(ctx context.Context, key string) error
	// AccountFailures ajoute un échec et renvoie le nombre d'échecs de ce compte dans la fenêtre (toutes IP).
	AccountFailures(ctx context.Context, email string, at time.Time, window time.Duration) (int, error)
	// MarkAlerted mémorise qu'une alerte a été envoyée jusqu'à `until` ; renvoie false si déjà alerté.
	MarkAlerted(ctx context.Context, email string, now, until time.Time) (bool, error)
}

// UnitOfWork exécute fn dans UNE transaction : les écritures des dépôts ET l'événement publié (outbox) sont validés
// ensemble ou annulés ensemble. Un appel imbriqué rejoint la transaction en cours.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// NoUnitOfWork exécute fn sans transaction (jamais en production : voir cmd/identity).
type NoUnitOfWork struct{}

func (NoUnitOfWork) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// PasswordHasher hache et vérifie les mots de passe (Argon2id en production ; substituable en test).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) (bool, error)
	Dummy() string // condensat factice pour égaliser les temps de réponse
}

// BreachChecker refuse les mots de passe connus comme compromis. La SOURCE de la liste n'est pas encore
// choisie (décision en attente) : l'implémentation par défaut est explicitement désactivée.
type BreachChecker interface {
	Compromised(ctx context.Context, password string) (bool, error)
}

// DisabledBreachChecker n'effectue AUCUN contrôle. À remplacer avant la production.
type DisabledBreachChecker struct{}

func (DisabledBreachChecker) Compromised(context.Context, string) (bool, error) { return false, nil }

// SettingsProvider fournit les réglages (identity-settings).
type SettingsProvider interface {
	Get(ctx context.Context) domain.Settings
}

// StaticSettings sert des réglages fixes (défauts ou tests).
type StaticSettings domain.Settings

func (s StaticSettings) Get(context.Context) domain.Settings { return domain.Settings(s) }

// StaffRepo persiste le personnel (pool séparé des clients), ses rôles et son 2FA.
type StaffRepo interface {
	CreateStaff(ctx context.Context, u *domain.StaffUser) error // ErrEmailTaken
	StaffByEmail(ctx context.Context, email string) (*domain.StaffUser, error)
	StaffByID(ctx context.Context, id string) (*domain.StaffUser, error)
	UpdateStaff(ctx context.Context, u *domain.StaffUser) error
	// AdvanceTOTPStep est ATOMIQUE : vrai si `step` est strictement postérieur au dernier pas accepté (un code ne sert qu'une fois).
	AdvanceTOTPStep(ctx context.Context, id string, step int64) (bool, error)
	// SetRecoveryCodes remplace les codes de secours (condensats) ; ConsumeRecoveryCode est ATOMIQUE.
	SetRecoveryCodes(ctx context.Context, id string, hashes []string) error
	ConsumeRecoveryCode(ctx context.Context, id, hash string, at time.Time) (bool, error)
	RolesByIDs(ctx context.Context, ids []string) ([]domain.Role, error)
	RoleByName(ctx context.Context, name string) (*domain.Role, error)
	CountActiveWithRole(ctx context.Context, roleID string) (int, error)
}

// ChallengeRepo persiste les jetons de défi 2FA.
type ChallengeRepo interface {
	PutChallenge(ctx context.Context, c *domain.Challenge) error
	ChallengeByID(ctx context.Context, id string) (*domain.Challenge, error)
	// ConsumeChallenge est ATOMIQUE : vrai pour un seul appelant (jeton valide, non utilisé, non expiré).
	ConsumeChallenge(ctx context.Context, id string, now time.Time) (bool, error)
	// FailChallenge ajoute un code erroné (atomique) et renvoie le total.
	FailChallenge(ctx context.Context, id string) (int, error)
}

// StaffAdminRepo : gestion du personnel et des rôles (écrans d'administration).
type StaffAdminRepo interface {
	ListStaff(ctx context.Context, query string, limit, offset int) ([]*domain.StaffUser, int, error)
	SetStaffRoles(ctx context.Context, staffID string, roleIDs []string) error
	ClearTwoFactor(ctx context.Context, staffID string) error
	// Invitations : un seul jeton valable par membre (le précédent est remplacé).
	PutStaffInvitation(ctx context.Context, hash, staffID string, expires time.Time) error
	// ConsumeStaffInvitation est ATOMIQUE : renvoie l'identifiant du membre si le jeton est valide, non utilisé, non expiré.
	ConsumeStaffInvitation(ctx context.Context, hash string, now time.Time) (staffID string, ok bool, err error)

	ListRoles(ctx context.Context, query string, limit, offset int) ([]domain.Role, int, error)
	RoleByID(ctx context.Context, id string) (*domain.Role, error)
	CreateRole(ctx context.Context, r *domain.Role) error // ErrRoleNameTaken
	UpdateRole(ctx context.Context, r *domain.Role) error
	DeleteRole(ctx context.Context, id string) error // ErrRoleInUse si encore attribué
	RolesByIDsExist(ctx context.Context, ids []string) (int, error)
}

// AuditRepo : journal d'audit append-only.
type AuditRepo interface {
	AppendAudit(ctx context.Context, e *domain.AuditEntry) error
	ListAudit(ctx context.Context, f domain.AuditFilter) ([]*domain.AuditEntry, string, error)
}

// Erreurs de dépôt supplémentaires.
var (
	ErrRoleNameTaken = errors.New("nom de rôle déjà utilisé")
	ErrRoleInUse     = errors.New("rôle encore attribué")
)

// CustomerAdminRepo : consultation des clients et des entreprises (écrans d'administration).
// Les modifications passent par UserRepo.Update (le compte porte son entreprise).
type CustomerAdminRepo interface {
	// ListCustomers : `kind` et `status` vides = tous. Tri : plus récents d'abord.
	ListCustomers(ctx context.Context, query, kind, status string, limit, offset int) ([]*domain.User, int, error)
	// UserByCompanyID renvoie le compte titulaire de l'entreprise (ErrNotFound si inconnue).
	UserByCompanyID(ctx context.Context, companyID string) (*domain.User, error)
	// ListCompanies : `status` vide = tous. Tri : plus anciennes en attente d'abord pour « pending », sinon plus récentes.
	ListCompanies(ctx context.Context, query, status string, limit, offset int) ([]*domain.Company, int, error)
	CountCompanies(ctx context.Context, status string) (int, error)
}

// SettingsRepo persiste les réglages modifiables d'identity (une seule ligne).
type SettingsRepo interface {
	LoadIdentitySettings(ctx context.Context) (domain.IdentitySettings, error)
	SaveIdentitySettings(ctx context.Context, s domain.IdentitySettings, updatedBy string, at time.Time) error
}

// OAuthRepo : liens entre un compte local et une identité chez un fournisseur (Google).
type OAuthRepo interface {
	// OAuthUser renvoie le compte lié à (fournisseur, sujet), ou found=false.
	OAuthUser(ctx context.Context, provider, subject string) (userID string, found bool, err error)
	// LinkOAuth lie l'identité au compte ; sans effet si le lien existe déjà pour ce compte.
	LinkOAuth(ctx context.Context, provider, subject, userID string, at time.Time) error
}

// PreferencesRepo persiste les préférences d'interface, par compte (personnel ou client).
type PreferencesRepo interface {
	// LoadPreferences renvoie nil quand le compte n'a jamais rien enregistré.
	LoadPreferences(ctx context.Context, kind domain.PrincipalKind, id string) (*domain.Preferences, error)
	// SavePreferences remplace les préférences ; ErrNotFound si le compte n'existe pas.
	SavePreferences(ctx context.Context, kind domain.PrincipalKind, id string, p domain.Preferences, at time.Time) error
}

// ConsentRepo : journal de consentements APPEND-ONLY (preuve).
type ConsentRepo interface {
	AppendConsent(ctx context.Context, c *domain.ConsentRecord) error // ErrNotFound si user_id inconnu
	// LatestUserConsent renvoie le consentement le plus récent du compte, ou ErrNotFound s'il n'en a jamais donné.
	LatestUserConsent(ctx context.Context, userID string) (*domain.ConsentRecord, error)
}
