-- +goose Up
-- Statut de validation d'une entreprise : en attente / approuvée / rejetée (motif et date de décision conservés).
ALTER TABLE companies
    ADD COLUMN status           text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    ADD COLUMN rejection_reason text,
    ADD COLUMN decided_at       timestamptz;
UPDATE companies SET status = 'approved', decided_at = created_at WHERE approved;
ALTER TABLE companies
    ADD CONSTRAINT companies_decision CHECK (
        (status = 'pending'  AND decided_at IS NULL AND rejection_reason IS NULL AND NOT approved) OR
        (status = 'approved' AND decided_at IS NOT NULL AND rejection_reason IS NULL AND approved) OR
        (status = 'rejected' AND decided_at IS NOT NULL AND btrim(rejection_reason) <> '' AND NOT approved));
CREATE INDEX companies_status_idx ON companies (status, created_at DESC);

-- +goose Down
DROP INDEX companies_status_idx;
ALTER TABLE companies DROP CONSTRAINT companies_decision, DROP COLUMN status, DROP COLUMN rejection_reason, DROP COLUMN decided_at;
