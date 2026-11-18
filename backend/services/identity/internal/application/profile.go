package application

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
)

// Événements du changement d'adresse (contrat : proto/events/v1).
const (
	EvEmailChangeRequested = "identity.email_change_requested.v1"
	EvEmailChangeConfirmed = "identity.email_change_confirmed.v1"
)

// UpdateProfile modifie PARTIELLEMENT le profil du compte (nom, téléphone, langue, devise). L'adresse e-mail et le
// mot de passe ont leurs propres parcours.
func (s *Service) UpdateProfile(ctx context.Context, userID string, p domain.ProfilePatch) (*domain.User, error) {
	u, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := p.Apply(u); err != nil {
		return nil, err
	}
	if err := s.Users.Update(ctx, u); err != nil {
		return nil, s.notFoundOr(err, "compte introuvable")
	}
	return u, nil
}

func (s *Service) notFoundOr(err error, msg string) error {
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound(msg)
	}
	return apperr.Internal(err)
}

// ── changement d'adresse e-mail ─────────────────────────────────────────────

// RequestEmailChange démarre le changement. Mot de passe EXIGÉ (avec le même ralentissement progressif que le
// changement de mot de passe). Réponse UNIFORME : si la nouvelle adresse appartient déjà à un compte, rien n'est
// envoyé mais l'appelant ne le saura pas. Un lien est envoyé à la NOUVELLE adresse ; un seul lien reste valable.
//
// Un compte SANS mot de passe (créé via Google) est refusé : la preuve de « connexion récente » n'est pas transmise
// à ce service (voir docs/IDENTITY-GATEWAY-MAPPING.md).
func (s *Service) RequestEmailChange(ctx context.Context, userID, newEmail, password string) error {
	set := s.Settings.Get(ctx)
	now := s.Now()
	u, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if u.PasswordHash == "" {
		return apperr.PermissionDenied(apperr.CodeForbidden, "connexion récente requise pour modifier l'adresse d'un compte sans mot de passe")
	}
	key := "chem|" + userID
	st, err := s.Failures.Pair(ctx, key)
	if err != nil {
		return apperr.Internal(err)
	}
	if wait := set.Throttle.Wait(st, now); wait > 0 {
		return apperr.RateLimited(wait)
	}
	ok, err := s.Hasher.Verify(password, u.PasswordHash)
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		_ = s.Failures.SetPair(ctx, key, set.Throttle.RecordFailure(st, now))
		return apperr.Unauthenticated(apperr.CodeInvalidCredentials, "mot de passe incorrect")
	}
	_ = s.Failures.ResetPair(ctx, key)

	next, err := domain.NormalizeEmail(newEmail)
	if err != nil {
		return apperr.Validation("adresse e-mail invalide", apperr.Field{Pointer: "/newEmail", Code: apperr.CodeValidation, Message: "adresse e-mail invalide"})
	}
	if next == u.Email {
		return apperr.Validation("adresse identique", apperr.Field{Pointer: "/newEmail", Code: apperr.CodeValidation, Message: "c'est déjà votre adresse"})
	}
	if _, err := s.Users.ByEmail(ctx, next); err == nil {
		return nil // adresse déjà utilisée : réponse identique, aucun e-mail
	} else if !errors.Is(err, ErrNotFound) {
		return apperr.Internal(err)
	}
	plain, hash, err := domain.NewOpaqueToken()
	if err != nil {
		return apperr.Internal(err)
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Tokens.InvalidateUserTokens(ctx, domain.PurposeEmailChange, userID, now); err != nil {
			return apperr.Internal(err)
		}
		if err := s.Tokens.Put(ctx, &domain.OneTimeToken{Hash: hash, Purpose: domain.PurposeEmailChange, UserID: userID, Payload: next, ExpiresAt: now.Add(set.EmailChangeTTL)}); err != nil {
			return apperr.Internal(err)
		}
		return s.publish(ctx, EvEmailChangeRequested, userID, &eventsv1.EmailChangeRequested{UserId: userID, OldEmail: u.Email, NewEmail: next, Token: plain, Locale: u.Locale}, u.Locale)
	})
}

// ConfirmEmailChange bascule l'adresse (jeton à usage unique reçu à la NOUVELLE adresse). L'adresse est alors
// vérifiée, TOUTES les sessions sont révoquées et l'ANCIENNE adresse est notifiée.
func (s *Service) ConfirmEmailChange(ctx context.Context, token string) error {
	return s.tx(ctx, func(ctx context.Context) error {
		now := s.Now()
		id, next, ok, err := s.Tokens.ConsumePayload(ctx, domain.PurposeEmailChange, domain.HashToken(token), now)
		if err != nil {
			return apperr.Internal(err)
		}
		if !ok || next == "" {
			return apperr.New(codes.InvalidArgument, apperr.CodeInvalidCode, "lien invalide ou expiré")
		}
		u, err := s.GetUser(ctx, id)
		if err != nil {
			return err
		}
		old := u.Email
		u.Email, u.EmailVerified = next, true
		switch err := s.Users.Update(ctx, u); {
		case errors.Is(err, ErrEmailTaken):
			return apperr.New(codes.AlreadyExists, apperr.CodeConflict, "cette adresse est désormais utilisée par un autre compte")
		case err != nil:
			return apperr.Internal(err)
		}
		if err := s.Sessions.RevokeAllExcept(ctx, id, "", now); err != nil {
			return apperr.Internal(err)
		}
		return s.publish(ctx, EvEmailChangeConfirmed, id, &eventsv1.EmailChangeConfirmed{UserId: id, OldEmail: old, NewEmail: next, Locale: u.Locale}, u.Locale)
	})
}

// ── entreprise du compte ────────────────────────────────────────────────────

// MyCompany renvoie l'entreprise du compte (NotFound pour un compte « particulier »).
func (s *Service) MyCompany(ctx context.Context, userID string) (*domain.Company, error) {
	u, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.Company == nil {
		return nil, apperr.NotFound("aucune entreprise sur ce compte")
	}
	return u.Company, nil
}

// PutMyCompany renseigne/modifie l'entreprise. Toute modification des données LÉGALES (raison sociale, identifiant
// fiscal, pays) remet le statut à « en attente » : les conditions professionnelles se referment jusqu'à nouvelle
// validation par le personnel. Changer seulement la fonction ne touche pas au statut. Un compte « particulier » ne
// peut pas porter d'entreprise (409).
func (s *Service) PutMyCompany(ctx context.Context, userID string, in domain.CompanyData) (*domain.Company, error) {
	d, err := in.Normalize()
	if err != nil {
		return nil, err
	}
	u, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.Company == nil {
		return nil, apperr.New(codes.AlreadyExists, apperr.CodeConflict, "un compte particulier n'a pas d'entreprise")
	}
	c := u.Company
	if c.LegalChanged(d) {
		c.Approved, c.Status, c.RejectionReason, c.DecidedAt = false, domain.CompanyPending, "", nil
	}
	c.LegalName, c.TaxID, c.Country, c.JobTitle = d.LegalName, d.TaxID, d.Country, d.JobTitle
	if err := s.Users.Update(ctx, u); err != nil {
		return nil, s.notFoundOr(err, "compte introuvable")
	}
	return c, nil
}

// ── consentements ───────────────────────────────────────────────────────────

// ConsentInput : un choix de consentement (visiteur ou client connecté).
type ConsentInput struct {
	UserID, AnonymousID string
	Purposes            domain.ConsentPurposes
	PolicyVersion       string
	Locale, ClientIP    string
}

// RecordConsent ajoute une PREUVE au journal (jamais de mise à jour). Visiteur sans identifiant : un identifiant
// anonyme est créé pour que la preuve reste rattachée à quelqu'un.
func (s *Service) RecordConsent(ctx context.Context, in ConsentInput) (*domain.ConsentRecord, error) {
	c := &domain.ConsentRecord{
		ID: s.NewID(), UserID: in.UserID, AnonymousID: strings.TrimSpace(in.AnonymousID), Purposes: in.Purposes,
		PolicyVersion: in.PolicyVersion, Locale: in.Locale, IPNet: truncateIP(in.ClientIP), RecordedAt: s.Now(),
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if c.UserID == "" && c.AnonymousID == "" {
		c.AnonymousID = s.NewID()
	}
	if err := s.Consents.AppendConsent(ctx, c); err != nil {
		return nil, s.notFoundOr(err, "compte introuvable")
	}
	return c, nil
}

// MyConsent renvoie le dernier consentement du compte (NotFound s'il n'en a jamais donné : rien n'est inventé).
func (s *Service) MyConsent(ctx context.Context, userID string) (*domain.ConsentRecord, error) {
	c, err := s.Consents.LatestUserConsent(ctx, userID)
	if err != nil {
		return nil, s.notFoundOr(err, "aucun consentement enregistré")
	}
	return c, nil
}

// truncateIP garde le réseau (IPv4 /24, IPv6 /48) : suffisant comme preuve, sans identifier la connexion.
func truncateIP(raw string) string {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.Mask(net.CIDRMask(24, 32)).String() + "/24"
	}
	return ip.Mask(net.CIDRMask(48, 128)).String() + "/48"
}

// ── contrôle de permission (appel interne des autres services) ──────────────

// CheckPermission indique si le membre du personnel ACTIF détient la permission, lue en base (pas dans un jeton).
// Compte inconnu ou désactivé : refus, jamais d'erreur (fail closed).
func (s *Service) CheckPermission(ctx context.Context, staffID, permission string) (bool, error) {
	if !domain.IsPermission(permission) || permission == domain.AllPermissions {
		return false, apperr.Validation("permission inconnue", apperr.Field{Pointer: "/permission", Code: apperr.CodeValidation, Message: "code absent du catalogue"})
	}
	u, err := s.Staff.StaffByID(ctx, staffID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, apperr.Internal(err)
	}
	if u.Status != domain.StatusActive {
		return false, nil
	}
	roles, err := s.Staff.RolesByIDs(ctx, u.RoleIDs)
	if err != nil {
		return false, apperr.Internal(err)
	}
	_, perms := rolesAndPerms(roles)
	return domain.HasPermission(perms, permission), nil
}
