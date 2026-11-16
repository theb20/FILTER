-- +goose Up
-- Administration : description des rôles, invitations du personnel, journal d'audit append-only.

ALTER TABLE staff_roles ADD COLUMN description text NOT NULL DEFAULT '';

-- Un rôle système ne peut ni être renommé, ni voir ses permissions changer, ni être supprimé : garanti PAR LA BASE.
-- +goose StatementBegin
CREATE FUNCTION staff_roles_protect_system() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.system THEN RAISE EXCEPTION 'rôle système protégé' USING ERRCODE = '23514'; END IF;
        RETURN OLD;
    END IF;
    IF OLD.system AND (NEW.name IS DISTINCT FROM OLD.name OR NEW.permissions IS DISTINCT FROM OLD.permissions OR NEW.system IS DISTINCT FROM OLD.system) THEN
        RAISE EXCEPTION 'rôle système protégé' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER staff_roles_protect_system BEFORE UPDATE OR DELETE ON staff_roles FOR EACH ROW EXECUTE FUNCTION staff_roles_protect_system();

-- Seul le rôle installé d'office peut porter le joker `*` ; les autres rôles n'ont que des permissions du catalogue.
ALTER TABLE staff_roles ADD CONSTRAINT staff_roles_wildcard_system CHECK (system OR NOT ('*' = ANY (permissions)));

CREATE TABLE staff_invitations (
    token_hash text        PRIMARY KEY CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    staff_id   uuid        NOT NULL REFERENCES staff_users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz
);
CREATE INDEX staff_invitations_staff_idx ON staff_invitations (staff_id);

CREATE TABLE audit_log (
    id          uuid        PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    actor_type  text        NOT NULL CHECK (actor_type IN ('staff', 'customer', 'system')),
    actor_id    text,
    action      text        NOT NULL CHECK (action <> ''),
    entity      text        NOT NULL CHECK (entity <> ''),
    entity_id   text,
    before      jsonb,
    after       jsonb,
    ip          text,
    trace_id    text
);
CREATE INDEX audit_log_at_idx ON audit_log (at DESC, id DESC);
CREATE INDEX audit_log_entity_idx ON audit_log (entity, entity_id, at DESC);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_id, at DESC);

-- Append-only : aucune modification, et aucune suppression sauf purge de rétention explicite (SET LOCAL ff.audit_purge = 'on').
-- +goose StatementBegin
CREATE FUNCTION audit_log_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' AND current_setting('ff.audit_purge', true) = 'on' THEN RETURN OLD; END IF;
    RAISE EXCEPTION 'journal d''audit en ajout seul' USING ERRCODE = '23514';
END $$;
-- +goose StatementEnd
CREATE TRIGGER audit_log_append_only BEFORE UPDATE OR DELETE ON audit_log FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();

-- +goose Down
DROP TRIGGER audit_log_append_only ON audit_log;
DROP FUNCTION audit_log_append_only();
DROP TABLE audit_log;
DROP TABLE staff_invitations;
ALTER TABLE staff_roles DROP CONSTRAINT staff_roles_wildcard_system;
DROP TRIGGER staff_roles_protect_system ON staff_roles;
DROP FUNCTION staff_roles_protect_system();
ALTER TABLE staff_roles DROP COLUMN description;
