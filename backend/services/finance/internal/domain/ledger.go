// Package domain : règles de la comptabilité (grand livre), sans dépendance technique.
// Montants : entiers en unités mineures de la devise. Jamais de flottant.
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/theb20/FILTER/backend/pkg/apperr"
)

const (
	StatusDraft  = "draft"
	StatusPosted = "posted"

	OriginManual    = "manual"
	OriginOperation = "operation"
	OriginReversal  = "reversal"

	PeriodOpen   = "open"
	PeriodClosed = "closed"

	ApprovalReopenPeriod = "reopen_period"
	ApprovalPending      = "pending"
	ApprovalApproved     = "approved"
	ApprovalRejected     = "rejected"

	DateLayout = "2006-01-02"
)

var (
	AccountKinds = []string{"asset", "liability", "equity", "income", "expense", "other"}
	JournalKinds = []string{"sales", "purchases", "bank", "cash", "general"}

	accountCodeRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,19}$`)
	journalCodeRE = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)
	currencyRE    = regexp.MustCompile(`^[A-Z]{3}$`)
)

type fieldErrs []apperr.Field

func (f *fieldErrs) add(ptr, msg string) {
	*f = append(*f, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
}
func (f fieldErrs) err(msg string) error {
	if len(f) == 0 {
		return nil
	}
	return apperr.Validation(msg, f...)
}

func oneOf(v string, set []string) bool {
	for _, s := range set {
		if v == s {
			return true
		}
	}
	return false
}

// ── réglages ──

type Settings struct {
	BaseCurrency      string
	FourEyesMinAmount int64
	UpdatedAt         time.Time
}

func (s *Settings) Validate() error {
	var f fieldErrs
	if !currencyRE.MatchString(s.BaseCurrency) {
		f.add("/baseCurrency", "code ISO 4217 sur 3 lettres majuscules")
	}
	if s.FourEyesMinAmount < 0 {
		f.add("/fourEyesMinAmount", "doit être supérieur ou égal à 0")
	}
	return f.err("réglages invalides")
}

// ── plan comptable ──

type Account struct {
	ID, Code, Label, Kind string
	Active                bool
	CreatedAt             time.Time
}

func (a *Account) Validate() error {
	var f fieldErrs
	a.Code, a.Label = strings.TrimSpace(a.Code), strings.TrimSpace(a.Label)
	if !accountCodeRE.MatchString(a.Code) {
		f.add("/code", "lettres, chiffres, point, tiret ou souligné (20 caractères maximum)")
	}
	if a.Label == "" || utf8.RuneCountInString(a.Label) > 200 {
		f.add("/label", "libellé obligatoire (200 caractères maximum)")
	}
	if !oneOf(a.Kind, AccountKinds) {
		f.add("/kind", "asset, liability, equity, income, expense ou other")
	}
	return f.err("compte invalide")
}

// ── journaux ──

type Journal struct {
	ID, Code, Label, Kind string
	Active                bool
}

func (j *Journal) Validate() error {
	var f fieldErrs
	j.Code, j.Label = strings.TrimSpace(j.Code), strings.TrimSpace(j.Label)
	if !journalCodeRE.MatchString(j.Code) {
		f.add("/code", "majuscules et chiffres uniquement (10 caractères maximum)")
	}
	if j.Label == "" || utf8.RuneCountInString(j.Label) > 200 {
		f.add("/label", "libellé obligatoire (200 caractères maximum)")
	}
	if !oneOf(j.Kind, JournalKinds) {
		f.add("/kind", "sales, purchases, bank, cash ou general")
	}
	return f.err("journal invalide")
}

// ── exercices et périodes ──

type FiscalPeriod struct {
	ID, FiscalYearID string
	Number           int
	Start, End       time.Time
	Status           string
	ClosedAt         *time.Time
	ClosedBy         *string
}

type FiscalYear struct {
	ID, Label  string
	Start, End time.Time
	Periods    []FiscalPeriod
}

// Status calculé : open (aucune période close), closed (toutes closes), partially_closed sinon.
func (y *FiscalYear) Status() string {
	closed := 0
	for _, p := range y.Periods {
		if p.Status == PeriodClosed {
			closed++
		}
	}
	switch {
	case closed == 0:
		return PeriodOpen
	case closed == len(y.Periods):
		return PeriodClosed
	}
	return "partially_closed"
}

// BuildPeriods découpe l'exercice en mois calendaires (le premier et le dernier peuvent être partiels).
func BuildPeriods(start, end time.Time) []FiscalPeriod {
	var out []FiscalPeriod
	cur := start
	for n := 1; !cur.After(end); n++ {
		next := time.Date(cur.Year(), cur.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		pend := next.AddDate(0, 0, -1)
		if pend.After(end) {
			pend = end
		}
		out = append(out, FiscalPeriod{Number: n, Start: cur, End: pend, Status: PeriodOpen})
		cur = next
	}
	return out
}

func (y *FiscalYear) Validate() error {
	var f fieldErrs
	y.Label = strings.TrimSpace(y.Label)
	if y.Label == "" || utf8.RuneCountInString(y.Label) > 100 {
		f.add("/label", "libellé obligatoire (100 caractères maximum)")
	}
	switch {
	case y.Start.IsZero():
		f.add("/startDate", "date de début obligatoire")
	case y.End.IsZero():
		f.add("/endDate", "date de fin obligatoire")
	case y.End.Before(y.Start):
		f.add("/endDate", "la fin doit suivre le début")
	case y.End.After(y.Start.AddDate(2, 0, 0)):
		f.add("/endDate", "un exercice ne dépasse pas 24 mois")
	}
	return f.err("exercice invalide")
}

func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(DateLayout, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("date AAAA-MM-JJ attendue")
	}
	return t.UTC(), nil
}

// ── écritures ──

type EntryLine struct {
	ID, AccountID, AccountCode, AccountLabel string
	Debit, Credit                            int64
	Memo                                     *string
}

type EntrySource struct{ Type, ID, Kind string }

type Entry struct {
	ID, JournalID, JournalCode, FiscalPeriodID, FiscalYearID, FiscalYearLabel string
	AccountingDate                                                            time.Time
	Status                                                                    string
	EntryNumber                                                               *int64
	Label                                                                     string
	Reference                                                                 *string
	Currency, Origin                                                          string
	Source                                                                    *EntrySource
	ReversesEntryID, ReversedByEntryID                                        *string
	ReversalReason                                                            *string
	Lines                                                                     []EntryLine
	TotalDebit, TotalCredit                                                   int64
	CreatedBy                                                                 string
	CreatedAt                                                                 time.Time
	PostedBy                                                                  *string
	PostedAt                                                                  *time.Time
}

// Number : JOURNAL-AAAA-000001 (l'année est celle de l'exercice, pas de la date).
func (e *Entry) Number() *string {
	if e.EntryNumber == nil {
		return nil
	}
	n := fmt.Sprintf("%s-%s-%06d", e.JournalCode, e.FiscalYearLabel, *e.EntryNumber)
	return &n
}

// ValidateDraft vérifie la forme d'une saisie (équilibre exigé seulement à la validation).
func (e *Entry) ValidateDraft() error {
	var f fieldErrs
	e.Label = strings.TrimSpace(e.Label)
	if e.Label == "" || utf8.RuneCountInString(e.Label) > 300 {
		f.add("/label", "libellé obligatoire (300 caractères maximum)")
	}
	if e.Reference != nil {
		r := strings.TrimSpace(*e.Reference)
		if r == "" {
			e.Reference = nil
		} else if utf8.RuneCountInString(r) > 100 {
			f.add("/reference", "100 caractères maximum")
		} else {
			e.Reference = &r
		}
	}
	if e.JournalID == "" {
		f.add("/journalId", "journal obligatoire")
	}
	if e.AccountingDate.IsZero() {
		f.add("/accountingDate", "date comptable obligatoire")
	}
	if len(e.Lines) < 2 {
		f.add("/lines", "au moins deux lignes sont nécessaires")
	}
	if len(e.Lines) > 200 {
		f.add("/lines", "200 lignes au maximum")
	}
	for i := range e.Lines {
		l := &e.Lines[i]
		ptr := fmt.Sprintf("/lines/%d", i)
		if l.AccountID == "" {
			f.add(ptr+"/accountId", "compte obligatoire")
		}
		if l.Debit < 0 || l.Credit < 0 {
			f.add(ptr, "un montant ne peut pas être négatif")
		} else if (l.Debit == 0) == (l.Credit == 0) {
			f.add(ptr, "une ligne porte un débit OU un crédit strictement positif")
		}
		if l.Memo != nil {
			m := strings.TrimSpace(*l.Memo)
			if m == "" {
				l.Memo = nil
			} else if utf8.RuneCountInString(m) > 300 {
				f.add(ptr+"/memo", "300 caractères maximum")
			} else {
				l.Memo = &m
			}
		}
	}
	return f.err("écriture invalide")
}

// Totals calcule les totaux avec contrôle de débordement (jamais de flottant).
func (e *Entry) Totals() error {
	const max = int64(1) << 62
	var d, c int64
	for _, l := range e.Lines {
		if d > max-l.Debit || c > max-l.Credit {
			return apperr.Validation("montants trop élevés", apperr.Field{Pointer: "/lines", Code: apperr.CodeValidation, Message: "montants trop élevés"})
		}
		d += l.Debit
		c += l.Credit
	}
	e.TotalDebit, e.TotalCredit = d, c
	return nil
}

// ValidateForPosting : conditions de la validation (hors période et approbateur, vérifiés par le service).
func (e *Entry) ValidateForPosting() error {
	if err := e.ValidateDraft(); err != nil {
		return err
	}
	if err := e.Totals(); err != nil {
		return err
	}
	var f fieldErrs
	if e.TotalDebit != e.TotalCredit {
		f.add("/lines", fmt.Sprintf("l'écriture n'est pas équilibrée : débits %d, crédits %d (écart %d)", e.TotalDebit, e.TotalCredit, e.TotalDebit-e.TotalCredit))
	}
	if e.TotalDebit == 0 {
		f.add("/lines", "le total de l'écriture est nul")
	}
	return f.err("validation impossible")
}

// Reversal construit les lignes inversées d'une écriture validée.
func (e *Entry) ReversalLines() []EntryLine {
	out := make([]EntryLine, len(e.Lines))
	for i, l := range e.Lines {
		out[i] = EntryLine{AccountID: l.AccountID, AccountCode: l.AccountCode, AccountLabel: l.AccountLabel, Debit: l.Credit, Credit: l.Debit, Memo: l.Memo}
	}
	return out
}

// ── approbations ──

type Approval struct {
	ID, Type, SubjectID, SubjectLabel, Reason, Status string
	RequestedBy                                       string
	RequestedAt                                       time.Time
	DecidedBy                                         *string
	DecidedAt                                         *time.Time
	DecisionNote                                      *string
}

func ValidateReason(r string, ptr string) (string, error) {
	r = strings.TrimSpace(r)
	if utf8.RuneCountInString(r) < 5 || utf8.RuneCountInString(r) > 500 {
		return "", apperr.Validation("motif invalide", apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: "le motif est obligatoire (5 à 500 caractères)"})
	}
	return r, nil
}
