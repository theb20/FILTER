-- +goose Up
-- Réglages modifiables d'identity : UNE seule ligne, contraintes de bornes en base (jamais seulement dans le code).
CREATE TABLE identity_settings (
    id                          smallint    PRIMARY KEY CHECK (id = 1),
    access_token_minutes        integer     NOT NULL CHECK (access_token_minutes >= 1),
    customer_refresh_days       integer     NOT NULL CHECK (customer_refresh_days >= 1),
    staff_refresh_hours         integer     NOT NULL CHECK (staff_refresh_hours >= 1),
    password_reset_code_minutes integer     NOT NULL CHECK (password_reset_code_minutes >= 1),
    password_reset_max_attempts integer     NOT NULL CHECK (password_reset_max_attempts >= 1),
    staff_challenge_minutes     integer     NOT NULL CHECK (staff_challenge_minutes >= 1),
    invitation_hours            integer     NOT NULL CHECK (invitation_hours >= 1),
    email_verification_hours    integer     NOT NULL CHECK (email_verification_hours >= 1),
    email_change_hours          integer     NOT NULL CHECK (email_change_hours >= 1),
    recent_login_minutes        integer     NOT NULL CHECK (recent_login_minutes >= 1),
    throttle_free_attempts      integer     NOT NULL CHECK (throttle_free_attempts >= 0),
    throttle_base_delay_seconds integer     NOT NULL CHECK (throttle_base_delay_seconds >= 1),
    throttle_max_delay_seconds  integer     NOT NULL CHECK (throttle_max_delay_seconds >= 1),
    login_alert_after_failures  integer     NOT NULL CHECK (login_alert_after_failures >= 1),
    personal_data_days          integer     CHECK (personal_data_days IS NULL OR personal_data_days >= 0),
    audit_log_days              integer     CHECK (audit_log_days IS NULL OR audit_log_days >= 0),
    updated_at                  timestamptz NOT NULL DEFAULT now(),
    updated_by                  text,
    CONSTRAINT identity_settings_throttle_order CHECK (throttle_max_delay_seconds >= throttle_base_delay_seconds)
);
-- Valeurs par défaut du contrat.
INSERT INTO identity_settings (id, access_token_minutes, customer_refresh_days, staff_refresh_hours, password_reset_code_minutes, password_reset_max_attempts,
    staff_challenge_minutes, invitation_hours, email_verification_hours, email_change_hours, recent_login_minutes,
    throttle_free_attempts, throttle_base_delay_seconds, throttle_max_delay_seconds, login_alert_after_failures)
VALUES (1, 15, 30, 8, 15, 5, 5, 72, 48, 24, 10, 3, 1, 900, 5);

-- +goose Down
DROP TABLE identity_settings;
