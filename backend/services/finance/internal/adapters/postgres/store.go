// Package postgres : adaptateur PostgreSQL du service finance. Les invariants comptables sont dans la base (triggers et contraintes).
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/pgxtx"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const MigrationsTable = "goose_finance"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// Migrate applique les migrations de l'outbox puis celles de finance.
func Migrate(ctx context.Context, url string) error {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := outbox.MigrateOutbox(ctx, db); err != nil {
		return err
	}
	return outbox.Migrate(ctx, db, Migrations(), MigrationsTable)
}

type Store struct{ Pool *pgxpool.Pool }

var _ application.Store = (*Store)(nil)

func New(ctx context.Context, url string) (*Store, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Store{Pool: p}, nil
}

func (s *Store) Close()                           { s.Pool.Close() }
func (s *Store) q(ctx context.Context) pgxtx.DBTX { return pgxtx.Querier(ctx, s.Pool) }
func (s *Store) Do(ctx context.Context, f func(context.Context) error) error {
	return pgxtx.Run(ctx, s.Pool, f)
}

func pgErr(err error) *pgconn.PgError {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe
	}
	return nil
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

// mapErr traduit les violations de contraintes et de triggers en erreurs métier.
func mapErr(err error) error {
	pe := pgErr(err)
	if pe == nil {
		return err
	}
	switch {
	case pe.Code == "23505" && (pe.ConstraintName == "accounts_code_key" || pe.ConstraintName == "journals_code_key"):
		return application.ErrCodeTaken
	case pe.Code == "23505" && pe.ConstraintName == "fiscal_years_label_key":
		return application.ErrLabelTaken
	case pe.Code == "23P01":
		return application.ErrOverlap
	case pe.Code == "23505" && pe.ConstraintName == "entries_source_key":
		return application.ErrDuplicateSource
	case pe.Code == "23505" && pe.ConstraintName == "entries_reverses_entry_id_key":
		return application.ErrAlreadyReversed
	case pe.Code == "23505" && pe.ConstraintName == "approvals_one_pending_key":
		return application.ErrPendingExists
	case pe.Code == "23505" && (pe.ConstraintName == "third_parties_name_key" || pe.ConstraintName == "operation_categories_label_key" || pe.ConstraintName == "bank_accounts_name_key"):
		return application.ErrNameTaken
	case pe.Code == "23505" && pe.ConstraintName == "third_parties_ref_key":
		return application.ErrCodeTaken
	case pe.Code == "23505" && pe.ConstraintName == "operations_recurring_key":
		return application.ErrRecurringTaken
	case pe.Code == "23505" && pe.ConstraintName == "attachments_dedupe_key":
		return application.ErrDuplicateFile
	case pe.Code == "23514" && pe.Message == "operation_frozen":
		return application.ErrFrozenOperation
	case pe.Code == "23514" && pe.Message == "recon_over_statement", pe.Code == "23514" && pe.Message == "recon_over_entry":
		return application.ErrReconOver
	case pe.Code == "23514" && pe.Message == "recon_wrong_direction":
		return application.ErrReconDirection
	case pe.Code == "23514" && (pe.Message == "recon_wrong_account" || pe.Message == "recon_wrong_bank_account" || pe.Message == "recon_before_opening"):
		return application.ErrReconAccount
	case pe.Code == "23514" && pe.Message == "recon_entry_not_posted":
		return application.ErrReconNotPosted
	case pe.Code == "23514" && pe.Message == "statement_immutable":
		return application.ErrImmutable
	case pe.Code == "23514" && pe.ConstraintName == "approvals_two_people":
		return application.ErrFourEyes
	case pe.Code == "23514":
		switch pe.Message {
		case "entry_immutable":
			return application.ErrImmutable
		case "period_closed":
			return application.ErrPeriodClosed
		case "period_has_drafts":
			return application.ErrHasDrafts
		case "reopen_not_approved":
			return application.ErrNotApproved
		case "entry_unbalanced":
			return application.ErrUnbalanced
		case "account_inactive":
			return application.ErrAccountInactive
		case "journal_inactive":
			return application.ErrJournalInactive
		case "four_eyes_required":
			return application.ErrFourEyes
		}
	case pe.Code == "23503":
		return application.ErrInUse
	}
	return err
}

func like(q string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
}

// ── réglages ────────────────────────────────────────────────────────────────

func (s *Store) GetSettings(ctx context.Context) (*domain.Settings, error) {
	var st domain.Settings
	err := s.q(ctx).QueryRow(ctx, `SELECT base_currency, four_eyes_min_amount, updated_at FROM finance_settings`).Scan(&st.BaseCurrency, &st.FourEyesMinAmount, &st.UpdatedAt)
	return &st, err
}

func (s *Store) PutSettings(ctx context.Context, st *domain.Settings, by string) error {
	var u any
	if by != "" {
		u = by
	}
	_, err := s.q(ctx).Exec(ctx, `UPDATE finance_settings SET base_currency=$1, four_eyes_min_amount=$2, updated_at=now(), updated_by=$3`, st.BaseCurrency, st.FourEyesMinAmount, u)
	return mapErr(err)
}

func (s *Store) AnyEntryExists(ctx context.Context) (bool, error) {
	var ok bool
	err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM entries)`).Scan(&ok)
	return ok, err
}

// ── comptes ─────────────────────────────────────────────────────────────────

const accCols = `id::text, code, label, kind, active, created_at`

func scanAccount(r pgx.Row) (*domain.Account, error) {
	var a domain.Account
	if err := r.Scan(&a.ID, &a.Code, &a.Label, &a.Kind, &a.Active, &a.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &a, nil
}

func (s *Store) CreateAccount(ctx context.Context, a *domain.Account) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO accounts (id, code, label, kind, active) VALUES ($1,$2,$3,$4,$5)`, a.ID, a.Code, a.Label, a.Kind, a.Active)
	return mapErr(err)
}

func (s *Store) UpdateAccount(ctx context.Context, a *domain.Account) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE accounts SET label=$2, kind=$3, active=$4, updated_at=now() WHERE id=$1`, a.ID, a.Label, a.Kind, a.Active)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) AccountByID(ctx context.Context, id string) (*domain.Account, error) {
	return scanAccount(s.q(ctx).QueryRow(ctx, `SELECT `+accCols+` FROM accounts WHERE id = $1`, id))
}

func (s *Store) AccountsByIDs(ctx context.Context, ids []string) (map[string]*domain.Account, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT `+accCols+` FROM accounts WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Account, error) { return scanAccount(r) })
	out := make(map[string]*domain.Account, len(items))
	for _, a := range items {
		out[a.ID] = a
	}
	return out, err
}

func (s *Store) ListAccounts(ctx context.Context, f application.AccountFilter, limit, offset int) ([]*domain.Account, int, error) {
	where, args := ` WHERE true`, []any{}
	if f.Query != "" {
		args = append(args, like(f.Query))
		where += fmt.Sprintf(` AND (code ILIKE $%[1]d OR label ILIKE $%[1]d)`, len(args))
	}
	if f.Kind != nil {
		args = append(args, *f.Kind)
		where += fmt.Sprintf(` AND kind = $%d`, len(args))
	}
	if f.Active != nil {
		args = append(args, *f.Active)
		where += fmt.Sprintf(` AND active = $%d`, len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM accounts`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+accCols+` FROM accounts`+where+fmt.Sprintf(` ORDER BY code LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Account, error) { return scanAccount(r) })
	return items, total, err
}

func (s *Store) ExistingAccountCodes(ctx context.Context, codes []string) (map[string]bool, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT lower(code) FROM accounts WHERE lower(code) = ANY($1)`, codes)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, pgx.RowTo[string])
	out := map[string]bool{}
	for _, c := range list {
		out[c] = true
	}
	return out, err
}

func (s *Store) AccountUsed(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM entry_lines WHERE account_id = $1)`, id).Scan(&ok)
	return ok, err
}

// ── journaux ────────────────────────────────────────────────────────────────

const jCols = `id::text, code, label, kind, active`

func scanJournal(r pgx.Row) (*domain.Journal, error) {
	var j domain.Journal
	if err := r.Scan(&j.ID, &j.Code, &j.Label, &j.Kind, &j.Active); err != nil {
		return nil, notFound(err)
	}
	return &j, nil
}

func (s *Store) CreateJournal(ctx context.Context, j *domain.Journal) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO journals (id, code, label, kind, active) VALUES ($1,$2,$3,$4,$5)`, j.ID, j.Code, j.Label, j.Kind, j.Active)
	return mapErr(err)
}

func (s *Store) UpdateJournal(ctx context.Context, j *domain.Journal) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE journals SET label=$2, kind=$3, active=$4, updated_at=now() WHERE id=$1`, j.ID, j.Label, j.Kind, j.Active)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) JournalByID(ctx context.Context, id string) (*domain.Journal, error) {
	return scanJournal(s.q(ctx).QueryRow(ctx, `SELECT `+jCols+` FROM journals WHERE id = $1`, id))
}

func (s *Store) ListJournals(ctx context.Context, active *bool) ([]*domain.Journal, error) {
	q, args := `SELECT `+jCols+` FROM journals`, []any{}
	if active != nil {
		q += ` WHERE active = $1`
		args = append(args, *active)
	}
	rows, err := s.q(ctx).Query(ctx, q+` ORDER BY code`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Journal, error) { return scanJournal(r) })
}

func (s *Store) ExistingJournalCodes(ctx context.Context, codes []string) (map[string]bool, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT code FROM journals WHERE code = ANY($1)`, codes)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, pgx.RowTo[string])
	out := map[string]bool{}
	for _, c := range list {
		out[c] = true
	}
	return out, err
}

func (s *Store) JournalUsed(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM entries WHERE journal_id = $1)`, id).Scan(&ok)
	return ok, err
}

// ── exercices et périodes ───────────────────────────────────────────────────

func (s *Store) CreateFiscalYear(ctx context.Context, y *domain.FiscalYear) error {
	if _, err := s.q(ctx).Exec(ctx, `INSERT INTO fiscal_years (id, label, start_date, end_date) VALUES ($1,$2,$3,$4)`, y.ID, y.Label, y.Start, y.End); err != nil {
		return mapErr(err)
	}
	for _, p := range y.Periods {
		if _, err := s.q(ctx).Exec(ctx, `INSERT INTO fiscal_periods (id, fiscal_year_id, number, start_date, end_date) VALUES ($1,$2,$3,$4,$5)`, p.ID, y.ID, p.Number, p.Start, p.End); err != nil {
			return mapErr(err)
		}
	}
	return nil
}

const perCols = `id::text, fiscal_year_id::text, number, start_date, end_date, status, closed_at, closed_by::text`

func scanPeriod(r pgx.Row) (*domain.FiscalPeriod, error) {
	var p domain.FiscalPeriod
	if err := r.Scan(&p.ID, &p.FiscalYearID, &p.Number, &p.Start, &p.End, &p.Status, &p.ClosedAt, &p.ClosedBy); err != nil {
		return nil, notFound(err)
	}
	return &p, nil
}

func (s *Store) loadPeriods(ctx context.Context, ys []*domain.FiscalYear) error {
	if len(ys) == 0 {
		return nil
	}
	by := map[string]*domain.FiscalYear{}
	ids := make([]string, len(ys))
	for i, y := range ys {
		by[y.ID], ids[i] = y, y.ID
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT `+perCols+` FROM fiscal_periods WHERE fiscal_year_id = ANY($1::uuid[]) ORDER BY fiscal_year_id, number`, ids)
	if err != nil {
		return err
	}
	ps, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.FiscalPeriod, error) { return scanPeriod(r) })
	if err != nil {
		return err
	}
	for _, p := range ps {
		by[p.FiscalYearID].Periods = append(by[p.FiscalYearID].Periods, *p)
	}
	return nil
}

func (s *Store) FiscalYearByID(ctx context.Context, id string) (*domain.FiscalYear, error) {
	var y domain.FiscalYear
	if err := s.q(ctx).QueryRow(ctx, `SELECT id::text, label, start_date, end_date FROM fiscal_years WHERE id = $1`, id).Scan(&y.ID, &y.Label, &y.Start, &y.End); err != nil {
		return nil, notFound(err)
	}
	return &y, s.loadPeriods(ctx, []*domain.FiscalYear{&y})
}

func (s *Store) ListFiscalYears(ctx context.Context) ([]*domain.FiscalYear, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT id::text, label, start_date, end_date FROM fiscal_years ORDER BY start_date DESC`)
	if err != nil {
		return nil, err
	}
	ys, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.FiscalYear, error) {
		var y domain.FiscalYear
		err := r.Scan(&y.ID, &y.Label, &y.Start, &y.End)
		return &y, err
	})
	if err != nil {
		return nil, err
	}
	return ys, s.loadPeriods(ctx, ys)
}

func (s *Store) PeriodByID(ctx context.Context, id string, lock bool) (*domain.FiscalPeriod, error) {
	q := `SELECT ` + perCols + ` FROM fiscal_periods WHERE id = $1`
	if lock {
		q += ` FOR UPDATE`
	}
	return scanPeriod(s.q(ctx).QueryRow(ctx, q, id))
}

func (s *Store) PeriodForDate(ctx context.Context, d time.Time) (*domain.FiscalPeriod, error) {
	p, err := scanPeriod(s.q(ctx).QueryRow(ctx, `SELECT `+perCols+` FROM fiscal_periods WHERE start_date <= $1 AND end_date >= $1`, d))
	if errors.Is(err, application.ErrNotFound) {
		return nil, application.ErrNoPeriod
	}
	return p, err
}

func (s *Store) ClosePeriod(ctx context.Context, id, by string, at time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE fiscal_periods SET status='closed', closed_at=$2, closed_by=$3 WHERE id=$1`, id, at, by)
	return mapErr(err)
}

func (s *Store) ReopenPeriod(ctx context.Context, id string) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE fiscal_periods SET status='open', closed_at=NULL, closed_by=NULL WHERE id=$1`, id)
	return mapErr(err)
}

func (s *Store) CountDrafts(ctx context.Context, periodID string) (int, error) {
	var n int
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM entries WHERE fiscal_period_id = $1 AND status = 'draft'`, periodID).Scan(&n)
	return n, err
}

// ── approbations ────────────────────────────────────────────────────────────

const apCols = `a.id::text, a.type, a.subject_id::text, coalesce(to_char(p.start_date,'YYYY-MM-DD') || ' → ' || to_char(p.end_date,'YYYY-MM-DD'), ''), a.reason, a.status, a.requested_by::text, a.requested_at, a.decided_by::text, a.decided_at, a.decision_note`
const apFrom = ` FROM approval_requests a LEFT JOIN fiscal_periods p ON a.type = 'reopen_period' AND p.id = a.subject_id`

func scanApproval(r pgx.Row) (*domain.Approval, error) {
	var a domain.Approval
	if err := r.Scan(&a.ID, &a.Type, &a.SubjectID, &a.SubjectLabel, &a.Reason, &a.Status, &a.RequestedBy, &a.RequestedAt, &a.DecidedBy, &a.DecidedAt, &a.DecisionNote); err != nil {
		return nil, notFound(err)
	}
	return &a, nil
}

func (s *Store) CreateApproval(ctx context.Context, a *domain.Approval) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO approval_requests (id, type, subject_id, reason, status, requested_by, requested_at) VALUES ($1,$2,$3,$4,'pending',$5,$6)`,
		a.ID, a.Type, a.SubjectID, a.Reason, a.RequestedBy, a.RequestedAt)
	return mapErr(err)
}

func (s *Store) ApprovalByID(ctx context.Context, id string, lock bool) (*domain.Approval, error) {
	q := `SELECT ` + apCols + apFrom + ` WHERE a.id = $1`
	if lock {
		q += ` FOR UPDATE OF a`
	}
	return scanApproval(s.q(ctx).QueryRow(ctx, q, id))
}

func (s *Store) ListApprovals(ctx context.Context, status *string, limit, offset int) ([]*domain.Approval, int, error) {
	where, args := ``, []any{}
	if status != nil {
		where, args = ` WHERE a.status = $1`, []any{*status}
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*)`+apFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+apCols+apFrom+where+fmt.Sprintf(` ORDER BY (a.status = 'pending') DESC, a.requested_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Approval, error) { return scanApproval(r) })
	return items, total, err
}

func (s *Store) DecideApproval(ctx context.Context, id, status, by string, note *string, at time.Time) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE approval_requests SET status=$2, decided_by=$3, decided_at=$4, decision_note=$5 WHERE id=$1 AND status='pending'`, id, status, by, at, note)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

// ── écritures ───────────────────────────────────────────────────────────────

const entCols = `e.id::text, e.journal_id::text, j.code, e.fiscal_period_id::text, e.fiscal_year_id::text, y.label, e.accounting_date, e.status, e.entry_number, e.label, e.reference, e.currency, e.origin,
	e.source_type, e.source_id::text, e.source_kind, e.reverses_entry_id::text, (SELECT r.id::text FROM entries r WHERE r.reverses_entry_id = e.id), e.reversal_reason,
	e.total_amount, e.created_by::text, e.created_at, e.posted_by::text, e.posted_at`
const entFrom = ` FROM entries e JOIN journals j ON j.id = e.journal_id JOIN fiscal_years y ON y.id = e.fiscal_year_id`

func scanEntry(r pgx.Row) (*domain.Entry, error) {
	var e domain.Entry
	var st, sid, sk *string
	var total int64
	if err := r.Scan(&e.ID, &e.JournalID, &e.JournalCode, &e.FiscalPeriodID, &e.FiscalYearID, &e.FiscalYearLabel, &e.AccountingDate, &e.Status, &e.EntryNumber, &e.Label, &e.Reference, &e.Currency, &e.Origin,
		&st, &sid, &sk, &e.ReversesEntryID, &e.ReversedByEntryID, &e.ReversalReason, &total, &e.CreatedBy, &e.CreatedAt, &e.PostedBy, &e.PostedAt); err != nil {
		return nil, notFound(err)
	}
	if st != nil {
		e.Source = &domain.EntrySource{Type: *st, ID: *sid, Kind: *sk}
	}
	return &e, nil
}

func (s *Store) loadLines(ctx context.Context, es []*domain.Entry) error {
	if len(es) == 0 {
		return nil
	}
	by := map[string]*domain.Entry{}
	ids := make([]string, len(es))
	for i, e := range es {
		by[e.ID], ids[i] = e, e.ID
		e.Lines, e.TotalDebit, e.TotalCredit = []domain.EntryLine{}, 0, 0
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT l.entry_id::text, l.id::text, l.account_id::text, a.code, a.label, l.debit, l.credit, l.memo
		FROM entry_lines l JOIN accounts a ON a.id = l.account_id WHERE l.entry_id = ANY($1::uuid[]) ORDER BY l.entry_id, l.line_no`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var eid string
		var l domain.EntryLine
		if err := rows.Scan(&eid, &l.ID, &l.AccountID, &l.AccountCode, &l.AccountLabel, &l.Debit, &l.Credit, &l.Memo); err != nil {
			return err
		}
		e := by[eid]
		e.Lines = append(e.Lines, l)
		e.TotalDebit += l.Debit
		e.TotalCredit += l.Credit
	}
	return rows.Err()
}

func (s *Store) writeLines(ctx context.Context, e *domain.Entry) error {
	for i, l := range e.Lines {
		if _, err := s.q(ctx).Exec(ctx, `INSERT INTO entry_lines (id, entry_id, line_no, account_id, debit, credit, memo) VALUES ($1,$2,$3,$4,$5,$6,$7)`, l.ID, e.ID, i+1, l.AccountID, l.Debit, l.Credit, l.Memo); err != nil {
			return mapErr(err)
		}
	}
	return nil
}

func (s *Store) CreateEntry(ctx context.Context, e *domain.Entry) error {
	var st, sid, sk any
	if e.Source != nil {
		st, sid, sk = e.Source.Type, e.Source.ID, e.Source.Kind
	}
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO entries (id, journal_id, fiscal_year_id, fiscal_period_id, accounting_date, status, label, reference, currency, origin, source_type, source_id, source_kind,
		reverses_entry_id, reversal_reason, created_by, created_at) VALUES ($1,$2,$3,$4,$5,'draft',$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		e.ID, e.JournalID, e.FiscalYearID, e.FiscalPeriodID, e.AccountingDate, e.Label, e.Reference, e.Currency, e.Origin, st, sid, sk, e.ReversesEntryID, e.ReversalReason, e.CreatedBy, e.CreatedAt)
	if err != nil {
		return mapErr(err)
	}
	return s.writeLines(ctx, e)
}

func (s *Store) ReplaceEntry(ctx context.Context, e *domain.Entry) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE entries SET journal_id=$2, fiscal_year_id=$3, fiscal_period_id=$4, accounting_date=$5, label=$6, reference=$7 WHERE id=$1`,
		e.ID, e.JournalID, e.FiscalYearID, e.FiscalPeriodID, e.AccountingDate, e.Label, e.Reference)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	if _, err := s.q(ctx).Exec(ctx, `DELETE FROM entry_lines WHERE entry_id = $1`, e.ID); err != nil {
		return mapErr(err)
	}
	return s.writeLines(ctx, e)
}

func (s *Store) DeleteEntry(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM entries WHERE id = $1`, id)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) EntryByID(ctx context.Context, id string, lock bool) (*domain.Entry, error) {
	q := `SELECT ` + entCols + entFrom + ` WHERE e.id = $1`
	if lock {
		q += ` FOR UPDATE OF e`
	}
	e, err := scanEntry(s.q(ctx).QueryRow(ctx, q, id))
	if err != nil {
		return nil, err
	}
	return e, s.loadLines(ctx, []*domain.Entry{e})
}

func (s *Store) ListEntries(ctx context.Context, f application.EntryFilter, limit, offset int) ([]*domain.Entry, int, error) {
	where, args := ` WHERE true`, []any{}
	if f.Query != "" {
		args = append(args, like(f.Query))
		where += fmt.Sprintf(` AND (e.label ILIKE $%[1]d OR e.reference ILIKE $%[1]d OR (j.code || '-' || y.label || '-' || lpad(e.entry_number::text, 6, '0')) ILIKE $%[1]d)`, len(args))
	}
	if f.Status != nil {
		args = append(args, *f.Status)
		where += fmt.Sprintf(` AND e.status = $%d`, len(args))
	}
	if f.JournalID != nil {
		args = append(args, *f.JournalID)
		where += fmt.Sprintf(` AND e.journal_id = $%d`, len(args))
	}
	if f.AccountID != nil {
		args = append(args, *f.AccountID)
		where += fmt.Sprintf(` AND EXISTS (SELECT 1 FROM entry_lines l WHERE l.entry_id = e.id AND l.account_id = $%d)`, len(args))
	}
	if f.From != nil {
		args = append(args, *f.From)
		where += fmt.Sprintf(` AND e.accounting_date >= $%d`, len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where += fmt.Sprintf(` AND e.accounting_date <= $%d`, len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*)`+entFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+entCols+entFrom+where+fmt.Sprintf(` ORDER BY e.accounting_date DESC, e.created_at DESC, e.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Entry, error) { return scanEntry(r) })
	if err != nil {
		return nil, 0, err
	}
	return items, total, s.loadLines(ctx, items)
}

// PostEntry : le compteur est verrouillé dans la transaction de l'appelant. Un échec annule le tout, y compris l'incrément : aucun trou.
func (s *Store) PostEntry(ctx context.Context, e *domain.Entry, by string, at time.Time) (int64, error) {
	q := s.q(ctx)
	if _, err := q.Exec(ctx, `INSERT INTO entry_counters (journal_id, fiscal_year_id, last_number) VALUES ($1,$2,0) ON CONFLICT DO NOTHING`, e.JournalID, e.FiscalYearID); err != nil {
		return 0, err
	}
	var n int64
	if err := q.QueryRow(ctx, `UPDATE entry_counters SET last_number = last_number + 1 WHERE journal_id = $1 AND fiscal_year_id = $2 RETURNING last_number`, e.JournalID, e.FiscalYearID).Scan(&n); err != nil {
		return 0, err
	}
	tag, err := q.Exec(ctx, `UPDATE entries SET status='posted', entry_number=$2, posted_by=$3, posted_at=$4 WHERE id=$1 AND status='draft'`, e.ID, n, by, at)
	if err != nil {
		return 0, mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return 0, application.ErrNotFound
	}
	return n, nil
}

// ── états ───────────────────────────────────────────────────────────────────

const numExpr = `(j.code || '-' || y.label || '-' || lpad(e.entry_number::text, 6, '0'))`

func (s *Store) GeneralLedger(ctx context.Context, accountID string, from, to *time.Time, limit, offset int) (*application.Ledger, error) {
	acc, err := s.AccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := &application.Ledger{Account: acc}
	q := s.q(ctx)
	if from != nil {
		if err := q.QueryRow(ctx, `SELECT coalesce(sum(l.debit - l.credit), 0) FROM entry_lines l JOIN entries e ON e.id = l.entry_id
			WHERE l.account_id = $1 AND e.status = 'posted' AND e.accounting_date < $2`, accountID, *from).Scan(&out.Opening); err != nil {
			return nil, err
		}
	}
	var sum int64
	if err := q.QueryRow(ctx, `SELECT count(*), coalesce(sum(l.debit - l.credit), 0) FROM entry_lines l JOIN entries e ON e.id = l.entry_id
		WHERE l.account_id = $1 AND e.status = 'posted' AND ($2::date IS NULL OR e.accounting_date >= $2) AND ($3::date IS NULL OR e.accounting_date <= $3)`, accountID, from, to).Scan(&out.Total, &sum); err != nil {
		return nil, err
	}
	out.Closing = out.Opening + sum
	rows, err := q.Query(ctx, `WITH x AS (
		SELECT e.id::text AS eid, `+numExpr+` AS num, e.accounting_date AS d, j.code AS jc, e.label, e.reference, l.memo, l.debit, l.credit,
		       $4::bigint + sum(l.debit - l.credit) OVER (ORDER BY e.accounting_date, j.code, e.entry_number, l.line_no) AS bal, e.entry_number, l.line_no
		FROM entry_lines l JOIN entries e ON e.id = l.entry_id JOIN journals j ON j.id = e.journal_id JOIN fiscal_years y ON y.id = e.fiscal_year_id
		WHERE l.account_id = $1 AND e.status = 'posted' AND ($2::date IS NULL OR e.accounting_date >= $2) AND ($3::date IS NULL OR e.accounting_date <= $3))
		SELECT eid, num, d, jc, label, reference, memo, debit, credit, bal FROM x ORDER BY d, jc, entry_number, line_no LIMIT $5 OFFSET $6`, accountID, from, to, out.Opening, limit, offset)
	if err != nil {
		return nil, err
	}
	out.Lines, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (application.LedgerLine, error) {
		var l application.LedgerLine
		err := r.Scan(&l.EntryID, &l.EntryNumber, &l.AccountingDate, &l.JournalCode, &l.Label, &l.Reference, &l.Memo, &l.Debit, &l.Credit, &l.Balance)
		return l, err
	})
	return out, err
}

func (s *Store) TrialBalance(ctx context.Context, from, to time.Time) ([]application.TrialRow, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT a.id::text, a.code, a.label, a.kind,
		coalesce(sum(l.debit - l.credit) FILTER (WHERE e.accounting_date < $1), 0),
		coalesce(sum(l.debit) FILTER (WHERE e.accounting_date >= $1), 0),
		coalesce(sum(l.credit) FILTER (WHERE e.accounting_date >= $1), 0)
		FROM accounts a JOIN entry_lines l ON l.account_id = a.id JOIN entries e ON e.id = l.entry_id AND e.status = 'posted' AND e.accounting_date <= $2
		GROUP BY a.id, a.code, a.label, a.kind ORDER BY a.code`, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (application.TrialRow, error) {
		var t application.TrialRow
		err := r.Scan(&t.AccountID, &t.Code, &t.Label, &t.Kind, &t.Opening, &t.Debit, &t.Credit)
		t.Closing = t.Opening + t.Debit - t.Credit
		return t, err
	})
}
