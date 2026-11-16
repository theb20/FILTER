package application

import (
	"context"
	"errors"

	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
)

// SearchParams : requête envoyée à une source.
type SearchParams struct {
	Query, Language, Country string
	Count                    int
}

// CallError classe l'échec d'un appel : seuls les échecs réseau et les 5xx sont rejoués ; un 4xx (dont 429) ne l'est jamais.
type CallError struct {
	Kind       string // network | http_5xx | http_4xx | quota | decode
	HTTPStatus int
	Retryable  bool
	// Sent : la requête a-t-elle pu atteindre la source (donc être facturée) ? Faux pour un échec de connexion.
	Sent bool
	Err  error
}

func (e *CallError) Error() string {
	if e.Err != nil {
		return e.Kind + ": " + e.Err.Error()
	}
	return e.Kind
}
func (e *CallError) Unwrap() error { return e.Err }

// Connector interroge UNE source externe.
type Connector interface {
	Search(ctx context.Context, p SearchParams) ([]domain.RawHit, error)
}

// ConnectorFactory choisit le connecteur d'une source (selon son adresse) et résout son secret.
type ConnectorFactory interface {
	For(src *domain.Source) (Connector, error)
}

// IsTimeout : le contexte a-t-il expiré ?
func IsTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
