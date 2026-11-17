package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var currencyExponent = map[string]int{"XOF": 0, "XAF": 0, "EUR": 2, "USD": 2, "GBP": 2, "MAD": 2}

// Exponent : nombre de décimales de l'unité mineure (XOF : 0 ; EUR : 2).
func Exponent(currency string) int {
	if e, ok := currencyExponent[currency]; ok {
		return e
	}
	return 2
}

// ParseAmountMinor convertit « -1 250,5 » en unités mineures SANS flottant. Refuse les décimales en excès.
func ParseAmountMinor(s, currency string) (int64, error) {
	t := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == ' ' || r == ' ' {
			return -1
		}
		return r
	}, s)
	t = strings.ReplaceAll(t, ",", ".")
	neg := false
	if strings.HasPrefix(t, "-") || strings.HasPrefix(t, "−") {
		neg, t = true, strings.TrimLeft(t, "-−")
	} else if strings.HasPrefix(t, "+") {
		t = t[1:]
	}
	if t == "" {
		return 0, fmt.Errorf("montant vide")
	}
	whole, frac, _ := strings.Cut(t, ".")
	if whole == "" || strings.ContainsAny(whole+frac, "+-−") {
		return 0, fmt.Errorf("montant invalide")
	}
	e := Exponent(currency)
	if len(frac) > e {
		return 0, fmt.Errorf("trop de décimales pour %s", currency)
	}
	for _, r := range whole + frac {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("montant invalide")
		}
	}
	n, err := strconv.ParseInt(whole+frac+strings.Repeat("0", e-len(frac)), 10, 64)
	if err != nil || n > maxAmount {
		return 0, fmt.Errorf("montant trop élevé")
	}
	if neg {
		n = -n
	}
	return n, nil
}

type BankAccount struct {
	ID, Name, BankName, Currency         string
	AccountID, AccountCode, AccountLabel string
	HasNumber                            bool
	NumberLast4                          *string
	OpeningBalance                       int64
	OpeningDate                          time.Time
	Active                               bool
	LedgerBalance, StatementBalance      int64
	LastStatementDate                    *time.Time
}

func (b *BankAccount) Validate() error {
	var f fieldErrs
	b.Name, b.BankName = strings.TrimSpace(b.Name), strings.TrimSpace(b.BankName)
	if b.Name == "" || utf8.RuneCountInString(b.Name) > 120 {
		f.add("/name", "nom obligatoire (120 caractères maximum)")
	}
	if b.BankName == "" || utf8.RuneCountInString(b.BankName) > 120 {
		f.add("/bankName", "banque obligatoire (120 caractères maximum)")
	}
	if b.AccountID == "" {
		f.add("/accountId", "compte du plan comptable obligatoire")
	}
	if b.OpeningDate.IsZero() {
		f.add("/openingDate", "date d'ouverture obligatoire")
	}
	return f.err("compte bancaire invalide")
}

// NormalizeBankNumber : lettres et chiffres uniquement, majuscules (IBAN, RIB, numéro local). Aucune structure n'est présumée.
func NormalizeBankNumber(s string) (full, last4 string, err error) {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	n := b.String()
	if len(n) < 6 || len(n) > 40 {
		return "", "", fmt.Errorf("numéro de compte invalide (6 à 40 lettres et chiffres)")
	}
	return n, n[len(n)-4:], nil
}

type Statement struct {
	ID, BankAccountID, Filename, SHA256    string
	PeriodStart, PeriodEnd                 time.Time
	LineCount, InsertedCount, SkippedCount int
	ClosingBalance                         *int64
	ImportedBy                             string
	ImportedAt                             time.Time
}

type ParsedLine struct {
	Date        time.Time
	Label       string
	Reference   *string
	Amount      int64
	Fingerprint string
}

type CSVError struct {
	Line    int
	Message string
}

func parseStatementDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "02/01/2006", "02-01-2006", "02.01.2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("date invalide (AAAA-MM-JJ ou JJ/MM/AAAA)")
}

func splitStatementLine(l string) []string {
	sep := ","
	if strings.Contains(l, ";") {
		sep = ";"
	} else if strings.Contains(l, "\t") {
		sep = "\t"
	}
	p := strings.Split(l, sep)
	for i := range p {
		p[i] = strings.Trim(strings.TrimSpace(p[i]), `"`)
	}
	return p
}

// ParseStatementCSV lit « date ; libellé ; référence ; montant ». Montant signé (négatif = sortie d'argent).
// Les lignes identiques d'un même fichier reçoivent un numéro d'occurrence : deux paiements identiques le même jour restent deux lignes.
func ParseStatementCSV(csv, currency, accountID string, notBefore time.Time) ([]ParsedLine, []CSVError) {
	var out []ParsedLine
	var errs []CSVError
	seen := map[string]int{}
	for i, raw := range strings.Split(strings.ReplaceAll(csv, "\r\n", "\n"), "\n") {
		raw = strings.TrimPrefix(strings.TrimSpace(raw), "\ufeff")
		if raw == "" {
			continue
		}
		f := splitStatementLine(raw)
		if i == 0 && len(f) > 0 && strings.EqualFold(strings.TrimSpace(f[0]), "date") {
			continue
		}
		if len(f) < 4 {
			errs = append(errs, CSVError{i + 1, "quatre colonnes attendues : date ; libellé ; référence ; montant"})
			continue
		}
		d, err := parseStatementDate(f[0])
		if err != nil {
			errs = append(errs, CSVError{i + 1, err.Error()})
			continue
		}
		if d.Before(notBefore) {
			errs = append(errs, CSVError{i + 1, "ligne antérieure à la date d'ouverture du compte (le solde d'ouverture la couvre déjà)"})
			continue
		}
		label := strings.TrimSpace(f[1])
		if label == "" || utf8.RuneCountInString(label) > 300 {
			errs = append(errs, CSVError{i + 1, "libellé obligatoire (300 caractères maximum)"})
			continue
		}
		var ref *string
		if r := strings.TrimSpace(f[2]); r != "" {
			if utf8.RuneCountInString(r) > 100 {
				errs = append(errs, CSVError{i + 1, "référence trop longue (100 caractères maximum)"})
				continue
			}
			ref = &r
		}
		amt, err := ParseAmountMinor(f[3], currency)
		if err != nil {
			errs = append(errs, CSVError{i + 1, err.Error()})
			continue
		}
		if amt == 0 {
			errs = append(errs, CSVError{i + 1, "un montant nul n'est pas un mouvement"})
			continue
		}
		key := fmt.Sprintf("%s|%s|%d|%s|%s", accountID, d.Format(DateLayout), amt, strings.ToLower(label), strings.ToLower(deref(ref)))
		seen[key]++
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d", key, seen[key])))
		out = append(out, ParsedLine{Date: d, Label: label, Reference: ref, Amount: amt, Fingerprint: hex.EncodeToString(sum[:])})
	}
	return out, errs
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

type Match struct {
	ID, EntryLineID, EntryID, EntryNumber, MatchedBy string
	Amount                                           int64
	MatchedAt                                        time.Time
}

type BankLine struct {
	ID, StatementID, BankAccountID, Label string
	LineDate                              time.Time
	Reference                             *string
	Amount, Matched, Remaining            int64
	Status                                string
	Matches                               []Match
}

func (l *BankLine) Compute() {
	abs := l.Amount
	if abs < 0 {
		abs = -abs
	}
	l.Remaining = abs - l.Matched
	switch {
	case l.Matched == 0:
		l.Status = "unmatched"
	case l.Remaining == 0:
		l.Status = "matched"
	default:
		l.Status = "partial"
	}
}

type LedgerCandidate struct {
	EntryLineID, EntryID, EntryNumber, Label string
	AccountingDate                           time.Time
	Reference                                *string
	Amount, Remaining                        int64
	Score                                    int
	ExactAmount                              bool
}

func tokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if utf8.RuneCountInString(f) >= 3 {
			out[f] = true
		}
	}
	return out
}

// ScoreCandidate : suggestion UNIQUEMENT (jamais appliquée seule). Montant, proximité de date, mots communs du libellé.
func ScoreCandidate(line *BankLine, c *LedgerCandidate) {
	want := line.Remaining
	c.ExactAmount = c.Remaining == want
	switch {
	case c.ExactAmount:
		c.Score += 100
	case c.Remaining < want:
		c.Score += 20
	}
	days := int(line.LineDate.Sub(c.AccountingDate).Hours() / 24)
	if days < 0 {
		days = -days
	}
	switch {
	case days <= 3:
		c.Score += 30
	case days <= 10:
		c.Score += 15
	case days <= 30:
		c.Score += 5
	}
	a, b := tokens(line.Label+" "+deref(line.Reference)), tokens(c.Label+" "+deref(c.Reference))
	common := 0
	for t := range a {
		if b[t] {
			common++
		}
	}
	if common > 3 {
		common = 3
	}
	c.Score += common * 10
}

func SortCandidates(cs []LedgerCandidate) {
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].Score != cs[j].Score {
			return cs[i].Score > cs[j].Score
		}
		return cs[i].AccountingDate.After(cs[j].AccountingDate)
	})
}

type ReconSummary struct {
	BankAccountID, Currency                       string
	LedgerBalance, StatementBalance, Difference   int64
	UnmatchedStatement, UnmatchedLedger           int64
	UnmatchedStatementCount, UnmatchedLedgerCount int
	Unexplained, OpeningGap                       int64
	LastStatementDate                             *time.Time
}

type ForecastBucket struct {
	Start, End                 time.Time
	Inflows, Outflows, Balance int64
}

type Forecast struct {
	Currency                        string
	AsOf                            time.Time
	HorizonDays                     int
	OpeningBalance                  int64
	Buckets                         []ForecastBucket
	OverdueInflows, OverdueOutflows int64
	UndatedInflows, UndatedOutflows int64
	ExcludedDrafts                  int
	ExcludedDraftsAmount            int64
	BankAccounts                    int
	Assumptions                     []string
}
