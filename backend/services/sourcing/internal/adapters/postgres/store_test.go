package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/postgres"
)

func newStore(t *testing.T) *postgres.Store {
	t.Helper()
	url := testsupport.Postgres(t)
	if err := postgres.Migrate(context.Background(), url); err != nil {
		t.Fatal(err)
	}
	s, err := postgres.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func state(t *testing.T, s *postgres.Store, q string, args ...any) string {
	t.Helper()
	_, err := s.Pool.Exec(context.Background(), q, args...)
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	return ""
}

const ins = `INSERT INTO sources (id, kind, name, base_url, priority, timeout_ms, max_retries, secret_ref, cost_per_call_micros, currency) VALUES ($1,$2,$3,'https://x.test',0,1000,2,'K',5000,$4)`

func newSrc(t *testing.T, s *postgres.Store, name string) string {
	t.Helper()
	id := uuid.NewString()
	if c := state(t, s, ins, id, "web_search_api", name, "USD"); c != "" {
		t.Fatalf("insertion : %s", c)
	}
	return id
}

// Les invariants du service sont garantis PAR LA BASE, même par un accès SQL direct.
func TestContraintesEnBase(t *testing.T) {
	s := newStore(t)
	a := newSrc(t, s, "A")
	const check, unique, fk = "23514", "23505", "23503"
	for name, c := range map[string]struct{ want, got string }{
		"nom en double (insensible à la casse)":  {unique, state(t, s, ins, uuid.NewString(), "web_search_api", "a", "USD")},
		"type inconnu":                           {check, state(t, s, ins, uuid.NewString(), "magie", "B", "USD")},
		"devise en minuscules":                   {check, state(t, s, ins, uuid.NewString(), "web_search_api", "C", "usd")},
		"coût négatif":                           {check, state(t, s, `UPDATE sources SET cost_per_call_micros = -1 WHERE id = $1`, a)},
		"délai nul":                              {check, state(t, s, `UPDATE sources SET timeout_ms = 0 WHERE id = $1`, a)},
		"cache nul":                              {check, state(t, s, `UPDATE sources SET max_cache_ttl_seconds = 0 WHERE id = $1`, a)},
		"secret vide":                            {check, state(t, s, `UPDATE sources SET secret_ref = '  ' WHERE id = $1`, a)},
		"stockage brut sans vérification datée":  {check, state(t, s, `UPDATE sources SET allows_raw_storage = true WHERE id = $1`, a)},
		"plafond journalier > budget mensuel":    {check, state(t, s, `INSERT INTO source_quotas VALUES ($1, 100, 200, 80, NULL)`, a)},
		"seuil d'alerte à 0":                     {check, state(t, s, `INSERT INTO source_quotas VALUES ($1, 100, 10, 0, NULL)`, a)},
		"secours = soi-même":                     {check, state(t, s, `INSERT INTO source_quotas VALUES ($1, 100, 10, 80, $1)`, a)},
		"secours inexistant":                     {fk, state(t, s, `INSERT INTO source_quotas VALUES ($1, 100, 10, 80, $2)`, a, uuid.NewString())},
		"réservation sans source":                {fk, state(t, s, `INSERT INTO source_reservations (id, source_id, search_id, amount_micros, day, month, created_at, expires_at) VALUES ($1,$2,$3,1,now(),now(),now(),now()+interval '1 minute')`, uuid.NewString(), uuid.NewString(), uuid.NewString())},
		"réservation de montant nul":             {check, state(t, s, `INSERT INTO source_reservations (id, source_id, search_id, amount_micros, day, month, created_at, expires_at) VALUES ($1,$2,$3,0,now(),now(),now(),now()+interval '1 minute')`, uuid.NewString(), a, uuid.NewString())},
		"réservation expirant avant sa création": {check, state(t, s, `INSERT INTO source_reservations (id, source_id, search_id, amount_micros, day, month, created_at, expires_at) VALUES ($1,$2,$3,5,now(),now(),now(),now()-interval '1 minute')`, uuid.NewString(), a, uuid.NewString())},
		"compteur réservé négatif":               {check, state(t, s, `INSERT INTO source_usage (source_id, period_kind, period_start, reserved_micros) VALUES ($1,'day',now(),-1)`, a)},
		"période inconnue":                       {check, state(t, s, `INSERT INTO source_usage (source_id, period_kind, period_start) VALUES ($1,'semaine',now())`, a)},
	} {
		if c.got != c.want {
			t.Errorf("%s : SQLSTATE %q, attendu %q", name, c.got, c.want)
		}
	}
}

func TestBasculeUnSeulSautEnBase(t *testing.T) {
	s := newStore(t)
	a, b, c := newSrc(t, s, "A"), newSrc(t, s, "B"), newSrc(t, s, "C")
	const check = "23514"
	q := `INSERT INTO source_quotas (source_id, monthly_budget_micros, daily_cap_micros, alert_threshold_pct, fallback_source_id) VALUES ($1, 100, 10, 80, $2)
		ON CONFLICT (source_id) DO UPDATE SET fallback_source_id = EXCLUDED.fallback_source_id`
	if got := state(t, s, q, a, b); got != "" {
		t.Fatalf("A→B : %q", got)
	}
	if got := state(t, s, q, b, c); got != check {
		t.Errorf("B→C alors que B est déjà un secours : %q", got)
	}
	if got := state(t, s, q, b, a); got != check {
		t.Errorf("B→A (boucle) : %q", got)
	}
	if got := state(t, s, q, c, a); got != check {
		t.Errorf("C→A alors que A a un secours : %q", got)
	}
	if got := state(t, s, q, c, b); got != "" {
		t.Errorf("deux sources peuvent partager un secours : %q", got)
	}
	if got := state(t, s, `UPDATE source_quotas SET fallback_source_id = NULL WHERE source_id = $1`, a); got != "" {
		t.Errorf("retirer un secours : %q", got)
	}
}

func TestUneSourceUneFoisParRechercheEnBase(t *testing.T) {
	s := newStore(t)
	a, b := newSrc(t, s, "A"), newSrc(t, s, "B")
	search := uuid.NewString()
	r := `INSERT INTO source_reservations (id, source_id, search_id, amount_micros, day, month, created_at, expires_at) VALUES ($1,$2,$3,5,now(),now(),now(),now()+interval '1 minute')`
	if got := state(t, s, r, uuid.NewString(), a, search); got != "" {
		t.Fatal(got)
	}
	if got := state(t, s, r, uuid.NewString(), a, search); got != "23505" {
		t.Errorf("même source, même recherche : %q", got)
	}
	if got := state(t, s, r, uuid.NewString(), b, search); got != "" {
		t.Errorf("une autre source pour la même recherche est permise : %q", got)
	}
}
