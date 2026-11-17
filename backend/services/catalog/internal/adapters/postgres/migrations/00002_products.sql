-- +goose Up
-- Catalogue · produits : fiche, médias (URL), références croisées, secteurs, versions. Invariants dans la BASE.

CREATE SEQUENCE product_sku_seq;

CREATE TABLE products (
    id                  uuid        PRIMARY KEY,
    sku                 text        NOT NULL CHECK (sku ~ '^[A-Z0-9]+(-[A-Z0-9]+)*$' AND length(sku) <= 64),
    slug                text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    name                jsonb       NOT NULL CHECK (jsonb_typeof(name) = 'object' AND btrim(name->>'fr') <> ''),
    description         jsonb       CHECK (description IS NULL OR (jsonb_typeof(description) = 'object' AND btrim(description->>'fr') <> '')),
    brand_id            uuid        REFERENCES brands(id) ON DELETE RESTRICT,
    category_id         uuid        NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    manufacturer_ref    text        CHECK (manufacturer_ref IS NULL OR (btrim(manufacturer_ref) <> '' AND length(manufacturer_ref) <= 100)),
    manufacturer_ref_norm text      CHECK ((manufacturer_ref IS NULL) = (manufacturer_ref_norm IS NULL)),
    status              text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'review', 'active', 'archived')),
    fulfillment_mode    text        NOT NULL CHECK (fulfillment_mode IN ('stock', 'made_to_order')),
    lead_time_days      integer     CHECK (lead_time_days IS NULL OR lead_time_days >= 0),
    price_mode          text        NOT NULL DEFAULT 'fixed' CHECK (price_mode IN ('fixed', 'on_quote')),
    price_amount        bigint      CHECK (price_amount IS NULL OR price_amount >= 0),   -- HORS TAXES, unités mineures de la devise de base
    gtin                text        CHECK (gtin IS NULL OR gtin ~ '^[0-9]{8,14}$'),
    sale_unit           text        CHECK (sale_unit IS NULL OR (btrim(sale_unit) <> '' AND length(sale_unit) <= 50)),
    min_order_quantity  integer     CHECK (min_order_quantity IS NULL OR min_order_quantity >= 1),
    weight_grams        integer     CHECK (weight_grams IS NULL OR weight_grams >= 0),
    length_mm           integer     CHECK (length_mm IS NULL OR length_mm >= 0),
    width_mm            integer     CHECK (width_mm IS NULL OR width_mm >= 0),
    height_mm           integer     CHECK (height_mm IS NULL OR height_mm >= 0),
    attributes          jsonb       NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(attributes) = 'object'),
    origins             jsonb       NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(origins) = 'object'),
    version             integer     NOT NULL DEFAULT 1 CHECK (version >= 1),
    published_at        timestamptz,
    created_by          uuid,
    updated_by          uuid,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT products_made_to_order_lead_time CHECK (fulfillment_mode <> 'made_to_order' OR lead_time_days IS NOT NULL),
    CONSTRAINT products_on_quote_no_price CHECK (price_mode <> 'on_quote' OR price_amount IS NULL),
    CONSTRAINT products_active_has_price CHECK (status <> 'active' OR price_mode = 'on_quote' OR price_amount IS NOT NULL),
    CONSTRAINT products_active_is_published CHECK (status <> 'active' OR published_at IS NOT NULL)
);
CREATE UNIQUE INDEX products_sku_key ON products (sku);
CREATE UNIQUE INDEX products_slug_key ON products (slug);
-- clé de fusion du moteur : référence constructeur normalisée + marque ; sans l'une des deux, pas de contrainte
CREATE UNIQUE INDEX products_brand_ref_key ON products (brand_id, manufacturer_ref_norm) WHERE brand_id IS NOT NULL AND manufacturer_ref_norm IS NOT NULL;
CREATE INDEX products_category_idx ON products (category_id);
CREATE INDEX products_brand_idx ON products (brand_id);
CREATE INDEX products_status_idx ON products (status, updated_at DESC);

-- SKU et slug sont figés dès la première publication (les commandes, devis et liens y renvoient).
-- +goose StatementBegin
CREATE FUNCTION products_frozen_identifiers() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.published_at IS NOT NULL AND (NEW.sku <> OLD.sku OR NEW.slug <> OLD.slug) THEN
        RAISE EXCEPTION 'product_identifiers_frozen' USING ERRCODE = '23514';
    END IF;
    IF OLD.published_at IS NOT NULL AND NEW.published_at IS DISTINCT FROM OLD.published_at THEN
        RAISE EXCEPTION 'product_identifiers_frozen' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER products_frozen_identifiers BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION products_frozen_identifiers();

CREATE TABLE product_sectors (
    product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sector_id  uuid NOT NULL REFERENCES sectors(id) ON DELETE RESTRICT,
    PRIMARY KEY (product_id, sector_id)
);

CREATE TABLE product_media (
    id         uuid    PRIMARY KEY,
    product_id uuid    NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url        text    NOT NULL CHECK (url ~ '^https?://' AND length(url) <= 2000),
    alt        text    CHECK (alt IS NULL OR length(alt) <= 300),
    sort_order integer NOT NULL CHECK (sort_order >= 0)
);
CREATE UNIQUE INDEX product_media_order_key ON product_media (product_id, sort_order);

CREATE TABLE product_cross_refs (
    id          uuid PRIMARY KEY,
    product_id  uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    brand       text NOT NULL CHECK (btrim(brand) <> '' AND length(brand) <= 100),
    number      text NOT NULL CHECK (btrim(number) <> '' AND length(number) <= 100),
    number_norm text NOT NULL CHECK (number_norm <> ''),
    kind        text NOT NULL CHECK (kind IN ('oem', 'equivalent', 'supersedes'))
);
CREATE UNIQUE INDEX product_cross_refs_key ON product_cross_refs (product_id, lower(brand), number_norm, kind);
CREATE INDEX product_cross_refs_lookup ON product_cross_refs (number_norm);

-- Historique : une ligne par version (état modifié champ par champ), non modifiable.
CREATE TABLE product_versions (
    product_id uuid        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    version    integer     NOT NULL,
    at         timestamptz NOT NULL DEFAULT now(),
    actor_id   uuid,
    action     text        NOT NULL CHECK (btrim(action) <> ''),
    changes    jsonb       NOT NULL CHECK (jsonb_typeof(changes) = 'object'),
    PRIMARY KEY (product_id, version)
);
-- +goose StatementBegin
CREATE FUNCTION product_versions_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'product_versions_immutable' USING ERRCODE = '23514'; END $$;
-- +goose StatementEnd
CREATE TRIGGER product_versions_immutable BEFORE UPDATE ON product_versions FOR EACH ROW EXECUTE FUNCTION product_versions_immutable();

-- +goose Down
DROP TABLE product_versions, product_cross_refs, product_media, product_sectors, products;
DROP SEQUENCE product_sku_seq;
DROP FUNCTION product_versions_immutable(), products_frozen_identifiers();
