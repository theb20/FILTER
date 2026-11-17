-- +goose Up
-- Finance · grand livre. Les invariants sont dans la BASE : équilibre, immuabilité, périodes, numérotation, double validation.

CREATE TABLE finance_settings (
    id                   boolean     PRIMARY KEY DEFAULT true CHECK (id),
    base_currency        text        NOT NULL DEFAULT 'XOF' CHECK (base_currency ~ '^[A-Z]{3}$'),
    four_eyes_min_amount bigint      NOT NULL DEFAULT 0 CHECK (four_eyes_min_amount >= 0),  -- 0 : une autre personne valide TOUJOURS
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           uuid
);
INSERT INTO finance_settings DEFAULT VALUES;

CREATE TABLE accounts (
    id         uuid        PRIMARY KEY,
    code       text        NOT NULL CHECK (code ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,19}$'),
    label      text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 200),
    kind       text        NOT NULL CHECK (kind IN ('asset', 'liability', 'equity', 'income', 'expense', 'other')),
    active     boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX accounts_code_key ON accounts (lower(code));

CREATE TABLE journals (
    id         uuid        PRIMARY KEY,
    code       text        NOT NULL CHECK (code ~ '^[A-Z0-9]{1,10}$'),
    label      text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 200),
    kind       text        NOT NULL CHECK (kind IN ('sales', 'purchases', 'bank', 'cash', 'general')),
    active     boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX journals_code_key ON journals (code);

CREATE TABLE fiscal_years (
    id         uuid        PRIMARY KEY,
    label      text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 100),
    start_date date        NOT NULL,
    end_date   date        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fiscal_years_order CHECK (end_date >= start_date AND end_date <= start_date + interval '24 months'),
    CONSTRAINT fiscal_years_no_overlap EXCLUDE USING gist (daterange(start_date, end_date, '[]') WITH &&)
);
CREATE UNIQUE INDEX fiscal_years_label_key ON fiscal_years (lower(label));

CREATE TABLE fiscal_periods (
    id             uuid        PRIMARY KEY,
    fiscal_year_id uuid        NOT NULL REFERENCES fiscal_years(id) ON DELETE RESTRICT,
    number         integer     NOT NULL CHECK (number >= 1),
    start_date     date        NOT NULL,
    end_date       date        NOT NULL CHECK (end_date >= start_date),
    status         text        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    closed_at      timestamptz,
    closed_by      uuid,
    UNIQUE (fiscal_year_id, number),
    CONSTRAINT fiscal_periods_closed_has_actor CHECK ((status = 'closed') = (closed_at IS NOT NULL AND closed_by IS NOT NULL))
);
CREATE INDEX fiscal_periods_dates_idx ON fiscal_periods (start_date, end_date);

-- Approbations (double validation).
CREATE TABLE approval_requests (
    id            uuid        PRIMARY KEY,
    type          text        NOT NULL CHECK (type IN ('reopen_period')),
    subject_id    uuid        NOT NULL,
    reason        text        NOT NULL CHECK (length(btrim(reason)) BETWEEN 5 AND 500),
    status        text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_by  uuid        NOT NULL,
    requested_at  timestamptz NOT NULL DEFAULT now(),
    decided_by    uuid,
    decided_at    timestamptz,
    decision_note text        CHECK (decision_note IS NULL OR length(decision_note) <= 500),
    consumed_at   timestamptz,
    CONSTRAINT approvals_decided_has_actor CHECK ((status = 'pending') = (decided_by IS NULL AND decided_at IS NULL)),
    CONSTRAINT approvals_two_people CHECK (status <> 'approved' OR decided_by <> requested_by),
    CONSTRAINT approvals_consumed_only_if_approved CHECK (consumed_at IS NULL OR status = 'approved')
);
CREATE UNIQUE INDEX approvals_one_pending_key ON approval_requests (type, subject_id) WHERE status = 'pending';

-- La réouverture d'une période exige une approbation (autre personne), consommée une seule fois ; la clôture exige l'absence de brouillons.
-- +goose StatementBegin
CREATE FUNCTION fiscal_periods_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE ap uuid;
BEGIN
    IF OLD.status = 'open' AND NEW.status = 'closed' THEN
        IF EXISTS (SELECT 1 FROM entries WHERE fiscal_period_id = OLD.id AND status = 'draft') THEN
            RAISE EXCEPTION 'period_has_drafts' USING ERRCODE = '23514';
        END IF;
    ELSIF OLD.status = 'closed' AND NEW.status = 'open' THEN
        UPDATE approval_requests SET consumed_at = now()
         WHERE id = (SELECT id FROM approval_requests WHERE type = 'reopen_period' AND subject_id = OLD.id AND status = 'approved' AND consumed_at IS NULL ORDER BY decided_at LIMIT 1)
        RETURNING id INTO ap;
        IF ap IS NULL THEN
            RAISE EXCEPTION 'reopen_not_approved' USING ERRCODE = '23514';
        END IF;
    END IF;
    IF NEW.fiscal_year_id <> OLD.fiscal_year_id OR NEW.start_date <> OLD.start_date OR NEW.end_date <> OLD.end_date THEN
        RAISE EXCEPTION 'period_dates_frozen' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TABLE entries (
    id               uuid        PRIMARY KEY,
    journal_id       uuid        NOT NULL REFERENCES journals(id) ON DELETE RESTRICT,
    fiscal_year_id   uuid        NOT NULL REFERENCES fiscal_years(id) ON DELETE RESTRICT,
    fiscal_period_id uuid        NOT NULL REFERENCES fiscal_periods(id) ON DELETE RESTRICT,
    accounting_date  date        NOT NULL,
    status           text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'posted')),
    entry_number     bigint      CHECK (entry_number IS NULL OR entry_number >= 1),
    label            text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 300),
    reference        text        CHECK (reference IS NULL OR (btrim(reference) <> '' AND length(reference) <= 100)),
    currency         text        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    origin           text        NOT NULL DEFAULT 'manual' CHECK (origin IN ('manual', 'operation', 'reversal')),
    source_type      text,
    source_id        uuid,
    source_kind      text,
    reverses_entry_id uuid       UNIQUE REFERENCES entries(id) ON DELETE RESTRICT,
    reversal_reason  text        CHECK (reversal_reason IS NULL OR length(btrim(reversal_reason)) BETWEEN 5 AND 500),
    total_amount     bigint      NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    created_by       uuid        NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    posted_by        uuid,
    posted_at        timestamptz,
    CONSTRAINT entries_posted_complete CHECK ((status = 'posted') = (entry_number IS NOT NULL AND posted_by IS NOT NULL AND posted_at IS NOT NULL)),
    CONSTRAINT entries_source_complete CHECK ((source_type IS NULL) = (source_id IS NULL) AND (source_type IS NULL) = (source_kind IS NULL)),
    CONSTRAINT entries_reversal_complete CHECK ((origin = 'reversal') = (reverses_entry_id IS NOT NULL) AND (origin = 'reversal') = (reversal_reason IS NOT NULL))
);
-- anti-doublon : une même source ne produit qu'UNE écriture par nature
CREATE UNIQUE INDEX entries_source_key ON entries (source_type, source_id, source_kind) WHERE source_type IS NOT NULL;
-- numérotation continue par journal et exercice (attribuée à la validation seulement)
CREATE UNIQUE INDEX entries_number_key ON entries (journal_id, fiscal_year_id, entry_number) WHERE entry_number IS NOT NULL;
CREATE INDEX entries_period_idx ON entries (fiscal_period_id, status);
CREATE INDEX entries_date_idx ON entries (accounting_date, id);
CREATE INDEX entries_status_idx ON entries (status, created_at DESC);

CREATE TABLE entry_lines (
    id         uuid    PRIMARY KEY,
    entry_id   uuid    NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    line_no    integer NOT NULL CHECK (line_no >= 1),
    account_id uuid    NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    debit      bigint  NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit     bigint  NOT NULL DEFAULT 0 CHECK (credit >= 0),
    memo       text    CHECK (memo IS NULL OR (btrim(memo) <> '' AND length(memo) <= 300)),
    UNIQUE (entry_id, line_no),
    CONSTRAINT entry_lines_one_side CHECK ((debit = 0) <> (credit = 0))
);
CREATE INDEX entry_lines_account_idx ON entry_lines (account_id, entry_id);

CREATE TABLE entry_counters (
    journal_id     uuid   NOT NULL REFERENCES journals(id) ON DELETE RESTRICT,
    fiscal_year_id uuid   NOT NULL REFERENCES fiscal_years(id) ON DELETE RESTRICT,
    last_number    bigint NOT NULL DEFAULT 0 CHECK (last_number >= 0),
    PRIMARY KEY (journal_id, fiscal_year_id)
);

-- ── garde-fous des écritures ──
-- +goose StatementBegin
CREATE FUNCTION entries_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE d bigint; c bigint; n integer; p fiscal_periods%ROWTYPE; min_amount bigint;
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'posted' THEN RAISE EXCEPTION 'entry_immutable' USING ERRCODE = '23514'; END IF;
        RETURN OLD;
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.status = 'posted' THEN
        RAISE EXCEPTION 'entry_immutable' USING ERRCODE = '23514';
    END IF;
    IF TG_OP = 'INSERT' AND NEW.status <> 'draft' THEN
        RAISE EXCEPTION 'entry_must_start_as_draft' USING ERRCODE = '23514';
    END IF;
    SELECT * INTO p FROM fiscal_periods WHERE id = NEW.fiscal_period_id;
    IF p.id IS NULL OR p.fiscal_year_id <> NEW.fiscal_year_id OR NEW.accounting_date < p.start_date OR NEW.accounting_date > p.end_date THEN
        RAISE EXCEPTION 'entry_date_outside_period' USING ERRCODE = '23514';
    END IF;
    IF p.status <> 'open' THEN
        RAISE EXCEPTION 'period_closed' USING ERRCODE = '23514';
    END IF;
    IF NEW.status = 'posted' THEN
        SELECT coalesce(sum(debit), 0), coalesce(sum(credit), 0), count(*) INTO d, c, n FROM entry_lines WHERE entry_id = NEW.id;
        IF n < 2 OR d <> c OR d = 0 THEN
            RAISE EXCEPTION 'entry_unbalanced' USING ERRCODE = '23514';
        END IF;
        IF EXISTS (SELECT 1 FROM entry_lines l JOIN accounts a ON a.id = l.account_id WHERE l.entry_id = NEW.id AND NOT a.active) THEN
            RAISE EXCEPTION 'account_inactive' USING ERRCODE = '23514';
        END IF;
        IF EXISTS (SELECT 1 FROM journals j WHERE j.id = NEW.journal_id AND NOT j.active) THEN
            RAISE EXCEPTION 'journal_inactive' USING ERRCODE = '23514';
        END IF;
        SELECT four_eyes_min_amount INTO min_amount FROM finance_settings;
        IF NEW.posted_by = NEW.created_by AND d >= min_amount THEN
            RAISE EXCEPTION 'four_eyes_required' USING ERRCODE = '23514';
        END IF;
        NEW.total_amount := d;
    END IF;
    NEW.updated_at := now();
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER entries_guard BEFORE INSERT OR UPDATE OR DELETE ON entries FOR EACH ROW EXECUTE FUNCTION entries_guard();
CREATE TRIGGER fiscal_periods_guard BEFORE UPDATE ON fiscal_periods FOR EACH ROW EXECUTE FUNCTION fiscal_periods_guard();

-- +goose StatementBegin
CREATE FUNCTION entry_lines_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE st text;
BEGIN
    SELECT status INTO st FROM entries WHERE id = CASE WHEN TG_OP = 'DELETE' THEN OLD.entry_id ELSE NEW.entry_id END;
    IF st = 'posted' THEN RAISE EXCEPTION 'entry_immutable' USING ERRCODE = '23514'; END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END $$;
-- +goose StatementEnd
CREATE TRIGGER entry_lines_guard BEFORE INSERT OR UPDATE OR DELETE ON entry_lines FOR EACH ROW EXECUTE FUNCTION entry_lines_guard();

-- +goose Down
DROP TABLE entry_counters, entry_lines, entries, approval_requests, fiscal_periods, fiscal_years, journals, accounts, finance_settings;
DROP FUNCTION entries_guard(), entry_lines_guard(), fiscal_periods_guard();
