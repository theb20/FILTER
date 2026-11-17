package server_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	catalogadminv1 "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// pubCatalog : un produit publié et un brouillon.
type pubCatalog struct{ *fakeCatalog }

func (f pubCatalog) GetProduct(_ context.Context, in *catalogadminv1.GetProductRequest, _ ...grpc.CallOption) (*catalogadminv1.GetProductResponse, error) {
	p := prod(in.GetId())
	if in.GetId() == "11111111-1111-1111-1111-111111111111" {
		p.Status = "active"
		p.Name = &catalogadminv1.Localized{Texts: map[string]string{"fr": "Pompe", "en": "Pump"}}
	}
	if in.GetId() == "22222222-2222-2222-2222-222222222222" {
		return nil, status.Error(codes.NotFound, "absent")
	}
	return &catalogadminv1.GetProductResponse{Product: p}, nil
}

func TestVitrineCatalogue(t *testing.T) {
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Catalog = pubCatalog{&fakeCatalog{calls: map[string]int{}}} })
	pub := "/v1/products/11111111-1111-1111-1111-111111111111"
	// sans aucun jeton
	rec := r.do(req{"GET", pub, "", map[string]string{"Accept-Language": "en"}})
	if rec.Code != 200 {
		t.Fatalf("produit publié : %d %s", rec.Code, rec.Body.String())
	}
	var p struct {
		Name         string
		Availability struct{ Status string }
		Price        struct {
			UnitPrice struct{ Amount, Currency string }
		}
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	if p.Name != "Pump" || p.Availability.Status != "on_request" || p.Price.UnitPrice.Currency != "XOF" {
		t.Fatalf("contenu inattendu : %s", rec.Body.String())
	}
	// un brouillon n'est jamais visible
	if rec := r.do(req{"GET", "/v1/products/" + uuid.NewString(), "", nil}); rec.Code != 404 {
		t.Fatalf("brouillon : %d, attendu 404", rec.Code)
	}
	if rec := r.do(req{"GET", "/v1/products/22222222-2222-2222-2222-222222222222", "", nil}); rec.Code != 404 {
		t.Fatalf("absent : %d, attendu 404", rec.Code)
	}
	for _, path := range []string{"/v1/products", "/v1/categories", "/v1/brands", "/v1/sectors"} {
		if rec := r.do(req{"GET", path, "", nil}); rec.Code != 200 {
			t.Errorf("%s : %d %s", path, rec.Code, rec.Body.String())
		}
	}
}

func (f pubCatalog) ListBrands(context.Context, *catalogadminv1.ListBrandsRequest, ...grpc.CallOption) (*catalogadminv1.ListBrandsResponse, error) {
	return &catalogadminv1.ListBrandsResponse{}, nil
}

func (f pubCatalog) ListSectors(context.Context, *catalogadminv1.ListSectorsRequest, ...grpc.CallOption) (*catalogadminv1.ListSectorsResponse, error) {
	return &catalogadminv1.ListSectorsResponse{}, nil
}

func (f pubCatalog) ListAttributeDefs(context.Context, *catalogadminv1.ListAttributeDefsRequest, ...grpc.CallOption) (*catalogadminv1.ListAttributeDefsResponse, error) {
	return &catalogadminv1.ListAttributeDefsResponse{}, nil
}
