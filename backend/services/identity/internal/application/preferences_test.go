package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// customerID crée un client et renvoie son identifiant.
func customerID(t *testing.T, e *env, email string) string {
	t.Helper()
	register(t, e, email, "correct-horse-battery")
	u, err := e.store.ByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	return u.ID
}

func testPreferencesDefautsEtValidation(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	got, err := e.svc.GetPreferences(ctx, domain.PrincipalStaff, a.ID)
	if err != nil || !got.Guide.Enabled || got.Guide.Animations != "auto" || got.Guide.Seen == nil || len(got.Guide.Seen) != 0 {
		t.Fatalf("défauts attendus (guide actif, animations auto, aucune visite) : %v %+v", err, got)
	}
	dup := []string{"a", "a"}
	tooMany := make([]string, domain.MaxGuideSeen+1)
	for i := range tooMany {
		tooMany[i] = "t" + strings.Repeat("x", i%5) + string(rune('a'+i%26)) + uuid.NewString()[:6]
	}
	for name, in := range map[string]domain.GuidePreferences{
		"animations inconnues":  {Enabled: true, Animations: "fast"},
		"identifiant en double": {Enabled: true, Animations: "auto", Seen: dup},
		"majuscules interdites": {Enabled: true, Animations: "auto", Seen: []string{"Dashboard"}},
		"identifiant vide":      {Enabled: true, Animations: "auto", Seen: []string{""}},
		"identifiant trop long": {Enabled: true, Animations: "auto", Seen: []string{strings.Repeat("a", 65)}},
		"trop de visites":       {Enabled: true, Animations: "auto", Seen: tooMany},
	} {
		_, err := e.svc.PutPreferences(ctx, domain.PrincipalStaff, a.ID, domain.Preferences{Guide: in})
		if code(err) != apperr.CodeValidation {
			t.Fatalf("%s : validation attendue, obtenu %v", name, err)
		}
	}
	if again, _ := e.svc.GetPreferences(ctx, domain.PrincipalStaff, a.ID); again.Guide.Animations != "auto" || len(again.Guide.Seen) != 0 {
		t.Fatal("des préférences invalides ont été enregistrées")
	}
}

func testPreferencesEnregistreesRemplaceesEtIsolees(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	cust := customerID(t, e, "client@exemple.ci")

	in := domain.Preferences{Guide: domain.GuidePreferences{Enabled: false, Animations: "off", Seen: []string{"dashboard", "content-faq-form"}}}
	if _, err := e.svc.PutPreferences(ctx, domain.PrincipalStaff, a.ID, in); err != nil {
		t.Fatal(err)
	}
	got, _ := e.svc.GetPreferences(ctx, domain.PrincipalStaff, a.ID)
	if got.Guide.Enabled || got.Guide.Animations != "off" || strings.Join(got.Guide.Seen, ",") != "dashboard,content-faq-form" {
		t.Fatalf("préférences relues différentes : %+v", got)
	}

	// Les deux mondes sont séparés : le personnel n'affecte pas les clients, et inversement.
	if c, _ := e.svc.GetPreferences(ctx, domain.PrincipalCustomer, cust); !c.Guide.Enabled || len(c.Guide.Seen) != 0 {
		t.Fatalf("le client doit garder les défauts : %+v", c)
	}
	if _, err := e.svc.PutPreferences(ctx, domain.PrincipalCustomer, cust, domain.Preferences{Guide: domain.GuidePreferences{Enabled: true, Animations: "auto", Seen: []string{"home"}}}); err != nil {
		t.Fatal(err)
	}
	if s, _ := e.svc.GetPreferences(ctx, domain.PrincipalStaff, a.ID); s.Guide.Enabled || len(s.Guide.Seen) != 2 {
		t.Fatalf("les préférences du client ont écrasé celles du personnel : %+v", s)
	}

	// Remplacement complet : une liste vide efface les visites (« tout marquer comme non vu »).
	if _, err := e.svc.PutPreferences(ctx, domain.PrincipalStaff, a.ID, domain.Preferences{Guide: domain.GuidePreferences{Enabled: true, Animations: "auto"}}); err != nil {
		t.Fatal(err)
	}
	if s, _ := e.svc.GetPreferences(ctx, domain.PrincipalStaff, a.ID); !s.Guide.Enabled || s.Guide.Seen == nil || len(s.Guide.Seen) != 0 {
		t.Fatalf("remplacement attendu avec liste vide non nulle : %+v", s)
	}

	// Une valeur renvoyée n'est jamais partagée avec le stockage.
	r1, _ := e.svc.GetPreferences(ctx, domain.PrincipalCustomer, cust)
	r1.Guide.Seen[0] = "modifie"
	if r2, _ := e.svc.GetPreferences(ctx, domain.PrincipalCustomer, cust); r2.Guide.Seen[0] != "home" {
		t.Fatal("la liste renvoyée est partagée avec le stockage")
	}
}

func testPreferencesCompteInconnuOuMondeInconnu(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	ok := domain.DefaultPreferences()
	if _, err := e.svc.PutPreferences(ctx, domain.PrincipalStaff, uuid.NewString(), ok); code(err) != apperr.CodeNotFound {
		t.Fatalf("compte du personnel inconnu : NOT_FOUND attendu, obtenu %v", err)
	}
	if _, err := e.svc.PutPreferences(ctx, domain.PrincipalCustomer, uuid.NewString(), ok); code(err) != apperr.CodeNotFound {
		t.Fatalf("client inconnu : NOT_FOUND attendu, obtenu %v", err)
	}
	// Un identifiant du personnel n'est pas un client : on ne peut pas écrire dans le mauvais monde.
	if _, err := e.svc.PutPreferences(ctx, domain.PrincipalCustomer, a.ID, ok); code(err) != apperr.CodeNotFound {
		t.Fatalf("identifiant du personnel utilisé comme client : NOT_FOUND attendu, obtenu %v", err)
	}
	if got, err := e.svc.GetPreferences(ctx, domain.PrincipalStaff, uuid.NewString()); err != nil || !got.Guide.Enabled {
		t.Fatalf("la lecture d'un compte sans préférences renvoie les défauts : %v", err)
	}
}
