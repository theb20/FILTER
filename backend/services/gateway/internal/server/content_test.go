package server_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	contentv1 "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeContent struct {
	contentv1.ContentAdminServiceClient
	calls map[string]int
	last  *structpb.Struct
	where map[string]string
}

func doc(kind string, m map[string]any) *contentv1.Document {
	s, _ := structpb.NewStruct(m)
	return &contentv1.Document{Id: uuid.NewString(), Kind: kind, Data: s, Version: 1, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}
}

func (f *fakeContent) ListDocuments(_ context.Context, in *contentv1.ListDocumentsRequest, _ ...grpc.CallOption) (*contentv1.ListDocumentsResponse, error) {
	f.calls["List:"+in.GetKind()]++
	f.where = in.GetWhere()
	var items []*contentv1.Document
	if in.GetKind() == "faq" {
		items = append(items, doc("faq", map[string]any{"question": map[string]any{"fr": "Quoi ?", "en": "What?"}, "answer": map[string]any{"fr": "Ça."}, "published": true}))
	}
	return &contentv1.ListDocumentsResponse{Items: items, Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: int64(len(items)), TotalPages: 1}}, nil
}

func (f *fakeContent) GetDocument(_ context.Context, in *contentv1.GetDocumentRequest, _ ...grpc.CallOption) (*contentv1.GetDocumentResponse, error) {
	f.calls["Get:"+in.GetKind()]++
	switch {
	case in.GetKind() == "page" && in.GetIdOrKey() == "brouillon":
		return &contentv1.GetDocumentResponse{Document: doc("page", map[string]any{"slug": "brouillon", "kind": "legal", "title": map[string]any{"fr": "T"}, "body": map[string]any{"fr": "B"}, "published": false})}, nil
	case in.GetKind() == "page" && in.GetIdOrKey() == "cgv":
		return &contentv1.GetDocumentResponse{Document: doc("page", map[string]any{"slug": "cgv", "kind": "legal", "title": map[string]any{"fr": "CGV"}, "body": map[string]any{"fr": "Texte"}, "published": true})}, nil
	case in.GetKind() == "promo_code" && in.GetIdOrKey() == "BIENVENUE10":
		return &contentv1.GetDocumentResponse{Document: doc("promo_code", map[string]any{"code": "BIENVENUE10", "type": "percent", "value": 10.0, "minSubtotal": 50000.0, "active": true})}, nil
	}
	return nil, status.Error(codes.NotFound, "absent")
}

func (f *fakeContent) CreateDocument(_ context.Context, in *contentv1.CreateDocumentRequest, _ ...grpc.CallOption) (*contentv1.CreateDocumentResponse, error) {
	f.calls["Create:"+in.GetKind()]++
	f.last = in.GetData()
	return &contentv1.CreateDocumentResponse{Document: doc(in.GetKind(), in.GetData().AsMap())}, nil
}

func (f *fakeContent) UpdateDocument(_ context.Context, in *contentv1.UpdateDocumentRequest, _ ...grpc.CallOption) (*contentv1.UpdateDocumentResponse, error) {
	f.calls["Update:"+in.GetKind()]++
	m := in.GetData().AsMap()
	if in.GetKind() == "article" { // le service renvoie le document complet
		m["slug"], m["title"], m["body"], m["kind"] = "a", map[string]any{"fr": "T"}, map[string]any{"fr": "B"}, "advice"
	}
	return &contentv1.UpdateDocumentResponse{Document: doc(in.GetKind(), m)}, nil
}

func contentRig(t *testing.T, f *fakeContent) *rig {
	return newRigDeps(t, nil, func(d *server.Deps) { d.Content = f })
}

func TestPermissionsContenu(t *testing.T) {
	f := &fakeContent{calls: map[string]int{}}
	r := contentRig(t, f)
	sub, id := uuid.NewString(), uuid.NewString()
	faq := `{"question":{"fr":"Q"},"answer":{"fr":"R"},"published":true}`
	for _, c := range []struct {
		name, method, path, body string
		perms                    []string
		want                     int
	}{
		{"liste sans permission", "GET", "/v1/admin/faq", "", nil, 403},
		{"liste avec content:read", "GET", "/v1/admin/faq", "", []string{"content:read"}, 200},
		{"création avec content:read seulement", "POST", "/v1/admin/faq", faq, []string{"content:read"}, 403},
		{"création avec content:write", "POST", "/v1/admin/faq", faq, []string{"content:write"}, 201},
		{"zones : shipping:write exigée", "POST", "/v1/admin/shipping-zones", `{"code":"abj","name":{"fr":"Abidjan"},"fee":1,"daysMin":1,"daysMax":2}`, []string{"content:write"}, 403},
		{"codes promo : pricing:write exigée", "PUT", "/v1/admin/promo-codes/" + id, `{"code":"ABC","type":"percent","value":5}`, []string{"shipping:write"}, 403},
		{"réglages du site", "PUT", "/v1/admin/site-settings", `{"name":"N","publisher":"P"}`, []string{"content:write"}, 200},
		{"publier un article : content:write ne suffit pas", "POST", "/v1/admin/articles/" + id + "/publish", "", []string{"content:write"}, 403},
		{"publier un article avec content:publish", "POST", "/v1/admin/articles/" + id + "/publish", "", []string{"content:publish"}, 200},
		{"l'état de publication ne se règle pas par le formulaire", "POST", "/v1/admin/articles", `{"slug":"a","title":{"fr":"T"},"body":{"fr":"B"},"kind":"advice","status":"published"}`, []string{"content:write"}, 400},
		{"champ inconnu refusé par le contrat", "POST", "/v1/admin/faq", `{"question":{"fr":"Q"},"answer":{"fr":"R"},"intrus":1}`, []string{"content:write"}, 400},
	} {
		rec := r.do(req{c.method, c.path, c.body, bearer(staffToken(t, r, sub, c.perms...))})
		if rec.Code != c.want {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
		}
	}
	if f.calls["Create:shipping_zone"] != 0 || f.calls["Update:promo_code"] != 0 {
		t.Fatalf("appels relayés malgré le refus : %v", f.calls)
	}
}

func TestVitrineContenu(t *testing.T) {
	f := &fakeContent{calls: map[string]int{}}
	r := contentRig(t, f)
	rec := r.do(req{"GET", "/v1/faq", "", map[string]string{"Accept-Language": "en"}})
	if rec.Code != 200 || f.where["published"] != "true" {
		t.Fatalf("faq : %d %s %v", rec.Code, rec.Body.String(), f.where)
	}
	if got := rec.Body.String(); !contains(got, `"question":"What?"`) || !contains(got, `"answer":"Ça."`) {
		t.Fatalf("langue ou repli : %s", got)
	}
	if rec := r.do(req{"GET", "/v1/pages/cgv", "", nil}); rec.Code != 200 {
		t.Fatalf("page publiée : %d", rec.Code)
	}
	if rec := r.do(req{"GET", "/v1/pages/brouillon", "", nil}); rec.Code != 404 {
		t.Fatalf("page non publiée : %d, attendu 404", rec.Code)
	}
	// formulaire : enregistré avec le statut « new », sans rien renvoyer
	rec = r.do(req{"POST", "/v1/contact", `{"name":"A","email":"a@b.co","message":"Bonjour"}`, nil})
	if rec.Code != 202 || f.last.AsMap()["status"] != "new" {
		t.Fatalf("contact : %d %s %v", rec.Code, rec.Body.String(), f.last)
	}
	// code promotionnel : valide, sous le minimum, inconnu
	for _, c := range []struct{ body, want string }{
		{`{"code":"bienvenue10","subtotal":100000}`, `"status":"valid"`},
		{`{"code":"BIENVENUE10","subtotal":1000}`, `"status":"minimum"`},
		{`{"code":"NOPE","subtotal":100000}`, `"status":"invalid"`},
	} {
		rec := r.do(req{"POST", "/v1/promo-codes/validate", c.body, nil})
		if rec.Code != 200 || !contains(rec.Body.String(), c.want) {
			t.Errorf("promo %s : %d %s", c.body, rec.Code, rec.Body.String())
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func (f *fakeContent) PutSettings(_ context.Context, in *contentv1.PutSettingsRequest, _ ...grpc.CallOption) (*contentv1.PutSettingsResponse, error) {
	f.calls["Put:"+in.GetKey()]++
	return &contentv1.PutSettingsResponse{Data: in.GetData(), Version: 1, UpdatedAt: timestamppb.Now()}, nil
}
