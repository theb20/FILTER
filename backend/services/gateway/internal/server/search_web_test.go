package server_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// webSourcing : service sourcing simulé, pour vérifier QUI est appelé et COMBIEN de fois (chaque appel est facturé).
type webSourcing struct {
	sourcingv1.SourcingServiceClient
	enabled bool
	images  bool
	status  sourcingv1.CallStatus
	calls   int
	query   string
}

func (f *webSourcing) ListSources(context.Context, *sourcingv1.ListSourcesRequest, ...grpc.CallOption) (*sourcingv1.ListSourcesResponse, error) {
	src := func(id, name, path string) *sourcingv1.Source {
		return &sourcingv1.Source{
			Id: id, Kind: sourcingv1.SourceKind_SOURCE_KIND_WEB_SEARCH_API, Name: name,
			BaseUrl: "https://api.search.brave.com/res/v1" + path, Enabled: f.enabled, Priority: 0, TimeoutMs: 3000,
			SecretRef: "BRAVE_SEARCH_API_KEY", CostPerCallMicros: 5000, Currency: "USD", CreatedAt: timestamppb.Now(),
		}
	}
	items := []*sourcingv1.Source{src("01a1ebe4-602c-7403-a667-8a40c900194d", "Brave Search", "/web/search")}
	if f.images {
		items = append(items, src("01a1ebe4-602c-7403-a667-8a40c900194e", "Brave Images", "/images/search"))
	}
	return &sourcingv1.ListSourcesResponse{Items: items, Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: int64(len(items)), TotalPages: 1}}, nil
}

func (f *webSourcing) CallSource(_ context.Context, in *sourcingv1.CallSourceRequest, _ ...grpc.CallOption) (*sourcingv1.CallSourceResponse, error) {
	f.calls++
	f.query = in.GetQuery()
	u := "https://exemple.test/filtre"
	return &sourcingv1.CallSourceResponse{
		Status: f.status,
		Hits: []*sourcingv1.RawHit{{Title: "Filtre hydraulique", Url: &u, PayloadJson: `{"description":"Un extrait de la page.","thumbnail":{"src":"https://imgs.exemple.test/v.jpg"},"profile":{"name":"HIFI FILTER"},"meta_url":{"hostname":"hifi-filter.com"}}`},
			{Title: "Vignette douteuse", Url: &u, PayloadJson: `{"thumbnail":{"src":"javascript:alert(1)"},"meta_url":{"hostname":"exemple.test"}}`}},
		Currency: "USD", CostMicros: 5000,
	}, nil
}

func webRig(t *testing.T, s *webSourcing) *rig {
	t.Helper()
	return newRigDeps(t, nil, func(d *server.Deps) { d.Sourcing = s })
}

func webSearch(r *rig, visitor, token string) int {
	h := map[string]string{"X-Visitor-Id": visitor}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	return r.do(req{"GET", "/v1/search/web?q=filtre+hydraulique", "", h}).Code
}

func TestRechercheWeb(t *testing.T) {
	src := &webSourcing{enabled: true, status: sourcingv1.CallStatus_CALL_STATUS_OK}
	r := webRig(t, src)

	// Requête trop courte : aucun appel payant n'est lancé.
	if rec := r.do(req{"GET", "/v1/search/web?q=a", "", nil}); rec.Code != 400 && rec.Code != 422 {
		t.Fatalf("requête trop courte : %d", rec.Code)
	}
	if src.calls != 0 {
		t.Fatalf("une requête invalide a provoqué %d appel(s) à la source", src.calls)
	}

	rec := r.do(req{"GET", "/v1/search/web?q=filtre+hydraulique", "", map[string]string{"X-Visitor-Id": "v-1"}})
	body := rec.Body.String()
	if rec.Code != 200 || !contains(body, `"status":"ok"`) || !contains(body, "Filtre hydraulique") || !contains(body, "Brave Search") {
		t.Fatalf("recherche : %d %s", rec.Code, body)
	}
	if !contains(body, "Un extrait de la page.") {
		t.Errorf("l'extrait de la source devrait être repris : %s", body)
	}
	// La carte affiche une vignette et le NOM DU SITE (à la place du prix, qu'une source web ne donne pas).
	if !contains(body, `"image":"https://imgs.exemple.test/v.jpg"`) || !contains(body, `"site":"HIFI FILTER"`) {
		t.Errorf("vignette ou site absent : %s", body)
	}
	// Une vignette qui n'est pas en https n'est jamais transmise au navigateur.
	if contains(body, "javascript:") {
		t.Fatalf("adresse d'image dangereuse transmise : %s", body)
	}
	if !contains(body, `"site":"exemple.test"`) { // repli sur le nom d'hôte quand le site n'a pas de nom
		t.Errorf("repli sur le nom d'hôte absent : %s", body)
	}
	if !contains(body, `"remaining":2`) { // 3 par jour pour un visiteur anonyme
		t.Errorf("quota restant absent ou faux : %s", body)
	}
	if src.query != "filtre hydraulique" {
		t.Errorf("requête transmise = %q", src.query)
	}
}

func TestRechercheWebPlafondsJournaliers(t *testing.T) {
	src := &webSourcing{enabled: true, status: sourcingv1.CallStatus_CALL_STATUS_OK}
	r := webRig(t, src)

	// Visiteur anonyme : 3 recherches par jour, la 4e est refusée SANS appeler la source.
	for i := 1; i <= 3; i++ {
		if code := webSearch(r, "v-anonyme", ""); code != 200 {
			t.Fatalf("recherche %d : %d", i, code)
		}
	}
	if code := webSearch(r, "v-anonyme", ""); code != 429 {
		t.Fatalf("4e recherche : %d (429 attendu)", code)
	}
	if src.calls != 3 {
		t.Fatalf("%d appels à la source pour 4 recherches : le plafond doit s'appliquer AVANT l'appel", src.calls)
	}

	// Client connecté : 10 par jour, donc une 4e recherche passe encore.
	tok := customerTokenFor(t, r, uuid.NewString())
	for i := 1; i <= 4; i++ {
		if code := webSearch(r, "v-client", tok); code != 200 {
			t.Fatalf("client connecté, recherche %d : %d", i, code)
		}
	}
}

func TestRechercheWebSansSourceOuSansBudget(t *testing.T) {
	// Aucune source active : 404 explicite, aucun appel.
	src := &webSourcing{enabled: false, status: sourcingv1.CallStatus_CALL_STATUS_OK}
	if code := webSearch(webRig(t, src), "v-2", ""); code != 404 {
		t.Fatalf("sans source active : %d", code)
	}
	if src.calls != 0 {
		t.Fatalf("appel lancé sans source active")
	}

	// Budget épuisé : ce n'est pas une panne du site, le visiteur reçoit une réponse claire.
	budget := &webSourcing{enabled: true, status: sourcingv1.CallStatus_CALL_STATUS_BUDGET_EXHAUSTED}
	rec := webRig(t, budget).do(req{"GET", "/v1/search/web?q=filtre+hydraulique", "", map[string]string{"X-Visitor-Id": "v-3"}})
	if rec.Code != 200 || !contains(rec.Body.String(), `"status":"budget"`) {
		t.Fatalf("budget épuisé : %d %s", rec.Code, rec.Body.String())
	}
}

// Les extraits des moteurs contiennent des balises et des entités HTML : le site les affiche comme du texte, donc le
// gateway les nettoie. Aucun fragment de HTML venu d'un tiers ne doit traverser l'API.
func TestExtraitsNettoyes(t *testing.T) {
	for raw, want := range map[string]string{
		"<strong>Filtre</strong> hydraulique": "Filtre hydraulique",
		"l&#x27;huile  &amp; l&#39;eau":       "l'huile & l'eau",
		"<script>alert(1)</script>Texte":      "alert(1)Texte",
		"  plusieurs   espaces\n et sauts  ":  "plusieurs espaces et sauts",
	} {
		if got := server.PlainTextForTest(raw); got != want {
			t.Errorf("plainText(%q) = %q, attendu %q", raw, got, want)
		}
	}
}

// Les photos d'une fiche viennent d'une source DISTINCTE (point d'entrée « images »), et consomment le même quota.
func TestPhotosDUnResultat(t *testing.T) {
	src := &webSourcing{enabled: true, images: true, status: sourcingv1.CallStatus_CALL_STATUS_OK}
	r := webRig(t, src)
	rec := r.do(req{"GET", "/v1/search/web/images?q=filtre+hydraulique", "", map[string]string{"X-Visitor-Id": "v-img"}})
	body := rec.Body.String()
	if rec.Code != 200 || !contains(body, `"status":"ok"`) || !contains(body, "Brave Images") {
		t.Fatalf("photos : %d %s", rec.Code, body)
	}
	if !contains(body, `"url":"https://imgs.exemple.test/v.jpg"`) {
		t.Errorf("vignette absente : %s", body)
	}
	if contains(body, "javascript:") { // la photo sans adresse https est écartée
		t.Fatalf("adresse dangereuse transmise : %s", body)
	}

	// Sans source « images » configurée, la fiche le dit clairement au lieu d'échouer en erreur serveur.
	noImg := &webSourcing{enabled: true, images: false, status: sourcingv1.CallStatus_CALL_STATUS_OK}
	if rec := webRig(t, noImg).do(req{"GET", "/v1/search/web/images?q=filtre", "", map[string]string{"X-Visitor-Id": "v-img2"}}); rec.Code != 404 {
		t.Fatalf("sans source images : %d", rec.Code)
	}

	// Les photos comptent dans le plafond journalier : 3 appels payants pour un visiteur anonyme, quel qu'en soit le type.
	q := &webSourcing{enabled: true, images: true, status: sourcingv1.CallStatus_CALL_STATUS_OK}
	rq := webRig(t, q)
	for i := 1; i <= 3; i++ {
		if code := webSearch(rq, "v-mix", ""); code != 200 {
			t.Fatalf("recherche %d : %d", i, code)
		}
	}
	if rec := rq.do(req{"GET", "/v1/search/web/images?q=filtre", "", map[string]string{"X-Visitor-Id": "v-mix"}}); rec.Code != 429 {
		t.Fatalf("les photos doivent partager le plafond : %d", rec.Code)
	}
}
