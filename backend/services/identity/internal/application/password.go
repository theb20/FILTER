package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EvPasswordResetRequested : consommé par la messagerie (en développement : la boîte d'envoi de développement).
const EvPasswordResetRequested = "identity.password_reset_requested.v1"

const (
	resetCodeTTL  = 15 * time.Minute
	resetTokenTTL = 15 * time.Minute
)

func resetCodeHash(email, code string) string { return domain.HashToken(email + "|" + code) }

func invalidCode() error {
	return apperr.New(codes.InvalidArgument, apperr.CodeInvalidCode, "code invalide ou expiré")
}

// ForgotPassword envoie un code à 6 chiffres. Répond TOUJOURS pareil : compte inconnu, suspendu ou demandes trop
// rapprochées ne se distinguent pas du succès (aucune fuite d'existence). Les demandes répétées sont ralenties.
func (s *Service) ForgotPassword(ctx context.Context, email, locale string) error {
	norm, err := domain.NormalizeEmail(email)
	if err != nil {
		return err // forme invalide : seule erreur renvoyée
	}
	set := s.Settings.Get(ctx)
	now := s.Now()
	key := "pwforgot|" + norm
	st, err := s.Failures.Pair(ctx, key)
	if err != nil {
		return apperr.Internal(err)
	}
	if set.Throttle.Wait(st, now) > 0 {
		return nil // trop de demandes : on n'envoie rien, la réponse reste identique
	}
	u, err := s.Users.ByEmail(ctx, norm)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return apperr.Internal(err)
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Failures.SetPair(ctx, key, set.Throttle.RecordFailure(st, now)); err != nil {
			return apperr.Internal(err)
		}
		if u == nil || u.Status != domain.StatusActive {
			return nil
		}
		n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
		if err != nil {
			return apperr.Internal(err)
		}
		code := fmt.Sprintf("%06d", n.Int64())
		if err := s.Tokens.Put(ctx, &domain.OneTimeToken{Hash: resetCodeHash(norm, code), Purpose: domain.PurposePasswordResetCode, UserID: u.ID, ExpiresAt: now.Add(resetCodeTTL)}); err != nil {
			return apperr.Internal(err)
		}
		return s.publish(ctx, EvPasswordResetRequested, u.ID, &eventsv1.PasswordResetRequested{Email: u.Email, Code: code, Locale: orDefault(locale, u.Locale), ExpiresAt: timestamppb.New(now.Add(resetCodeTTL))}, u.Locale)
	})
}

// VerifyPasswordCode échange le code contre un jeton de réinitialisation. Les essais sont RALENTIS par adresse
// (1 million de codes possibles : on ne laisse pas deviner).
func (s *Service) VerifyPasswordCode(ctx context.Context, email, code string) (token string, ttl time.Duration, err error) {
	norm, nerr := domain.NormalizeEmail(email)
	if nerr != nil {
		return "", 0, invalidCode()
	}
	set := s.Settings.Get(ctx)
	now := s.Now()
	key := "pwverify|" + norm
	st, err := s.Failures.Pair(ctx, key)
	if err != nil {
		return "", 0, apperr.Internal(err)
	}
	if wait := set.Throttle.Wait(st, now); wait > 0 {
		return "", 0, apperr.RateLimited(wait)
	}
	var plain string
	var bad bool
	err = s.tx(ctx, func(ctx context.Context) error {
		id, ok, err := s.Tokens.Consume(ctx, domain.PurposePasswordResetCode, resetCodeHash(norm, code), now)
		if err != nil {
			return apperr.Internal(err)
		}
		if !ok {
			bad = true
			return s.Failures.SetPair(ctx, key, set.Throttle.RecordFailure(st, now)) // l'échec est enregistré, la transaction validée
		}
		p, h, err := domain.NewOpaqueToken()
		if err != nil {
			return apperr.Internal(err)
		}
		if err := s.Tokens.Put(ctx, &domain.OneTimeToken{Hash: h, Purpose: domain.PurposePasswordReset, UserID: id, ExpiresAt: now.Add(resetTokenTTL)}); err != nil {
			return apperr.Internal(err)
		}
		plain = p
		return s.Failures.ResetPair(ctx, key)
	})
	if err != nil {
		return "", 0, err
	}
	if bad {
		return "", 0, invalidCode()
	}
	return plain, resetTokenTTL, nil
}

// ResetPassword définit le nouveau mot de passe et DÉCONNECTE tous les appareils. Prouver le contrôle de la boîte
// e-mail (le code) vaut aussi vérification de l'adresse.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	set := s.Settings.Get(ctx)
	if err := domain.ValidatePasswordShape(newPassword, set); err != nil {
		return err
	}
	if bad, err := s.Breach.Compromised(ctx, newPassword); err != nil {
		return apperr.Internal(err)
	} else if bad {
		return apperr.Validation("mot de passe compromis", apperr.Field{Pointer: "/newPassword", Code: apperr.CodePasswordCompromised, Message: "ce mot de passe a fuité : choisissez-en un autre"})
	}
	hash, err := s.Hasher.Hash(newPassword)
	if err != nil {
		return apperr.Internal(err)
	}
	now := s.Now()
	return s.tx(ctx, func(ctx context.Context) error {
		id, ok, err := s.Tokens.Consume(ctx, domain.PurposePasswordReset, domain.HashToken(token), now)
		if err != nil {
			return apperr.Internal(err)
		}
		if !ok {
			return invalidCode()
		}
		u, err := s.Users.ByID(ctx, id)
		if err != nil {
			return apperr.Internal(err)
		}
		u.PasswordHash, u.EmailVerified = hash, true
		if err := s.Users.Update(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		return s.wrap(s.Sessions.RevokeAllExcept(ctx, u.ID, "", now))
	})
}
