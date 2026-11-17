-- +goose Up
-- Identity : toutes les invariants d'intégrité vivent ICI (contraintes), jamais seulement dans le code.

CREATE TABLE companies (
    id         uuid        PRIMARY KEY,
    legal_name text        NOT NULL CHECK (btrim(legal_name) <> ''),
    tax_id     text        NOT NULL CHECK (btrim(tax_id) <> ''),
    country    text        NOT NULL CHECK (country ~ '^[A-Z]{2}$'),
    approved   boolean     NOT NULL DEFAULT false, -- validation MANUELLE (décision métier)
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id                 uuid        PRIMARY KEY,
    email              text        NOT NULL CHECK (email = lower(email) AND email = btrim(email) AND position('@' in email) > 1),
    password_hash      text        NOT NULL DEFAULT '', -- vide : compte créé via un fournisseur OAuth
    email_verified     boolean     NOT NULL DEFAULT false,
    kind               text        NOT NULL CHECK (kind IN ('individual', 'company')),
    first_name         text        NOT NULL DEFAULT '',
    last_name          text        NOT NULL DEFAULT '',
    locale             text        NOT NULL DEFAULT 'fr' CHECK (locale <> ''),
    preferred_currency text        NOT NULL DEFAULT 'XOF' CHECK (preferred_currency ~ '^[A-Z]{3}$'),
    status             text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    company_id         uuid        REFERENCES companies (id) DEFERRABLE INITIALLY DEFERRED,
    created_at         timestamptz NOT NULL DEFAULT now(),
    -- un compte « entreprise » a une entreprise, un compte « particulier » n'en a pas
    CONSTRAINT users_kind_company CHECK ((kind = 'company') = (company_id IS NOT NULL))
);
-- Unicité de l'adresse, insensible à la casse, GARANTIE par la base (course entre deux inscriptions comprise).
CREATE UNIQUE INDEX users_email_key ON users (email);
CREATE INDEX users_company_idx ON users (company_id) WHERE company_id IS NOT NULL;

-- Une famille de jetons = une session. Révoquer la famille invalide tous ses jetons.
CREATE TABLE token_families (
    id         uuid        PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);
CREATE INDEX token_families_user_idx ON token_families (user_id) WHERE revoked_at IS NULL;

CREATE TABLE refresh_tokens (
    id         uuid        PRIMARY KEY,
    family_id  uuid        NOT NULL REFERENCES token_families (id) ON DELETE CASCADE,
    token_hash text        NOT NULL CHECK (token_hash ~ '^[0-9a-f]{64}$'), -- SHA-256 : le jeton en clair n'est JAMAIS stocké
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    ip         text        NOT NULL DEFAULT '',
    user_agent text        NOT NULL DEFAULT '',
    CONSTRAINT refresh_tokens_expiry CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX refresh_tokens_hash_key ON refresh_tokens (token_hash);
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id);

-- Jetons à usage unique (vérification d'e-mail ; changement d'e-mail, invitation… s'ajouteront par migration).
CREATE TABLE one_time_tokens (
    purpose    text        NOT NULL CHECK (purpose IN ('email_verification')),
    token_hash text        NOT NULL CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    PRIMARY KEY (purpose, token_hash)
);
CREATE INDEX one_time_tokens_user_idx ON one_time_tokens (user_id);

-- Ralentissement progressif par couple (compte, IP).
CREATE TABLE login_throttle (
    key          text        PRIMARY KEY CHECK (key <> ''),
    failures     integer     NOT NULL CHECK (failures > 0),
    last_failure timestamptz NOT NULL
);

-- Échecs par compte (fenêtre glissante) et marque « alerte déjà envoyée ».
CREATE TABLE login_failures (
    id     bigserial   PRIMARY KEY,
    email  text        NOT NULL CHECK (email <> ''),
    at     timestamptz NOT NULL
);
CREATE INDEX login_failures_email_idx ON login_failures (email, at);

CREATE TABLE login_alerts (
    email text        PRIMARY KEY CHECK (email <> ''),
    until timestamptz NOT NULL
);

-- +goose Down
DROP TABLE login_alerts;
DROP TABLE login_failures;
DROP TABLE login_throttle;
DROP TABLE one_time_tokens;
DROP TABLE refresh_tokens;
DROP TABLE token_families;
DROP TABLE users;
DROP TABLE companies;
