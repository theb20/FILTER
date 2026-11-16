// Package application : cas d'usage du service sourcing. Ne dépend que des ports ci-dessous.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
)

var (
	ErrNotFound      = errors.New("introuvable")
	ErrNameTaken     = errors.New("nom de source déjà utilisé")
	ErrBudget        = errors.New("budget atteint")
	ErrAlreadyCalled = errors.New("source déjà appelée pour cette recherche")
	ErrFallbackChain = errors.New("la bascule est limitée à un seul saut")
)

// Reservation : budget retenu AVANT un appel. Une ligne par réservation, avec une échéance (libérée par ReleaseExpired).
type Reservation struct {
	ID, SourceID, SearchID string
	AmountMicros           int64
	Day, Month             time.Time // périodes (UTC) dont les compteurs sont mouvementés
	DailyCap, MonthlyCap   int64
	CreatedAt, ExpiresAt   time.Time
}

// CallLog : une ligne du journal des appels (rétention 90 jours).
type CallLog struct {
	ID, SourceID, SearchID string
	StartedAt              time.Time
	DurationMs             int
	Status                 string
	HTTPStatus             int
	Attempts               int
	CostMicros             int64
	Currency               string
	ErrorKind              string
}

// CallStats : agrégats de santé d'une source sur une fenêtre.
type CallStats struct {
	Calls, Errors            int64
	P50Ms, P95Ms             int
	LastSuccessAt, LastErrAt *time.Time
	LastErrorKind            string
}

// CostLine : coûts agrégés par source et devise.
type CostLine struct {
	SourceID, SourceName, Currency  string
	Calls, CachedCalls, FailedCalls int64
	CostMicros, PerSearchMicros     int64
}

// ActiveSource : ce dont le service search a besoin pour planifier une recherche.
type ActiveSource struct {
	ID, Name           string
	Priority           int
	FallbackSourceID   *string
	MaxCacheTTLSeconds *int
}

// Store : persistance des sources, du budget et du journal.
type Store interface {
	CreateSource(ctx context.Context, s *domain.Source) error // ErrNameTaken
	UpdateSource(ctx context.Context, s *domain.Source) error // ErrNameTaken, ErrNotFound
	SourceByID(ctx context.Context, id string) (*domain.Source, error)
	ListSources(ctx context.Context, query string, limit, offset int) ([]*domain.Source, int, error)
	SetEnabled(ctx context.Context, id string, enabled bool) error
	SetSuspendedUntil(ctx context.Context, id string, until *time.Time) error
	SetRawStorage(ctx context.Context, id string, allowed bool, checkedAt time.Time) error
	ListActive(ctx context.Context, now time.Time) ([]ActiveSource, error)

	QuotaByID(ctx context.Context, id string, now time.Time) (*domain.Quota, error)
	PutQuota(ctx context.Context, q *domain.Quota) error // ErrFallbackChain

	// Reserve est ATOMIQUE : refuse (ErrBudget) si le plafond du jour OU du mois serait dépassé ; ErrAlreadyCalled si la source a déjà
	// été appelée pour ce search_id.
	Reserve(ctx context.Context, r Reservation) error
	// Settle règle la réservation : `actualCost` ajouté aux dépenses, montant réservé libéré. Sans effet si déjà réglée.
	Settle(ctx context.Context, id string, actualCost int64, now time.Time) error
	Release(ctx context.Context, id string, now time.Time) error
	ReleaseExpired(ctx context.Context, now time.Time) (int, error)
	Reconcile(ctx context.Context, now time.Time) error

	LogCall(ctx context.Context, c CallLog) error
	Stats(ctx context.Context, sourceID string, since time.Time) (CallStats, error)
	Costs(ctx context.Context, sourceID string, from, to time.Time) ([]CostLine, error)
	PurgeCalls(ctx context.Context, before time.Time) (int64, error)

	CountTests(ctx context.Context, actorID string, since time.Time) (int, error)
	RecordTest(ctx context.Context, actorID string, at time.Time) error
}
