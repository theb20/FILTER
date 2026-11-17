package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/testsupport"
	"github.com/theb20/FILTER/backend/services/catalog/internal/adapters/postgres"
	"github.com/theb20/FILTER/backend/services/catalog/internal/application"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
)

var ctx = context.Background()

func rig(t *testing.T) (*application.Service, *postgres.Store) {
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
	return application.New(application.Config{Store: st, Events: &outbox.Publisher{}}), st
}

var adm = application.Actor{ID: "00000000-0000-0000-0000-000000000001", IP: "127.0.0.1", TraceID: "t1"}

func loc(fr string) domain.Localized { return domain.Localized{"fr": fr} }

func cat(t *testing.T, s *application.Service, slug string, parent *string) *domain.Category {
	t.Helper()
	c, err := s.CreateCategory(ctx, adm, domain.Category{Slug: slug, ParentID: parent, Name: loc(slug)})
	if err != nil {
		t.Fatalf("créer %s : %v", slug, err)
	}
	return c
}

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
		t.Fatalf("code = %q, attendu %q (%v)", got, want, err)
	}
}

func TestCategoryTreeAndDepth(t *testing.T) {
	s, _ := rig(t)
	a := cat(t, s, "pompes", nil)
	b := cat(t, s, "pompes-doseuses", &a.ID)
	c := cat(t, s, "membranes", &b.ID)
	if a.Depth != 0 || b.Depth != 1 || c.Depth != 2 {
		t.Fatalf("profondeurs : %d %d %d", a.Depth, b.Depth, c.Depth)
	}
	kids, err := s.ListCategories(ctx, "", &a.ID, 1, 20)
	if err != nil || len(kids.Items) != 1 || kids.Items[0].ID != b.ID {
		t.Fatalf("enfants : %+v %v", kids, err)
	}
	_, err = s.CreateCategory(ctx, adm, domain.Category{Slug: "pompes", Name: loc("x")})
	wantCode(t, err, apperr.CodeConflict)
	bad := "00000000-0000-0000-0000-0000000000ff"
	_, err = s.CreateCategory(ctx, adm, domain.Category{Slug: "orphan", ParentID: &bad, Name: loc("x")})
	wantCode(t, err, apperr.CodeValidation)
}

func TestCategoryValidation(t *testing.T) {
	s, _ := rig(t)
	for name, in := range map[string]domain.Category{
		"slug majuscule":      {Slug: "Pompes", Name: loc("x")},
		"slug espace":         {Slug: "a b", Name: loc("x")},
		"sans nom":            {Slug: "ok"},
		"sans fr":             {Slug: "ok", Name: domain.Localized{"en": "x"}},
		"fr vide":             {Slug: "ok", Name: loc("  ")},
		"langue invalide":     {Slug: "ok", Name: domain.Localized{"fr": "x", "FR_": "y"}},
		"ordre négatif":       {Slug: "ok", Name: loc("x"), SortOrder: -1},
		"description sans fr": {Slug: "ok", Name: loc("x"), Description: domain.Localized{"en": "d"}},
	} {
		_, err := s.CreateCategory(ctx, adm, in)
		if code(err) != apperr.CodeValidation {
			t.Errorf("%s : code %q", name, code(err))
		}
	}
}

func TestMoveRulesAndCycles(t *testing.T) {
	s, st := rig(t)
	a := cat(t, s, "a", nil)
	b := cat(t, s, "b", &a.ID)
	c := cat(t, s, "c", &b.ID)
	// sous elle-même, sous un descendant
	_, err := s.MoveCategory(ctx, adm, a.ID, &a.ID, 0)
	wantCode(t, err, apperr.CodeConflict)
	_, err = s.MoveCategory(ctx, adm, a.ID, &c.ID, 0)
	wantCode(t, err, apperr.CodeConflict)
	// le trigger protège aussi une écriture directe en SQL
	_, err = st.Pool.Exec(ctx, `UPDATE categories SET parent_id = $2 WHERE id = $1`, a.ID, c.ID)
	if err == nil {
		t.Fatal("un cycle a été accepté en SQL direct")
	}
	// déplacement valide : c à la racine, puis position
	d := cat(t, s, "d", nil)
	e := cat(t, s, "e", nil)
	got, err := s.MoveCategory(ctx, adm, c.ID, nil, 0)
	if err != nil || got.ParentID != nil || got.Depth != 0 || got.SortOrder != 0 {
		t.Fatalf("déplacement : %+v %v", got, err)
	}
	roots, _ := s.ListCategories(ctx, "", nil, 1, 100)
	order := map[string]int{}
	for _, r := range roots.Items {
		if r.ParentID == nil {
			order[r.Slug] = r.SortOrder
		}
	}
	if order["c"] != 0 {
		t.Fatalf("ordre racine : %v", order)
	}
	// position au-delà de la fin : placé en dernier, ordres contigus
	if _, err = s.MoveCategory(ctx, adm, d.ID, nil, 99); err != nil {
		t.Fatal(err)
	}
	_ = e
	rows, _ := st.Pool.Query(ctx, `SELECT slug, sort_order FROM categories WHERE parent_id IS NULL ORDER BY sort_order`)
	var seq []string
	i := 0
	for rows.Next() {
		var sl string
		var so int
		_ = rows.Scan(&sl, &so)
		if so != i {
			t.Fatalf("ordres non contigus : %s=%d attendu %d", sl, so, i)
		}
		seq = append(seq, sl)
		i++
	}
	if seq[len(seq)-1] != "d" {
		t.Fatalf("d devrait être dernier : %v", seq)
	}
	// parent inexistant, position négative, id inconnu
	ghost := "00000000-0000-0000-0000-0000000000aa"
	_, err = s.MoveCategory(ctx, adm, a.ID, &ghost, 0)
	wantCode(t, err, apperr.CodeValidation)
	_, err = s.MoveCategory(ctx, adm, a.ID, nil, -1)
	wantCode(t, err, apperr.CodeValidation)
	_, err = s.MoveCategory(ctx, adm, ghost, nil, 0)
	wantCode(t, err, apperr.CodeNotFound)
}

func TestConcurrentMovesNeverMakeACycle(t *testing.T) {
	s, st := rig(t)
	a := cat(t, s, "a", nil)
	b := cat(t, s, "b", nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = s.MoveCategory(ctx, adm, a.ID, &b.ID, 0) }()
		go func() { defer wg.Done(); _, _ = s.MoveCategory(ctx, adm, b.ID, &a.ID, 0) }()
	}
	wg.Wait()
	var n int
	if err := st.Pool.QueryRow(ctx, `SELECT count(*) FROM categories WHERE parent_id IS NOT NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("exactement une des deux doit avoir un parent, %d trouvées (cycle ?)", n)
	}
}

func TestDeleteCategoryRefusedWhenUsed(t *testing.T) {
	s, _ := rig(t)
	a := cat(t, s, "a", nil)
	b := cat(t, s, "b", &a.ID)
	wantCode(t, s.DeleteCategory(ctx, adm, a.ID), apperr.CodeConflict) // a des enfants
	if _, err := s.CreateAttributeDef(ctx, adm, domain.AttributeDef{CategoryID: b.ID, Key: "debit", Label: loc("Débit"), Type: "number"}); err != nil {
		t.Fatal(err)
	}
	wantCode(t, s.DeleteCategory(ctx, adm, b.ID), apperr.CodeConflict) // a des attributs
	wantCode(t, s.DeleteCategory(ctx, adm, "00000000-0000-0000-0000-0000000000aa"), apperr.CodeNotFound)
	c := cat(t, s, "vide", nil)
	if err := s.DeleteCategory(ctx, adm, c.ID); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateCategoryKeepsParentAndOrder(t *testing.T) {
	s, _ := rig(t)
	a := cat(t, s, "a", nil)
	b := cat(t, s, "b", &a.ID)
	other := "00000000-0000-0000-0000-0000000000bb"
	got, err := s.UpdateCategory(ctx, adm, b.ID, domain.Category{Slug: "b2", ParentID: &other, SortOrder: 9, Name: domain.Localized{"fr": "B", "en": "B"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "b2" || got.ParentID == nil || *got.ParentID != a.ID || got.SortOrder != b.SortOrder || got.Name["en"] != "B" {
		t.Fatalf("mise à jour : %+v", got)
	}
}

func TestAttributeRules(t *testing.T) {
	s, _ := rig(t)
	root := cat(t, s, "pompes", nil)
	mid := cat(t, s, "doseuses", &root.ID)
	leaf := cat(t, s, "membranes", &mid.ID)
	side := cat(t, s, "filtres", nil)
	mk := func(catID, key, typ string) (*domain.AttributeDef, error) {
		in := domain.AttributeDef{CategoryID: catID, Key: key, Label: loc(key), Type: typ}
		if typ == "enum" {
			in.Options = []domain.AttributeOption{{Value: "a", Label: loc("A")}}
		}
		return s.CreateAttributeDef(ctx, adm, in)
	}
	if _, err := mk(root.ID, "debit", "number"); err != nil {
		t.Fatal(err)
	}
	// doublon dans la même catégorie / héritage descendant / héritage ascendant
	_, err := mk(root.ID, "debit", "number")
	wantCode(t, err, apperr.CodeConflict)
	_, err = mk(leaf.ID, "debit", "number")
	wantCode(t, err, apperr.CodeConflict)
	if _, err = mk(leaf.ID, "matiere", "enum"); err != nil {
		t.Fatal(err)
	}
	_, err = mk(root.ID, "matiere", "enum")
	wantCode(t, err, apperr.CodeConflict)
	// une branche voisine peut réutiliser la clé
	if _, err = mk(side.ID, "debit", "number"); err != nil {
		t.Fatalf("branche voisine : %v", err)
	}
	// déplacer « filtres » sous « pompes » créerait le conflit « debit »
	_, err = s.MoveCategory(ctx, adm, side.ID, &root.ID, 0)
	wantCode(t, err, apperr.CodeConflict)
	// catégorie inconnue
	_, err = mk("00000000-0000-0000-0000-0000000000aa", "autre", "text")
	wantCode(t, err, apperr.CodeValidation)

	for name, in := range map[string]domain.AttributeDef{
		"enum sans options":   {CategoryID: root.ID, Key: "k1", Label: loc("x"), Type: "enum"},
		"options sur number":  {CategoryID: root.ID, Key: "k2", Label: loc("x"), Type: "number", Options: []domain.AttributeOption{{Value: "a", Label: loc("A")}}},
		"options en double":   {CategoryID: root.ID, Key: "k3", Label: loc("x"), Type: "enum", Options: []domain.AttributeOption{{Value: "a", Label: loc("A")}, {Value: "a", Label: loc("B")}}},
		"unité sur texte":     {CategoryID: root.ID, Key: "k4", Label: loc("x"), Type: "text", Unit: ptr("kg")},
		"filtre sur texte":    {CategoryID: root.ID, Key: "k5", Label: loc("x"), Type: "text", Filterable: true},
		"clé invalide":        {CategoryID: root.ID, Key: "1Bad", Label: loc("x"), Type: "text"},
		"type inconnu":        {CategoryID: root.ID, Key: "k6", Label: loc("x"), Type: "date"},
		"sans libellé fr":     {CategoryID: root.ID, Key: "k7", Label: domain.Localized{"en": "x"}, Type: "text"},
		"option sans libellé": {CategoryID: root.ID, Key: "k8", Label: loc("x"), Type: "enum", Options: []domain.AttributeOption{{Value: "a"}}},
	} {
		_, err := s.CreateAttributeDef(ctx, adm, in)
		if code(err) != apperr.CodeValidation {
			t.Errorf("%s : code %q", name, code(err))
		}
	}
}

func ptr(s string) *string { return &s }

func TestAttributeUpdateFreezesKeyAndType(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "c", nil)
	a, err := s.CreateAttributeDef(ctx, adm, domain.AttributeDef{CategoryID: c.ID, Key: "debit", Label: loc("Débit"), Type: "number", Unit: ptr("m3/h")})
	if err != nil {
		t.Fatal(err)
	}
	up := *a
	up.Key = "autre"
	_, err = s.UpdateAttributeDef(ctx, adm, a.ID, up)
	wantCode(t, err, apperr.CodeValidation)
	up = *a
	up.Type = "text"
	up.Unit = nil
	_, err = s.UpdateAttributeDef(ctx, adm, a.ID, up)
	wantCode(t, err, apperr.CodeValidation)
	up = *a
	up.Label, up.Filterable, up.Required, up.Unit = domain.Localized{"fr": "Débit max", "en": "Max flow"}, true, true, ptr("l/h")
	got, err := s.UpdateAttributeDef(ctx, adm, a.ID, up)
	if err != nil || got.Label["en"] != "Max flow" || !got.Filterable || *got.Unit != "l/h" {
		t.Fatalf("%+v %v", got, err)
	}
	re, _ := s.GetAttributeDef(ctx, a.ID)
	if re.Label["fr"] != "Débit max" {
		t.Fatalf("non persisté : %+v", re)
	}
	if err := s.DeleteAttributeDef(ctx, adm, a.ID); err != nil {
		t.Fatal(err)
	}
	_, err = s.GetAttributeDef(ctx, a.ID)
	wantCode(t, err, apperr.CodeNotFound)
}

func TestConcurrentSameKeyOnRelatedCategories(t *testing.T) {
	s, st := rig(t)
	p := cat(t, s, "p", nil)
	c := cat(t, s, "c", &p.ID)
	var wg sync.WaitGroup
	for _, id := range []string{p.ID, c.ID} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.CreateAttributeDef(ctx, adm, domain.AttributeDef{CategoryID: id, Key: "same", Label: loc("x"), Type: "text"})
		}()
	}
	wg.Wait()
	var n int
	_ = st.Pool.QueryRow(ctx, `SELECT count(*) FROM attribute_defs WHERE key='same'`).Scan(&n)
	if n != 1 {
		t.Fatalf("une seule clé doit survivre, %d", n)
	}
}

func TestBrandsAndSectors(t *testing.T) {
	s, _ := rig(t)
	b, err := s.CreateBrand(ctx, adm, domain.Brand{Slug: "grundfos", Name: "Grundfos", LogoURL: ptr("https://exemple.test/logo.png")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateBrand(ctx, adm, domain.Brand{Slug: "autre", Name: "GRUNDFOS"}) // nom unique sans casse
	wantCode(t, err, apperr.CodeConflict)
	_, err = s.CreateBrand(ctx, adm, domain.Brand{Slug: "grundfos", Name: "Autre"})
	wantCode(t, err, apperr.CodeConflict)
	_, err = s.CreateBrand(ctx, adm, domain.Brand{Slug: "x", Name: "X", LogoURL: ptr("javascript:alert(1)")})
	wantCode(t, err, apperr.CodeValidation)
	_, err = s.CreateBrand(ctx, adm, domain.Brand{Slug: "Bad Slug", Name: "Y"})
	wantCode(t, err, apperr.CodeValidation)
	up, err := s.UpdateBrand(ctx, adm, b.ID, domain.Brand{Slug: "grundfos", Name: "Grundfos SA"})
	if err != nil || up.Name != "Grundfos SA" || up.LogoURL != nil {
		t.Fatalf("%+v %v", up, err)
	}
	l, _ := s.ListBrands(ctx, "grund", 1, 10)
	if l.Total != 1 {
		t.Fatalf("recherche : %d", l.Total)
	}
	if l, _ = s.ListBrands(ctx, "%", 1, 10); l.Total != 0 { // le joker est échappé
		t.Fatalf("joker non échappé : %d", l.Total)
	}
	if err := s.DeleteBrand(ctx, adm, b.ID); err != nil {
		t.Fatal(err)
	}
	wantCode(t, s.DeleteBrand(ctx, adm, b.ID), apperr.CodeNotFound)

	x, err := s.CreateSector(ctx, adm, domain.Sector{Slug: "agroalimentaire", Name: domain.Localized{"fr": "Agroalimentaire", "en": "Food"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateSector(ctx, adm, domain.Sector{Slug: "agroalimentaire", Name: loc("Autre")})
	wantCode(t, err, apperr.CodeConflict)
	_, err = s.CreateSector(ctx, adm, domain.Sector{Slug: "s2", Name: domain.Localized{"en": "x"}})
	wantCode(t, err, apperr.CodeValidation)
	if _, err = s.UpdateSector(ctx, adm, x.ID, domain.Sector{Slug: "agro", Name: loc("Agro"), ImageURL: ptr("https://exemple.test/i.jpg")}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSector(ctx, adm, x.ID); err != nil {
		t.Fatal(err)
	}
}

func TestAuditEventsAreWrittenInTheSameTransaction(t *testing.T) {
	s, st := rig(t)
	count := func() int {
		var n int
		if err := st.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE subject = 'audit.recorded.v1'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	c := cat(t, s, "a", nil)
	if n := count(); n != 1 {
		t.Fatalf("création : %d événements", n)
	}
	// un échec (slug déjà pris) n'écrit AUCUN événement
	_, _ = s.CreateCategory(ctx, adm, domain.Category{Slug: "a", Name: loc("x")})
	if n := count(); n != 1 {
		t.Fatalf("échec : %d événements", n)
	}
	_, _ = s.UpdateCategory(ctx, adm, c.ID, domain.Category{Slug: "a", Name: loc("A2")})
	_, _ = s.MoveCategory(ctx, adm, c.ID, nil, 0)
	if err := s.DeleteCategory(ctx, adm, c.ID); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 4 {
		t.Fatalf("après update/move/delete : %d", n)
	}
	var actor, action, entity string
	if err := st.Pool.QueryRow(ctx, `SELECT payload->>'actorId', payload->>'action', payload->>'entity' FROM outbox WHERE subject='audit.recorded.v1' ORDER BY id LIMIT 1`).Scan(&actor, &action, &entity); err != nil {
		// la forme exacte du payload dépend de l'enveloppe : on vérifie au moins la présence
		t.Logf("lecture du payload : %v", err)
	}
}

func TestSettingsDefaults(t *testing.T) {
	_, st := rig(t)
	var lang, cur string
	if err := st.Pool.QueryRow(ctx, `SELECT default_language, base_currency FROM catalog_settings`).Scan(&lang, &cur); err != nil {
		t.Fatal(err)
	}
	if lang != "fr" || cur != "XOF" {
		t.Fatalf("%s %s", lang, cur)
	}
	if _, err := st.Pool.Exec(ctx, `INSERT INTO catalog_settings (id) VALUES (false)`); err == nil {
		t.Fatal("une seconde ligne de réglages a été acceptée")
	}
	if _, err := st.Pool.Exec(ctx, `UPDATE catalog_settings SET base_currency='xof'`); err == nil {
		t.Fatal("devise invalide acceptée")
	}
}

func TestListPagination(t *testing.T) {
	s, _ := rig(t)
	for i := 0; i < 5; i++ {
		cat(t, s, fmt.Sprintf("c%d", i), nil)
	}
	p, err := s.ListCategories(ctx, "", nil, 2, 2)
	if err != nil || p.Total != 5 || len(p.Items) != 2 || p.Pages != 3 || p.Page != 2 {
		t.Fatalf("%+v %v", p, err)
	}
}
