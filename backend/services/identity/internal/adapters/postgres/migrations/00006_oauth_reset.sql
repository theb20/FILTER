-- +goose Up
-- Réinitialisation du mot de passe (code puis jeton) et connexion via un fournisseur (Google).

ALTER TABLE one_time_tokens DROP CONSTRAINT one_time_tokens_purpose_check;
ALTER TABLE one_time_tokens ADD CONSTRAINT one_time_tokens_purpose_check
    CHECK (purpose IN ('email_verification', 'password_reset_code', 'password_reset', 'oauth_exchange'));

-- Identité chez un fournisseur : unique par (fournisseur, sujet), rattachée à UN compte.
CREATE TABLE oauth_identities (
    provider   text        NOT NULL CHECK (provider IN ('google')),
    subject    text        NOT NULL CHECK (btrim(subject) <> ''),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);
CREATE INDEX oauth_identities_user_idx ON oauth_identities (user_id);

-- +goose Down
DROP TABLE oauth_identities;
ALTER TABLE one_time_tokens DROP CONSTRAINT one_time_tokens_purpose_check;
ALTER TABLE one_time_tokens ADD CONSTRAINT one_time_tokens_purpose_check CHECK (purpose IN ('email_verification'));
