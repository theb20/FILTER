-- +goose Up
-- Catalogue · taxonomie : catégories (arbre), attributs par catégorie (hérités), marques, secteurs, réglages.
-- Les invariants sont dans la BASE : slugs, langue par défaut obligatoire, absence de cycle, clés d'attribut sans conflit d'héritage.

CREATE TABLE catalog_settings (
    id               boolean     PRIMARY KEY DEFAULT true CHECK (id),
    default_language text        NOT NULL DEFAULT 'fr' CHECK (default_language ~ '^[a-z]{2,3}$'),
    base_currency    text        NOT NULL DEFAULT 'XOF' CHECK (base_currency ~ '^[A-Z]{3}$'),  -- à confirmer par la direction (XOF ou EUR)
    updated_at       timestamptz NOT NULL DEFAULT now()
);
INSERT INTO catalog_settings DEFAULT VALUES;

CREATE TABLE categories (
    id          uuid        PRIMARY KEY,
    slug        text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    parent_id   uuid        REFERENCES categories(id) ON DELETE RESTRICT,
    sort_order  integer     NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    name        jsonb       NOT NULL CHECK (jsonb_typeof(name) = 'object' AND btrim(name->>'fr') <> ''),
    description jsonb       CHECK (description IS NULL OR (jsonb_typeof(description) = 'object' AND btrim(description->>'fr') <> '')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT categories_not_own_parent CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE UNIQUE INDEX categories_slug_key ON categories (slug);
CREATE INDEX categories_parent_idx ON categories (parent_id, sort_order, id);

CREATE TABLE attribute_defs (
    id          uuid        PRIMARY KEY,
    category_id uuid        NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    key         text        NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]{1,63}$'),
    label       jsonb       NOT NULL CHECK (jsonb_typeof(label) = 'object' AND btrim(label->>'fr') <> ''),
    type        text        NOT NULL CHECK (type IN ('text', 'number', 'boolean', 'enum')),
    unit        text        CHECK (unit IS NULL OR (btrim(unit) <> '' AND length(unit) <= 20)),
    filterable  boolean     NOT NULL DEFAULT false,
    required    boolean     NOT NULL DEFAULT false,
    options     jsonb       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(options) = 'array'),
    sort_order  integer     NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT attribute_defs_key_per_category UNIQUE (category_id, key),
    CONSTRAINT attribute_defs_enum_options CHECK ((type = 'enum') = (jsonb_array_length(options) > 0)),
    CONSTRAINT attribute_defs_unit_number CHECK (unit IS NULL OR type = 'number'),
    CONSTRAINT attribute_defs_no_free_text_filter CHECK (NOT (filterable AND type = 'text'))
);
CREATE INDEX attribute_defs_category_idx ON attribute_defs (category_id, sort_order, key);

CREATE TABLE brands (
    id         uuid        PRIMARY KEY,
    slug       text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    name       text        NOT NULL CHECK (btrim(name) <> '' AND length(name) <= 120),
    logo_url   text        CHECK (logo_url IS NULL OR (logo_url ~ '^https?://' AND length(logo_url) <= 2000)),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX brands_slug_key ON brands (slug);
CREATE UNIQUE INDEX brands_name_key ON brands (lower(name));

CREATE TABLE sectors (
    id         uuid        PRIMARY KEY,
    slug       text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    name       jsonb       NOT NULL CHECK (jsonb_typeof(name) = 'object' AND btrim(name->>'fr') <> ''),
    image_url  text        CHECK (image_url IS NULL OR (image_url ~ '^https?://' AND length(image_url) <= 2000)),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX sectors_slug_key ON sectors (slug);

-- ── arbre : aucun cycle, quel que soit le chemin d'écriture ──
-- +goose StatementBegin
CREATE FUNCTION categories_no_cycle() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.parent_id IS NULL THEN RETURN NEW; END IF;
    PERFORM pg_advisory_xact_lock(hashtext('catalog_tree'));
    IF EXISTS (
        WITH RECURSIVE up AS (
            SELECT id, parent_id FROM categories WHERE id = NEW.parent_id
            UNION ALL
            SELECT c.id, c.parent_id FROM categories c JOIN up ON c.id = up.parent_id
        ) SELECT 1 FROM up WHERE id = NEW.id
    ) THEN
        RAISE EXCEPTION 'category_cycle' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER categories_no_cycle BEFORE INSERT OR UPDATE OF parent_id ON categories
    FOR EACH ROW EXECUTE FUNCTION categories_no_cycle();

-- ── héritage : une clé d'attribut ne peut pas exister sur une catégorie ET sur l'un de ses ancêtres / descendants ──
-- +goose StatementBegin
CREATE FUNCTION attribute_defs_key_inheritance() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtext('attr_key:' || NEW.key));
    IF EXISTS (
        WITH RECURSIVE up AS (
            SELECT id, parent_id FROM categories WHERE id = NEW.category_id
            UNION ALL
            SELECT c.id, c.parent_id FROM categories c JOIN up ON c.id = up.parent_id
        ), down AS (
            SELECT id FROM categories WHERE id = NEW.category_id
            UNION ALL
            SELECT c.id FROM categories c JOIN down ON c.parent_id = down.id
        )
        SELECT 1 FROM attribute_defs a
         WHERE a.key = NEW.key AND a.id <> NEW.id
           AND (a.category_id IN (SELECT id FROM up) OR a.category_id IN (SELECT id FROM down))
    ) THEN
        RAISE EXCEPTION 'attribute_key_inherited' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER attribute_defs_key_inheritance BEFORE INSERT OR UPDATE OF key, category_id ON attribute_defs
    FOR EACH ROW EXECUTE FUNCTION attribute_defs_key_inheritance();

-- Déplacer une catégorie ne doit pas créer de conflit de clés entre son sous-arbre et ses nouveaux ancêtres.
-- +goose StatementBegin
CREATE FUNCTION categories_move_key_check() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.parent_id IS NOT DISTINCT FROM OLD.parent_id THEN RETURN NEW; END IF;
    PERFORM pg_advisory_xact_lock(hashtext('catalog_tree'));
    IF NEW.parent_id IS NOT NULL AND EXISTS (
        WITH RECURSIVE up AS (
            SELECT id, parent_id FROM categories WHERE id = NEW.parent_id
            UNION ALL
            SELECT c.id, c.parent_id FROM categories c JOIN up ON c.id = up.parent_id
        ), down AS (
            SELECT id FROM categories WHERE id = NEW.id
            UNION ALL
            SELECT c.id FROM categories c JOIN down ON c.parent_id = down.id
        )
        SELECT 1 FROM attribute_defs a1 JOIN attribute_defs a2 ON a1.key = a2.key
         WHERE a1.category_id IN (SELECT id FROM down) AND a2.category_id IN (SELECT id FROM up)
    ) THEN
        RAISE EXCEPTION 'attribute_key_inherited' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER categories_move_key_check BEFORE UPDATE OF parent_id ON categories
    FOR EACH ROW EXECUTE FUNCTION categories_move_key_check();

-- +goose Down
DROP TABLE sectors, brands, attribute_defs, categories, catalog_settings;
DROP FUNCTION categories_no_cycle(), attribute_defs_key_inheritance(), categories_move_key_check();
