package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
)

// ── tiers ───────────────────────────────────────────────────────────────────

const tpCols = `id::text, kind, external_ref, display_name, active`

func scanTP(r pgx.Row) (*domain.ThirdParty, error) {
	var t domain.ThirdParty
	if err := r.Scan(&t.ID, &t.Kind, &t.ExternalRef, &t.DisplayName, &t.Active); err != nil {
		return nil, notFound(err)
	}
	return &t, nil
}

func (s *Store) CreateThirdParty(ctx context.Context, t *domain.ThirdParty) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO third_parties (id, kind, external_ref, display_name, active) VALUES ($1,$2,$3,$4,$5)`, t.ID, t.Kind, t.ExternalRef, t.DisplayName, t.Active)
	return mapErr(err)
}

func (s *Store) UpdateThirdParty(ctx context.Context, t *domain.ThirdParty) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE third_parties SET external_ref=$2, display_name=$3, active=$4, updated_at=now() WHERE id=$1`, t.ID, t.ExternalRef, t.DisplayName, t.Active)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) ThirdPartyByID(ctx context.Context, id string) (*domain.ThirdParty, error) {
	return scanTP(s.q(ctx).QueryRow(ctx, `SELECT `+tpCols+` FROM third_parties WHERE id = $1`, id))
}

func (s *Store) ListThirdParties(ctx context.Context, query string, kind *string, limit, offset int) ([]*domain.ThirdParty, int, error) {
	where, args := ` WHERE true`, []any{}
	if query != "" {
		args = append(args, like(query))
		where += fmt.Sprintf(` AND (display_name ILIKE $%[1]d OR external_ref ILIKE $%[1]d)`, len(args))
	}
	if kind != nil {
		args = append(args, *kind)
		where += fmt.Sprintf(` AND kind = $%d`, len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM third_parties`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+tpCols+` FROM third_parties`+where+fmt.Sprintf(` ORDER BY lower(display_name) LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.ThirdParty, error) { return scanTP(r) })
	return items, total, err
}

// ── catégories ──────────────────────────────────────────────────────────────

const catCols = `c.id::text, c.kind, c.label, c.account_id::text, a.code, a.label, c.nature, c.active`

func scanCat(r pgx.Row) (*domain.OpCategory, error) {
	var c domain.OpCategory
	if err := r.Scan(&c.ID, &c.Kind, &c.Label, &c.AccountID, &c.AccountCode, &c.AccountLabel, &c.Nature, &c.Active); err != nil {
		return nil, notFound(err)
	}
	return &c, nil
}

func (s *Store) CreateOpCategory(ctx context.Context, c *domain.OpCategory) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO operation_categories (id, kind, label, account_id, nature, active) VALUES ($1,$2,$3,$4,$5,$6)`, c.ID, c.Kind, c.Label, c.AccountID, c.Nature, c.Active)
	return mapErr(err)
}

func (s *Store) UpdateOpCategory(ctx context.Context, c *domain.OpCategory) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE operation_categories SET label=$2, account_id=$3, nature=$4, active=$5, updated_at=now() WHERE id=$1`, c.ID, c.Label, c.AccountID, c.Nature, c.Active)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) OpCategoryByID(ctx context.Context, id string) (*domain.OpCategory, error) {
	return scanCat(s.q(ctx).QueryRow(ctx, `SELECT `+catCols+` FROM operation_categories c JOIN accounts a ON a.id = c.account_id WHERE c.id = $1`, id))
}

func (s *Store) ListOpCategories(ctx context.Context, kind *string, active *bool) ([]*domain.OpCategory, error) {
	where, args := ` WHERE true`, []any{}
	if kind != nil {
		args = append(args, *kind)
		where += fmt.Sprintf(` AND c.kind = $%d`, len(args))
	}
	if active != nil {
		args = append(args, *active)
		where += fmt.Sprintf(` AND c.active = $%d`, len(args))
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT `+catCols+` FROM operation_categories c JOIN accounts a ON a.id = c.account_id`+where+` ORDER BY c.kind, lower(c.label)`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.OpCategory, error) { return scanCat(r) })
}

// ── opérations ──────────────────────────────────────────────────────────────

const opList = `o.id::text AS id, o.kind, o.operation_date, o.due_date, o.third_party_id::text AS third_party_id, tp.display_name AS third_party_name, o.category_id::text AS category_id, c.label AS category_label,
	c.account_id::text AS category_account_id, o.label, o.reference, o.amount_net, o.tax_amount, o.tax_account_id::text AS tax_account_id, o.counterparty_account_id::text AS counterparty_account_id, ca.code AS counterparty_code,
	o.settlement_mode, o.nature, o.currency,
	CASE WHEN o.entry_id IS NULL THEN 'draft' WHEN e.status = 'draft' THEN 'entry_draft'
	     WHEN EXISTS (SELECT 1 FROM entries r WHERE r.reverses_entry_id = e.id AND r.status = 'posted') THEN 'reversed' ELSE 'recorded' END AS status,
	o.entry_id::text AS entry_id, CASE WHEN e.status = 'posted' THEN j.code || '-' || y.label || '-' || lpad(e.entry_number::text, 6, '0') END AS entry_number,
	COALESCE((SELECT sum(p.amount) FROM operation_payments p JOIN entries pe ON pe.id = p.entry_id WHERE p.operation_id = o.id
	          AND NOT EXISTS (SELECT 1 FROM entries r WHERE r.reverses_entry_id = pe.id AND r.status = 'posted')), 0)::bigint AS settled,
	o.duplicate_confirmed, o.recurring_template_id::text AS recurring_template_id, (SELECT count(*) FROM attachments at WHERE at.operation_id = o.id)::int AS attachment_count, o.created_by::text AS created_by, o.created_at`
const opFrom = ` FROM operations o JOIN operation_categories c ON c.id = o.category_id JOIN accounts ca ON ca.id = o.counterparty_account_id
	LEFT JOIN third_parties tp ON tp.id = o.third_party_id LEFT JOIN entries e ON e.id = o.entry_id LEFT JOIN journals j ON j.id = e.journal_id LEFT JOIN fiscal_years y ON y.id = e.fiscal_year_id`

func scanOp(r pgx.Row) (*domain.Operation, error) {
	var o domain.Operation
	if err := r.Scan(&o.ID, &o.Kind, &o.OperationDate, &o.DueDate, &o.ThirdPartyID, &o.ThirdPartyName, &o.CategoryID, &o.CategoryLabel, &o.CategoryAccountID, &o.Label, &o.Reference, &o.AmountNet, &o.TaxAmount,
		&o.TaxAccountID, &o.CounterpartyAccountID, &o.CounterpartyAccountCode, &o.SettlementMode, &o.Nature, &o.Currency, &o.Status, &o.EntryID, &o.EntryNumber, &o.SettledAmount,
		&o.DuplicateConfirmed, &o.RecurringTemplateID, &o.AttachmentCount, &o.CreatedBy, &o.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &o, nil
}

func (s *Store) loadPayments(ctx context.Context, ops []*domain.Operation) error {
	if len(ops) == 0 {
		return nil
	}
	by := map[string]*domain.Operation{}
	ids := make([]string, len(ops))
	for i, o := range ops {
		by[o.ID], ids[i] = o, o.ID
		o.Payments = []domain.OperationPayment{}
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT p.operation_id::text, p.id::text, p.payment_date, p.amount, p.account_id::text, a.code, p.entry_id::text, e.status,
		CASE WHEN e.status = 'posted' THEN j.code || '-' || y.label || '-' || lpad(e.entry_number::text, 6, '0') END
		FROM operation_payments p JOIN accounts a ON a.id = p.account_id JOIN entries e ON e.id = p.entry_id JOIN journals j ON j.id = e.journal_id JOIN fiscal_years y ON y.id = e.fiscal_year_id
		WHERE p.operation_id = ANY($1::uuid[]) ORDER BY p.payment_date, p.created_at`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var oid string
		var p domain.OperationPayment
		if err := rows.Scan(&oid, &p.ID, &p.PaymentDate, &p.Amount, &p.AccountID, &p.AccountCode, &p.EntryID, &p.EntryStatus, &p.EntryNumber); err != nil {
			return err
		}
		by[oid].Payments = append(by[oid].Payments, p)
	}
	return rows.Err()
}

func (s *Store) CreateOperation(ctx context.Context, o *domain.Operation) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO operations (id, kind, operation_date, due_date, third_party_id, category_id, label, reference, amount_net, tax_amount, tax_account_id, counterparty_account_id,
		settlement_mode, nature, currency, duplicate_confirmed, recurring_template_id, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		o.ID, o.Kind, o.OperationDate, o.DueDate, o.ThirdPartyID, o.CategoryID, o.Label, o.Reference, o.AmountNet, o.TaxAmount, o.TaxAccountID, o.CounterpartyAccountID, o.SettlementMode, o.Nature, o.Currency,
		o.DuplicateConfirmed, o.RecurringTemplateID, o.CreatedBy, o.CreatedAt)
	return mapErr(err)
}

func (s *Store) CreateOperationIfAbsent(ctx context.Context, o *domain.Operation, templateID string, month time.Time) (bool, error) {
	tag, err := s.q(ctx).Exec(ctx, `INSERT INTO operations (id, kind, operation_date, third_party_id, category_id, label, amount_net, tax_amount, tax_account_id, counterparty_account_id,
		settlement_mode, nature, currency, recurring_template_id, recurring_month, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (recurring_template_id, recurring_month) WHERE recurring_template_id IS NOT NULL DO NOTHING`,
		o.ID, o.Kind, o.OperationDate, o.ThirdPartyID, o.CategoryID, o.Label, o.AmountNet, o.TaxAmount, o.TaxAccountID, o.CounterpartyAccountID, o.SettlementMode, o.Nature, o.Currency, templateID, month, o.CreatedBy, o.CreatedAt)
	if err != nil {
		return false, mapErr(err)
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) UpdateOperation(ctx context.Context, o *domain.Operation) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE operations SET operation_date=$2, due_date=$3, third_party_id=$4, category_id=$5, label=$6, reference=$7, amount_net=$8, tax_amount=$9, tax_account_id=$10,
		counterparty_account_id=$11, settlement_mode=$12, nature=$13, currency=$14, duplicate_confirmed=$15 WHERE id=$1`,
		o.ID, o.OperationDate, o.DueDate, o.ThirdPartyID, o.CategoryID, o.Label, o.Reference, o.AmountNet, o.TaxAmount, o.TaxAccountID, o.CounterpartyAccountID, o.SettlementMode, o.Nature, o.Currency, o.DuplicateConfirmed)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) OperationByID(ctx context.Context, id string, lock bool) (*domain.Operation, error) {
	q := `SELECT ` + opList + opFrom + ` WHERE o.id = $1`
	if lock {
		q += ` FOR UPDATE OF o`
	}
	o, err := scanOp(s.q(ctx).QueryRow(ctx, q, id))
	if err != nil {
		return nil, err
	}
	return o, s.loadPayments(ctx, []*domain.Operation{o})
}

func (s *Store) ListOperations(ctx context.Context, f application.OperationFilter, limit, offset int) ([]*domain.Operation, application.OpTotals, int, error) {
	where, args := ` WHERE true`, []any{}
	if f.Query != "" {
		args = append(args, like(f.Query))
		where += fmt.Sprintf(` AND (x.label ILIKE $%[1]d OR x.reference ILIKE $%[1]d OR x.third_party_name ILIKE $%[1]d OR x.category_label ILIKE $%[1]d)`, len(args))
	}
	for col, v := range map[string]*string{"kind": f.Kind, "status": f.Status, "category_id": f.CategoryID, "third_party_id": f.ThirdPartyID} {
		if v != nil {
			args = append(args, *v)
			where += fmt.Sprintf(` AND x.%s = $%d`, col, len(args))
		}
	}
	if f.From != nil {
		args = append(args, *f.From)
		where += fmt.Sprintf(` AND x.operation_date >= $%d`, len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where += fmt.Sprintf(` AND x.operation_date <= $%d`, len(args))
	}
	base := `(SELECT ` + opList + opFrom + `) x`
	var tot application.OpTotals
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*), COALESCE(sum(x.amount_net),0)::bigint, COALESCE(sum(x.tax_amount),0)::bigint FROM `+base+where, args...).Scan(&total, &tot.Net, &tot.Tax); err != nil {
		return nil, tot, 0, err
	}
	tot.Gross = tot.Net + tot.Tax
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT x.* FROM `+base+where+fmt.Sprintf(` ORDER BY x.operation_date DESC, x.created_at DESC, x.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, tot, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Operation, error) { return scanOp(r) })
	if err != nil {
		return nil, tot, 0, err
	}
	return items, tot, total, s.loadPayments(ctx, items)
}

func (s *Store) DeleteOperation(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM operations WHERE id = $1`, id)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) SetOperationEntry(ctx context.Context, opID string, entryID *string) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE operations SET entry_id = $2 WHERE id = $1`, opID, entryID)
	return mapErr(err)
}

func (s *Store) DuplicateOperationExists(ctx context.Context, o *domain.Operation) (bool, error) {
	var ok bool
	err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM operations WHERE third_party_id = $1 AND lower(reference) = lower($2) AND kind = $3 AND amount_net = $4 AND tax_amount = $5 AND id <> $6)`,
		o.ThirdPartyID, o.Reference, o.Kind, o.AmountNet, o.TaxAmount, o.ID).Scan(&ok)
	return ok, err
}

func (s *Store) AddOperationPayment(ctx context.Context, opID string, p domain.OperationPayment, by string) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO operation_payments (id, operation_id, payment_date, amount, account_id, entry_id, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7)`, p.ID, opID, p.PaymentDate, p.Amount, p.AccountID, p.EntryID, by)
	return mapErr(err)
}

// ── récurrences ─────────────────────────────────────────────────────────────

const recCols = `r.id::text, r.kind, r.category_id::text, c.label, r.third_party_id::text, r.label, r.amount_net, r.tax_amount, r.tax_account_id::text, r.counterparty_account_id::text, r.journal_id::text,
	r.settlement_mode, r.nature, r.day_of_month, r.start_date, r.end_date, r.active, r.last_generated_month, r.created_by::text`
const recFrom = ` FROM recurring_templates r JOIN operation_categories c ON c.id = r.category_id`

func scanRec(r pgx.Row) (*domain.Recurring, error) {
	var x domain.Recurring
	if err := r.Scan(&x.ID, &x.Kind, &x.CategoryID, &x.CategoryLabel, &x.ThirdPartyID, &x.Label, &x.AmountNet, &x.TaxAmount, &x.TaxAccountID, &x.CounterpartyAccountID, &x.JournalID,
		&x.SettlementMode, &x.Nature, &x.DayOfMonth, &x.Start, &x.End, &x.Active, &x.LastGeneratedMonth, &x.CreatedBy); err != nil {
		return nil, notFound(err)
	}
	return &x, nil
}

func (s *Store) CreateRecurring(ctx context.Context, r *domain.Recurring) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO recurring_templates (id, kind, category_id, third_party_id, label, amount_net, tax_amount, tax_account_id, counterparty_account_id, journal_id, settlement_mode, nature,
		day_of_month, start_date, end_date, active, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		r.ID, r.Kind, r.CategoryID, r.ThirdPartyID, r.Label, r.AmountNet, r.TaxAmount, r.TaxAccountID, r.CounterpartyAccountID, r.JournalID, r.SettlementMode, r.Nature, r.DayOfMonth, r.Start, r.End, r.Active, r.CreatedBy)
	return mapErr(err)
}

func (s *Store) UpdateRecurring(ctx context.Context, r *domain.Recurring) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE recurring_templates SET category_id=$2, third_party_id=$3, label=$4, amount_net=$5, tax_amount=$6, tax_account_id=$7, counterparty_account_id=$8, journal_id=$9,
		settlement_mode=$10, nature=$11, day_of_month=$12, end_date=$13, active=$14, updated_at=now() WHERE id=$1`,
		r.ID, r.CategoryID, r.ThirdPartyID, r.Label, r.AmountNet, r.TaxAmount, r.TaxAccountID, r.CounterpartyAccountID, r.JournalID, r.SettlementMode, r.Nature, r.DayOfMonth, r.End, r.Active)
	if err = mapErr(err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) RecurringByID(ctx context.Context, id string) (*domain.Recurring, error) {
	return scanRec(s.q(ctx).QueryRow(ctx, `SELECT `+recCols+recFrom+` WHERE r.id = $1`, id))
}

func (s *Store) ListRecurring(ctx context.Context) ([]*domain.Recurring, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT `+recCols+recFrom+` ORDER BY r.active DESC, lower(r.label)`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Recurring, error) { return scanRec(r) })
}

func (s *Store) SetRecurringLastMonth(ctx context.Context, id string, month time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE recurring_templates SET last_generated_month = $2, updated_at = now() WHERE id = $1`, id, month)
	return err
}

// ── pièces ──────────────────────────────────────────────────────────────────

func (s *Store) AddAttachment(ctx context.Context, a *domain.Attachment, data []byte, by string) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO attachments (id, operation_id, filename, content_type, size, sha256, data, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, a.OperationID, a.Filename, a.ContentType, a.Size, a.SHA256, data, by, a.CreatedAt)
	return mapErr(err)
}

const attCols = `id::text, operation_id::text, filename, content_type, size, sha256, created_at`

func scanAtt(r pgx.Row) (*domain.Attachment, error) {
	var a domain.Attachment
	if err := r.Scan(&a.ID, &a.OperationID, &a.Filename, &a.ContentType, &a.Size, &a.SHA256, &a.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &a, nil
}

func (s *Store) ListAttachments(ctx context.Context, opID string) ([]*domain.Attachment, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT `+attCols+` FROM attachments WHERE operation_id = $1 ORDER BY created_at`, opID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Attachment, error) { return scanAtt(r) })
}

func (s *Store) AttachmentByID(ctx context.Context, id string) (*domain.Attachment, []byte, error) {
	var a domain.Attachment
	var data []byte
	if err := s.q(ctx).QueryRow(ctx, `SELECT `+attCols+`, data FROM attachments WHERE id = $1`, id).Scan(&a.ID, &a.OperationID, &a.Filename, &a.ContentType, &a.Size, &a.SHA256, &a.CreatedAt, &data); err != nil {
		return nil, nil, notFound(err)
	}
	return &a, data, nil
}

func (s *Store) DeleteAttachment(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}
