-- +goose Up
-- Image de catégorie, facultative : adresse https ou chemin du site (/img/…).
ALTER TABLE categories ADD COLUMN image_url text CHECK (image_url IS NULL OR (image_url ~ '^(https?://|/)' AND length(image_url) <= 2000));

-- +goose Down
ALTER TABLE categories DROP COLUMN image_url;
