package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/content/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/content/internal/application"
)

var ctx = context.Background()

func rig(t *testing.T) *application.Service {
	t.Helper()
	url := testsupport.Postgres(t)
	if err := postgres.Migrate(ctx, url); err != nil {
		t.Fatal(err)
	}
	st, err := postgres.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return application.New(application.Config{Store: st, Events: &outbox.Publisher{}})
}

var adm = application.Actor{ID: "00000000-0000-0000-0000-000000000001", IP: "127.0.0.1", TraceID: "t1"}

func code(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	if err == nil {
		return ""
	}
	return "OTHER:" + err.Error()
}

func wantCode(t *testing.T, err error, want string) {
	t.Helper()
	if got := code(err); got != want {
		t.Fatalf("code %q, attendu %q (%v)", got, want, err)
	}
}

type m = map[string]any

func TestValidationDesChamps(t *testing.T) {
	s := rig(t)
	// langue par défaut obligatoire
	_, err := s.Create(ctx, adm, "faq", m{"question": m{"en": "Q"}, "answer": m{"fr": "R"}})
	wantCode(t, err, apperr.CodeValidation)
	// champ inconnu
	_, err = s.Create(ctx, adm, "faq", m{"question": m{"fr": "Q"}, "answer": m{"fr": "R"}, "intrus": 1})
	wantCode(t, err, apperr.CodeValidation)
	// type incorrect
	_, err = s.Create(ctx, adm, "faq", m{"question": m{"fr": "Q"}, "answer": m{"fr": "R"}, "sortOrder": "3"})
	wantCode(t, err, apperr.CodeValidation)
	// collection inconnue
	_, err = s.Create(ctx, adm, "inconnue", m{})
	wantCode(t, err, apperr.CodeNotFound)
	// page : slug mal formé
	_, err = s.Create(ctx, adm, "page", m{"slug": "Mal Formé", "kind": "legal", "title": m{"fr": "T"}, "body": m{"fr": "B"}})
	wantCode(t, err, apperr.CodeValidation)
	// règles croisées
	_, err = s.Create(ctx, adm, "promo_code", m{"code": "PROMO", "type": "percent", "value": 150.0})
	wantCode(t, err, apperr.CodeValidation)
	_, err = s.Create(ctx, adm, "shipping_zone", m{"code": "abj", "name": m{"fr": "Abidjan"}, "fee": 5000.0, "daysMin": 5.0, "daysMax": 2.0})
	wantCode(t, err, apperr.CodeValidation)
	_, err = s.Create(ctx, adm, "newsletter_subscriber", m{"email": "pas-un-email", "status": "subscribed"})
	wantCode(t, err, apperr.CodeValidation)
}

func TestCrudUniciteEtRecherche(t *testing.T) {
	s := rig(t)
	d, err := s.Create(ctx, adm, "page", m{"slug": "cgv", "kind": "legal", "title": m{"fr": "CGV"}, "body": m{"fr": "Texte"}, "published": true})
	if err != nil {
		t.Fatal(err)
	}
	if d.Version != 1 || d.Data["slug"] != "cgv" {
		t.Fatalf("document inattendu : %+v", d)
	}
	_, err = s.Create(ctx, adm, "page", m{"slug": "cgv", "kind": "legal", "title": m{"fr": "Autre"}, "body": m{"fr": "x"}})
	wantCode(t, err, apperr.CodeConflict)

	// lecture par clé unique
	byKey, err := s.Get(ctx, "page", "cgv")
	if err != nil || byKey.ID != d.ID {
		t.Fatalf("lecture par slug : %v %+v", err, byKey)
	}
	// remplacement complet : un champ facultatif omis disparaît
	up, err := s.Update(ctx, adm, "page", d.ID, m{"slug": "cgv", "kind": "legal", "title": m{"fr": "CGV v2"}, "body": m{"fr": "Texte"}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, has := up.Data["published"]; has || up.Version != 2 {
		t.Fatalf("remplacement : %+v", up)
	}
	// modification partielle : le reste est conservé
	up, err = s.Update(ctx, adm, "page", d.ID, m{"published": true}, false)
	if err != nil || up.Data["title"].(map[string]any)["fr"] != "CGV v2" || up.Data["published"] != true {
		t.Fatalf("partielle : %v %+v", err, up)
	}
	// filtre et recherche
	l, err := s.List(ctx, "page", "", map[string]string{"published": "true"}, 1, 20)
	if err != nil || l.Total != 1 {
		t.Fatalf("filtre : %v %+v", err, l)
	}
	l, _ = s.List(ctx, "page", "introuvable", nil, 1, 20)
	if l.Total != 0 {
		t.Fatalf("recherche : %d", l.Total)
	}
	_, err = s.List(ctx, "page", "", map[string]string{"inconnu": "x"}, 1, 20)
	wantCode(t, err, apperr.CodeValidation)

	if err := s.Delete(ctx, adm, "page", d.ID); err != nil {
		t.Fatal(err)
	}
	_, err = s.Get(ctx, "page", d.ID)
	wantCode(t, err, apperr.CodeNotFound)
}

func TestCodePromoEnMajusculesEtOrdre(t *testing.T) {
	s := rig(t)
	p, err := s.Create(ctx, adm, "promo_code", m{"code": "bienvenue10", "type": "percent", "value": 10.0, "active": true})
	if err != nil || p.Data["code"] != "BIENVENUE10" {
		t.Fatalf("code : %v %+v", err, p)
	}
	if _, err := s.Get(ctx, "promo_code", "bienvenue10"); err != nil { // recherche insensible à la casse
		t.Fatal(err)
	}
	_, err = s.Create(ctx, adm, "promo_code", m{"code": "BIENVENUE10", "type": "fixed", "value": 500.0})
	wantCode(t, err, apperr.CodeConflict)
	for i, n := range []float64{20, 10} {
		if _, err := s.Create(ctx, adm, "faq", m{"question": m{"fr": "Q"}, "answer": m{"fr": "R"}, "sortOrder": n, "published": i >= 0}); err != nil {
			t.Fatal(err)
		}
	}
	l, _ := s.List(ctx, "faq", "", nil, 1, 20)
	if l.Items[0].Data["sortOrder"].(float64) != 10 {
		t.Fatalf("tri par ordre d'affichage : %+v", l.Items[0].Data)
	}
}

func TestReglages(t *testing.T) {
	s := rig(t)
	site, err := s.GetSettings(ctx, "site") // repris de l'ancienne configuration du site
	if err != nil || site.Data["name"] == nil || site.Version != 1 {
		t.Fatalf("réglages initiaux : %v %+v", err, site)
	}
	c, err := s.GetSettings(ctx, "commerce") // jamais enregistrés : vides
	if err != nil || len(c.Data) != 0 || c.Version != 0 {
		t.Fatalf("commerce vide : %v %+v", err, c)
	}
	if _, err := s.PutSettings(ctx, adm, "commerce", m{"vatRate": 18.0, "freeShippingThreshold": 500000.0}); err != nil {
		t.Fatal(err)
	}
	c, _ = s.GetSettings(ctx, "commerce")
	if c.Data["vatRate"] != 18.0 || c.Version != 1 {
		t.Fatalf("commerce : %+v", c)
	}
	_, err = s.PutSettings(ctx, adm, "commerce", m{"vatRate": 150.0})
	wantCode(t, err, apperr.CodeValidation)
	_, err = s.GetSettings(ctx, "autre")
	wantCode(t, err, apperr.CodeNotFound)
}

func TestDevisEtBrouillon(t *testing.T) {
	s := rig(t)
	owner := "11111111-1111-1111-1111-111111111111"
	d, err := s.Create(ctx, adm, "quote_draft", m{"userId": owner, "subject": "Cuves GPL", "deliveryDelay": "3 semaines", "items": []any{m{"reference": "HF-100", "quantity": 2.0}}})
	if err != nil {
		t.Fatal(err)
	}
	// un seul brouillon par compte (clé unique), retrouvé par l'identifiant du compte
	if _, err := s.Create(ctx, adm, "quote_draft", m{"userId": owner}); code(err) != apperr.CodeConflict {
		t.Fatalf("second brouillon du même compte : %v", code(err))
	}
	got, err := s.Get(ctx, "quote_draft", owner)
	if err != nil || got.ID != d.ID {
		t.Fatalf("lecture par compte : %v", err)
	}
	// la demande de devis reprend les éléments d'un devis
	q, err := s.Create(ctx, adm, "quote_request", m{"name": "A", "email": "a@b.co", "status": "new", "subject": "Cuves GPL", "reference": "ENGPLA030267", "attention": "Mme K.",
		"deliveryPlace": "Abidjan", "deliveryDelay": "3 semaines", "paymentTerms": "30 jours fin de mois", "items": []any{m{"reference": "VRN20 Vanne", "quantity": 126.0}}})
	if err != nil || q.Data["reference"] != "ENGPLA030267" {
		t.Fatalf("demande : %v", err)
	}
	if _, err := s.Create(ctx, adm, "quote_request", m{"name": "A", "email": "a@b.co", "status": "new", "items": []any{m{"quantity": 1.0}}}); code(err) != apperr.CodeValidation {
		t.Fatalf("ligne sans référence : %v", code(err))
	}
	site, _ := s.GetSettings(ctx, "site")
	if site.Data["rccm"] == nil || site.Data["taxNumber"] == nil || site.Data["capital"] == nil {
		t.Fatalf("identifiants légaux du devis : %v", site.Data)
	}
}
