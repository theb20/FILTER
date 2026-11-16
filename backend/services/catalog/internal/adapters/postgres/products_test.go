package postgres_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/catalog/internal/application"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
)

func i(v int) *int           { return &v }
func i64(v int64) *int64     { return &v }
func f64(v float64) *float64 { return &v }
func str(v string) *string   { return &v }

func base(catID string) domain.Product {
	return domain.Product{Name: loc("Pompe doseuse à membrane"), CategoryID: catID, FulfillmentMode: domain.FulfillStock, PriceMode: domain.PriceFixed, PriceAmount: i64(125000)}
}

func mk(t *testing.T, s *application.Service, in domain.Product) *domain.Product {
	t.Helper()
	p, err := s.CreateProduct(ctx, adm, in, "")
	if err != nil {
		t.Fatalf("création : %v", err)
	}
	return p
}

func fieldPointers(err error) []string {
	var ae *apperr.Error
	var out []string
	if e, ok := err.(*apperr.Error); ok {
		ae = e
	}
	if ae != nil {
		for _, f := range ae.Fields {
			out = append(out, f.Pointer)
		}
	}
	return out
}

func TestProductCreateDefaults(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "pompes", nil)
	p := mk(t, s, base(c.ID))
	if p.SKU != "FF-000001" || p.Slug != "pompe-doseuse-a-membrane" || p.Status != "draft" || p.Version != 1 || p.Currency != "XOF" {
		t.Fatalf("valeurs par défaut : %+v", p)
	}
	q := mk(t, s, base(c.ID))
	if q.SKU != "FF-000002" || q.Slug != "pompe-doseuse-a-membrane-2" {
		t.Fatalf("second produit : %s %s", q.SKU, q.Slug)
	}
	if p.Origins["name"] != "manual" || p.Origins["priceAmount"] != "manual" {
		t.Fatalf("origines : %v", p.Origins)
	}
	vs, err := s.ProductVersions(ctx, p.ID)
	if err != nil || len(vs) != 1 || vs[0].Action != "create" || vs[0].Version != 1 {
		t.Fatalf("versions : %+v %v", vs, err)
	}
	cc, _ := s.GetCategory(ctx, c.ID)
	if cc.ProductCount != 2 {
		t.Fatalf("compteur de la catégorie : %d", cc.ProductCount)
	}
}

func TestProductShapeValidation(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "pompes", nil)
	cases := map[string]func(*domain.Product){
		"SKU invalide":             func(p *domain.Product) { p.SKU = "pas valide!" },
		"à la commande sans délai": func(p *domain.Product) { p.FulfillmentMode = domain.FulfillMadeToOrder },
		"sur devis avec prix":      func(p *domain.Product) { p.PriceMode = domain.PriceOnQuote },
		"prix négatif":             func(p *domain.Product) { p.PriceAmount = i64(-1) },
		"mode de prix inconnu":     func(p *domain.Product) { p.PriceMode = "gratuit" },
		"GTIN de contrôle faux":    func(p *domain.Product) { p.GTIN = str("4006381333932") },
		"GTIN trop court":          func(p *domain.Product) { p.GTIN = str("123") },
		"quantité minimale 0":      func(p *domain.Product) { p.MinOrderQuantity = i(0) },
		"poids négatif":            func(p *domain.Product) { p.WeightGrams = i(-5) },
		"sans nom français":        func(p *domain.Product) { p.Name = domain.Localized{"en": "x"} },
		"sans catégorie":           func(p *domain.Product) { p.CategoryID = "" },
		"média non http":           func(p *domain.Product) { p.Media = []domain.Media{{URL: "javascript:alert(1)"}} },
		"référence croisée en double": func(p *domain.Product) {
			p.CrossRefs = []domain.CrossRef{{Brand: "SKF", Number: "6204-2RS", Kind: "oem"}, {Brand: "skf", Number: "6204 2RS", Kind: "oem"}}
		},
		"type de référence croisée": func(p *domain.Product) { p.CrossRefs = []domain.CrossRef{{Brand: "SKF", Number: "1", Kind: "clone"}} },
		"secteur en double":         func(p *domain.Product) { p.SectorIDs = []string{"a", "a"} },
		"attribut à deux valeurs": func(p *domain.Product) {
			p.Attributes = []domain.AttrValue{{Key: "debit", Text: str("x"), Bool: new(bool)}}
		},
		"attribut en double": func(p *domain.Product) {
			p.Attributes = []domain.AttrValue{{Key: "debit", Text: str("x")}, {Key: "debit", Text: str("y")}}
		},
	}
	for name, mutate := range cases {
		in := base(c.ID)
		mutate(&in)
		if _, err := s.CreateProduct(ctx, adm, in, ""); code(err) != apperr.CodeValidation {
			t.Errorf("%s : code %q (%v)", name, code(err), err)
		}
	}
	in := base(c.ID)
	in.GTIN = str("4006381333931") // EAN-13 valide
	in.SKU = "ab-12"               // mis en majuscules
	p, err := s.CreateProduct(ctx, adm, in, "")
	if err != nil || p.SKU != "AB-12" || *p.GTIN != "4006381333931" {
		t.Fatalf("%+v %v", p, err)
	}
	_, err = s.CreateProduct(ctx, adm, func() domain.Product { x := base(c.ID); x.SKU = "AB-12"; return x }(), "")
	wantCode(t, err, apperr.CodeConflict)
	bad := base("00000000-0000-0000-0000-0000000000aa")
	_, err = s.CreateProduct(ctx, adm, bad, "")
	wantCode(t, err, apperr.CodeValidation)
	bad = base(c.ID)
	bad.BrandID = str("00000000-0000-0000-0000-0000000000bb")
	_, err = s.CreateProduct(ctx, adm, bad, "")
	wantCode(t, err, apperr.CodeValidation)
}

func TestProductAttributesAgainstDefinitions(t *testing.T) {
	s, _ := rig(t)
	root := cat(t, s, "pompes", nil)
	leaf := cat(t, s, "doseuses", &root.ID)
	mustAttr := func(catID string, a domain.AttributeDef) {
		a.CategoryID = catID
		if _, err := s.CreateAttributeDef(ctx, adm, a); err != nil {
			t.Fatal(err)
		}
	}
	mustAttr(root.ID, domain.AttributeDef{Key: "debit", Label: loc("Débit"), Type: "number", Required: true})
	mustAttr(leaf.ID, domain.AttributeDef{Key: "matiere", Label: loc("Matière"), Type: "enum", Options: []domain.AttributeOption{{Value: "inox", Label: loc("Inox")}}})
	mustAttr(leaf.ID, domain.AttributeDef{Key: "auto", Label: loc("Auto"), Type: "boolean"})

	in := base(leaf.ID)
	in.Attributes = []domain.AttrValue{{Key: "inconnu", Text: str("x")}}
	_, err := s.CreateProduct(ctx, adm, in, "")
	wantCode(t, err, apperr.CodeValidation)
	in.Attributes = []domain.AttrValue{{Key: "debit", Text: str("beaucoup")}}
	_, err = s.CreateProduct(ctx, adm, in, "")
	wantCode(t, err, apperr.CodeValidation)
	in.Attributes = []domain.AttrValue{{Key: "matiere", Text: str("plomb")}}
	_, err = s.CreateProduct(ctx, adm, in, "")
	wantCode(t, err, apperr.CodeValidation)
	in.Attributes = []domain.AttrValue{{Key: "auto", Number: f64(1)}}
	_, err = s.CreateProduct(ctx, adm, in, "")
	wantCode(t, err, apperr.CodeValidation)

	// brouillon : l'attribut obligatoire « debit » (hérité) peut manquer ; la publication le refuse
	in.Attributes = []domain.AttrValue{{Key: "matiere", Text: str("inox")}}
	p := mk(t, s, in)
	_, err = s.PublishProduct(ctx, adm, p.ID)
	wantCode(t, err, apperr.CodeValidation)
	if ptrs := fieldPointers(err); len(ptrs) != 1 || ptrs[0] != "/attributes" {
		t.Fatalf("manques : %v", ptrs)
	}
	in.Attributes = append(in.Attributes, domain.AttrValue{Key: "debit", Number: f64(12.5)})
	if _, err := s.UpdateProduct(ctx, adm, p.ID, in, nil, ""); err != nil {
		t.Fatal(err)
	}
	pub, err := s.PublishProduct(ctx, adm, p.ID)
	if err != nil || pub.Status != "active" || pub.PublishedAt == nil {
		t.Fatalf("%+v %v", pub, err)
	}
	// valeurs relues avec leur type
	var got float64
	for _, a := range pub.Attributes {
		if a.Key == "debit" && a.Number != nil {
			got = *a.Number
		}
	}
	if got != 12.5 {
		t.Fatalf("débit relu : %v", pub.Attributes)
	}
	// un produit publié reste soumis aux règles : retirer l'attribut obligatoire est refusé
	in.Attributes = []domain.AttrValue{{Key: "matiere", Text: str("inox")}}
	_, err = s.UpdateProduct(ctx, adm, p.ID, in, nil, "")
	wantCode(t, err, apperr.CodeValidation)
	// l'attribut utilisé ne peut plus être supprimé
	defs, _ := s.ListAttributeDefs(ctx, "debit", &root.ID, 1, 10)
	wantCode(t, s.DeleteAttributeDef(ctx, adm, defs.Items[0].ID), apperr.CodeConflict)
}

func TestProductPublishRulesDependOnPriceMode(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "pompes", nil)
	in := base(c.ID)
	in.PriceAmount = nil
	p := mk(t, s, in)
	_, err := s.PublishProduct(ctx, adm, p.ID)
	wantCode(t, err, apperr.CodeValidation)
	if ptrs := fieldPointers(err); len(ptrs) != 1 || ptrs[0] != "/priceAmount" {
		t.Fatalf("manques : %v", ptrs)
	}
	in.PriceMode = domain.PriceOnQuote // sur devis : publiable sans prix
	if _, err := s.UpdateProduct(ctx, adm, p.ID, in, nil, ""); err != nil {
		t.Fatal(err)
	}
	if pub, err := s.PublishProduct(ctx, adm, p.ID); err != nil || pub.Status != "active" {
		t.Fatalf("%+v %v", pub, err)
	}
}

func TestProductLifecycle(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "pompes", nil)
	p := mk(t, s, base(c.ID))
	for name, f := range map[string]func() (*domain.Product, error){
		"restaurer un brouillon": func() (*domain.Product, error) { return s.RestoreProduct(ctx, adm, p.ID) },
	} {
		if _, err := f(); code(err) != apperr.CodeConflict {
			t.Errorf("%s : %v", name, err)
		}
	}
	if r, err := s.SubmitProductReview(ctx, adm, p.ID); err != nil || r.Status != "review" {
		t.Fatalf("%+v %v", r, err)
	}
	_, err := s.SubmitProductReview(ctx, adm, p.ID)
	wantCode(t, err, apperr.CodeConflict)
	if a, err := s.PublishProduct(ctx, adm, p.ID); err != nil || a.Status != "active" {
		t.Fatalf("%+v %v", a, err)
	}
	_, err = s.PublishProduct(ctx, adm, p.ID)
	wantCode(t, err, apperr.CodeConflict)
	if a, err := s.ArchiveProduct(ctx, adm, p.ID); err != nil || a.Status != "archived" {
		t.Fatalf("%+v %v", a, err)
	}
	// archivé : non modifiable tant qu'il n'est pas restauré
	in := base(c.ID)
	in.Name = loc("Autre nom")
	_, err = s.UpdateProduct(ctx, adm, p.ID, in, nil, "")
	wantCode(t, err, apperr.CodeConflict)
	r, err := s.RestoreProduct(ctx, adm, p.ID)
	if err != nil || r.Status != "draft" || r.PublishedAt == nil {
		t.Fatalf("restauré : %+v %v", r, err)
	}
	vs, _ := s.ProductVersions(ctx, p.ID)
	var actions []string
	for _, v := range vs {
		actions = append(actions, v.Action)
	}
	want := []string{"restore", "archive", "publish", "submit_review", "create"}
	if fmt.Sprint(actions) != fmt.Sprint(want) {
		t.Fatalf("historique : %v", actions)
	}
}

func TestProductIdentifiersFrozenAfterPublication(t *testing.T) {
	s, st := rig(t)
	c := cat(t, s, "pompes", nil)
	p := mk(t, s, base(c.ID))
	// avant publication : modifiables
	in := base(c.ID)
	in.SKU, in.Slug = "NOUVEAU-1", "nouveau-slug"
	p, err := s.UpdateProduct(ctx, adm, p.ID, in, nil, "")
	if err != nil || p.SKU != "NOUVEAU-1" || p.Slug != "nouveau-slug" {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := s.PublishProduct(ctx, adm, p.ID); err != nil {
		t.Fatal(err)
	}
	in.SKU = "AUTRE-2"
	_, err = s.UpdateProduct(ctx, adm, p.ID, in, nil, "")
	wantCode(t, err, apperr.CodeValidation)
	in.SKU, in.Slug = "NOUVEAU-1", "autre-slug"
	_, err = s.UpdateProduct(ctx, adm, p.ID, in, nil, "")
	wantCode(t, err, apperr.CodeValidation)
	// SKU et slug vides = inchangés
	in.SKU, in.Slug = "", ""
	in.Name = loc("Nom modifié après publication")
	if q, err := s.UpdateProduct(ctx, adm, p.ID, in, nil, ""); err != nil || q.SKU != "NOUVEAU-1" || q.Slug != "nouveau-slug" {
		t.Fatalf("%+v %v", q, err)
	}
	// la base protège aussi les écritures directes
	for _, q := range []string{`UPDATE products SET sku='X-1' WHERE id=$1`, `UPDATE products SET slug='x' WHERE id=$1`, `UPDATE products SET published_at=NULL WHERE id=$1`} {
		if _, err := st.Pool.Exec(ctx, q, p.ID); err == nil {
			t.Errorf("écriture directe acceptée : %s", q)
		}
	}
	if _, err := st.Pool.Exec(ctx, `UPDATE product_versions SET action='x' WHERE product_id=$1`, p.ID); err == nil {
		t.Error("l'historique est modifiable")
	}
}

func TestProductUpdateVersionsAndConcurrency(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "pompes", nil)
	p := mk(t, s, base(c.ID))
	// aucun changement : pas de nouvelle version
	same := base(c.ID)
	same.SKU, same.Slug = p.SKU, p.Slug
	q, err := s.UpdateProduct(ctx, adm, p.ID, same, nil, "")
	if err != nil || q.Version != 1 {
		t.Fatalf("mise à jour sans changement : %+v %v", q, err)
	}
	// changement de prix, origine « import »
	same.PriceAmount = i64(99000)
	q, err = s.UpdateProduct(ctx, adm, p.ID, same, i(1), "import")
	if err != nil || q.Version != 2 || q.Origins["priceAmount"] != "import" || q.Origins["name"] != "manual" {
		t.Fatalf("%+v %v", q, err)
	}
	vs, _ := s.ProductVersions(ctx, p.ID)
	ch := vs[0].Changes["priceAmount"].(map[string]any)
	if vs[0].Version != 2 || len(vs[0].Changes) != 1 || ch["before"].(float64) != 125000 || ch["after"].(float64) != 99000 || ch["origin"] != "import" {
		t.Fatalf("changement enregistré : %+v", vs[0])
	}
	// version périmée
	same.PriceAmount = i64(1)
	_, err = s.UpdateProduct(ctx, adm, p.ID, same, i(1), "")
	wantCode(t, err, apperr.CodeConflict)

	// concurrence : 8 écritures avec la même version attendue, une seule passe
	cur, _ := s.GetProduct(ctx, p.ID)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := same
			in.PriceAmount = i64(int64(1000 + n))
			_, err := s.UpdateProduct(ctx, adm, p.ID, in, i(cur.Version), "")
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				ok++
			} else if code(err) != apperr.CodeConflict {
				t.Errorf("l'écriture perdante doit être un conflit propre, pas : %v", err)
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("%d écritures concurrentes ont réussi, une seule attendue", ok)
	}
	end, _ := s.GetProduct(ctx, p.ID)
	if end.Version != cur.Version+1 {
		t.Fatalf("version : %d", end.Version)
	}
}

func TestProductConcurrentPublishHappensOnce(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "pompes", nil)
	p := mk(t, s, base(c.ID))
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for n := 0; n < 6; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.PublishProduct(ctx, adm, p.ID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				ok++
			} else if code(err) != apperr.CodeConflict {
				t.Errorf("la publication perdante doit être un conflit propre, pas : %v", err)
			}
		}()
	}
	wg.Wait()
	vs, _ := s.ProductVersions(ctx, p.ID)
	if ok != 1 || len(vs) != 2 {
		t.Fatalf("%d publications réussies, %d versions", ok, len(vs))
	}
}

func TestProductManufacturerRefUniqueness(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "roulements", nil)
	b1, _ := s.CreateBrand(ctx, adm, domain.Brand{Slug: "skf", Name: "SKF"})
	b2, _ := s.CreateBrand(ctx, adm, domain.Brand{Slug: "fag", Name: "FAG"})
	with := func(brand *string, ref string) domain.Product {
		p := base(c.ID)
		p.BrandID, p.ManufacturerRef = brand, str(ref)
		return p
	}
	mk(t, s, with(&b1.ID, "6204-2RS/C3"))
	_, err := s.CreateProduct(ctx, adm, with(&b1.ID, "6204 2rs c3"), "") // même référence normalisée, même marque
	wantCode(t, err, apperr.CodeConflict)
	mk(t, s, with(&b2.ID, "6204-2RS/C3")) // autre marque : permis
	mk(t, s, with(nil, "6204-2RS/C3"))    // sans marque : pas de clé de fusion
	mk(t, s, with(nil, "6204-2RS/C3"))
	// marque utilisée : suppression refusée
	wantCode(t, s.DeleteBrand(ctx, adm, b1.ID), apperr.CodeConflict)
	// catégorie utilisée : suppression refusée
	wantCode(t, s.DeleteCategory(ctx, adm, c.ID), apperr.CodeConflict)
}

func TestProductChildrenRoundTrip(t *testing.T) {
	s, _ := rig(t)
	c := cat(t, s, "pompes", nil)
	sec1, _ := s.CreateSector(ctx, adm, domain.Sector{Slug: "agro", Name: loc("Agro")})
	sec2, _ := s.CreateSector(ctx, adm, domain.Sector{Slug: "mines", Name: loc("Mines")})
	in := base(c.ID)
	in.SectorIDs = []string{sec1.ID, sec2.ID}
	in.Media = []domain.Media{{URL: "https://exemple.test/b.jpg", Alt: str("B")}, {URL: "https://exemple.test/a.jpg"}}
	in.CrossRefs = []domain.CrossRef{{Brand: "Grundfos", Number: "CM 3-5", Kind: "equivalent"}, {Brand: "Grundfos", Number: "CM 3-4", Kind: "oem"}}
	p := mk(t, s, in)
	if len(p.SectorIDs) != 2 || len(p.Media) != 2 || p.Media[0].URL != "https://exemple.test/b.jpg" || p.Media[1].SortOrder != 1 || len(p.CrossRefs) != 2 {
		t.Fatalf("%+v", p)
	}
	wantCode(t, s.DeleteSector(ctx, adm, sec1.ID), apperr.CodeConflict)
	// remplacement : l'ordre des médias suit l'entrée, les listes remplacent les précédentes
	in.Media = []domain.Media{{URL: "https://exemple.test/a.jpg"}, {URL: "https://exemple.test/b.jpg"}}
	in.CrossRefs = nil
	in.SectorIDs = []string{sec2.ID}
	in.SKU, in.Slug = p.SKU, p.Slug
	q, err := s.UpdateProduct(ctx, adm, p.ID, in, nil, "")
	if err != nil || q.Media[0].URL != "https://exemple.test/a.jpg" || len(q.CrossRefs) != 0 || len(q.SectorIDs) != 1 {
		t.Fatalf("%+v %v", q, err)
	}
	if err := s.DeleteSector(ctx, adm, sec1.ID); err != nil {
		t.Fatal(err)
	}
	// liste : filtres par catégorie (sous-catégories incluses), statut, recherche
	sub := cat(t, s, "doseuses", &c.ID)
	mk(t, s, base(sub.ID))
	l, _ := s.ListProducts(ctx, application.ProductFilter{CategoryID: &c.ID}, 1, 20)
	if l.Total != 2 {
		t.Fatalf("catégorie + descendants : %d", l.Total)
	}
	l, _ = s.ListProducts(ctx, application.ProductFilter{CategoryID: &sub.ID}, 1, 20)
	if l.Total != 1 {
		t.Fatalf("sous-catégorie : %d", l.Total)
	}
	act := "active"
	if l, _ = s.ListProducts(ctx, application.ProductFilter{Status: &act}, 1, 20); l.Total != 0 {
		t.Fatalf("publiés : %d", l.Total)
	}
	if l, _ = s.ListProducts(ctx, application.ProductFilter{Query: "doseuse"}, 1, 20); l.Total != 2 {
		t.Fatalf("recherche : %d", l.Total)
	}
	if l, _ = s.ListProducts(ctx, application.ProductFilter{Query: "%"}, 1, 20); l.Total != 0 {
		t.Fatalf("joker non échappé : %d", l.Total)
	}
}

func TestProductDeleteRules(t *testing.T) {
	s, st := rig(t)
	c := cat(t, s, "pompes", nil)
	draft := mk(t, s, base(c.ID))
	if err := s.DeleteProduct(ctx, adm, draft.ID); err != nil {
		t.Fatal(err)
	}
	_, err := s.GetProduct(ctx, draft.ID)
	wantCode(t, err, apperr.CodeNotFound)
	var n int
	_ = st.Pool.QueryRow(ctx, `SELECT count(*) FROM product_versions WHERE product_id=$1`, draft.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("historique orphelin : %d", n)
	}
	used := mk(t, s, base(c.ID))
	if _, err := s.PublishProduct(ctx, adm, used.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProduct(ctx, adm, used.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProduct(ctx, used.ID)
	if err != nil || got.Status != "archived" {
		t.Fatalf("un produit publié doit être archivé, pas supprimé : %+v %v", got, err)
	}
	wantCode(t, s.DeleteProduct(ctx, adm, "00000000-0000-0000-0000-0000000000aa"), apperr.CodeNotFound)
}

func TestProductAuditEvents(t *testing.T) {
	s, st := rig(t)
	count := func() int {
		var n int
		_ = st.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE subject = 'audit.recorded.v1'`).Scan(&n)
		return n
	}
	c := cat(t, s, "pompes", nil)
	start := count()
	p := mk(t, s, base(c.ID))
	_, _ = s.CreateProduct(ctx, adm, func() domain.Product { x := base(c.ID); x.SKU = "bad sku!"; return x }(), "") // échec : aucun événement
	in := base(c.ID)
	in.SKU, in.Slug = p.SKU, p.Slug
	in.Name = loc("Autre")
	_, _ = s.UpdateProduct(ctx, adm, p.ID, in, nil, "")
	_, _ = s.PublishProduct(ctx, adm, p.ID)
	_, _ = s.PublishProduct(ctx, adm, p.ID) // refusé : aucun événement
	if got := count() - start; got != 3 {
		t.Fatalf("%d événements d'audit, 3 attendus (création, modification, publication)", got)
	}
}
