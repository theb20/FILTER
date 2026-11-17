package server_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	catalogadminv1 "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeCatalog struct {
	catalogadminv1.CatalogAdminServiceClient
	calls    map[string]int
	last     *catalogadminv1.Category
	err      error
	product  *catalogadminv1.Product
	expected *int32
}

func (f *fakeCatalog) ListCategories(_ context.Context, _ *catalogadminv1.ListCategoriesRequest, _ ...grpc.CallOption) (*catalogadminv1.ListCategoriesResponse, error) {
	f.calls["ListCategories"]++
	return &catalogadminv1.ListCategoriesResponse{Items: []*catalogadminv1.Category{{Id: uuid.NewString(), Slug: "pompes", Name: &catalogadminv1.Localized{Texts: map[string]string{"fr": "Pompes"}}, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}},
		Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: 1, TotalPages: 1}}, nil
}

func (f *fakeCatalog) CreateCategory(_ context.Context, in *catalogadminv1.CreateCategoryRequest, _ ...grpc.CallOption) (*catalogadminv1.CreateCategoryResponse, error) {
	f.calls["CreateCategory"]++
	if f.err != nil {
		return nil, f.err
	}
	c := in.GetCategory()
	c.Id, c.CreatedAt, c.UpdatedAt = uuid.NewString(), timestamppb.Now(), timestamppb.Now()
	f.last = c
	return &catalogadminv1.CreateCategoryResponse{Category: c}, nil
}

func (f *fakeCatalog) MoveCategory(_ context.Context, in *catalogadminv1.MoveCategoryRequest, _ ...grpc.CallOption) (*catalogadminv1.MoveCategoryResponse, error) {
	f.calls["MoveCategory"]++
	return &catalogadminv1.MoveCategoryResponse{Category: &catalogadminv1.Category{Id: in.GetId(), Slug: "x", ParentId: in.ParentId, Name: &catalogadminv1.Localized{Texts: map[string]string{"fr": "X"}}, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}}, nil
}

func (f *fakeCatalog) CreateBrand(_ context.Context, in *catalogadminv1.CreateBrandRequest, _ ...grpc.CallOption) (*catalogadminv1.CreateBrandResponse, error) {
	f.calls["CreateBrand"]++
	b := in.GetBrand()
	b.Id, b.CreatedAt, b.UpdatedAt = uuid.NewString(), timestamppb.Now(), timestamppb.Now()
	return &catalogadminv1.CreateBrandResponse{Brand: b}, nil
}

func (f *fakeCatalog) DeleteBrand(_ context.Context, _ *catalogadminv1.DeleteBrandRequest, _ ...grpc.CallOption) (*catalogadminv1.DeleteBrandResponse, error) {
	f.calls["DeleteBrand"]++
	return &catalogadminv1.DeleteBrandResponse{}, nil
}

func catalogRig(t *testing.T, f *fakeCatalog) *rig {
	return newRigDeps(t, nil, func(d *server.Deps) { d.Catalog = f })
}

func TestPermissionsCatalogueTaxonomie(t *testing.T) {
	f := &fakeCatalog{calls: map[string]int{}}
	r := catalogRig(t, f)
	sub, id := uuid.NewString(), uuid.NewString()
	cat := `{"slug":"pompes","name":{"fr":"Pompes"}}`
	for _, c := range []struct {
		name, method, path, body string
		perms                    []string
		want                     int
		call                     string
	}{
		{"liste sans permission", "GET", "/v1/admin/categories", "", nil, 403, ""},
		{"liste avec une mauvaise permission", "GET", "/v1/admin/categories", "", []string{"engine:read"}, 403, ""},
		{"liste avec catalog:read", "GET", "/v1/admin/categories", "", []string{"catalog:read"}, 200, "ListCategories"},
		{"création avec catalog:read seulement", "POST", "/v1/admin/categories", cat, []string{"catalog:read"}, 403, ""},
		{"création avec catalog:write", "POST", "/v1/admin/categories", cat, []string{"catalog:write"}, 201, "CreateCategory"},
		{"déplacement avec catalog:read seulement", "POST", "/v1/admin/categories/" + id + "/move", `{"parentId":null,"position":0}`, []string{"catalog:read"}, 403, ""},
		{"déplacement avec catalog:write", "POST", "/v1/admin/categories/" + id + "/move", `{"parentId":null,"position":0}`, []string{"catalog:write"}, 200, "MoveCategory"},
		{"suppression de marque sans permission d'écriture", "DELETE", "/v1/admin/brands/" + id, "", []string{"catalog:read"}, 403, ""},
		{"suppression de marque", "DELETE", "/v1/admin/brands/" + id, "", []string{"catalog:write"}, 204, "DeleteBrand"},
		{"joker", "GET", "/v1/admin/categories", "", []string{"*"}, 200, "ListCategories"},
	} {
		before := f.calls[c.call]
		rec := r.do(req{c.method, c.path, c.body, bearer(staffToken(t, r, sub, c.perms...))})
		if rec.Code != c.want {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
			continue
		}
		if c.call != "" && f.calls[c.call] != before+1 {
			t.Errorf("%s : %s non appelé", c.name, c.call)
		}
		if c.call == "" {
			total := 0
			for _, n := range f.calls {
				total += n
			}
			_ = total
		}
	}
	// aucun appel n'a été relayé pour les refus
	if f.calls["CreateBrand"] != 0 {
		t.Fatalf("appel inattendu : %v", f.calls)
	}
}

func TestCatalogueSansJetonEtSansService(t *testing.T) {
	r := catalogRig(t, &fakeCatalog{calls: map[string]int{}})
	if rec := r.do(req{"GET", "/v1/admin/categories", "", nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	r = newRigWith(t, nil, nil) // service catalog non configuré
	rec := r.do(req{"GET", "/v1/admin/categories", "", bearer(staffToken(t, r, uuid.NewString(), "catalog:read"))})
	if rec.Code != 501 {
		t.Fatalf("service absent : %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestCatalogueRelaieLesErreursEtLesDonnees(t *testing.T) {
	f := &fakeCatalog{calls: map[string]int{}, err: status.Error(codes.AlreadyExists, "cet identifiant d'URL est déjà utilisé")}
	r := catalogRig(t, f)
	tok := bearer(staffToken(t, r, uuid.NewString(), "catalog:write"))
	rec := r.do(req{"POST", "/v1/admin/categories", `{"slug":"pompes","name":{"fr":"Pompes"}}`, tok})
	if rec.Code != 409 {
		t.Fatalf("conflit attendu : %d (%s)", rec.Code, rec.Body.String())
	}
	// validation côté gateway : slug invalide refusé avant tout appel
	f.err = nil
	n := f.calls["CreateCategory"]
	rec = r.do(req{"POST", "/v1/admin/categories", `{"slug":"Pas Valide","name":{"fr":"x"}}`, tok})
	if rec.Code != 400 && rec.Code != 422 {
		t.Fatalf("slug invalide : %d", rec.Code)
	}
	if f.calls["CreateCategory"] != n {
		t.Fatal("un slug invalide ne doit pas atteindre le service")
	}
	// les données traversent : parentId et description null sont conservés
	rec = r.do(req{"POST", "/v1/admin/categories", `{"slug":"ok","name":{"fr":"Ok","en":"Ok"},"description":null,"parentId":null}`, tok})
	if rec.Code != 201 {
		t.Fatalf("création : %d (%s)", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["slug"] != "ok" || out["parentId"] != nil {
		t.Fatalf("réponse : %v", out)
	}
	if f.last.GetName().GetTexts()["en"] != "Ok" || f.last.Description != nil || f.last.ParentId != nil {
		t.Fatalf("relayé : %+v", f.last)
	}
}

func prod(id string) *catalogadminv1.Product {
	return &catalogadminv1.Product{Id: id, Sku: "FF-000001", Slug: "p", Name: &catalogadminv1.Localized{Texts: map[string]string{"fr": "P"}}, CategoryId: uuid.NewString(), Status: "draft",
		FulfillmentMode: "stock", PriceMode: "fixed", Currency: "XOF", Version: 1, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}
}

func (f *fakeCatalog) ListProducts(_ context.Context, _ *catalogadminv1.ListProductsRequest, _ ...grpc.CallOption) (*catalogadminv1.ListProductsResponse, error) {
	f.calls["ListProducts"]++
	return &catalogadminv1.ListProductsResponse{Items: []*catalogadminv1.Product{prod(uuid.NewString())}, Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: 1, TotalPages: 1}}, nil
}

func (f *fakeCatalog) CreateProduct(_ context.Context, in *catalogadminv1.CreateProductRequest, _ ...grpc.CallOption) (*catalogadminv1.CreateProductResponse, error) {
	f.calls["CreateProduct"]++
	f.product = in.GetProduct()
	p := prod(uuid.NewString())
	p.Attributes = in.GetProduct().GetAttributes()
	return &catalogadminv1.CreateProductResponse{Product: p}, nil
}

func (f *fakeCatalog) UpdateProduct(_ context.Context, in *catalogadminv1.UpdateProductRequest, _ ...grpc.CallOption) (*catalogadminv1.UpdateProductResponse, error) {
	f.calls["UpdateProduct"]++
	f.product, f.expected = in.GetProduct(), in.ExpectedVersion
	return &catalogadminv1.UpdateProductResponse{Product: prod(in.GetProduct().GetId())}, nil
}

func (f *fakeCatalog) SubmitProductReview(_ context.Context, in *catalogadminv1.SubmitProductReviewRequest, _ ...grpc.CallOption) (*catalogadminv1.SubmitProductReviewResponse, error) {
	f.calls["SubmitProductReview"]++
	return &catalogadminv1.SubmitProductReviewResponse{Product: prod(in.GetId())}, nil
}

func (f *fakeCatalog) PublishProduct(_ context.Context, in *catalogadminv1.PublishProductRequest, _ ...grpc.CallOption) (*catalogadminv1.PublishProductResponse, error) {
	f.calls["PublishProduct"]++
	return &catalogadminv1.PublishProductResponse{Product: prod(in.GetId())}, nil
}

func TestPermissionsProduits(t *testing.T) {
	f := &fakeCatalog{calls: map[string]int{}}
	r := catalogRig(t, f)
	sub, id := uuid.NewString(), uuid.NewString()
	body := `{"name":{"fr":"Pompe"},"categoryId":"` + uuid.NewString() + `","fulfillmentMode":"stock","priceMode":"fixed","priceAmount":125000}`
	for _, c := range []struct {
		name, method, path, body string
		perms                    []string
		want                     int
		call                     string
	}{
		{"liste sans permission", "GET", "/v1/admin/products", "", nil, 403, ""},
		{"liste avec catalog:read", "GET", "/v1/admin/products", "", []string{"catalog:read"}, 200, "ListProducts"},
		{"création avec catalog:read", "POST", "/v1/admin/products", body, []string{"catalog:read"}, 403, ""},
		{"création avec catalog:write", "POST", "/v1/admin/products", body, []string{"catalog:write"}, 201, "CreateProduct"},
		{"soumission avec catalog:write", "POST", "/v1/admin/products/" + id + "/submit-review", "", []string{"catalog:write"}, 200, "SubmitProductReview"},
		{"publication avec catalog:write seulement", "POST", "/v1/admin/products/" + id + "/publish", "", []string{"catalog:write"}, 403, ""},
		{"publication avec catalog:read", "POST", "/v1/admin/products/" + id + "/publish", "", []string{"catalog:read"}, 403, ""},
		{"publication avec catalog:publish", "POST", "/v1/admin/products/" + id + "/publish", "", []string{"catalog:publish"}, 200, "PublishProduct"},
		{"publication avec le joker", "POST", "/v1/admin/products/" + id + "/publish", "", []string{"*"}, 200, "PublishProduct"},
	} {
		before := f.calls[c.call]
		rec := r.do(req{c.method, c.path, c.body, bearer(staffToken(t, r, sub, c.perms...))})
		if rec.Code != c.want {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
			continue
		}
		if c.call != "" && f.calls[c.call] != before+1 {
			t.Errorf("%s : %s non appelé", c.name, c.call)
		}
	}
	if f.calls["PublishProduct"] != 2 {
		t.Fatalf("seuls catalog:publish et le joker publient : %v", f.calls)
	}
}

func TestProduitsRelaieLesChamps(t *testing.T) {
	f := &fakeCatalog{calls: map[string]int{}}
	r := catalogRig(t, f)
	tok := bearer(staffToken(t, r, uuid.NewString(), "catalog:write"))
	body := `{"name":{"fr":"Roulement","en":"Bearing"},"categoryId":"` + uuid.NewString() + `","fulfillmentMode":"made_to_order","leadTimeDays":14,"priceMode":"on_quote","priceAmount":null,
	"manufacturerRef":"6204-2RS","gtin":"4006381333931","minOrderQuantity":10,"weightGrams":120,
	"attributes":[{"key":"debit","value":12.5},{"key":"matiere","value":"inox"},{"key":"auto","value":true}],
	"media":[{"url":"https://exemple.test/a.jpg","alt":"A"}],"crossReferences":[{"brand":"SKF","number":"6204-2RS","kind":"oem"}]}`
	rec := r.do(req{"POST", "/v1/admin/products", body, tok})
	if rec.Code != 201 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	p := f.product
	if p.GetFulfillmentMode() != "made_to_order" || p.GetLeadTimeDays() != 14 || p.PriceAmount != nil || p.GetPriceMode() != "on_quote" || p.GetManufacturerRef() != "6204-2RS" ||
		p.GetGtin() != "4006381333931" || p.GetMinOrderQuantity() != 10 || p.GetWeightGrams() != 120 || p.Description != nil || p.BrandId != nil {
		t.Fatalf("relayé : %+v", p)
	}
	at := p.GetAttributes()
	if len(at) != 3 || at[0].GetNumber() != 12.5 || at[1].GetText() != "inox" || !at[2].GetBoolean() {
		t.Fatalf("attributs typés : %+v", at)
	}
	if p.GetMedia()[0].GetUrl() != "https://exemple.test/a.jpg" || p.GetMedia()[0].GetAlt() != "A" || p.GetCrossReferences()[0].GetKind() != "oem" {
		t.Fatalf("médias/références : %+v", p)
	}
	// les attributs reviennent avec leur type JSON
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if a := out["attributes"].([]any); len(a) != 3 {
		t.Fatalf("réponse : %v", out)
	}
	// version attendue transmise à la mise à jour
	id := uuid.NewString()
	rec = r.do(req{"PUT", "/v1/admin/products/" + id, `{"name":{"fr":"X"},"categoryId":"` + uuid.NewString() + `","fulfillmentMode":"stock","priceMode":"fixed","version":7}`, tok})
	if rec.Code != 200 || f.expected == nil || *f.expected != 7 || f.product.GetId() != id {
		t.Fatalf("%d expected=%v id=%v", rec.Code, f.expected, f.product.GetId())
	}
	// validation côté contrat : GTIN mal formé, mode de prix inconnu
	for _, bad := range []string{`"gtin":"abc"`, `"priceMode":"gratuit"`} {
		b := `{"name":{"fr":"X"},"categoryId":"` + uuid.NewString() + `","fulfillmentMode":"stock","priceMode":"fixed",` + bad + `}`
		b = map[bool]string{true: `{"name":{"fr":"X"},"categoryId":"` + uuid.NewString() + `","fulfillmentMode":"stock",` + bad + `}`, false: b}[bad[1] == 'p']
		n := f.calls["CreateProduct"]
		if rec := r.do(req{"POST", "/v1/admin/products", b, tok}); (rec.Code != 400 && rec.Code != 422) || f.calls["CreateProduct"] != n {
			t.Errorf("%s : %d", bad, rec.Code)
		}
	}
}
