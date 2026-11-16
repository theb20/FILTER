package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
	"google.golang.org/grpc/codes"
)

// Paramètres techniques du service (valeurs proposées du contrat).
const (
	TestsPerHour            = 10
	ReservationMargin       = 30 * time.Second // échéance d'une réservation = délai de l'appel + marge
	BreakerThreshold        = 5                // échecs consécutifs avant ouverture du disjoncteur
	BreakerCooldown         = 30 * time.Second
	DefaultCallResultsCount = 10
)

// Service regroupe les cas d'usage.
type Service struct {
	Store      Store
	Connectors ConnectorFactory
	Now        func() time.Time
	NewID      func() string

	mu       sync.Mutex
	breakers map[string]*breaker
}

type breaker struct {
	fails     int
	openUntil time.Time
}

// Config : dépendances du service.
type Config struct {
	Store      Store
	Connectors ConnectorFactory
	Now        func() time.Time
	NewID      func() string
}

func New(c Config) *Service {
	s := &Service{Store: c.Store, Connectors: c.Connectors, Now: c.Now, NewID: c.NewID, breakers: map[string]*breaker{}}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.NewID == nil {
		s.NewID = func() string { return uuid.Must(uuid.NewV7()).String() }
	}
	return s
}

func invalid(ptr, msg string) error {
	return apperr.Validation(msg, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
}
func conflict(msg string) error { return apperr.New(codes.AlreadyExists, apperr.CodeConflict, msg) }

// ── sources ─────────────────────────────────────────────────────────────────

func (s *Service) CreateSource(ctx context.Context, in domain.Source) (*domain.Source, error) {
	in.ID, in.CreatedAt = s.NewID(), s.Now()
	in.Enabled = false // une source est créée DÉSACTIVÉE : on la teste, puis on l'active
	in.AllowsRawStorage, in.TermsCheckedAt = false, nil
	if err := domain.ValidateSource(&in); err != nil {
		return nil, err
	}
	if _, err := s.Connectors.For(&in); err != nil {
		return nil, invalid("/baseUrl", "aucun connecteur disponible pour cette adresse")
	}
	if err := s.Store.CreateSource(ctx, &in); errors.Is(err, ErrNameTaken) {
		return nil, conflict("une source porte déjà ce nom")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return &in, nil
}

func (s *Service) UpdateSource(ctx context.Context, id string, in domain.Source) (*domain.Source, error) {
	cur, err := s.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	// ni l'état, ni le droit de stockage, ni la date de création ne se modifient ici
	in.ID, in.Enabled, in.SuspendedUntil, in.AllowsRawStorage, in.TermsCheckedAt, in.CreatedAt = cur.ID, cur.Enabled, cur.SuspendedUntil, cur.AllowsRawStorage, cur.TermsCheckedAt, cur.CreatedAt
	if err := domain.ValidateSource(&in); err != nil {
		return nil, err
	}
	if in.Currency != cur.Currency {
		if q, qerr := s.Store.QuotaByID(ctx, id, s.Now()); qerr == nil && (q.MonthSpent > 0 || q.MonthlyBudgetMicros > 0) {
			return nil, invalid("/currency", "la devise ne peut plus changer une fois un budget défini (les montants sont dans la devise de la source)")
		}
	}
	if _, err := s.Connectors.For(&in); err != nil {
		return nil, invalid("/baseUrl", "aucun connecteur disponible pour cette adresse")
	}
	if err := s.Store.UpdateSource(ctx, &in); errors.Is(err, ErrNameTaken) {
		return nil, conflict("une source porte déjà ce nom")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return &in, nil
}

func (s *Service) GetSource(ctx context.Context, id string) (*domain.Source, error) {
	src, err := s.Store.SourceByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperr.NotFound("source introuvable")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return src, nil
}

type Page[T any] struct {
	Items                        []T
	Page, PageSize, Total, Pages int
}

func (s *Service) ListSources(ctx context.Context, query string, page, size int) (*Page[*domain.Source], error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items, total, err := s.Store.ListSources(ctx, strings.TrimSpace(query), size, (page-1)*size)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &Page[*domain.Source]{Items: items, Page: page, PageSize: size, Total: total, Pages: (total + size - 1) / size}, nil
}

// EnableSource : activation refusée tant qu'aucun budget n'est défini (une source sans plafond ne doit jamais être appelée).
func (s *Service) EnableSource(ctx context.Context, id string) (*domain.Source, error) {
	src, err := s.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	q, err := s.Store.QuotaByID(ctx, id, s.Now())
	if err != nil || q.MonthlyBudgetMicros <= 0 || q.DailyCapMicros <= 0 {
		return nil, conflict("définissez d'abord le budget mensuel et le plafond journalier de cette source")
	}
	if err := s.Store.SetEnabled(ctx, id, true); err != nil {
		return nil, apperr.Internal(err)
	}
	src.Enabled = true
	return src, nil
}

func (s *Service) DisableSource(ctx context.Context, id string) (*domain.Source, error) {
	src, err := s.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.Store.SetEnabled(ctx, id, false); err != nil {
		return nil, apperr.Internal(err)
	}
	src.Enabled = false
	return src, nil
}

// SetRawStorage : à n'autoriser qu'après vérification des conditions d'usage ; refusé pour une source sans date de vérification.
func (s *Service) SetRawStorage(ctx context.Context, id string, allowed bool) (*domain.Source, error) {
	src, err := s.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.Store.SetRawStorage(ctx, id, allowed, s.Now()); err != nil {
		return nil, apperr.Internal(err)
	}
	src.AllowsRawStorage = allowed
	return s.GetSource(ctx, id)
}

// ── quota ───────────────────────────────────────────────────────────────────

func (s *Service) GetQuota(ctx context.Context, id string) (*domain.Quota, error) {
	if _, err := s.GetSource(ctx, id); err != nil {
		return nil, err
	}
	q, err := s.Store.QuotaByID(ctx, id, s.Now())
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return q, nil
}

// PutQuota remplace le budget. Relever le budget lève la suspension automatique en cours. La bascule est limitée à un saut.
func (s *Service) PutQuota(ctx context.Context, in domain.Quota) (*domain.Quota, error) {
	src, err := s.GetSource(ctx, in.SourceID)
	if err != nil {
		return nil, err
	}
	in.Currency = src.Currency
	if err := domain.ValidateQuota(&in); err != nil {
		return nil, err
	}
	if in.FallbackSourceID != nil {
		if _, err := s.GetSource(ctx, *in.FallbackSourceID); err != nil {
			return nil, invalid("/fallbackSourceId", "la source de secours n'existe pas")
		}
	}
	if err := s.Store.PutQuota(ctx, &in); errors.Is(err, ErrFallbackChain) {
		return nil, invalid("/fallbackSourceId", "la bascule est limitée à un seul saut : le secours ne peut pas avoir de secours, ni être lui-même un secours")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	if src.SuspendedUntil != nil { // le budget a pu être relevé : on relève la suspension, l'appel suivant la rétablira si besoin
		_ = s.Store.SetSuspendedUntil(ctx, src.ID, nil)
	}
	return s.GetQuota(ctx, in.SourceID)
}

// ── appel d'une source ──────────────────────────────────────────────────────

// CallInput : un appel demandé par le service search. `SearchID` rend l'appel idempotent : une source n'est appelée qu'UNE fois par recherche.
type CallInput struct {
	SourceID, SearchID, Query, Language, Country string
	Deadline                                     time.Duration
	IsTest                                       bool
}

type CallOutput struct {
	Status           string
	Hits             []domain.RawHit
	Duration         time.Duration
	Attempts         int
	CostMicros       int64
	Currency         string
	FallbackSourceID *string
	ErrorKind        string
}

func (s *Service) CallSource(ctx context.Context, in CallInput) (*CallOutput, error) {
	src, err := s.GetSource(ctx, in.SourceID)
	if err != nil {
		return nil, err
	}
	return s.callSource(ctx, src, in)
}

// callSource fait l'appel pour une source déjà chargée (un test appelle avec une copie activée, sans toucher à la base).
func (s *Service) callSource(ctx context.Context, src *domain.Source, in CallInput) (*CallOutput, error) {
	now := s.Now()
	out := &CallOutput{Currency: src.Currency}
	fallback := func() {
		if q, err := s.Store.QuotaByID(ctx, src.ID, now); err == nil {
			out.FallbackSourceID = q.FallbackSourceID
		}
	}
	if !src.Enabled {
		out.Status = domain.CallDisabled
		return out, nil
	}
	if src.Suspended(now) {
		out.Status = domain.CallBudgetExhausted
		fallback()
		return out, nil
	}
	if s.breakerOpen(src.ID, now) {
		out.Status = domain.CallCircuitOpen
		fallback()
		return out, nil
	}
	conn, err := s.Connectors.For(src)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	timeout := time.Duration(src.TimeoutMs) * time.Millisecond
	if in.Deadline > 0 && in.Deadline < timeout {
		timeout = in.Deadline
	}
	// ── 1. réservation atomique (échec = l'appel n'a PAS lieu) ──
	var resID string
	if src.CostPerCallMicros > 0 {
		q, err := s.Store.QuotaByID(ctx, src.ID, now)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		resID = s.NewID()
		err = s.Store.Reserve(ctx, Reservation{
			ID: resID, SourceID: src.ID, SearchID: in.SearchID, AmountMicros: src.CostPerCallMicros, Day: domain.DayStart(now), Month: domain.MonthStart(now),
			DailyCap: q.DailyCapMicros, MonthlyCap: q.MonthlyBudgetMicros, CreatedAt: now,
			ExpiresAt: now.Add(timeout*time.Duration(src.MaxRetries+1) + ReservationMargin),
		})
		switch {
		case errors.Is(err, ErrAlreadyCalled):
			out.Status = domain.CallAlreadyCalled
			return out, nil
		case errors.Is(err, ErrBudget):
			// suspension automatique jusqu'à la fin de la période épuisée (jour ou mois)
			until := domain.NextDay(now)
			if q.MonthSpent+q.MonthReserved+src.CostPerCallMicros > q.MonthlyBudgetMicros {
				until = domain.NextMonth(now)
			}
			_ = s.Store.SetSuspendedUntil(ctx, src.ID, &until)
			out.Status = domain.CallBudgetExhausted
			fallback()
			return out, nil
		case err != nil:
			return nil, apperr.Internal(err)
		}
	}
	// ── 2. appel (délai par tentative, tentatives sur erreurs réseau et 5xx seulement) ──
	started := s.Now()
	hits, attempts, callErr := s.callWithRetries(ctx, conn, SearchParams{Query: in.Query, Language: in.Language, Country: in.Country, Count: DefaultCallResultsCount}, timeout, src.MaxRetries)
	out.Duration, out.Attempts = s.Now().Sub(started), attempts
	// ── 3. règlement de la réservation ──
	switch {
	case callErr == nil:
		out.Status, out.Hits, out.CostMicros = domain.CallOK, hits, src.CostPerCallMicros
		s.breakerSuccess(src.ID)
	default:
		var ce *CallError
		isCE := errors.As(callErr, &ce)
		out.ErrorKind = "unknown"
		if isCE {
			out.ErrorKind = ce.Kind
		}
		if IsTimeout(callErr) || (isCE && IsTimeout(ce.Err)) {
			out.Status, out.ErrorKind = domain.CallTimeout, "timeout"
			out.CostMicros = src.CostPerCallMicros // la requête est partie : on compte le coût estimé (prudence)
		} else {
			out.Status = domain.CallError
			if isCE && ce.Sent && ce.Kind != "http_4xx" && ce.Kind != "quota" && ce.Kind != "http_5xx" {
				out.CostMicros = src.CostPerCallMicros
			}
		}
		s.breakerFailure(src.ID, s.Now())
		fallback()
	}
	if resID != "" {
		sctx := context.WithoutCancel(ctx) // le règlement doit avoir lieu même si l'appelant a annulé
		if out.CostMicros > 0 {
			err = s.Store.Settle(sctx, resID, out.CostMicros, s.Now())
		} else {
			err = s.Store.Release(sctx, resID, s.Now())
		}
		if err != nil {
			return nil, apperr.Internal(err)
		}
	}
	_ = s.Store.LogCall(context.WithoutCancel(ctx), CallLog{ID: s.NewID(), SourceID: src.ID, SearchID: in.SearchID, StartedAt: started, DurationMs: int(out.Duration.Milliseconds()),
		Status: out.Status, Attempts: attempts, CostMicros: out.CostMicros, Currency: src.Currency, ErrorKind: out.ErrorKind})
	return out, nil
}

func (s *Service) callWithRetries(ctx context.Context, c Connector, p SearchParams, perAttempt time.Duration, maxRetries int) ([]domain.RawHit, int, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries+1; attempt++ {
		actx, cancel := context.WithTimeout(ctx, perAttempt)
		hits, err := c.Search(actx, p)
		cancel()
		if err == nil {
			return hits, attempt, nil
		}
		lastErr = err
		var ce *CallError
		if ctx.Err() != nil || !errors.As(err, &ce) || !ce.Retryable { // 4xx, quota, délai global dépassé : jamais rejoué
			return nil, attempt, err
		}
	}
	return nil, maxRetries + 1, lastErr
}

// ── disjoncteur (par instance) ──────────────────────────────────────────────

func (s *Service) breakerOpen(id string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.breakers[id]
	return b != nil && now.Before(b.openUntil)
}

func (s *Service) breakerSuccess(id string) {
	s.mu.Lock()
	delete(s.breakers, id)
	s.mu.Unlock()
}

func (s *Service) breakerFailure(id string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.breakers[id]
	if b == nil {
		b = &breaker{}
		s.breakers[id] = b
	}
	b.fails++
	if b.fails >= BreakerThreshold {
		b.openUntil = now.Add(BreakerCooldown) // après le délai, un appel d'essai passe : un échec rouvre aussitôt
		b.fails = BreakerThreshold - 1
	}
}

// ── test d'une source ───────────────────────────────────────────────────────

type TestResult struct {
	OK          bool
	Duration    time.Duration
	ResultCount int
	CostMicros  int64
	Currency    string
	ErrorKind   string
	Sample      []string
}

// TestSource consomme un appel RÉEL (réservé, compté, journalisé) ; 10 tests par heure et par membre.
func (s *Service) TestSource(ctx context.Context, actorID, sourceID, query, language, country string) (*TestResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, invalid("/query", "la requête de test est obligatoire")
	}
	n, err := s.Store.CountTests(ctx, actorID, s.Now().Add(-time.Hour))
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if n >= TestsPerHour {
		return nil, apperr.RateLimited(time.Hour)
	}
	src, err := s.GetSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if err := s.Store.RecordTest(ctx, actorID, s.Now()); err != nil {
		return nil, apperr.Internal(err)
	}
	// un test n'exige pas que la source soit activée : on l'active virtuellement pour l'appel (on teste AVANT d'activer)
	probe := *src
	probe.Enabled = true
	out, err := s.callSource(ctx, &probe, CallInput{SourceID: sourceID, SearchID: s.NewID(), Query: query, Language: language, Country: country, IsTest: true})
	if err != nil {
		return nil, err
	}
	res := &TestResult{OK: out.Status == domain.CallOK, Duration: out.Duration, ResultCount: len(out.Hits), CostMicros: out.CostMicros, Currency: out.Currency, ErrorKind: out.ErrorKind}
	if !res.OK && res.ErrorKind == "" {
		res.ErrorKind = out.Status
	}
	for i, h := range out.Hits {
		if i == 5 {
			break
		}
		res.Sample = append(res.Sample, h.Title)
	}
	return res, nil
}

// ── santé et coûts ──────────────────────────────────────────────────────────

type Health struct {
	SourceID                   string
	Status                     string
	CircuitOpen                bool
	P50Ms, P95Ms               int
	ErrorRate                  float64
	Calls24h                   int64
	LastSuccessAt, LastErrorAt *time.Time
	LastErrorKind              string
}

func (s *Service) GetHealth(ctx context.Context, id string) (*Health, error) {
	src, err := s.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	now := s.Now()
	st, err := s.Store.Stats(ctx, id, now.Add(-24*time.Hour))
	if err != nil {
		return nil, apperr.Internal(err)
	}
	h := &Health{SourceID: id, CircuitOpen: s.breakerOpen(id, now), P50Ms: st.P50Ms, P95Ms: st.P95Ms, Calls24h: st.Calls, LastSuccessAt: st.LastSuccessAt, LastErrorAt: st.LastErrAt, LastErrorKind: st.LastErrorKind}
	if st.Calls > 0 {
		h.ErrorRate = float64(st.Errors) / float64(st.Calls)
	}
	switch {
	case !src.Enabled:
		h.Status = "disabled"
	case src.Suspended(now):
		h.Status = "suspended"
	case h.CircuitOpen:
		h.Status = "down"
	case h.ErrorRate >= 0.2:
		h.Status = "degraded"
	default:
		h.Status = "healthy"
	}
	return h, nil
}

func (s *Service) Costs(ctx context.Context, sourceID string, from, to time.Time) ([]CostLine, error) {
	if !to.After(from) {
		return nil, invalid("/to", "la fin de période doit suivre le début")
	}
	lines, err := s.Store.Costs(ctx, sourceID, from, to)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].SourceName < lines[j].SourceName })
	return lines, nil
}

func (s *Service) ListActive(ctx context.Context) ([]ActiveSource, error) {
	a, err := s.Store.ListActive(ctx, s.Now())
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return a, nil
}

// ── tâches périodiques ──────────────────────────────────────────────────────

// RunMaintenance libère les réservations expirées (chaque minute) et réconcilie les compteurs (chaque heure) ; purge le journal des appels (90 jours).
func (s *Service) RunMaintenance(ctx context.Context, log func(msg string, kv ...any)) {
	tick, hour := time.NewTicker(time.Minute), time.NewTicker(time.Hour)
	defer tick.Stop()
	defer hour.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if n, err := s.Store.ReleaseExpired(ctx, s.Now()); err != nil {
				log("libération des réservations expirées", "error", err.Error())
			} else if n > 0 {
				log("réservations expirées libérées", "count", n)
			}
		case <-hour.C:
			if err := s.Store.Reconcile(ctx, s.Now()); err != nil {
				log("réconciliation du budget", "error", err.Error())
			}
			if _, err := s.Store.PurgeCalls(ctx, s.Now().AddDate(0, 0, -90)); err != nil {
				log("purge du journal des appels", "error", err.Error())
			}
		}
	}
}
