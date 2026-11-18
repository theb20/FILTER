// Package domain : règles du service sourcing (sources externes, budget, quotas), sans dépendance technique.
package domain

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
)

// Types de source.
const (
	KindWebSearchAPI    = "web_search_api"
	KindManufacturerAPI = "manufacturer_api"
	KindDistributorFeed = "distributor_feed"
)

// Source est une source de données externe. Les montants sont en micro-unités de `Currency` (1 unité = 1 000 000).
type Source struct {
	ID                 string
	Kind               string
	Name               string
	BaseURL            string
	Enabled            bool
	SuspendedUntil     *time.Time // suspension automatique (budget atteint) ; levée à la période suivante ou au relèvement du budget
	Priority           int        // plus bas = interrogé plus tôt ; même valeur = en parallèle
	TimeoutMs          int
	MaxRetries         int // erreurs réseau et 5xx uniquement
	SecretRef          string
	CostPerCallMicros  int64 // estimation du coût MAXIMAL d'un appel : sert à la réservation
	Currency           string
	AllowsRawStorage   bool
	TermsCheckedAt     *time.Time
	MaxCacheTTLSeconds *int // nil = aucune mise en cache (conditions d'usage non vérifiées)
	CreatedAt          time.Time
}

// Suspended : la suspension automatique est-elle en cours à l'instant `now` ?
func (s *Source) Suspended(now time.Time) bool {
	return s.SuspendedUntil != nil && now.Before(*s.SuspendedUntil)
}

// Quota : budget et plafonds d'une source, avec la consommation courante.
type Quota struct {
	SourceID            string
	Currency            string
	MonthlyBudgetMicros int64
	DailyCapMicros      int64
	AlertThresholdPct   int
	FallbackSourceID    *string
	MonthSpent          int64
	MonthReserved       int64
	DaySpent            int64
	DayReserved         int64
}

// Alert : le seuil d'alerte du budget mensuel est-il atteint ?
func (q Quota) Alert() bool {
	if q.MonthlyBudgetMicros <= 0 {
		return q.MonthSpent+q.MonthReserved > 0
	}
	return (q.MonthSpent+q.MonthReserved)*100 >= q.MonthlyBudgetMicros*int64(q.AlertThresholdPct)
}

// RawHit : résultat brut d'une source. `Reference` ET `Brand` sont requis pour fusionner deux résultats (clé de la phase 1).
type RawHit struct {
	Title       string
	Reference   *string
	Brand       *string
	URL         *string
	PriceMicros *string
	Currency    *string
	PayloadJSON string
}

// Statuts d'un appel.
const (
	CallOK              = "ok"
	CallError           = "error"
	CallTimeout         = "timeout"
	CallBudgetExhausted = "budget_exhausted"
	CallCircuitOpen     = "circuit_open"
	CallDisabled        = "disabled"
	CallCached          = "cached"
	CallAlreadyCalled   = "already_called"
)

// ── périodes (UTC) ──────────────────────────────────────────────────────────

func DayStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func MonthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
func NextDay(t time.Time) time.Time   { return DayStart(t).AddDate(0, 0, 1) }
func NextMonth(t time.Time) time.Time { return MonthStart(t).AddDate(0, 1, 0) }

// ── validation ──────────────────────────────────────────────────────────────

var currencyRE = regexp.MustCompile(`^[A-Z]{3}$`)

// secretRefRE : nom de variable d'environnement (majuscules, chiffres, soulignés). Interdit « = », les espaces et les
// minuscules, c'est-à-dire tout ce qui ressemblerait à une clé collée par mégarde dans ce champ.
var secretRefRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,63}$`)

// ValidateSource applique les bornes du contrat ; chaque champ fautif est signalé.
func ValidateSource(s *Source) error {
	var f []apperr.Field
	bad := func(ptr, msg string) {
		f = append(f, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
	}
	switch s.Kind {
	case KindWebSearchAPI, KindManufacturerAPI, KindDistributorFeed:
	default:
		bad("/kind", "type de source inconnu")
	}
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" || len(s.Name) > 120 {
		bad("/name", "le nom est obligatoire (120 caractères maximum)")
	}
	if u, err := url.Parse(s.BaseURL); err != nil || u.Scheme == "" || u.Host == "" && u.Scheme != "test" {
		bad("/baseUrl", "adresse invalide")
	}
	if s.Priority < 0 {
		bad("/priority", "doit être supérieur ou égal à 0")
	}
	if s.TimeoutMs < 1 {
		bad("/timeoutMs", "doit être supérieur ou égal à 1")
	}
	if s.MaxRetries < 0 {
		bad("/maxRetries", "doit être supérieur ou égal à 0")
	}
	// Le champ ne porte que le NOM d'une variable d'environnement. Cette forme imposée empêche d'y coller la clé
	// elle-même (ou toute la ligne « NOM=valeur ») : un secret ne doit JAMAIS entrer en base, ni dans l'audit.
	s.SecretRef = strings.TrimSpace(s.SecretRef)
	switch {
	case s.SecretRef == "":
		bad("/secretRef", "la référence du secret est obligatoire (jamais la clé elle-même)")
	case !secretRefRE.MatchString(s.SecretRef):
		bad("/secretRef", "nom de variable d'environnement attendu (majuscules, chiffres et soulignés, ex. BRAVE_SEARCH_API_KEY) : jamais la clé elle-même")
	}
	if s.CostPerCallMicros < 0 {
		bad("/costPerCallMicros", "doit être supérieur ou égal à 0")
	}
	if !currencyRE.MatchString(s.Currency) {
		bad("/currency", "code ISO 4217 attendu (ex. USD)")
	}
	if s.MaxCacheTTLSeconds != nil && *s.MaxCacheTTLSeconds < 1 {
		bad("/maxCacheTtlSeconds", "doit être supérieur ou égal à 1, ou absent")
	}
	if len(f) > 0 {
		return apperr.Validation("source invalide", f...)
	}
	return nil
}

// ValidateQuota : plafond journalier ≤ budget mensuel, seuil entre 1 et 100.
func ValidateQuota(q *Quota) error {
	var f []apperr.Field
	bad := func(ptr, msg string) {
		f = append(f, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
	}
	if q.MonthlyBudgetMicros < 0 {
		bad("/monthlyBudgetMicros", "doit être supérieur ou égal à 0")
	}
	if q.DailyCapMicros < 0 {
		bad("/dailyCapMicros", "doit être supérieur ou égal à 0")
	}
	if q.DailyCapMicros > q.MonthlyBudgetMicros {
		bad("/dailyCapMicros", "le plafond journalier ne peut pas dépasser le budget mensuel")
	}
	if q.AlertThresholdPct < 1 || q.AlertThresholdPct > 100 {
		bad("/alertThresholdPct", "entre 1 et 100")
	}
	if q.FallbackSourceID != nil && *q.FallbackSourceID == q.SourceID {
		bad("/fallbackSourceId", "une source ne peut pas être son propre secours")
	}
	if len(f) > 0 {
		return apperr.Validation("quota invalide", f...)
	}
	return nil
}
