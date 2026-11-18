-- +goose Up
-- Profil (téléphone, fonction), changement d'e-mail (donnée liée au jeton) et journal de consentements.

ALTER TABLE users
    ADD COLUMN phone text NOT NULL DEFAULT '' CHECK (phone = '' OR phone ~ '^\+[1-9][0-9]{6,14}$');

ALTER TABLE companies
    ADD COLUMN job_title text NOT NULL DEFAULT '' CHECK (char_length(job_title) <= 100);

-- Le jeton de changement d'e-mail porte la NOUVELLE adresse (le lien ne contient que le jeton opaque).
ALTER TABLE one_time_tokens ADD COLUMN payload text NOT NULL DEFAULT '';
ALTER TABLE one_time_tokens DROP CONSTRAINT one_time_tokens_purpose_check;
ALTER TABLE one_time_tokens ADD CONSTRAINT one_time_tokens_purpose_check
    CHECK (purpose IN ('email_verification', 'password_reset_code', 'password_reset', 'oauth_exchange', 'email_change'));

-- Preuve de consentement : APPEND-ONLY (aucune mise à jour, aucune suppression applicative). L'adresse IP est
-- tronquée avant écriture. Un compte supprimé garde la preuve, détachée de lui (user_id → NULL) : voir la décision RGPD.
CREATE TABLE consents (
    id             uuid        PRIMARY KEY,
    user_id        uuid        REFERENCES users (id) ON DELETE SET NULL,
    anonymous_id   text        NOT NULL DEFAULT '' CHECK (anonymous_id = '' OR anonymous_id ~ '^[A-Za-z0-9_-]{8,64}$'),
    necessary      boolean     NOT NULL CHECK (necessary),
    preferences    boolean     NOT NULL,
    functional     boolean     NOT NULL,
    analytics      boolean     NOT NULL,
    marketing      boolean     NOT NULL,
    policy_version text        NOT NULL CHECK (policy_version ~ '^[A-Za-z0-9._-]{1,32}$'),
    locale         text        NOT NULL DEFAULT '',
    ip_net         text        NOT NULL DEFAULT '',
    recorded_at    timestamptz NOT NULL,
    CONSTRAINT consents_has_subject CHECK (user_id IS NOT NULL OR anonymous_id <> '')
);
CREATE INDEX consents_user_idx ON consents (user_id, recorded_at DESC) WHERE user_id IS NOT NULL;
CREATE INDEX consents_anon_idx ON consents (anonymous_id, recorded_at DESC) WHERE anonymous_id <> '';

-- +goose Down
DROP TABLE consents;
ALTER TABLE one_time_tokens DROP CONSTRAINT one_time_tokens_purpose_check;
ALTER TABLE one_time_tokens ADD CONSTRAINT one_time_tokens_purpose_check
    CHECK (purpose IN ('email_verification', 'password_reset_code', 'password_reset', 'oauth_exchange'));
ALTER TABLE one_time_tokens DROP COLUMN payload;
ALTER TABLE companies DROP COLUMN job_title;
ALTER TABLE users DROP COLUMN phone;
