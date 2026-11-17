-- +goose Up
-- Finance · F2 : tiers, catégories, dépenses et revenus, règlements, récurrences, pièces justificatives.
-- Une opération génère une écriture en BROUILLON ; la validation, l'immuabilité et la double validation restent celles du grand livre.

CREATE TABLE third_parties (
    id           uuid        PRIMARY KEY,
    kind         text        NOT NULL CHECK (kind IN ('customer', 'supplier')),
    external_ref text        CHECK (external_ref IS NULL OR (btrim(external_ref) <> '' AND length(external_ref) <= 100)),
    display_name text        NOT NULL CHECK (btrim(display_name) <> '' AND length(display_name) <= 200),
    active       boolean     NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX third_parties_name_key ON third_parties (kind, lower(display_name));
CREATE UNIQUE INDEX third_parties_ref_key ON third_parties (kind, external_ref) WHERE external_ref IS NOT NULL;

CREATE TABLE operation_categories (
    id         uuid        PRIMARY KEY,
    kind       text        NOT NULL CHECK (kind IN ('expense', 'income')),
    label      text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 120),
    account_id uuid        NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    nature     text        CHECK (nature IS NULL OR nature IN ('fixed', 'variable')),
    active     boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT operation_categories_nature_expense CHECK (kind = 'expense' OR nature IS NULL)
);
CREATE UNIQUE INDEX operation_categories_label_key ON operation_categories (kind, lower(label));

CREATE TABLE recurring_templates (
    id                      uuid        PRIMARY KEY,
    kind                    text        NOT NULL CHECK (kind IN ('expense', 'income')),
    category_id             uuid        NOT NULL REFERENCES operation_categories(id) ON DELETE RESTRICT,
    third_party_id          uuid        REFERENCES third_parties(id) ON DELETE RESTRICT,
    label                   text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 300),
    amount_net              bigint      NOT NULL CHECK (amount_net > 0),
    tax_amount              bigint      NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    tax_account_id          uuid        REFERENCES accounts(id) ON DELETE RESTRICT,
    counterparty_account_id uuid        NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    journal_id              uuid        NOT NULL REFERENCES journals(id) ON DELETE RESTRICT,
    settlement_mode         text        NOT NULL CHECK (settlement_mode IN ('immediate', 'on_account')),
    nature                  text        CHECK (nature IS NULL OR nature IN ('fixed', 'variable')),
    day_of_month            integer     NOT NULL CHECK (day_of_month BETWEEN 1 AND 28),
    start_date              date        NOT NULL,
    end_date                date,
    active                  boolean     NOT NULL DEFAULT true,
    last_generated_month    date,
    created_by              uuid        NOT NULL,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT recurring_tax_account CHECK (tax_amount = 0 OR tax_account_id IS NOT NULL),
    CONSTRAINT recurring_dates CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE TABLE operations (
    id                      uuid        PRIMARY KEY,
    kind                    text        NOT NULL CHECK (kind IN ('expense', 'income')),
    operation_date          date        NOT NULL,
    due_date                date,
    third_party_id          uuid        REFERENCES third_parties(id) ON DELETE RESTRICT,
    category_id             uuid        NOT NULL REFERENCES operation_categories(id) ON DELETE RESTRICT,
    label                   text        NOT NULL CHECK (btrim(label) <> '' AND length(label) <= 300),
    reference               text        CHECK (reference IS NULL OR (btrim(reference) <> '' AND length(reference) <= 100)),
    amount_net              bigint      NOT NULL CHECK (amount_net > 0),
    tax_amount              bigint      NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    tax_account_id          uuid        REFERENCES accounts(id) ON DELETE RESTRICT,
    counterparty_account_id uuid        NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    settlement_mode         text        NOT NULL CHECK (settlement_mode IN ('immediate', 'on_account')),
    nature                  text        CHECK (nature IS NULL OR nature IN ('fixed', 'variable')),
    currency                text        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    entry_id                uuid        UNIQUE REFERENCES entries(id) ON DELETE SET NULL,
    duplicate_confirmed     boolean     NOT NULL DEFAULT false,
    recurring_template_id   uuid        REFERENCES recurring_templates(id) ON DELETE RESTRICT,
    recurring_month         date,
    created_by              uuid        NOT NULL,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT operations_tax_account CHECK (tax_amount = 0 OR tax_account_id IS NOT NULL),
    CONSTRAINT operations_due CHECK (due_date IS NULL OR due_date >= operation_date),
    CONSTRAINT operations_recurring_pair CHECK ((recurring_template_id IS NULL) = (recurring_month IS NULL))
);
CREATE UNIQUE INDEX operations_recurring_key ON operations (recurring_template_id, recurring_month) WHERE recurring_template_id IS NOT NULL;
CREATE INDEX operations_date_idx ON operations (operation_date DESC, id);
CREATE INDEX operations_kind_idx ON operations (kind, operation_date DESC);
CREATE INDEX operations_dup_idx ON operations (third_party_id, lower(reference)) WHERE reference IS NOT NULL;

-- Une opération liée à une écriture est figée : seul le détachement (annulation du brouillon, entry_id → NULL) est possible.
-- +goose StatementBegin
CREATE FUNCTION operations_frozen() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.entry_id IS NOT NULL AND NEW.entry_id IS NOT NULL AND (
        NEW.kind <> OLD.kind OR NEW.operation_date <> OLD.operation_date OR NEW.category_id <> OLD.category_id OR NEW.amount_net <> OLD.amount_net OR NEW.tax_amount <> OLD.tax_amount
        OR NEW.counterparty_account_id <> OLD.counterparty_account_id OR NEW.settlement_mode <> OLD.settlement_mode OR NEW.label <> OLD.label
        OR NEW.tax_account_id IS DISTINCT FROM OLD.tax_account_id OR NEW.third_party_id IS DISTINCT FROM OLD.third_party_id OR NEW.reference IS DISTINCT FROM OLD.reference) THEN
        RAISE EXCEPTION 'operation_frozen' USING ERRCODE = '23514';
    END IF;
    NEW.updated_at := now();
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER operations_frozen BEFORE UPDATE ON operations FOR EACH ROW EXECUTE FUNCTION operations_frozen();

CREATE TABLE operation_payments (
    id           uuid        PRIMARY KEY,
    operation_id uuid        NOT NULL REFERENCES operations(id) ON DELETE RESTRICT,
    payment_date date        NOT NULL,
    amount       bigint      NOT NULL CHECK (amount > 0),
    account_id   uuid        NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    entry_id     uuid        NOT NULL UNIQUE REFERENCES entries(id) ON DELETE CASCADE,  -- supprimer le brouillon supprime le règlement
    created_by   uuid        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX operation_payments_op_idx ON operation_payments (operation_id);

-- Pièces justificatives : stockées en base derrière un port (un stockage objet pourra les remplacer, sans changer l'API).
CREATE TABLE attachments (
    id           uuid        PRIMARY KEY,
    operation_id uuid        NOT NULL REFERENCES operations(id) ON DELETE CASCADE,
    filename     text        NOT NULL CHECK (btrim(filename) <> '' AND length(filename) <= 200),
    content_type text        NOT NULL CHECK (content_type IN ('application/pdf', 'image/png', 'image/jpeg')),
    size         bigint      NOT NULL CHECK (size > 0 AND size <= 5242880),
    sha256       text        NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    data         bytea       NOT NULL,
    created_by   uuid        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT attachments_size_matches CHECK (octet_length(data) = size)
);
CREATE UNIQUE INDEX attachments_dedupe_key ON attachments (operation_id, sha256);

-- +goose Down
DROP TABLE attachments, operation_payments, operations, recurring_templates, operation_categories, third_parties;
DROP FUNCTION operations_frozen();
