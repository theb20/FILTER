-- +goose Up
-- Préférences d'interface par compte (guide visuel…), synchronisées entre appareils.
-- UN propriétaire exactement : un client OU un membre du personnel (deux mondes séparés). La suppression du compte efface ses préférences.
CREATE TABLE user_preferences (
    customer_id     uuid        REFERENCES users (id) ON DELETE CASCADE,
    staff_id        uuid        REFERENCES staff_users (id) ON DELETE CASCADE,
    guide_enabled   boolean     NOT NULL DEFAULT true,
    guide_animations text       NOT NULL DEFAULT 'auto' CHECK (guide_animations IN ('auto', 'off')),
    guide_seen      text[]      NOT NULL DEFAULT '{}' CHECK (cardinality(guide_seen) <= 200),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_preferences_one_owner CHECK ((customer_id IS NULL) <> (staff_id IS NULL))
);
CREATE UNIQUE INDEX user_preferences_customer_uq ON user_preferences (customer_id) WHERE customer_id IS NOT NULL;
CREATE UNIQUE INDEX user_preferences_staff_uq    ON user_preferences (staff_id)    WHERE staff_id    IS NOT NULL;

-- +goose Down
DROP TABLE user_preferences;
