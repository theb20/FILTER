package server_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"google.golang.org/grpc"
)

var lastPrefsGet *identityv1.GetPreferencesRequest
var lastPrefsPut *identityv1.PutPreferencesRequest

func (f *fakeID) GetPreferences(_ context.Context, in *identityv1.GetPreferencesRequest, _ ...grpc.CallOption) (*identityv1.GetPreferencesResponse, error) {
	lastPrefsGet = in
	return &identityv1.GetPreferencesResponse{Preferences: &identityv1.Preferences{Guide: &identityv1.GuidePreferences{Enabled: true, Animations: "auto"}}}, nil
}

func (f *fakeID) PutPreferences(_ context.Context, in *identityv1.PutPreferencesRequest, _ ...grpc.CallOption) (*identityv1.PutPreferencesResponse, error) {
	lastPrefsPut = in
	return &identityv1.PutPreferencesResponse{Preferences: in.GetPreferences()}, nil
}

func customerTokenFor(t *testing.T, r *rig, sub string) string {
	t.Helper()
	tok, err := r.keys[jwtx.AudCustomer].Sign(jwtx.AudCustomer, sub, time.Minute, time.Now(), jwtx.Claims{SID: "sid-c", Kind: "individual"})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

const prefsBody = `{"guide":{"enabled":false,"animations":"off","seen":["dashboard","content-faq-form"]}}`

func TestPreferencesDuPersonnel(t *testing.T) {
	r := newRig(t, nil)
	sub := uuid.NewString()
	tok := staffToken(t, r, sub) // aucune permission : c'est du libre-service

	if rec := r.do(req{"GET", "/v1/admin/me/preferences", "", nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	if rec := r.do(req{"GET", "/v1/admin/me/preferences", "", bearer(customerToken(t, r, "s", time.Minute))}); rec.Code != 401 {
		t.Fatalf("jeton client sur une route du personnel : %d", rec.Code)
	}

	rec := r.do(req{"GET", "/v1/admin/me/preferences", "", bearer(tok)})
	if rec.Code != 200 {
		t.Fatalf("lecture : %d %s", rec.Code, rec.Body.String())
	}
	if lastPrefsGet.GetKind() != identityv1.PrincipalKind_PRINCIPAL_KIND_STAFF || lastPrefsGet.GetPrincipalId() != sub {
		t.Fatalf("le compte doit être celui du jeton, dans le monde du personnel : %+v", lastPrefsGet)
	}
	var got map[string]map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if seen, ok := got["guide"]["seen"].([]any); !ok || len(seen) != 0 {
		t.Fatalf("`seen` doit être [] et non null : %s", rec.Body.String())
	}

	rec = r.do(req{"PUT", "/v1/admin/me/preferences", prefsBody, bearer(tok)})
	if rec.Code != 200 {
		t.Fatalf("écriture : %d %s", rec.Code, rec.Body.String())
	}
	g := lastPrefsPut.GetPreferences().GetGuide()
	if lastPrefsPut.GetKind() != identityv1.PrincipalKind_PRINCIPAL_KIND_STAFF || lastPrefsPut.GetPrincipalId() != sub || g.GetEnabled() || g.GetAnimations() != "off" || len(g.GetSeen()) != 2 {
		t.Fatalf("requête transmise à identity : %+v", lastPrefsPut)
	}
}

func TestPreferencesDuClient(t *testing.T) {
	r := newRig(t, nil)
	sub := uuid.NewString()
	tok := customerTokenFor(t, r, sub)

	if rec := r.do(req{"PUT", "/v1/me/preferences", prefsBody, nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	if rec := r.do(req{"PUT", "/v1/me/preferences", prefsBody, bearer(staffToken(t, r, sub, "*"))}); rec.Code != 401 {
		t.Fatalf("jeton du personnel sur une route client : %d", rec.Code)
	}
	if rec := r.do(req{"PUT", "/v1/me/preferences", prefsBody, bearer(tok)}); rec.Code != 200 {
		t.Fatalf("écriture : %d", rec.Code)
	}
	if lastPrefsPut.GetKind() != identityv1.PrincipalKind_PRINCIPAL_KIND_CUSTOMER || lastPrefsPut.GetPrincipalId() != sub {
		t.Fatalf("compte client attendu : %+v", lastPrefsPut)
	}
	if rec := r.do(req{"GET", "/v1/me/preferences", "", bearer(tok)}); rec.Code != 200 || lastPrefsGet.GetKind() != identityv1.PrincipalKind_PRINCIPAL_KIND_CUSTOMER {
		t.Fatalf("lecture : %d %+v", rec.Code, lastPrefsGet)
	}
}

// Le compte n'est jamais lu dans la requête : un identifiant glissé dans le corps est refusé (additionalProperties: false)
// et, de toute façon, jamais utilisé.
func TestPreferencesRefusentLesChampsInconnusEtLesValeursInvalides(t *testing.T) {
	r := newRig(t, nil)
	tok := bearer(customerTokenFor(t, r, uuid.NewString()))
	for name, body := range map[string]string{
		"identifiant d'un autre compte":  `{"userId":"` + uuid.NewString() + `","guide":{"enabled":true,"animations":"auto","seen":[]}}`,
		"animations inconnues":           `{"guide":{"enabled":true,"animations":"fast","seen":[]}}`,
		"identifiant de visite invalide": `{"guide":{"enabled":true,"animations":"auto","seen":["Dashboard"]}}`,
		"visites en double":              `{"guide":{"enabled":true,"animations":"auto","seen":["a","a"]}}`,
		"section guide absente":          `{}`,
	} {
		lastPrefsPut = nil
		rec := r.do(req{"PUT", "/v1/me/preferences", body, tok})
		if rec.Code != 400 && rec.Code != 422 {
			t.Errorf("%s : %d, attendu 400 ou 422", name, rec.Code)
		}
		if lastPrefsPut != nil {
			t.Errorf("%s : identity a été appelé malgré une requête invalide", name)
		}
	}
}
