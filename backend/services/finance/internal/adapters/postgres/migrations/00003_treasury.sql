-- +goose Up
-- Finance · F3 : comptes bancaires (numéro chiffré), relevés importés (immuables), rapprochement avec écarts conservés.

CREATE TABLE bank_accounts (
    id               uuid        PRIMARY KEY,
    name             text        NOT NULL CHECK (btrim(name) <> '' AND length(name) <= 120),
    bank_name        text        NOT NULL CHECK (btrim(bank_name) <> '' AND length(bank_name) <= 120),
    currency         text        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    account_id       uuid        NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    number_encrypted bytea,                                   -- AES-256-GCM ; clé hors base ; jamais renvoyé par l'API
    number_last4     text        CHECK (number_last4 IS NULL OR number_last4 ~ '^[A-Za-z0-9]{1,4}$'),
    opening_balance  bigint      NOT NULL DEFAULT 0,          -- solde du relevé à la date d'ouverture
    opening_date     date        NOT NULL,
    active           boolean     NOT NULL DEFAULT true,
    created_by       uuid        NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT bank_accounts_number_pair CHECK ((number_encrypted IS NULL) = (number_last4 IS NULL))
);
CREATE UNIQUE INDEX bank_accounts_name_key ON bank_accounts (lower(name));
CREATE UNIQUE INDEX bank_accounts_ledger_key ON bank_accounts (account_id);  -- un compte comptable = un compte bancaire : aucune ambiguïté de rapprochement

CREATE TABLE bank_statements (
    id              uuid        PRIMARY KEY,
    bank_account_id uuid        NOT NULL REFERENCES bank_accounts(id) ON DELETE RESTRICT,
    filename        text        NOT NULL CHECK (btrim(filename) <> '' AND length(filename) <= 200),
    sha256          text        NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    period_start    date        NOT NULL,
    period_end      date        NOT NULL CHECK (period_end >= period_start),
    line_count      integer     NOT NULL CHECK (line_count >= 0),
    inserted_count  integer     NOT NULL CHECK (inserted_count >= 0),
    skipped_count   integer     NOT NULL CHECK (skipped_count >= 0),
    closing_balance bigint,
    imported_by     uuid        NOT NULL,
    imported_at     timestamptz NOT NULL DEFAULT now()
);
-- le même fichier ne s'importe qu'une fois par compte
CREATE UNIQUE INDEX bank_statements_file_key ON bank_statements (bank_account_id, sha256);

CREATE TABLE bank_statement_lines (
    id              uuid        PRIMARY KEY,
    statement_id    uuid        NOT NULL REFERENCES bank_statements(id) ON DELETE RESTRICT,
    bank_account_id uuid        NOT NULL REFERENCES bank_accounts(id) ON DELETE RESTRICT,
    line_date       date        NOT NULL,
    label           text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 300),
    reference       text        CHECK (reference IS NULL OR (btrim(reference) <> '' AND length(reference) <= 100)),
    amount          bigint      NOT NULL CHECK (amount <> 0),   -- signé : positif = entrée d'argent
    fingerprint     text        NOT NULL CHECK (fingerprint ~ '^[0-9a-f]{64}$'),
    created_at      timestamptz NOT NULL DEFAULT now()
);
-- anti-doublon entre relevés qui se recouvrent : une ligne identique (même occurrence) n'est importée qu'une fois
CREATE UNIQUE INDEX bank_statement_lines_fp_key ON bank_statement_lines (bank_account_id, fingerprint);
CREATE INDEX bank_statement_lines_date_idx ON bank_statement_lines (bank_account_id, line_date DESC, id);

-- Un relevé importé est un document : il ne se modifie ni ne se supprime.
-- +goose StatementBegin
CREATE FUNCTION statement_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'statement_immutable' USING ERRCODE = '23514'; END $$;
-- +goose StatementEnd
CREATE TRIGGER bank_statement_lines_immutable BEFORE UPDATE OR DELETE ON bank_statement_lines FOR EACH ROW EXECUTE FUNCTION statement_immutable();
CREATE TRIGGER bank_statements_immutable BEFORE UPDATE OR DELETE ON bank_statements FOR EACH ROW EXECUTE FUNCTION statement_immutable();

CREATE TABLE reconciliations (
    id                uuid        PRIMARY KEY,
    bank_account_id   uuid        NOT NULL REFERENCES bank_accounts(id) ON DELETE RESTRICT,
    statement_line_id uuid        NOT NULL REFERENCES bank_statement_lines(id) ON DELETE RESTRICT,
    entry_line_id     uuid        NOT NULL REFERENCES entry_lines(id) ON DELETE RESTRICT,
    amount            bigint      NOT NULL CHECK (amount > 0),
    matched_by        uuid        NOT NULL,
    matched_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (statement_line_id, entry_line_id)
);
CREATE INDEX reconciliations_entry_line_idx ON reconciliations (entry_line_id);

-- Un rapprochement alloue un MONTANT entre une ligne de relevé et une ligne d'écriture VALIDÉE du compte banque, dans le bon sens ;
-- jamais plus que ce qui reste de chaque côté. L'écart non alloué reste visible : il n'est jamais absorbé.
-- +goose StatementBegin
CREATE FUNCTION reconciliations_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE sl bank_statement_lines%ROWTYPE; el entry_lines%ROWTYPE; est text; ed date; ba bank_accounts%ROWTYPE; used bigint;
BEGIN
    IF TG_OP = 'UPDATE' THEN RAISE EXCEPTION 'reconciliation_immutable' USING ERRCODE = '23514'; END IF;
    PERFORM pg_advisory_xact_lock(hashtext('recon_sl:' || NEW.statement_line_id::text));
    PERFORM pg_advisory_xact_lock(hashtext('recon_el:' || NEW.entry_line_id::text));
    SELECT * INTO sl FROM bank_statement_lines WHERE id = NEW.statement_line_id;
    SELECT * INTO el FROM entry_lines WHERE id = NEW.entry_line_id;
    SELECT status, accounting_date INTO est, ed FROM entries WHERE id = el.entry_id;
    SELECT * INTO ba FROM bank_accounts WHERE id = sl.bank_account_id;
    IF NEW.bank_account_id <> sl.bank_account_id THEN RAISE EXCEPTION 'recon_wrong_bank_account' USING ERRCODE = '23514'; END IF;
    IF est <> 'posted' THEN RAISE EXCEPTION 'recon_entry_not_posted' USING ERRCODE = '23514'; END IF;
    IF el.account_id <> ba.account_id THEN RAISE EXCEPTION 'recon_wrong_account' USING ERRCODE = '23514'; END IF;
    IF ed < ba.opening_date THEN RAISE EXCEPTION 'recon_before_opening' USING ERRCODE = '23514'; END IF;  -- avant l'ouverture : couvert par le solde d'ouverture, pas par les lignes du relevé
    IF (sl.amount > 0 AND el.debit = 0) OR (sl.amount < 0 AND el.credit = 0) THEN RAISE EXCEPTION 'recon_wrong_direction' USING ERRCODE = '23514'; END IF;
    SELECT coalesce(sum(amount), 0) INTO used FROM reconciliations WHERE statement_line_id = NEW.statement_line_id;
    IF used + NEW.amount > abs(sl.amount) THEN RAISE EXCEPTION 'recon_over_statement' USING ERRCODE = '23514'; END IF;
    SELECT coalesce(sum(amount), 0) INTO used FROM reconciliations WHERE entry_line_id = NEW.entry_line_id;
    IF used + NEW.amount > greatest(el.debit, el.credit) THEN RAISE EXCEPTION 'recon_over_entry' USING ERRCODE = '23514'; END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER reconciliations_guard BEFORE INSERT OR UPDATE ON reconciliations FOR EACH ROW EXECUTE FUNCTION reconciliations_guard();

-- +goose Down
DROP TABLE reconciliations, bank_statement_lines, bank_statements, bank_accounts;
DROP FUNCTION reconciliations_guard(), statement_immutable();
