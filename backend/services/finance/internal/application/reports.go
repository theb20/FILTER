package application

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

type LedgerPage struct {
	*Ledger
	Currency              string
	Page, PageSize, Pages int
}

type TrialBalance struct {
	From, To                time.Time
	Currency                string
	Rows                    []TrialRow
	TotalDebit, TotalCredit int64
	Balanced                bool
}

func checkRange(from, to *time.Time) error {
	if from != nil && to != nil && to.Before(*from) {
		return invalid("/to", "la date de fin précède la date de début")
	}
	return nil
}

// GeneralLedger : écritures VALIDÉES d'un compte, avec solde cumulé (débit − crédit). Calcul entièrement en base.
func (s *Service) GeneralLedger(ctx context.Context, accountID string, from, to *time.Time, page, size int) (*LedgerPage, error) {
	if err := checkRange(from, to); err != nil {
		return nil, err
	}
	page, size = paging(page, size)
	st, err := s.Store.GetSettings(ctx)
	if err != nil {
		return nil, fail(err)
	}
	l, err := s.Store.GeneralLedger(ctx, accountID, from, to, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return &LedgerPage{Ledger: l, Currency: st.BaseCurrency, Page: page, PageSize: size, Pages: (l.Total + size - 1) / size}, nil
}

// GetTrialBalance : soldes d'ouverture, mouvements et soldes de clôture par compte ; `Balanced` doit toujours être vrai.
func (s *Service) GetTrialBalance(ctx context.Context, from, to time.Time) (*TrialBalance, error) {
	if err := checkRange(&from, &to); err != nil {
		return nil, err
	}
	st, err := s.Store.GetSettings(ctx)
	if err != nil {
		return nil, fail(err)
	}
	rows, err := s.Store.TrialBalance(ctx, from, to)
	if err != nil {
		return nil, fail(err)
	}
	tb := &TrialBalance{From: from, To: to, Currency: st.BaseCurrency, Rows: rows}
	var open, closing int64
	for _, r := range rows {
		tb.TotalDebit += r.Debit
		tb.TotalCredit += r.Credit
		open += r.Opening
		closing += r.Closing
	}
	tb.Balanced = tb.TotalDebit == tb.TotalCredit && open == 0 && closing == 0
	return tb, nil
}

// csvText neutralise l'injection de formules (=, +, -, @) dans les champs texte saisis par les utilisateurs.
func csvText(s string) string {
	if s != "" && strings.ContainsAny(s[:1], "=+-@\t\r") {
		return "'" + s
	}
	return s
}

func i64(n int64) string { return strconv.FormatInt(n, 10) }

func csvBytes(rows [][]string) []byte {
	var b bytes.Buffer
	b.WriteString("\xef\xbb\xbf") // BOM UTF-8 : Excel reconnaît l'encodage
	w := csv.NewWriter(&b)
	w.Comma = ';'
	_ = w.WriteAll(rows)
	return b.Bytes()
}

func (s *Service) ExportTrialBalance(ctx context.Context, from, to time.Time) ([]byte, error) {
	tb, err := s.GetTrialBalance(ctx, from, to)
	if err != nil {
		return nil, err
	}
	rows := [][]string{{"compte", "libelle", "nature", "solde_ouverture", "debit", "credit", "solde_cloture", "devise"}}
	for _, r := range tb.Rows {
		rows = append(rows, []string{csvText(r.Code), csvText(r.Label), r.Kind, i64(r.Opening), i64(r.Debit), i64(r.Credit), i64(r.Closing), tb.Currency})
	}
	rows = append(rows, []string{"TOTAL", "", "", "", i64(tb.TotalDebit), i64(tb.TotalCredit), "", tb.Currency})
	return csvBytes(rows), nil
}

func (s *Service) ExportGeneralLedger(ctx context.Context, accountID string, from, to *time.Time) ([]byte, error) {
	if err := checkRange(from, to); err != nil {
		return nil, err
	}
	st, err := s.Store.GetSettings(ctx)
	if err != nil {
		return nil, fail(err)
	}
	l, err := s.Store.GeneralLedger(ctx, accountID, from, to, 200000, 0)
	if err != nil {
		return nil, fail(err)
	}
	rows := [][]string{{"compte", "date", "numero", "journal", "libelle", "reference", "memo", "debit", "credit", "solde", "devise"}}
	opt := func(p *string) string {
		if p == nil {
			return ""
		}
		return csvText(*p)
	}
	rows = append(rows, []string{csvText(l.Account.Code), "", "", "", "Solde d'ouverture", "", "", "", "", i64(l.Opening), st.BaseCurrency})
	for _, x := range l.Lines {
		rows = append(rows, []string{csvText(l.Account.Code), x.AccountingDate.Format(domain.DateLayout), x.EntryNumber, x.JournalCode, csvText(x.Label), opt(x.Reference), opt(x.Memo), i64(x.Debit), i64(x.Credit), i64(x.Balance), st.BaseCurrency})
	}
	rows = append(rows, []string{csvText(l.Account.Code), "", "", "", "Solde de clôture", "", "", "", "", i64(l.Closing), st.BaseCurrency})
	return csvBytes(rows), nil
}
