package domain

import (
	"bytes"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	KindExpense = "expense"
	KindIncome  = "income"

	SettleImmediate = "immediate"
	SettleOnAccount = "on_account"

	OpDraft      = "draft"       // aucune écriture
	OpEntryDraft = "entry_draft" // écriture en brouillon
	OpRecorded   = "recorded"    // écriture validée
	OpReversed   = "reversed"    // écriture contre-passée (validée)

	MaxAttachmentSize = 5 << 20
)

type ThirdParty struct {
	ID, Kind    string
	ExternalRef *string
	DisplayName string
	Active      bool
}

func (t *ThirdParty) Validate() error {
	var f fieldErrs
	t.DisplayName = strings.TrimSpace(t.DisplayName)
	if t.Kind != "customer" && t.Kind != "supplier" {
		f.add("/kind", "customer ou supplier")
	}
	if t.DisplayName == "" || utf8.RuneCountInString(t.DisplayName) > 200 {
		f.add("/displayName", "nom obligatoire (200 caractères maximum)")
	}
	if t.ExternalRef != nil {
		r := strings.TrimSpace(*t.ExternalRef)
		if r == "" {
			t.ExternalRef = nil
		} else if utf8.RuneCountInString(r) > 100 {
			f.add("/externalRef", "100 caractères maximum")
		} else {
			t.ExternalRef = &r
		}
	}
	return f.err("tiers invalide")
}

type OpCategory struct {
	ID, Kind, Label, AccountID, AccountCode, AccountLabel string
	Nature                                                *string
	Active                                                bool
}

func validNature(n *string, ptr string, f *fieldErrs) {
	if n != nil && *n != "fixed" && *n != "variable" {
		f.add(ptr, "fixed ou variable")
	}
}

func (c *OpCategory) Validate() error {
	var f fieldErrs
	c.Label = strings.TrimSpace(c.Label)
	if c.Kind != KindExpense && c.Kind != KindIncome {
		f.add("/kind", "expense ou income")
	}
	if c.Label == "" || utf8.RuneCountInString(c.Label) > 120 {
		f.add("/label", "libellé obligatoire (120 caractères maximum)")
	}
	if c.AccountID == "" {
		f.add("/accountId", "le compte du plan comptable est obligatoire")
	}
	validNature(c.Nature, "/nature", &f)
	if c.Kind == KindIncome && c.Nature != nil {
		f.add("/nature", "la nature fixe/variable ne concerne que les dépenses")
	}
	return f.err("catégorie invalide")
}

type OperationPayment struct {
	ID                     string
	PaymentDate            time.Time
	Amount                 int64
	AccountID, AccountCode string
	EntryID, EntryStatus   string
	EntryNumber            *string
}

type Operation struct {
	ID, Kind                                            string
	OperationDate                                       time.Time
	DueDate                                             *time.Time
	ThirdPartyID, ThirdPartyName                        *string
	CategoryID, CategoryLabel, CategoryAccountID, Label string
	Reference                                           *string
	AmountNet, TaxAmount                                int64
	TaxAccountID                                        *string
	CounterpartyAccountID, CounterpartyAccountCode      string
	SettlementMode                                      string
	Nature                                              *string
	Currency, Status                                    string
	EntryID, EntryNumber                                *string
	SettledAmount                                       int64
	DuplicateConfirmed                                  bool
	RecurringTemplateID                                 *string
	AttachmentCount                                     int
	Payments                                            []OperationPayment
	CreatedBy                                           string
	CreatedAt                                           time.Time
}

func (o *Operation) Gross() int64 { return o.AmountNet + o.TaxAmount }
func (o *Operation) Remaining() int64 {
	if o.SettlementMode != SettleOnAccount {
		return 0
	}
	if r := o.Gross() - o.SettledAmount; r > 0 {
		return r
	}
	return 0
}

const maxAmount = int64(1) << 60

func (o *Operation) Validate() error {
	var f fieldErrs
	o.Label = strings.TrimSpace(o.Label)
	if o.Kind != KindExpense && o.Kind != KindIncome {
		f.add("/kind", "expense ou income")
	}
	if o.OperationDate.IsZero() {
		f.add("/operationDate", "date obligatoire")
	}
	if o.DueDate != nil && !o.OperationDate.IsZero() && o.DueDate.Before(o.OperationDate) {
		f.add("/dueDate", "l'échéance ne peut pas précéder la date de l'opération")
	}
	if o.Label == "" || utf8.RuneCountInString(o.Label) > 300 {
		f.add("/label", "libellé obligatoire (300 caractères maximum)")
	}
	if o.Reference != nil {
		r := strings.TrimSpace(*o.Reference)
		if r == "" {
			o.Reference = nil
		} else if utf8.RuneCountInString(r) > 100 {
			f.add("/reference", "100 caractères maximum")
		} else {
			o.Reference = &r
		}
	}
	if o.AmountNet <= 0 || o.AmountNet > maxAmount {
		f.add("/amountNet", "le montant hors taxes doit être strictement positif")
	}
	if o.TaxAmount < 0 || o.TaxAmount > maxAmount {
		f.add("/taxAmount", "la taxe ne peut pas être négative")
	}
	if o.TaxAmount > 0 && (o.TaxAccountID == nil || *o.TaxAccountID == "") {
		f.add("/taxAccountId", "choisissez le compte de taxe (aucun taux ni compte n'est supposé)")
	}
	if o.TaxAmount == 0 {
		o.TaxAccountID = nil
	}
	if o.CategoryID == "" {
		f.add("/categoryId", "catégorie obligatoire")
	}
	if o.CounterpartyAccountID == "" {
		f.add("/counterpartyAccountId", "compte de contrepartie obligatoire")
	}
	if o.SettlementMode != SettleImmediate && o.SettlementMode != SettleOnAccount {
		f.add("/settlementMode", "immediate ou on_account")
	}
	validNature(o.Nature, "/nature", &f)
	if o.Kind == KindIncome {
		o.Nature = nil
	}
	return f.err("opération invalide")
}

// EntryLines construit les lignes de l'écriture de constatation : charge (ou produit) HT, taxe, contrepartie TTC.
func (o *Operation) EntryLines(categoryAccountID string) []EntryLine {
	memo := o.Reference
	var l []EntryLine
	if o.Kind == KindExpense {
		l = append(l, EntryLine{AccountID: categoryAccountID, Debit: o.AmountNet, Memo: memo})
		if o.TaxAmount > 0 {
			l = append(l, EntryLine{AccountID: *o.TaxAccountID, Debit: o.TaxAmount, Memo: memo})
		}
		l = append(l, EntryLine{AccountID: o.CounterpartyAccountID, Credit: o.Gross(), Memo: memo})
		return l
	}
	l = append(l, EntryLine{AccountID: o.CounterpartyAccountID, Debit: o.Gross(), Memo: memo})
	l = append(l, EntryLine{AccountID: categoryAccountID, Credit: o.AmountNet, Memo: memo})
	if o.TaxAmount > 0 {
		l = append(l, EntryLine{AccountID: *o.TaxAccountID, Credit: o.TaxAmount, Memo: memo})
	}
	return l
}

// PaymentLines : règlement d'une opération « en compte » (dette ou créance → banque ou caisse).
func (o *Operation) PaymentLines(settleAccountID string, amount int64) []EntryLine {
	if o.Kind == KindExpense {
		return []EntryLine{{AccountID: o.CounterpartyAccountID, Debit: amount}, {AccountID: settleAccountID, Credit: amount}}
	}
	return []EntryLine{{AccountID: settleAccountID, Debit: amount}, {AccountID: o.CounterpartyAccountID, Credit: amount}}
}

type Recurring struct {
	ID, Kind, CategoryID, CategoryLabel, Label string
	ThirdPartyID                               *string
	AmountNet, TaxAmount                       int64
	TaxAccountID                               *string
	CounterpartyAccountID, JournalID           string
	SettlementMode                             string
	Nature                                     *string
	DayOfMonth                                 int
	Start                                      time.Time
	End, LastGeneratedMonth                    *time.Time
	Active                                     bool
	CreatedBy                                  string
}

func (r *Recurring) Validate() error {
	op := Operation{Kind: r.Kind, OperationDate: r.Start, Label: r.Label, AmountNet: r.AmountNet, TaxAmount: r.TaxAmount, TaxAccountID: r.TaxAccountID, CategoryID: r.CategoryID,
		CounterpartyAccountID: r.CounterpartyAccountID, SettlementMode: r.SettlementMode, Nature: r.Nature}
	var f fieldErrs
	if err := op.Validate(); err != nil {
		return err
	}
	r.Label, r.TaxAccountID, r.Nature = op.Label, op.TaxAccountID, op.Nature
	if r.DayOfMonth < 1 || r.DayOfMonth > 28 {
		f.add("/dayOfMonth", "entre 1 et 28 (valable pour tous les mois)")
	}
	if r.JournalID == "" {
		f.add("/journalId", "journal obligatoire")
	}
	if r.End != nil && r.End.Before(r.Start) {
		f.add("/endDate", "la fin précède le début")
	}
	return f.err("modèle récurrent invalide")
}

// DueMonths : mois (1er du mois) à générer, du premier mois non encore généré jusqu'à `upTo` inclus.
func (r *Recurring) DueMonths(upTo time.Time) []time.Time {
	first := time.Date(r.Start.Year(), r.Start.Month(), 1, 0, 0, 0, 0, time.UTC)
	if r.LastGeneratedMonth != nil {
		first = r.LastGeneratedMonth.AddDate(0, 1, 0)
		first = time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	var out []time.Time
	for m := first; ; m = m.AddDate(0, 1, 0) {
		day := time.Date(m.Year(), m.Month(), r.DayOfMonth, 0, 0, 0, 0, time.UTC)
		if day.After(upTo) || (r.End != nil && day.After(*r.End)) {
			break
		}
		if day.Before(r.Start) {
			continue
		}
		out = append(out, m)
		if len(out) >= 60 {
			break
		}
	}
	return out
}

type Attachment struct {
	ID, OperationID, Filename, ContentType, SHA256 string
	Size                                           int64
	CreatedAt                                      time.Time
}

// DetectContentType : le type est déduit du CONTENU (jamais de la seule extension ni de l'en-tête déclaré).
func DetectContentType(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("%PDF-")):
		return "application/pdf"
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	}
	return ""
}

// CleanFilename retire tout chemin et tout caractère de contrôle.
func CleanFilename(n string) string {
	n = strings.ReplaceAll(n, "\\", "/")
	if i := strings.LastIndex(n, "/"); i >= 0 {
		n = n[i+1:]
	}
	var b strings.Builder
	for _, r := range n {
		if r >= 32 && r != 127 && r != '"' && r != '<' && r != '>' {
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(out) > 200 {
		out = string([]rune(out)[:200])
	}
	return out
}
