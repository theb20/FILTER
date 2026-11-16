package server_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeSourcing struct {
	sourcingv1.SourcingServiceClient
	calls  map[string]int
	actors []string
}

func (f *fakeSourcing) note(ctx context.Context, name string) {
	f.calls[name]++
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if v := md.Get("x-actor-id"); len(v) > 0 {
			f.actors = append(f.actors, v[0])
		}
	}
}

func (f *fakeSourcing) ListSources(ctx context.Context, _ *sourcingv1.ListSourcesRequest, _ ...grpc.CallOption) (*sourcingv1.ListSourcesResponse, error) {
	f.note(ctx, "ListSources")
	return &sourcingv1.ListSourcesResponse{Items: []*sourcingv1.Source{{Id: uuid.NewString(), Kind: sourcingv1.SourceKind_SOURCE_KIND_WEB_SEARCH_API, Name: "Brave", BaseUrl: "https://api.search.brave.com/res/v1/web/search",
		Priority: 0, TimeoutMs: 3000, MaxRetries: 2, SecretRef: "BRAVE_SEARCH_API_KEY", CostPerCallMicros: 5000, Currency: "USD", CreatedAt: timestamppb.Now()}}, Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: 1, TotalPages: 1}}, nil
}

func (f *fakeSourcing) CreateSource(ctx context.Context, in *sourcingv1.CreateSourceRequest, _ ...grpc.CallOption) (*sourcingv1.CreateSourceResponse, error) {
	f.note(ctx, "CreateSource")
	s := in.GetSource()
	s.Id, s.CreatedAt = uuid.NewString(), timestamppb.Now()
	return &sourcingv1.CreateSourceResponse{Source: s}, nil
}

func (f *fakeSourcing) PutSourceQuota(ctx context.Context, in *sourcingv1.PutSourceQuotaRequest, _ ...grpc.CallOption) (*sourcingv1.PutSourceQuotaResponse, error) {
	f.note(ctx, "PutSourceQuota")
	return &sourcingv1.PutSourceQuotaResponse{Quota: in.GetQuota()}, nil
}

func (f *fakeSourcing) ExportSourceCosts(ctx context.Context, _ *sourcingv1.ExportSourceCostsRequest, _ ...grpc.CallOption) (*sourcingv1.ExportSourceCostsResponse, error) {
	f.note(ctx, "ExportSourceCosts")
	return &sourcingv1.ExportSourceCostsResponse{Csv: []byte("source;devise\nBrave;USD\n")}, nil
}

const braveBody = `{"kind":"web_search_api","name":"Brave","baseUrl":"https://api.search.brave.com/res/v1/web/search","priority":0,"timeoutMs":3000,"maxRetries":2,"secretRef":"BRAVE_SEARCH_API_KEY","costPerCallMicros":5000,"currency":"USD"}`

func TestPermissionsEngineSurLesRoutesDesSources(t *testing.T) {
	f := &fakeSourcing{calls: map[string]int{}}
	r := newRigWith(t, nil, f)
	sub := uuid.NewString()
	id := uuid.NewString()
	quota := `{"monthlyBudgetMicros":5000000,"dailyCapMicros":333333,"alertThresholdPct":80,"fallbackSourceId":null}`
	for _, c := range []struct {
		name, method, path, body string
		perms                    []string
		want                     int
		call                     string
	}{
		{"liste sans permission", "GET", "/v1/admin/sources", "", nil, 403, ""},
		{"liste avec une mauvaise permission", "GET", "/v1/admin/sources", "", []string{"catalog:read"}, 403, ""},
		{"liste avec engine:read", "GET", "/v1/admin/sources", "", []string{"engine:read"}, 200, "ListSources"},
		{"création avec engine:read seulement", "POST", "/v1/admin/sources", braveBody, []string{"engine:read"}, 403, ""},
		{"création avec engine:write", "POST", "/v1/admin/sources", braveBody, []string{"engine:write"}, 201, "CreateSource"},
		{"budget avec engine:write seulement (engage de l'argent)", "PUT", "/v1/admin/sources/" + id + "/quota", quota, []string{"engine:write"}, 403, ""},
		{"budget avec engine:budget", "PUT", "/v1/admin/sources/" + id + "/quota", quota, []string{"engine:budget"}, 200, "PutSourceQuota"},
		{"export sans engine:export", "GET", "/v1/admin/source-costs/export", "", []string{"engine:read"}, 403, ""},
		{"export avec engine:export", "GET", "/v1/admin/source-costs/export", "", []string{"engine:export"}, 200, "ExportSourceCosts"},
		{"joker", "GET", "/v1/admin/sources", "", []string{"*"}, 200, "ListSources"},
	} {
		before := f.calls[c.call]
		rec := r.do(req{c.method, c.path, c.body, bearer(staffToken(t, r, sub, c.perms...))})
		if rec.Code != c.want {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
			continue
		}
		if c.call == "" && len(f.calls) > 0 {
			for name, n := range f.calls {
				if n > 0 && name == "" {
					t.Errorf("%s : appel interdit", c.name)
				}
			}
		}
		if c.call != "" && f.calls[c.call] != before+1 {
			t.Errorf("%s : le service sourcing n'a pas été appelé", c.name)
		}
	}
	if len(f.actors) == 0 || f.actors[0] != sub {
		t.Fatalf("l'acteur doit être transmis à sourcing : %v", f.actors)
	}
}

func TestExportCSVDesCouts(t *testing.T) {
	f := &fakeSourcing{calls: map[string]int{}}
	r := newRigWith(t, nil, f)
	rec := r.do(req{"GET", "/v1/admin/source-costs/export", "", bearer(staffToken(t, r, uuid.NewString(), "engine:export"))})
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/csv") || !strings.Contains(rec.Body.String(), "Brave;USD") {
		t.Fatalf("export : %d %q %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
}

func TestSourcesSansServiceConfigureRepondent501(t *testing.T) {
	r := newRig(t, nil)
	rec := r.do(req{"GET", "/v1/admin/sources", "", bearer(staffToken(t, r, uuid.NewString(), "engine:read"))})
	if rec.Code != 501 {
		t.Fatalf("service non configuré : %d %s", rec.Code, rec.Body.String())
	}
}

func TestSourceInvalideRefuseeParLeContratAvantIdentity(t *testing.T) {
	f := &fakeSourcing{calls: map[string]int{}}
	r := newRigWith(t, nil, f)
	tok := bearer(staffToken(t, r, uuid.NewString(), "engine:write"))
	for name, body := range map[string]string{
		"type inconnu":    strings.Replace(braveBody, "web_search_api", "magie", 1),
		"champ inconnu":   strings.Replace(braveBody, `"currency":"USD"`, `"currency":"USD","apiKey":"secret"`, 1),
		"devise invalide": strings.Replace(braveBody, `"USD"`, `"usd"`, 1),
		"délai nul":       strings.Replace(braveBody, `"timeoutMs":3000`, `"timeoutMs":0`, 1),
	} {
		rec := r.do(req{"POST", "/v1/admin/sources", body, tok})
		if rec.Code != 400 && rec.Code != 422 {
			t.Errorf("%s : %d %s", name, rec.Code, rec.Body.String())
		}
	}
	if f.calls["CreateSource"] != 0 {
		t.Fatal("une requête invalide ne doit pas atteindre le service")
	}
}
