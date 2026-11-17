-- +goose Up
-- Contenu du site : collections (FAQ, pages, blog, zones de livraison, codes promotionnels, avis, messages…) et réglages uniques.
-- Les règles par collection sont validées par le service ; la base garantit l'unicité et la forme.

CREATE TABLE documents (
    id          uuid        PRIMARY KEY,
    kind        text        NOT NULL CHECK (kind ~ '^[a-z_]{2,40}$'),
    uniq        text        CHECK (uniq IS NULL OR (btrim(uniq) <> '' AND length(uniq) <= 254)),   -- slug, code ou e-mail
    sort_order  integer     NOT NULL DEFAULT 0,
    data        jsonb       NOT NULL CHECK (jsonb_typeof(data) = 'object'),
    version     integer     NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX documents_uniq ON documents (kind, uniq) WHERE uniq IS NOT NULL;
CREATE INDEX documents_list ON documents (kind, sort_order, created_at DESC);

CREATE TABLE settings (
    key         text        PRIMARY KEY CHECK (key ~ '^[a-z_]{2,40}$'),
    data        jsonb       NOT NULL CHECK (jsonb_typeof(data) = 'object'),
    version     integer     NOT NULL DEFAULT 1 CHECK (version >= 1),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Informations du site reprises de l'ancienne configuration du site (modifiables ensuite au backoffice).
INSERT INTO settings (key, data) VALUES ('site', '{
  "name": "Holding Ledan & Associe",
  "publisher": "Holding Ledan Et Associé (HLEA)",
  "phone": "+225 07 47 225 087",
  "email": "ledan.associe@gmail.com",
  "address": "RCI ABIDJAN Palmeraie Résidence KAHA, Appartement E3",
  "linkedinUrl": "https://www.linkedin.com/company/holding-ledan-et-associ%C3%A9/",
  "crossReferencesClaim": "+5000",
  "stockValueClaim": "1000",
  "foundedYear": "2011"
}'::jsonb);

-- +goose Down
DROP TABLE settings;
DROP TABLE documents;
