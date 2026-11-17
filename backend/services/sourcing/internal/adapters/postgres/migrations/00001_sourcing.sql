-- +goose Up
-- Sourcing : sources externes, budget (réservations à échéance), journal des appels. Les invariants sont dans la BASE.

CREATE TABLE sources (
    id                    uuid        PRIMARY KEY,
    kind                  text        NOT NULL CHECK (kind IN ('web_search_api', 'manufacturer_api', 'distributor_feed')),
    name                  text        NOT NULL CHECK (btrim(name) <> '' AND length(name) <= 120),
    base_url              text        NOT NULL CHECK (base_url <> ''),
    enabled               boolean     NOT NULL DEFAULT false,
    suspended_until       timestamptz,                         -- suspension automatique (budget atteint)
    priority              integer     NOT NULL CHECK (priority >= 0),
    timeout_ms            integer     NOT NULL CHECK (timeout_ms > 0),
    max_retries           integer     NOT NULL CHECK (max_retries >= 0),
    secret_ref            text        NOT NULL CHECK (btrim(secret_ref) <> ''), -- nom du secret : JAMAIS la clé
    cost_per_call_micros  bigint      NOT NULL CHECK (cost_per_call_micros >= 0),
    currency              text        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    allows_raw_storage    boolean     NOT NULL DEFAULT false,
    terms_checked_at      timestamptz,
    max_cache_ttl_seconds integer     CHECK (max_cache_ttl_seconds IS NULL OR max_cache_ttl_seconds > 0),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    -- le stockage des résultats bruts n'est autorisé qu'après vérification datée des conditions d'usage
    CONSTRAINT sources_raw_storage_needs_check CHECK (NOT allows_raw_storage OR terms_checked_at IS NOT NULL)
);
CREATE UNIQUE INDEX sources_name_key ON sources (lower(name));

CREATE TABLE source_quotas (
    source_id             uuid    PRIMARY KEY REFERENCES sources (id),
    monthly_budget_micros bigint  NOT NULL CHECK (monthly_budget_micros >= 0),
    daily_cap_micros      bigint  NOT NULL CHECK (daily_cap_micros >= 0),
    alert_threshold_pct   integer NOT NULL CHECK (alert_threshold_pct BETWEEN 1 AND 100),
    fallback_source_id    uuid    REFERENCES sources (id),
    CONSTRAINT source_quotas_daily_le_monthly CHECK (daily_cap_micros <= monthly_budget_micros),
    CONSTRAINT source_quotas_no_self_fallback CHECK (fallback_source_id IS NULL OR fallback_source_id <> source_id)
);
CREATE INDEX source_quotas_fallback_idx ON source_quotas (fallback_source_id) WHERE fallback_source_id IS NOT NULL;

-- Bascule en UN SEUL SAUT : le secours n'a pas de secours, et une source déjà désignée comme secours n'en a pas (ni A→B→C, ni A→B→A).
-- +goose StatementBegin
CREATE FUNCTION source_quotas_single_hop() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.fallback_source_id IS NULL THEN RETURN NEW; END IF;
    IF EXISTS (SELECT 1 FROM source_quotas WHERE source_id = NEW.fallback_source_id AND fallback_source_id IS NOT NULL) THEN
        RAISE EXCEPTION 'le secours ne peut pas avoir de secours' USING ERRCODE = '23514';
    END IF;
    IF EXISTS (SELECT 1 FROM source_quotas WHERE fallback_source_id = NEW.source_id AND source_id <> NEW.source_id) THEN
        RAISE EXCEPTION 'cette source est déjà le secours d''une autre' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER source_quotas_single_hop BEFORE INSERT OR UPDATE ON source_quotas FOR EACH ROW EXECUTE FUNCTION source_quotas_single_hop();

-- Compteurs de budget par période (UTC).
CREATE TABLE source_usage (
    source_id       uuid   NOT NULL REFERENCES sources (id),
    period_kind     text   NOT NULL CHECK (period_kind IN ('day', 'month')),
    period_start    date   NOT NULL,
    spent_micros    bigint NOT NULL DEFAULT 0 CHECK (spent_micros >= 0),
    reserved_micros bigint NOT NULL DEFAULT 0 CHECK (reserved_micros >= 0),
    PRIMARY KEY (source_id, period_kind, period_start)
);

-- Une ligne PAR réservation, avec échéance : un plantage ne bloque jamais une source (voir ReleaseExpired).
CREATE TABLE source_reservations (
    id            uuid        PRIMARY KEY,
    source_id     uuid        NOT NULL REFERENCES sources (id),
    search_id     uuid        NOT NULL,
    amount_micros bigint      NOT NULL CHECK (amount_micros > 0),
    day           date        NOT NULL,
    month         date        NOT NULL,
    status        text        NOT NULL DEFAULT 'held' CHECK (status IN ('held', 'settled', 'released', 'expired')),
    created_at    timestamptz NOT NULL,
    expires_at    timestamptz NOT NULL,
    settled_at    timestamptz,
    CONSTRAINT source_reservations_expiry CHECK (expires_at > created_at),
    -- UNE source n'est appelée qu'UNE FOIS par recherche, garanti par la base
    CONSTRAINT source_reservations_once_per_search UNIQUE (search_id, source_id)
);
CREATE INDEX source_reservations_held_idx ON source_reservations (expires_at) WHERE status = 'held';

CREATE TABLE source_calls (
    id          uuid        PRIMARY KEY,
    source_id   uuid        NOT NULL REFERENCES sources (id),
    search_id   uuid        NOT NULL,
    started_at  timestamptz NOT NULL,
    duration_ms integer     NOT NULL CHECK (duration_ms >= 0),
    status      text        NOT NULL,
    http_status integer,
    attempts    integer     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    cost_micros bigint      NOT NULL DEFAULT 0 CHECK (cost_micros >= 0),
    currency    text        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    error_kind  text
);
CREATE INDEX source_calls_source_idx ON source_calls (source_id, started_at DESC);
CREATE INDEX source_calls_started_idx ON source_calls (started_at);

CREATE TABLE source_tests (
    id       bigserial   PRIMARY KEY,
    actor_id text        NOT NULL CHECK (actor_id <> ''),
    at       timestamptz NOT NULL
);
CREATE INDEX source_tests_actor_idx ON source_tests (actor_id, at DESC);

-- +goose Down
DROP TABLE source_tests;
DROP TABLE source_calls;
DROP TABLE source_reservations;
DROP TABLE source_usage;
DROP TRIGGER source_quotas_single_hop ON source_quotas;
DROP FUNCTION source_quotas_single_hop();
DROP TABLE source_quotas;
DROP TABLE sources;
