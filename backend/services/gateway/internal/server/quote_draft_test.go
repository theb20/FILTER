package server_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	contentv1 "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// draftContent : un seul brouillon par compte, retrouvé par la clé (identifiant du compte).
type draftContent struct {
	contentv1.ContentAdminServiceClient
	docs map[string]*contentv1.Document
}

func (f *draftContent) GetDocument(_ context.Context, in *contentv1.GetDocumentRequest, _ ...grpc.CallOption) (*contentv1.GetDocumentResponse, error) {
	if d, ok := f.docs[in.GetIdOrKey()]; ok {
		return &contentv1.GetDocumentResponse{Document: d}, nil
	}
	return nil, status.Error(codes.NotFound, "absent")
}

func (f *draftContent) CreateDocument(_ context.Context, in *contentv1.CreateDocumentRequest, _ ...grpc.CallOption) (*contentv1.CreateDocumentResponse, error) {
	d := &contentv1.Document{Id: uuid.NewString(), Kind: in.GetKind(), Data: in.GetData(), Version: 1, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}
	f.docs[in.GetData().GetFields()["userId"].GetStringValue()] = d
	return &contentv1.CreateDocumentResponse{Document: d}, nil
}

func (f *draftContent) UpdateDocument(_ context.Context, in *contentv1.UpdateDocumentRequest, _ ...grpc.CallOption) (*contentv1.UpdateDocumentResponse, error) {
	for k, d := range f.docs {
		if d.GetId() == in.GetId() {
			d.Data, d.UpdatedAt = in.GetData(), timestamppb.Now()
			f.docs[k] = d
			return &contentv1.UpdateDocumentResponse{Document: d}, nil
		}
	}
	return nil, status.Error(codes.NotFound, "absent")
}

func (f *draftContent) DeleteDocument(_ context.Context, in *contentv1.DeleteDocumentRequest, _ ...grpc.CallOption) (*contentv1.DeleteDocumentResponse, error) {
	for k, d := range f.docs {
		if d.GetId() == in.GetId() {
			delete(f.docs, k)
		}
	}
	return &contentv1.DeleteDocumentResponse{}, nil
}

var _ = structpb.NewStruct

func TestBrouillonDeDevisParCompte(t *testing.T) {
	f := &draftContent{docs: map[string]*contentv1.Document{}}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Content = f })
	alice, bob := uuid.NewString(), uuid.NewString()
	ta, tb := customerTokenFor(t, r, alice), customerTokenFor(t, r, bob)

	if rec := r.do(req{"GET", "/v1/me/quote-draft", "", nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	if rec := r.do(req{"GET", "/v1/me/quote-draft", "", bearer(ta)}); rec.Code != 404 {
		t.Fatalf("aucun brouillon : %d", rec.Code)
	}
	body := `{"company":"Société","subject":"Cuves GPL","items":[{"reference":"HF-100","quantity":3}]}`
	if rec := r.do(req{"PUT", "/v1/me/quote-draft", body, bearer(ta)}); rec.Code != 200 || !contains(rec.Body.String(), `"subject":"Cuves GPL"`) || !contains(rec.Body.String(), "updatedAt") {
		t.Fatalf("enregistrer : %d %s", rec.Code, rec.Body.String())
	}
	// remplacement (et non second brouillon)
	if rec := r.do(req{"PUT", "/v1/me/quote-draft", `{"subject":"Autre"}`, bearer(ta)}); rec.Code != 200 || len(f.docs) != 1 {
		t.Fatalf("remplacer : %d %d", rec.Code, len(f.docs))
	}
	if rec := r.do(req{"GET", "/v1/me/quote-draft", "", bearer(ta)}); rec.Code != 200 || !contains(rec.Body.String(), `"subject":"Autre"`) || contains(rec.Body.String(), "userId") {
		t.Fatalf("lire : %d %s", rec.Code, rec.Body.String())
	}
	// un autre compte ne voit jamais ce brouillon
	if rec := r.do(req{"GET", "/v1/me/quote-draft", "", bearer(tb)}); rec.Code != 404 {
		t.Fatalf("isolation : %d", rec.Code)
	}
	if rec := r.do(req{"DELETE", "/v1/me/quote-draft", "", bearer(ta)}); rec.Code != 204 || len(f.docs) != 0 {
		t.Fatalf("supprimer : %d", rec.Code)
	}
	if rec := r.do(req{"DELETE", "/v1/me/quote-draft", "", bearer(ta)}); rec.Code != 204 {
		t.Fatalf("supprimer deux fois : %d", rec.Code)
	}
}
