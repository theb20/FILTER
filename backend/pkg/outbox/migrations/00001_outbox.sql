-- +goose Up
-- Outbox : l'événement est écrit DANS LA MÊME TRANSACTION que la donnée métier, puis relayé vers NATS.
CREATE TABLE outbox (
    id              uuid        PRIMARY KEY,
    subject         text        NOT NULL CHECK (subject <> ''),
    aggregate_id    text        NOT NULL DEFAULT '',
    payload         bytea       NOT NULL CHECK (length(payload) > 0), -- events.v1.Envelope sérialisé
    created_at      timestamptz NOT NULL DEFAULT now(),
    published_at    timestamptz,
    attempts        integer     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error      text
);
CREATE INDEX outbox_pending_idx ON outbox (next_attempt_at, created_at) WHERE published_at IS NULL;

-- Consommateurs IDEMPOTENTS : un événement n'est traité qu'une fois par consommateur.
CREATE TABLE processed_events (
    consumer     text        NOT NULL CHECK (consumer <> ''),
    event_id     uuid        NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (consumer, event_id)
);

-- +goose Down
DROP TABLE processed_events;
DROP TABLE outbox;
