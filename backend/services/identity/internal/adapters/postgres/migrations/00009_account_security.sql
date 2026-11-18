-- +goose Up
-- Sécurité du compte client (décisions du 18 nov. 2026) : historique des connexions RÉUSSIES et appareils connus,
-- conservés 90 jours (purge applicative à chaque connexion), photo de profil et couverture, lien « ce n'était pas moi ».

-- Historique des connexions. La position est APPROXIMATIVE (base DB-IP locale) et figée au moment de la connexion.
CREATE TABLE login_events (
    id           uuid             PRIMARY KEY,
    user_id      uuid             NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    at           timestamptz      NOT NULL,
    ip           text             NOT NULL DEFAULT '' CHECK (char_length(ip) <= 64),
    user_agent   text             NOT NULL DEFAULT '' CHECK (char_length(user_agent) <= 512),
    method       text             NOT NULL CHECK (method IN ('password', 'google')),
    new_device   boolean          NOT NULL,
    city         text             NOT NULL DEFAULT '',
    region       text             NOT NULL DEFAULT '',
    country_code text             NOT NULL DEFAULT '' CHECK (country_code = '' OR country_code ~ '^[A-Z]{2}$'),
    country      text             NOT NULL DEFAULT '',
    latitude     double precision CHECK (latitude BETWEEN -90 AND 90),
    longitude    double precision CHECK (longitude BETWEEN -180 AND 180),
    CONSTRAINT login_events_coords CHECK ((latitude IS NULL) = (longitude IS NULL))
);
CREATE INDEX login_events_user_idx ON login_events (user_id, at DESC);
CREATE INDEX login_events_at_idx ON login_events (at); -- purge

-- Appareils connus d'un compte : condensat (navigateur, système, type) — l'en-tête brut n'y est pas conservé.
CREATE TABLE known_devices (
    user_id       uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    device_key    text        NOT NULL CHECK (device_key ~ '^[0-9a-f]{64}$'),
    first_seen_at timestamptz NOT NULL,
    last_seen_at  timestamptz NOT NULL,
    PRIMARY KEY (user_id, device_key)
);
CREATE INDEX known_devices_seen_idx ON known_devices (last_seen_at); -- purge

-- Photo de profil et couverture : contenu vérifié par le service (PNG, JPEG, WebP), 600 Kio au plus.
CREATE TABLE user_images (
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('avatar', 'cover')),
    content_type text        NOT NULL CHECK (content_type IN ('image/png', 'image/jpeg', 'image/webp')),
    data         bytea       NOT NULL CHECK (octet_length(data) BETWEEN 1 AND 614400),
    updated_at   timestamptz NOT NULL,
    PRIMARY KEY (user_id, kind)
);

ALTER TABLE one_time_tokens DROP CONSTRAINT one_time_tokens_purpose_check;
ALTER TABLE one_time_tokens ADD CONSTRAINT one_time_tokens_purpose_check
    CHECK (purpose IN ('email_verification', 'password_reset_code', 'password_reset', 'oauth_exchange', 'email_change', 'login_alert'));

-- +goose Down
DELETE FROM one_time_tokens WHERE purpose = 'login_alert';
ALTER TABLE one_time_tokens DROP CONSTRAINT one_time_tokens_purpose_check;
ALTER TABLE one_time_tokens ADD CONSTRAINT one_time_tokens_purpose_check
    CHECK (purpose IN ('email_verification', 'password_reset_code', 'password_reset', 'oauth_exchange', 'email_change'));
DROP TABLE user_images;
DROP TABLE known_devices;
DROP TABLE login_events;
