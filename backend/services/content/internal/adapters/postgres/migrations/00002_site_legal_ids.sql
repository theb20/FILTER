-- +goose Up
-- Mentions d'identification de l'entreprise imprimées sur les devis (reprises de votre modèle de devis ; modifiables au backoffice).
UPDATE settings
   SET data = data || '{"rccm": "CI-ABJ-03-2019-B12-14381", "taxNumber": "1938849 A", "capital": "5 000 000 FCFA/XOF"}'::jsonb
 WHERE key = 'site' AND NOT (data ? 'rccm');

-- +goose Down
UPDATE settings SET data = data - 'rccm' - 'taxNumber' - 'capital' WHERE key = 'site';
