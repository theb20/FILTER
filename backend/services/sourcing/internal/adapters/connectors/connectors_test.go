package connectors_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/services/sourcing/internal/adapters/connectors"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/application"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
)

const cleTest = "cle-de-test-factice"

func brave(t *testing.T, h http.HandlerFunc, key string) (*connectors.Brave, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &connectors.Brave{Endpoint: srv.URL + "/res/v1/web/search", Key: func() string { return key }, Client: srv.Client()}, srv
}

func TestBraveEnvoieLaCleLesParametresEtLitLesResultats(t *testing.T) {
	var gotKey, gotPath, gotQ, gotCountry, gotLang, gotCount string
	b, _ := brave(t, func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotPath = r.Header.Get("X-Subscription-Token"), r.URL.Path
		gotQ, gotCountry, gotLang, gotCount = r.URL.Query().Get("q"), r.URL.Query().Get("country"), r.URL.Query().Get("search_lang"), r.URL.Query().Get("count")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"Roulement 6204","url":"https://a.test/1","description":"x"},{"title":"","url":"https://a.test/2"},{"title":"Autre","url":"https://a.test/3"}]}}`))
	}, cleTest)
	hits, err := b.Search(context.Background(), application.SearchParams{Query: "roulement 6204 & co", Language: "FR", Country: "ci", Count: 10})
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != cleTest || gotPath != "/res/v1/web/search" || gotQ != "roulement 6204 & co" || gotCountry != "CI" || gotLang != "fr" || gotCount != "10" {
		t.Fatalf("requête : clé=%q chemin=%q q=%q pays=%q langue=%q count=%q", gotKey, gotPath, gotQ, gotCountry, gotLang, gotCount)
	}
	if len(hits) != 2 || hits[0].Title != "Roulement 6204" || *hits[0].URL != "https://a.test/1" {
		t.Fatalf("résultats : %+v", hits)
	}
	// aucune référence ni marque structurées : ces résultats ne seront JAMAIS fusionnés (clé de fusion de la phase 1)
	if hits[0].Reference != nil || hits[0].Brand != nil {
		t.Fatal("Brave ne fournit ni référence ni marque : elles doivent rester absentes")
	}
}

func TestBraveClassementDesErreurs(t *testing.T) {
	for _, c := range []struct {
		status    int
		kind      string
		retryable bool
		sent      bool
	}{{429, "quota", false, true}, {500, "http_5xx", true, true}, {503, "http_5xx", true, true}, {401, "http_4xx", false, true}, {422, "http_4xx", false, true}} {
		b, _ := brave(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(c.status) }, cleTest)
		_, err := b.Search(context.Background(), application.SearchParams{Query: "x"})
		var ce *application.CallError
		if !errors.As(err, &ce) || ce.Kind != c.kind || ce.Retryable != c.retryable || ce.Sent != c.sent || ce.HTTPStatus != c.status {
			t.Errorf("statut %d : %+v", c.status, ce)
		}
	}
	b, _ := brave(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }, cleTest)
	if _, err := b.Search(context.Background(), application.SearchParams{Query: "x"}); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("réponse illisible : %v", err)
	}
}

func TestBraveSansCleOuReseauHorsService(t *testing.T) {
	b, _ := brave(t, func(w http.ResponseWriter, _ *http.Request) { t.Error("aucun appel sans clé") }, "  ")
	_, err := b.Search(context.Background(), application.SearchParams{Query: "x"})
	var ce *application.CallError
	if !errors.As(err, &ce) || ce.Kind != "secret_missing" || ce.Retryable {
		t.Fatalf("clé absente : %v", err)
	}
	b2, srv := brave(t, func(http.ResponseWriter, *http.Request) {}, cleTest)
	srv.Close()
	_, err = b2.Search(context.Background(), application.SearchParams{Query: "x"})
	if !errors.As(err, &ce) || ce.Kind != "network" || !ce.Retryable || ce.Sent {
		t.Fatalf("réseau : %v", err)
	}
	if strings.Contains(err.Error(), cleTest) || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("l'erreur ne doit révéler ni la clé ni l'adresse : %v", err)
	}
}

func TestBraveRespecteLeDelai(t *testing.T) {
	b, _ := brave(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(300 * time.Millisecond) }, cleTest)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := b.Search(ctx, application.SearchParams{Query: "x"})
	if !application.IsTimeout(err) || time.Since(start) > 250*time.Millisecond {
		t.Fatalf("délai : %v en %v", err, time.Since(start))
	}
}

func TestLaCleNEstJamaisDansLesErreursNiLesJournaux(t *testing.T) {
	b, _ := brave(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }, cleTest)
	_, err := b.Search(context.Background(), application.SearchParams{Query: cleTest})
	if err == nil || strings.Contains(err.Error(), "BSA-") && !strings.Contains(err.Error(), "5") {
		t.Fatalf("erreur : %v", err)
	}
}

func TestChoixDuConnecteur(t *testing.T) {
	env := func(n string) string { return map[string]string{"BRAVE_SEARCH_API_KEY": cleTest}[n] }
	f := &connectors.Factory{Env: env}
	src := func(u string) *domain.Source { return &domain.Source{BaseURL: u, SecretRef: "BRAVE_SEARCH_API_KEY"} }
	if c, err := f.For(src("https://api.search.brave.com/res/v1/web/search")); err != nil || c == nil {
		t.Fatalf("Brave : %v", err)
	}
	if _, err := f.For(src("https://autre.example/search")); err == nil {
		t.Fatal("adresse inconnue : aucun connecteur attendu")
	}
	if _, err := f.For(src("http://api.search.brave.com/res/v1/web/search")); err == nil {
		t.Fatal("Brave n'est joint qu'en HTTPS")
	}
	if _, err := f.For(src("test://x")); err == nil {
		t.Fatal("le connecteur de test est interdit tant que AllowTest est faux (production)")
	}
	f.AllowTest = true
	f.RegisterTest("x", &connectors.Fake{})
	if _, err := f.For(src("test://x")); err != nil {
		t.Fatal(err)
	}
}
