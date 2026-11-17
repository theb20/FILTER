package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

const numExprE = `(j.code || '-' || y.label || '-' || lpad(e.entry_number::text, 6, '0'))`

const baCols = `b.id::text, b.name, b.bank_name, b.currency, b.account_id::text, a.code, a.label, b.number_last4, b.opening_balance, b.opening_date, b.active,
	COALESCE((SELECT sum(l.debit - l.credit) FROM entry_lines l JOIN entries e ON e.id = l.entry_id AND e.status = 'posted' WHERE l.account_id = b.account_id), 0)::bigint,
	(b.opening_balance + COALESCE((SELECT sum(sl.amount) FROM bank_statement_lines sl WHERE sl.bank_account_id = b.id), 0))::bigint,
	(SELECT max(sl.line_date) FROM bank_statement_lines sl WHERE sl.bank_account_id = b.id)`
const baFrom = ` FROM bank_accounts b JOIN accounts a ON a.id = b.account_id`

func scanBA(r pgx.Row) (*domain.BankAccount, error) {
	var b domain.BankAccount
	if err := r.Scan(&b.ID, &b.Name, &b.BankName, &b.Currency, &b.AccountID, &b.AccountCode, &b.AccountLabel, &b.NumberLast4, &b.OpeningBalance, &b.OpeningDate, &b.Active, &b.LedgerBalance, &b.StatementBalance, &b.LastStatementDate); err != nil {
		return nil, notFound(err)
	}
	b.HasNumber = b.NumberLast4 != nil
	return &b, nil
}

func (s *Store) CreateBankAccount(ctx context.Context, b *domain.BankAccount, enc []byte, last4 *string, by string) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO bank_accounts (id, name, bank_name, currency, account_id, number_encrypted, number_last4, opening_balance, opening_date, active, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, b.ID, b.Name, b.BankName, b.Currency, b.AccountID, enc, last4, b.OpeningBalance, b.OpeningDate, b.Active, by)
	if pe := pgErr(err); pe != nil && pe.Code == "23505" && pe.ConstraintName == "bank_accounts_ledger_key" {
		return application.ErrBankLinked
	}
	return mapErr(err)
}

func (s *Store) UpdateBankAccount(ctx context.Context, b *domain.BankAccount, enc []byte, last4 *string, replace bool) error {
	q := `UPDATE bank_accounts SET name=$2, bank_name=$3, account_id=$4, opening_balance=$5, opening_date=$6, active=$7, updated_at=now()`
	args := []any{b.ID, b.Name, b.BankName, b.AccountID, b.OpeningBalance, b.OpeningDate, b.Active}
	if replace {
		q += `, number_encrypted=$8, number_last4=$9`
		args = append(args, enc, last4)
	}
	tag, err := s.q(ctx).Exec(ctx, q+` WHERE id=$1`, args...)
	if pe := pgErr(err); pe != nil && pe.Code == "23505" && pe.ConstraintName == "bank_accounts_ledger_key" {
		return application.ErrBankLinked
	}
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) BankAccountByID(ctx context.Context, id string) (*domain.BankAccount, error) {
	return scanBA(s.q(ctx).QueryRow(ctx, `SELECT `+baCols+baFrom+` WHERE b.id = $1`, id))
}

func (s *Store) ListBankAccounts(ctx context.Context, active *bool) ([]*domain.BankAccount, error) {
	q, args := `SELECT `+baCols+baFrom, []any{}
	if active != nil {
		q += ` WHERE b.active = $1`
		args = append(args, *active)
	}
	rows, err := s.q(ctx).Query(ctx, q+` ORDER BY lower(b.name)`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.BankAccount, error) { return scanBA(r) })
}

func (s *Store) StatementCount(ctx context.Context, id string) (int, error) {
	var n int
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM bank_statements WHERE bank_account_id = $1`, id).Scan(&n)
	return n, err
}

func (s *Store) StatementFileExists(ctx context.Context, id, sha string) (bool, error) {
	var ok bool
	err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM bank_statements WHERE bank_account_id = $1 AND sha256 = $2)`, id, sha).Scan(&ok)
	return ok, err
}

func (s *Store) ExistingFingerprints(ctx context.Context, id string, fps []string) (map[string]bool, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT fingerprint FROM bank_statement_lines WHERE bank_account_id = $1 AND fingerprint = ANY($2)`, id, fps)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, pgx.RowTo[string])
	out := map[string]bool{}
	for _, f := range list {
		out[f] = true
	}
	return out, err
}

func (s *Store) InsertStatement(ctx context.Context, st *domain.Statement) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO bank_statements (id, bank_account_id, filename, sha256, period_start, period_end, line_count, inserted_count, skipped_count, closing_balance, imported_by, imported_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, st.ID, st.BankAccountID, st.Filename, st.SHA256, st.PeriodStart, st.PeriodEnd, st.LineCount, st.InsertedCount, st.SkippedCount, st.ClosingBalance, st.ImportedBy, st.ImportedAt)
	if pe := pgErr(err); pe != nil && pe.Code == "23505" && pe.ConstraintName == "bank_statements_file_key" {
		return application.ErrFileImported
	}
	return mapErr(err)
}

func (s *Store) InsertStatementLines(ctx context.Context, st *domain.Statement, lines []domain.ParsedLine, newID func() string) (int, error) {
	n := 0
	for _, l := range lines {
		tag, err := s.q(ctx).Exec(ctx, `INSERT INTO bank_statement_lines (id, statement_id, bank_account_id, line_date, label, reference, amount, fingerprint) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (bank_account_id, fingerprint) DO NOTHING`, newID(), st.ID, st.BankAccountID, l.Date, l.Label, l.Reference, l.Amount, l.Fingerprint)
		if err != nil {
			return 0, mapErr(err)
		}
		n += int(tag.RowsAffected())
	}
	return n, nil
}

func (s *Store) ListStatements(ctx context.Context, id string) ([]*domain.Statement, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT id::text, bank_account_id::text, filename, sha256, period_start, period_end, line_count, inserted_count, skipped_count, closing_balance, imported_by::text, imported_at
		FROM bank_statements WHERE bank_account_id = $1 ORDER BY imported_at DESC`, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Statement, error) {
		var x domain.Statement
		err := r.Scan(&x.ID, &x.BankAccountID, &x.Filename, &x.SHA256, &x.PeriodStart, &x.PeriodEnd, &x.LineCount, &x.InsertedCount, &x.SkippedCount, &x.ClosingBalance, &x.ImportedBy, &x.ImportedAt)
		return &x, err
	})
}

const blCols = `l.id::text AS id, l.statement_id::text AS statement_id, l.bank_account_id::text AS bank_account_id, l.line_date, l.label, l.reference, l.amount,
	COALESCE((SELECT sum(r.amount) FROM reconciliations r WHERE r.statement_line_id = l.id), 0)::bigint AS matched`

func scanBL(r pgx.Row) (*domain.BankLine, error) {
	var l domain.BankLine
	if err := r.Scan(&l.ID, &l.StatementID, &l.BankAccountID, &l.LineDate, &l.Label, &l.Reference, &l.Amount, &l.Matched); err != nil {
		return nil, notFound(err)
	}
	l.Compute()
	return &l, nil
}

func (s *Store) loadMatches(ctx context.Context, ls []*domain.BankLine) error {
	if len(ls) == 0 {
		return nil
	}
	by := map[string]*domain.BankLine{}
	ids := make([]string, len(ls))
	for i, l := range ls {
		by[l.ID], ids[i] = l, l.ID
		l.Matches = []domain.Match{}
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT r.statement_line_id::text, r.id::text, r.entry_line_id::text, e.id::text, `+numExprE+`, r.amount, r.matched_at, r.matched_by::text
		FROM reconciliations r JOIN entry_lines el ON el.id = r.entry_line_id JOIN entries e ON e.id = el.entry_id JOIN journals j ON j.id = e.journal_id JOIN fiscal_years y ON y.id = e.fiscal_year_id
		WHERE r.statement_line_id = ANY($1::uuid[]) ORDER BY r.matched_at`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var lid string
		var m domain.Match
		if err := rows.Scan(&lid, &m.ID, &m.EntryLineID, &m.EntryID, &m.EntryNumber, &m.Amount, &m.MatchedAt, &m.MatchedBy); err != nil {
			return err
		}
		by[lid].Matches = append(by[lid].Matches, m)
	}
	return rows.Err()
}

func (s *Store) BankLineByID(ctx context.Context, id string) (*domain.BankLine, error) {
	l, err := scanBL(s.q(ctx).QueryRow(ctx, `SELECT `+blCols+` FROM bank_statement_lines l WHERE l.id = $1`, id))
	if err != nil {
		return nil, err
	}
	return l, s.loadMatches(ctx, []*domain.BankLine{l})
}

func (s *Store) ListBankLines(ctx context.Context, f application.BankLineFilter, limit, offset int) ([]*domain.BankLine, int, error) {
	where, args := ` WHERE x.bank_account_id = $1`, []any{f.BankAccountID}
	if f.Query != "" {
		args = append(args, like(f.Query))
		where += fmt.Sprintf(` AND (x.label ILIKE $%[1]d OR x.reference ILIKE $%[1]d)`, len(args))
	}
	if f.Status != nil {
		switch *f.Status {
		case "unmatched":
			where += ` AND x.matched = 0`
		case "partial":
			where += ` AND x.matched > 0 AND x.matched < abs(x.amount)`
		case "matched":
			where += ` AND x.matched = abs(x.amount)`
		}
	}
	if f.From != nil {
		args = append(args, *f.From)
		where += fmt.Sprintf(` AND x.line_date >= $%d`, len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where += fmt.Sprintf(` AND x.line_date <= $%d`, len(args))
	}
	base := `(SELECT ` + blCols + ` FROM bank_statement_lines l) x`
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM `+base+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT x.* FROM `+base+where+fmt.Sprintf(` ORDER BY x.line_date DESC, x.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.BankLine, error) { return scanBL(r) })
	if err != nil {
		return nil, 0, err
	}
	return items, total, s.loadMatches(ctx, items)
}

const llBase = `SELECT l.id::text AS line_id, e.id::text AS entry_id, ` + numExprE + ` AS num, e.accounting_date AS d, e.label AS label, e.reference AS reference, (l.debit - l.credit) AS amount,
	(greatest(l.debit, l.credit) - COALESCE((SELECT sum(r.amount) FROM reconciliations r WHERE r.entry_line_id = l.id), 0)) AS remaining
	FROM entry_lines l JOIN entries e ON e.id = l.entry_id AND e.status = 'posted' JOIN journals j ON j.id = e.journal_id JOIN fiscal_years y ON y.id = e.fiscal_year_id
	JOIN bank_accounts b ON b.account_id = l.account_id AND e.accounting_date >= b.opening_date WHERE b.id = $1`

func scanLC(r pgx.Row) (domain.LedgerCandidate, error) {
	var c domain.LedgerCandidate
	err := r.Scan(&c.EntryLineID, &c.EntryID, &c.EntryNumber, &c.AccountingDate, &c.Label, &c.Reference, &c.Amount, &c.Remaining)
	return c, err
}

func (s *Store) ListLedgerLines(ctx context.Context, bankAccountID string, onlyUnmatched bool, query string, limit, offset int) ([]domain.LedgerCandidate, int, error) {
	where, args := ` WHERE true`, []any{bankAccountID}
	if onlyUnmatched {
		where += ` AND x.remaining > 0`
	}
	if query != "" {
		args = append(args, like(query))
		where += fmt.Sprintf(` AND (x.label ILIKE $%[1]d OR x.reference ILIKE $%[1]d OR x.num ILIKE $%[1]d)`, len(args))
	}
	base := `(` + llBase + `) x`
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM `+base+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT x.line_id, x.entry_id, x.num, x.d, x.label, x.reference, x.amount, x.remaining FROM `+base+where+fmt.Sprintf(` ORDER BY x.d DESC, x.num DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.LedgerCandidate, error) { return scanLC(r) })
	return items, total, err
}

func (s *Store) LedgerCandidates(ctx context.Context, bankAccountID string, inflow bool, limit int) ([]domain.LedgerCandidate, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT x.line_id, x.entry_id, x.num, x.d, x.label, x.reference, x.amount, x.remaining FROM (`+llBase+`) x
		WHERE x.remaining > 0 AND (x.amount > 0) = $2 ORDER BY x.d DESC LIMIT $3`, bankAccountID, inflow, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.LedgerCandidate, error) { return scanLC(r) })
}

func (s *Store) LedgerLineRemaining(ctx context.Context, entryLineID string) (string, int64, error) {
	var acc string
	var rem int64
	err := s.q(ctx).QueryRow(ctx, `SELECT l.account_id::text, greatest(l.debit, l.credit) - COALESCE((SELECT sum(r.amount) FROM reconciliations r WHERE r.entry_line_id = l.id), 0) FROM entry_lines l WHERE l.id = $1`, entryLineID).Scan(&acc, &rem)
	return acc, rem, notFound(err)
}

func (s *Store) CreateReconciliation(ctx context.Context, id, bankAccountID, lineID, entryLineID string, amount int64, by string, at time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO reconciliations (id, bank_account_id, statement_line_id, entry_line_id, amount, matched_by, matched_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, bankAccountID, lineID, entryLineID, amount, by, at)
	return mapErr(err)
}

func (s *Store) ReconciliationLine(ctx context.Context, id string) (string, error) {
	var l string
	err := s.q(ctx).QueryRow(ctx, `SELECT statement_line_id::text FROM reconciliations WHERE id = $1`, id).Scan(&l)
	return l, notFound(err)
}

func (s *Store) DeleteReconciliation(ctx context.Context, id string) error {
	_, err := s.q(ctx).Exec(ctx, `DELETE FROM reconciliations WHERE id = $1`, id)
	return err
}

func (s *Store) ReconSummary(ctx context.Context, id string) (*domain.ReconSummary, error) {
	out := &domain.ReconSummary{BankAccountID: id}
	var openingBalance int64
	var openingDate time.Time
	var accID string
	if err := s.q(ctx).QueryRow(ctx, `SELECT currency, account_id::text, opening_balance, opening_date FROM bank_accounts WHERE id = $1`, id).Scan(&out.Currency, &accID, &openingBalance, &openingDate); err != nil {
		return nil, notFound(err)
	}
	var ledgerBefore, matchedStmt, matchedLedger, sumLines int64
	err := s.q(ctx).QueryRow(ctx, `SELECT
		COALESCE(sum(l.debit - l.credit), 0)::bigint,
		COALESCE(sum(l.debit - l.credit) FILTER (WHERE e.accounting_date < $2), 0)::bigint
		FROM entry_lines l JOIN entries e ON e.id = l.entry_id AND e.status = 'posted' WHERE l.account_id = $1`, accID, openingDate).Scan(&out.LedgerBalance, &ledgerBefore)
	if err != nil {
		return nil, err
	}
	err = s.q(ctx).QueryRow(ctx, `SELECT COALESCE(sum(x.amount), 0)::bigint, COALESCE(sum(x.matched), 0)::bigint,
		COALESCE(sum(CASE WHEN x.amount > 0 THEN 1 ELSE -1 END * (abs(x.amount) - x.matched)), 0)::bigint, count(*) FILTER (WHERE abs(x.amount) - x.matched > 0)::int,
		max(x.line_date)
		FROM (SELECT l.amount, l.line_date, COALESCE((SELECT sum(r.amount) FROM reconciliations r WHERE r.statement_line_id = l.id), 0) AS matched FROM bank_statement_lines l WHERE l.bank_account_id = $1) x`, id).
		Scan(&sumLines, &matchedStmt, &out.UnmatchedStatement, &out.UnmatchedStatementCount, &out.LastStatementDate)
	if err != nil {
		return nil, err
	}
	err = s.q(ctx).QueryRow(ctx, `SELECT COALESCE(sum(CASE WHEN x.amount > 0 THEN 1 ELSE -1 END * x.remaining) FILTER (WHERE x.remaining > 0), 0)::bigint, count(*) FILTER (WHERE x.remaining > 0)::int,
		COALESCE(sum(CASE WHEN x.amount > 0 THEN 1 ELSE -1 END * (abs(x.amount) - x.remaining)), 0)::bigint
		FROM (`+llBase+`) x`, id).Scan(&out.UnmatchedLedger, &out.UnmatchedLedgerCount, &matchedLedger)
	if err != nil {
		return nil, err
	}
	out.StatementBalance = openingBalance + sumLines
	out.Difference = out.LedgerBalance - out.StatementBalance
	out.OpeningGap = ledgerBefore - openingBalance
	// contrôle de cohérence : tout l'écart est expliqué par les éléments non rapprochés et par l'écart d'ouverture. Doit TOUJOURS valoir 0.
	out.Unexplained = out.Difference - (out.UnmatchedLedger - out.UnmatchedStatement) - out.OpeningGap
	_ = matchedStmt
	_ = matchedLedger
	return out, nil
}

func (s *Store) ForecastData(ctx context.Context, today time.Time) (*application.ForecastData, error) {
	d := &application.ForecastData{}
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*)::int, COALESCE(sum((SELECT sum(l.debit - l.credit) FROM entry_lines l JOIN entries e ON e.id = l.entry_id AND e.status = 'posted' AND e.accounting_date <= $1 WHERE l.account_id = b.account_id)), 0)::bigint
		FROM bank_accounts b WHERE b.active`, today).Scan(&d.BankAccounts, &d.OpeningBalance)
	if err != nil {
		return nil, err
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT x.kind, x.due_date, (x.amount_net + x.tax_amount - x.settled)::bigint FROM (SELECT `+opList+opFrom+`) x
		WHERE x.status = 'recorded' AND x.settlement_mode = 'on_account' AND x.amount_net + x.tax_amount - x.settled > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var f application.OpenFlow
		if err := rows.Scan(&kind, &f.Due, &f.Amount); err != nil {
			return nil, err
		}
		f.Inflow = kind == domain.KindIncome
		d.Open = append(d.Open, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	err = s.q(ctx).QueryRow(ctx, `SELECT count(*)::int, COALESCE(sum(CASE WHEN x.kind = 'income' THEN 1 ELSE -1 END * (x.amount_net + x.tax_amount)), 0)::bigint FROM (SELECT `+opList+opFrom+`) x WHERE x.status IN ('draft', 'entry_draft')`).
		Scan(&d.DraftCount, &d.DraftNetAmount)
	return d, err
}
