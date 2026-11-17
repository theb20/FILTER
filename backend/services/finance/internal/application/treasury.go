package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
	"google.golang.org/grpc/codes"
)

func bankSnap(b *domain.BankAccount) map[string]any {
	return map[string]any{"name": b.Name, "bank": b.BankName, "accountId": b.AccountID, "currency": b.Currency, "hasNumber": b.HasNumber, "last4": strOrNil(b.NumberLast4),
		"openingBalance": b.OpeningBalance, "openingDate": b.OpeningDate.Format(domain.DateLayout), "active": b.Active}
}

func (s *Service) sealNumber(number *string) (enc []byte, last4 *string, err error) {
	if number == nil || strings.TrimSpace(*number) == "" {
		return nil, nil, nil
	}
	if s.Box == nil {
		return nil, nil, precondition("le chiffrement des numéros de compte n'est pas configuré sur ce serveur : le numéro n'a pas été enregistré")
	}
	full, l4, err := domain.NormalizeBankNumber(*number)
	if err != nil {
		return nil, nil, invalid("/number", err.Error())
	}
	enc, err = s.Box.Seal([]byte(full))
	if err != nil {
		return nil, nil, apperr.Internal(err)
	}
	return enc, &l4, nil
}

func (s *Service) CreateBankAccount(ctx context.Context, a Actor, in domain.BankAccount, number *string) (*domain.BankAccount, error) {
	if a.ID == "" {
		return nil, apperr.Unauthenticated(apperr.CodeUnauthenticated, "auteur inconnu")
	}
	in.ID, in.Active = s.NewID(), true
	if err := in.Validate(); err != nil {
		return nil, err
	}
	enc, l4, err := s.sealNumber(number)
	if err != nil {
		return nil, err
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		st, err := s.Store.GetSettings(ctx)
		if err != nil {
			return err
		}
		in.Currency = st.BaseCurrency // la comptabilité est mono-devise : un compte bancaire est dans la devise de la comptabilité
		acc, err := s.Store.AccountByID(ctx, in.AccountID)
		if err != nil || !acc.Active {
			return invalid("/accountId", "compte du plan comptable introuvable ou inactif")
		}
		in.HasNumber, in.NumberLast4 = enc != nil, l4
		if err := s.Store.CreateBankAccount(ctx, &in, enc, l4, a.ID); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.bank_account.create", "finance_bank_account", in.ID, nil, bankSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.Store.BankAccountByID(ctx, in.ID)
}

// UpdateBankAccount : nom, banque, état et numéro. Le compte comptable lié et l'ouverture ne changent plus dès qu'un relevé est importé.
func (s *Service) UpdateBankAccount(ctx context.Context, a Actor, id string, in domain.BankAccount, active bool, number *string) (*domain.BankAccount, error) {
	cur, err := s.Store.BankAccountByID(ctx, id)
	if err != nil {
		return nil, fail(err)
	}
	in.ID, in.Active, in.Currency = id, active, cur.Currency
	if err := in.Validate(); err != nil {
		return nil, err
	}
	enc, l4, err := s.sealNumber(number)
	if err != nil {
		return nil, err
	}
	err = s.tx(ctx, func(ctx context.Context) error {
		n, err := s.Store.StatementCount(ctx, id)
		if err != nil {
			return err
		}
		if n > 0 && (in.AccountID != cur.AccountID || !in.OpeningDate.Equal(cur.OpeningDate) || in.OpeningBalance != cur.OpeningBalance) {
			return precondition("le compte comptable lié et l'ouverture ne peuvent plus changer : des relevés ont été importés")
		}
		if in.AccountID != cur.AccountID {
			if acc, err := s.Store.AccountByID(ctx, in.AccountID); err != nil || !acc.Active {
				return invalid("/accountId", "compte du plan comptable introuvable ou inactif")
			}
		}
		in.HasNumber, in.NumberLast4 = cur.HasNumber, cur.NumberLast4
		if enc != nil {
			in.HasNumber, in.NumberLast4 = true, l4
		}
		if err := s.Store.UpdateBankAccount(ctx, &in, enc, l4, enc != nil); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.bank_account.update", "finance_bank_account", id, bankSnap(cur), bankSnap(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.Store.BankAccountByID(ctx, id)
}

func (s *Service) GetBankAccount(ctx context.Context, id string) (*domain.BankAccount, error) {
	b, err := s.Store.BankAccountByID(ctx, id)
	return b, fail(err)
}

func (s *Service) ListBankAccounts(ctx context.Context, active *bool) ([]*domain.BankAccount, error) {
	v, err := s.Store.ListBankAccounts(ctx, active)
	return v, fail(err)
}

func (s *Service) ListStatements(ctx context.Context, id string) ([]*domain.Statement, error) {
	if _, err := s.GetBankAccount(ctx, id); err != nil {
		return nil, err
	}
	v, err := s.Store.ListStatements(ctx, id)
	return v, fail(err)
}

type StatementImport struct {
	Simulated                bool
	Total, Inserted, Skipped int
	Errors                   []domain.CSVError
	PeriodStart, PeriodEnd   *time.Time
	SumAmount                int64
	Statement                *domain.Statement
}

// ImportBankStatement : CSV à colonnes fixes. La SIMULATION ne modifie rien. L'import est « tout ou rien » : une seule ligne en erreur le refuse
// (importer partiellement fausserait silencieusement le solde). Un relevé importé est ensuite immuable.
func (s *Service) ImportBankStatement(ctx context.Context, a Actor, bankAccountID, filename, csv string, closing *int64, simulate bool) (*StatementImport, error) {
	if a.ID == "" {
		return nil, apperr.Unauthenticated(apperr.CodeUnauthenticated, "auteur inconnu")
	}
	b, err := s.GetBankAccount(ctx, bankAccountID)
	if err != nil {
		return nil, err
	}
	if !b.Active {
		return nil, precondition("ce compte bancaire est inactif")
	}
	name := domain.CleanFilename(filename)
	if name == "" {
		name = "releve.csv"
	}
	if len(csv) > 900_000 || strings.Count(csv, "\n") > 5000 {
		return nil, invalid("/csv", "fichier trop volumineux (5000 lignes au maximum)")
	}
	lines, perrs := domain.ParseStatementCSV(csv, b.Currency, b.ID, b.OpeningDate)
	res := &StatementImport{Simulated: simulate, Total: len(lines) + len(perrs), Errors: perrs}
	for _, l := range lines {
		res.SumAmount += l.Amount
		d := l.Date
		if res.PeriodStart == nil || d.Before(*res.PeriodStart) {
			res.PeriodStart = &d
		}
		if res.PeriodEnd == nil || d.After(*res.PeriodEnd) {
			res.PeriodEnd = &d
		}
	}
	if len(lines) == 0 && len(perrs) == 0 {
		return nil, invalid("/csv", "le fichier ne contient aucune ligne : date ; libellé ; référence ; montant")
	}
	fps := make([]string, len(lines))
	for i, l := range lines {
		fps[i] = l.Fingerprint
	}
	exist, err := s.Store.ExistingFingerprints(ctx, b.ID, fps)
	if err != nil {
		return nil, fail(err)
	}
	for _, l := range lines {
		if exist[l.Fingerprint] {
			res.Skipped++
		} else {
			res.Inserted++
		}
	}
	sum := sha256.Sum256([]byte(csv))
	sha := hex.EncodeToString(sum[:])
	already, err := s.Store.StatementFileExists(ctx, b.ID, sha)
	if err != nil {
		return nil, fail(err)
	}
	if simulate {
		return res, nil
	}
	if len(perrs) > 0 {
		return res, apperr.New(codes.FailedPrecondition, apperr.CodeValidation, fmt.Sprintf("import refusé : %d ligne(s) en erreur. Corrigez le fichier (la simulation les détaille) : un import partiel fausserait le solde.", len(perrs)))
	}
	if already {
		return res, fail(ErrFileImported)
	}
	st := &domain.Statement{ID: s.NewID(), BankAccountID: b.ID, Filename: name, SHA256: sha, PeriodStart: *res.PeriodStart, PeriodEnd: *res.PeriodEnd, LineCount: len(lines),
		InsertedCount: res.Inserted, SkippedCount: res.Skipped, ClosingBalance: closing, ImportedBy: a.ID, ImportedAt: s.Now()}
	err = s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.InsertStatement(ctx, st); err != nil {
			return err
		}
		n, err := s.Store.InsertStatementLines(ctx, st, lines, s.NewID)
		if err != nil {
			return err
		}
		if n != res.Inserted { // une écriture concurrente a inséré une des lignes entre-temps : l'import est annulé, jamais approximatif
			return conflict("le relevé a changé pendant l'import : relancez la simulation puis l'import")
		}
		return s.audit(ctx, a, "finance.bank_statement.import", "finance_bank_account", b.ID, nil,
			map[string]any{"file": name, "sha256": sha, "lines": len(lines), "inserted": res.Inserted, "skipped": res.Skipped, "sum": res.SumAmount})
	})
	if err != nil {
		return nil, err
	}
	res.Statement = st
	return res, nil
}

func (s *Service) ListBankLines(ctx context.Context, f BankLineFilter, page, size int) (*Page[*domain.BankLine], error) {
	if _, err := s.GetBankAccount(ctx, f.BankAccountID); err != nil {
		return nil, err
	}
	page, size = paging(page, size)
	f.Query = strings.TrimSpace(f.Query)
	items, total, err := s.Store.ListBankLines(ctx, f, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

func (s *Service) ListLedgerLines(ctx context.Context, bankAccountID string, onlyUnmatched bool, query string, page, size int) (*Page[domain.LedgerCandidate], error) {
	if _, err := s.GetBankAccount(ctx, bankAccountID); err != nil {
		return nil, err
	}
	page, size = paging(page, size)
	items, total, err := s.Store.ListLedgerLines(ctx, bankAccountID, onlyUnmatched, strings.TrimSpace(query), size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

// ListCandidates : suggestions de rapprochement (score), JAMAIS appliquées automatiquement.
func (s *Service) ListCandidates(ctx context.Context, bankLineID string) ([]domain.LedgerCandidate, error) {
	line, err := s.Store.BankLineByID(ctx, bankLineID)
	if err != nil {
		return nil, fail(err)
	}
	if line.Remaining <= 0 {
		return []domain.LedgerCandidate{}, nil
	}
	cs, err := s.Store.LedgerCandidates(ctx, line.BankAccountID, line.Amount > 0, 300)
	if err != nil {
		return nil, fail(err)
	}
	for i := range cs {
		domain.ScoreCandidate(line, &cs[i])
	}
	domain.SortCandidates(cs)
	if len(cs) > 10 {
		cs = cs[:10]
	}
	return cs, nil
}

// CreateReconciliation alloue un montant entre une ligne de relevé et une ligne d'écriture validée du compte banque.
// Montant 0 = le plus grand possible. Ce qui reste d'un côté ou de l'autre reste visible : aucun écart n'est absorbé.
func (s *Service) CreateReconciliation(ctx context.Context, a Actor, bankLineID, entryLineID string, amount int64) (*domain.BankLine, error) {
	if a.ID == "" {
		return nil, apperr.Unauthenticated(apperr.CodeUnauthenticated, "auteur inconnu")
	}
	if amount < 0 {
		return nil, invalid("/amount", "le montant ne peut pas être négatif")
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		line, err := s.Store.BankLineByID(ctx, bankLineID)
		if err != nil {
			return err
		}
		accID, rem, err := s.Store.LedgerLineRemaining(ctx, entryLineID)
		if err != nil {
			return err
		}
		b, err := s.Store.BankAccountByID(ctx, line.BankAccountID)
		if err != nil {
			return err
		}
		if accID != b.AccountID {
			return ErrReconAccount
		}
		if amount == 0 {
			amount = min(line.Remaining, rem)
		}
		if amount <= 0 {
			return ErrReconOver
		}
		id := s.NewID()
		if err := s.Store.CreateReconciliation(ctx, id, line.BankAccountID, bankLineID, entryLineID, amount, a.ID, s.Now()); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.reconciliation.create", "finance_bank_account", line.BankAccountID, nil, map[string]any{"reconciliation": id, "bankLine": bankLineID, "entryLine": entryLineID, "amount": amount})
	})
	if err != nil {
		return nil, err
	}
	l, err := s.Store.BankLineByID(ctx, bankLineID)
	return l, fail(err)
}

func (s *Service) DeleteReconciliation(ctx context.Context, a Actor, id string) (*domain.BankLine, error) {
	var lineID string
	err := s.tx(ctx, func(ctx context.Context) error {
		l, err := s.Store.ReconciliationLine(ctx, id)
		if err != nil {
			return err
		}
		lineID = l
		bl, err := s.Store.BankLineByID(ctx, l)
		if err != nil {
			return err
		}
		if err := s.Store.DeleteReconciliation(ctx, id); err != nil {
			return err
		}
		return s.audit(ctx, a, "finance.reconciliation.delete", "finance_bank_account", bl.BankAccountID, map[string]any{"reconciliation": id, "bankLine": l}, nil)
	})
	if err != nil {
		return nil, err
	}
	l, err := s.Store.BankLineByID(ctx, lineID)
	return l, fail(err)
}

func (s *Service) GetReconciliationSummary(ctx context.Context, bankAccountID string) (*domain.ReconSummary, error) {
	if _, err := s.GetBankAccount(ctx, bankAccountID); err != nil {
		return nil, err
	}
	v, err := s.Store.ReconSummary(ctx, bankAccountID)
	return v, fail(err)
}

// GetCashForecast : projection hebdomadaire des FLUX CONNUS uniquement (opérations comptabilisées en compte non réglées, avec leur échéance, et occurrences
// récurrentes à venir non encore générées). Aucune statistique ni hypothèse cachée : tout ce qui n'est pas daté ou est en retard est affiché à part.
func (s *Service) GetCashForecast(ctx context.Context, horizonDays int) (*domain.Forecast, error) {
	switch horizonDays {
	case 0:
		horizonDays = 90
	case 30, 60, 90, 180:
	default:
		return nil, invalid("/horizonDays", "30, 60, 90 ou 180 jours")
	}
	today := s.Now().Truncate(24 * time.Hour)
	st, err := s.Store.GetSettings(ctx)
	if err != nil {
		return nil, fail(err)
	}
	data, err := s.Store.ForecastData(ctx, today)
	if err != nil {
		return nil, fail(err)
	}
	end := today.AddDate(0, 0, horizonDays-1)
	f := &domain.Forecast{Currency: st.BaseCurrency, AsOf: today, HorizonDays: horizonDays, OpeningBalance: data.OpeningBalance, BankAccounts: data.BankAccounts, ExcludedDrafts: data.DraftCount, ExcludedDraftsAmount: data.DraftNetAmount}
	weeks := (horizonDays + 6) / 7
	for i := 0; i < weeks; i++ {
		st := today.AddDate(0, 0, 7*i)
		en := st.AddDate(0, 0, 6)
		if en.After(end) {
			en = end
		}
		f.Buckets = append(f.Buckets, domain.ForecastBucket{Start: st, End: en})
	}
	place := func(inflow bool, due time.Time, amount int64) {
		for i := range f.Buckets {
			if !due.Before(f.Buckets[i].Start) && !due.After(f.Buckets[i].End) {
				if inflow {
					f.Buckets[i].Inflows += amount
				} else {
					f.Buckets[i].Outflows += amount
				}
				return
			}
		}
	}
	for _, o := range data.Open {
		switch {
		case o.Due == nil && o.Inflow:
			f.UndatedInflows += o.Amount
		case o.Due == nil:
			f.UndatedOutflows += o.Amount
		case o.Due.Before(today) && o.Inflow:
			f.OverdueInflows += o.Amount
		case o.Due.Before(today):
			f.OverdueOutflows += o.Amount
		case !o.Due.After(end):
			place(o.Inflow, *o.Due, o.Amount)
		}
	}
	tpls, err := s.Store.ListRecurring(ctx)
	if err != nil {
		return nil, fail(err)
	}
	recurring := 0
	for _, t := range tpls {
		if !t.Active {
			continue
		}
		for _, m := range t.DueMonths(end) {
			day := time.Date(m.Year(), m.Month(), t.DayOfMonth, 0, 0, 0, 0, time.UTC)
			if day.Before(today) {
				continue // occurrence échue non générée : à générer, donc connue comme brouillon d'opération (non projetée)
			}
			place(t.Kind == domain.KindIncome, day, t.AmountNet+t.TaxAmount)
			recurring++
		}
	}
	bal := f.OpeningBalance
	for i := range f.Buckets {
		bal += f.Buckets[i].Inflows - f.Buckets[i].Outflows
		f.Buckets[i].Balance = bal
	}
	f.Assumptions = []string{
		"Solde de départ : grand livre des comptes bancaires actifs, écritures validées datées d'aujourd'hui ou avant.",
		"Encaissements et décaissements : opérations comptabilisées « en compte », non réglées, à leur date d'échéance.",
		fmt.Sprintf("Occurrences récurrentes à venir non encore générées : %d (à leur date).", recurring),
		"Non projetés, affichés à part : montants en retard (échéance passée) et sans échéance. Rien n'est supposé sur leur date de règlement.",
		"Non inclus : opérations non encore comptabilisées (brouillons), salaires et taxes non saisis, ventes futures. Aucune statistique ni prévision automatique.",
	}
	return f, nil
}
