package application

import (
	"context"
	"errors"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// GetPreferences renvoie les préférences du compte, ou les défauts s'il n'a jamais rien enregistré.
func (s *Service) GetPreferences(ctx context.Context, kind domain.PrincipalKind, id string) (domain.Preferences, error) {
	p, err := s.Preferences.LoadPreferences(ctx, kind, id)
	if err != nil {
		return domain.Preferences{}, apperr.Internal(err)
	}
	if p == nil {
		return domain.DefaultPreferences(), nil
	}
	return p.Clone(), nil
}

// PutPreferences remplace les préférences (validées). Pas d'entrée d'audit : ce sont des réglages d'affichage personnels,
// sans effet sur la sécurité ni sur les données de l'entreprise (les écrire au journal le noierait).
func (s *Service) PutPreferences(ctx context.Context, kind domain.PrincipalKind, id string, in domain.Preferences) (domain.Preferences, error) {
	if err := in.Validate(); err != nil {
		return domain.Preferences{}, err
	}
	in = in.Clone()
	if err := s.Preferences.SavePreferences(ctx, kind, id, in, s.Now()); err != nil {
		if errors.Is(err, ErrNotFound) {
			return domain.Preferences{}, apperr.NotFound("compte introuvable")
		}
		return domain.Preferences{}, apperr.Internal(err)
	}
	return in, nil
}
