package server_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	contentv1 "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
)

type verifiedID struct {
	*fakeID
	verified bool
}

func (v verifiedID) GetUser(_ context.Context, _ *identityv1.GetUserRequest, _ ...grpc.CallOption) (*identityv1.GetUserResponse, error) {
	u := testUser()
	u.EmailVerified = v.verified
	return &identityv1.GetUserResponse{User: u}, nil
}

type quotesContent struct {
	contentv1.ContentAdminServiceClient
	where map[string]string
}

func (q *quotesContent) ListDocuments(_ context.Context, in *contentv1.ListDocumentsRequest, _ ...grpc.CallOption) (*contentv1.ListDocumentsResponse, error) {
	q.where = in.GetWhere()
	d := doc("quote_request", map[string]any{"name": "A", "email": "awa@exemple.ci", "status": "in_progress", "subject": "Cuves GPL", "items": []any{map[string]any{"reference": "X", "quantity": 1.0}, map[string]any{"reference": "Y", "quantity": 2.0}}})
	return &contentv1.ListDocumentsResponse{Items: []*contentv1.Document{d}, Page: &commonv1.PageResponse{Page: 1, PageSize: 100, Total: 1, TotalPages: 1}}, nil
}

func TestSuiviDesDemandesDeDevis(t *testing.T) {
	q := &quotesContent{}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Identity = verifiedID{&fakeID{}, true}; d.Content = q })
	tok := customerTokenFor(t, r, uuid.NewString())
	if rec := r.do(req{"GET", "/v1/me/quote-requests", "", nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	rec := r.do(req{"GET", "/v1/me/quote-requests", "", bearer(tok)})
	if rec.Code != 200 || !contains(rec.Body.String(), `"status":"in_progress"`) || !contains(rec.Body.String(), `"itemCount":2`) {
		t.Fatalf("liste : %d %s", rec.Code, rec.Body.String())
	}
	if q.where["email"] != "awa@exemple.ci" { // jamais un filtre fourni par l'appelant
		t.Fatalf("filtre : %v", q.where)
	}
	// adresse non vérifiée : aucune demande n'est lisible
	r2 := newRigDeps(t, nil, func(d *server.Deps) { d.Identity = verifiedID{&fakeID{}, false}; d.Content = &quotesContent{} })
	if rec := r2.do(req{"GET", "/v1/me/quote-requests", "", bearer(customerTokenFor(t, r2, uuid.NewString()))}); rec.Code != 403 || !contains(rec.Body.String(), "EMAIL_NOT_VERIFIED") {
		t.Fatalf("non vérifiée : %d %s", rec.Code, rec.Body.String())
	}
}

// Le client ne voit un devis chiffré qu'une fois ENVOYÉ, et jamais les notes internes.
type answeredContent struct {
	contentv1.ContentAdminServiceClient
	respStatus string
}

func (a *answeredContent) ListDocuments(_ context.Context, _ *contentv1.ListDocumentsRequest, _ ...grpc.CallOption) (*contentv1.ListDocumentsResponse, error) {
	d := doc("quote_request", map[string]any{"name": "A", "email": "awa@exemple.ci", "status": "quoted", "items": []any{map[string]any{"reference": "X", "quantity": 2.0}}})
	return &contentv1.ListDocumentsResponse{Items: []*contentv1.Document{d}, Page: &commonv1.PageResponse{Page: 1, PageSize: 100, Total: 1, TotalPages: 1}}, nil
}

func (a *answeredContent) GetDocument(_ context.Context, in *contentv1.GetDocumentRequest, _ ...grpc.CallOption) (*contentv1.GetDocumentResponse, error) {
	if in.GetKind() != "quote_response" {
		return nil, nil
	}
	return &contentv1.GetDocumentResponse{Document: doc("quote_response", map[string]any{
		"status": a.respStatus, "number": "DEV-20261117-AB12", "validUntil": "2030-01-01T00:00:00Z", "sentAt": "2026-11-17T10:00:00Z", "internalNote": "marge 30 % : SECRET",
		"discountPct": 0.0, "shipping": 0.0, "vatRate": 18.0, "subtotal": 2000.0, "vatAmount": 360.0, "total": 2360.0,
		"incoterm": "DDP, Abidjan", "acompte": 360.0, "annexes": []any{map[string]any{"url": "/v1/files/0123456789abcdef0123456789abcdef.jpg", "caption": "Fiche GS50"}},
		"lines": []any{map[string]any{"reference": "X", "quantity": 2.0, "unitPrice": 1000.0, "note": "visible", "internal": "coût 600 : SECRET", "image": "/v1/files/fedcba9876543210fedcba9876543210.png"}},
	})}, nil
}

func TestDevisChiffreVisibleSeulementSiEnvoye(t *testing.T) {
	for _, tc := range []struct {
		status string
		shown  bool
	}{{"draft", false}, {"sent", true}} {
		r := newRigDeps(t, nil, func(d *server.Deps) {
			d.Identity = verifiedID{&fakeID{}, true}
			d.Content = &answeredContent{respStatus: tc.status}
		})
		rec := r.do(req{"GET", "/v1/me/quote-requests", "", bearer(customerTokenFor(t, r, uuid.NewString()))})
		body := rec.Body.String()
		if rec.Code != 200 || contains(body, "SECRET") {
			t.Fatalf("%s : %d fuite possible : %s", tc.status, rec.Code, body)
		}
		if has := contains(body, `"total":2360`) && contains(body, "DEV-20261117-AB12") && contains(body, `"lineTotal":2000`) && contains(body, `"remaining":2000`) && contains(body, "Fiche GS50") && contains(body, "fedcba9876543210fedcba9876543210.png"); has != tc.shown {
			t.Fatalf("%s : devis visible = %v : %s", tc.status, has, body)
		}
	}
}
