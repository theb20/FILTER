-- +goose Up
-- Personnel : pool de comptes SÉPARÉ des clients (tables, sessions et jetons distincts).

CREATE TABLE staff_roles (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text        NOT NULL CHECK (btrim(name) <> ''),
    permissions text[]      NOT NULL DEFAULT '{}',
    system      boolean     NOT NULL DEFAULT false, -- rôle système : ni renommé ni supprimé
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX staff_roles_name_key ON staff_roles (lower(name));

-- Rôle installé d'office (nom « suggéré, à confirmer » dans le contrat). `*` = toutes les permissions.
INSERT INTO staff_roles (name, permissions, system) VALUES ('super-admin', ARRAY['*'], true);

CREATE TABLE staff_users (
    id                uuid        PRIMARY KEY,
    email             text        NOT NULL CHECK (email = lower(email) AND email = btrim(email) AND position('@' in email) > 1),
    name              text        NOT NULL CHECK (btrim(name) <> ''),
    password_hash     text        NOT NULL DEFAULT '',
    status            text        NOT NULL DEFAULT 'invited' CHECK (status IN ('invited', 'active', 'deactivated')),
    totp_secret       bytea,       -- chiffré (AES-256-GCM) : jamais en clair en base
    two_factor_enabled boolean    NOT NULL DEFAULT false,
    totp_last_step    bigint      NOT NULL DEFAULT 0,
    last_login_at     timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    -- le 2FA ne peut être actif que si un secret existe
    CONSTRAINT staff_users_2fa_needs_secret CHECK (NOT two_factor_enabled OR totp_secret IS NOT NULL),
    -- un compte actif a un mot de passe
    CONSTRAINT staff_users_active_has_password CHECK (status <> 'active' OR password_hash <> '')
);
CREATE UNIQUE INDEX staff_users_email_key ON staff_users (email);

CREATE TABLE staff_user_roles (
    staff_id uuid NOT NULL REFERENCES staff_users (id) ON DELETE CASCADE,
    role_id  uuid NOT NULL REFERENCES staff_roles (id) ON DELETE RESTRICT, -- un rôle attribué ne se supprime pas
    PRIMARY KEY (staff_id, role_id)
);
CREATE INDEX staff_user_roles_role_idx ON staff_user_roles (role_id);

CREATE TABLE staff_recovery_codes (
    staff_id  uuid        NOT NULL REFERENCES staff_users (id) ON DELETE CASCADE,
    code_hash text        NOT NULL CHECK (code_hash ~ '^[0-9a-f]{64}$'),
    used_at   timestamptz,
    PRIMARY KEY (staff_id, code_hash)
);

-- Jetons de défi 2FA : usage unique, courte durée, liés à l'IP, 5 codes erronés maximum (le JWT porte `jti` = id).
CREATE TABLE staff_challenges (
    id         text        PRIMARY KEY CHECK (id <> ''),
    staff_id   uuid        NOT NULL REFERENCES staff_users (id) ON DELETE CASCADE,
    scope      text        NOT NULL CHECK (scope IN ('verify', 'enroll', 'confirm')),
    ip         text        NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    failures   integer     NOT NULL DEFAULT 0 CHECK (failures >= 0)
);
CREATE INDEX staff_challenges_staff_idx ON staff_challenges (staff_id);

CREATE TABLE staff_token_families (
    id         uuid        PRIMARY KEY,
    staff_id   uuid        NOT NULL REFERENCES staff_users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);
CREATE INDEX staff_token_families_staff_idx ON staff_token_families (staff_id) WHERE revoked_at IS NULL;

CREATE TABLE staff_refresh_tokens (
    id         uuid        PRIMARY KEY,
    family_id  uuid        NOT NULL REFERENCES staff_token_families (id) ON DELETE CASCADE,
    token_hash text        NOT NULL CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    ip         text        NOT NULL DEFAULT '',
    user_agent text        NOT NULL DEFAULT '',
    CONSTRAINT staff_refresh_tokens_expiry CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX staff_refresh_tokens_hash_key ON staff_refresh_tokens (token_hash);
CREATE INDEX staff_refresh_tokens_family_idx ON staff_refresh_tokens (family_id);

-- +goose Down
DROP TABLE staff_refresh_tokens;
DROP TABLE staff_token_families;
DROP TABLE staff_challenges;
DROP TABLE staff_recovery_codes;
DROP TABLE staff_user_roles;
DROP TABLE staff_users;
DROP TABLE staff_roles;
