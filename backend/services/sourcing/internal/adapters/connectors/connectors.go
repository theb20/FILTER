// Package connectors : connecteurs vers les sources externes. Brave Search API en production ; un connecteur de TEST contrôlé
// (schéma test://) pour les tests automatisés, jamais utilisable pour une source réelle.
package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/theb20/FILTER/backend/services/sourcing/internal/application"
	"github.com/theb20/FILTER/backend/services/sourcing/internal/domain"
)

const braveHost = "api.search.brave.com"

// Factory choisit le connecteur d'après l'adresse de la source. Le secret est lu À L'APPEL (jamais mis en cache ni journalisé).
type Factory struct {
	Env        func(name string) string // résolution du secret (os.Getenv en production)
	HTTP       *http.Client
	AllowTest  bool // autorise le schéma test:// (tests et développement)
	mu         sync.Mutex
	testByName map[string]application.Connector
}

var _ application.ConnectorFactory = (*Factory)(nil)

// RegisterTest enregistre un connecteur de test sous test://<name>.
func (f *Factory) RegisterTest(name string, c application.Connector) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.testByName == nil {
		f.testByName = map[string]application.Connector{}
	}
	f.testByName[name] = c
}

func (f *Factory) For(src *domain.Source) (application.Connector, error) {
	u, err := url.Parse(src.BaseURL)
	if err != nil {
		return nil, err
	}
	switch {
	case u.Scheme == "test":
		if !f.AllowTest {
			return nil, errors.New("connecteur de test interdit ici")
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if c, ok := f.testByName[u.Host]; ok {
			return c, nil
		}
		return nil, fmt.Errorf("connecteur de test %q inconnu", u.Host)
	case u.Scheme == "https" && u.Host == braveHost:
		cl := f.HTTP
		if cl == nil {
			cl = &http.Client{}
		}
		env := f.Env
		return &Brave{Endpoint: src.BaseURL, Key: func() string { return env(src.SecretRef) }, Client: cl}, nil
	}
	return nil, fmt.Errorf("aucun connecteur pour %s", src.BaseURL)
}

// ── Brave Search API ────────────────────────────────────────────────────────

// Brave interroge `GET /res/v1/web/search` (en-tête X-Subscription-Token ; paramètres q, country, search_lang, count).
// Les résultats de Brave ne portent ni référence ni marque structurées : Reference et Brand restent absents, donc ces
// résultats ne sont JAMAIS fusionnés entre eux (règle de la phase 1).
type Brave struct {
	Endpoint string
	Key      func() string
	Client   *http.Client
}

func (b *Brave) Search(ctx context.Context, p application.SearchParams) ([]domain.RawHit, error) {
	key := ""
	if b.Key != nil {
		key = strings.TrimSpace(b.Key())
	}
	if key == "" {
		return nil, &application.CallError{Kind: "secret_missing", Err: errors.New("clé d'API absente")}
	}
	q := url.Values{"q": {p.Query}}
	if p.Language != "" {
		q.Set("search_lang", strings.ToLower(p.Language))
	}
	if p.Country != "" {
		q.Set("country", strings.ToUpper(p.Country))
	}
	if p.Count > 0 {
		q.Set("count", strconv.Itoa(p.Count))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.Endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, &application.CallError{Kind: "network", Err: err}
	}
	req.Header.Set("X-Subscription-Token", key)
	req.Header.Set("Accept", "application/json")
	res, err := b.Client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var ne net.Error
		_ = errors.As(err, &ne)
		return nil, &application.CallError{Kind: "network", Retryable: true, Err: errors.New("échec réseau")} // le détail (adresse, clé en en-tête) n'est jamais conservé
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	switch {
	case res.StatusCode == http.StatusTooManyRequests:
		return nil, &application.CallError{Kind: "quota", HTTPStatus: 429, Sent: true, Err: errors.New("quota ou débit dépassé")}
	case res.StatusCode >= 500:
		return nil, &application.CallError{Kind: "http_5xx", HTTPStatus: res.StatusCode, Retryable: true, Sent: true, Err: fmt.Errorf("réponse %d", res.StatusCode)}
	case res.StatusCode >= 400:
		return nil, &application.CallError{Kind: "http_4xx", HTTPStatus: res.StatusCode, Sent: true, Err: fmt.Errorf("réponse %d", res.StatusCode)}
	}
	var doc struct {
		Web struct {
			Results []struct {
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, &application.CallError{Kind: "decode", Sent: true, Err: errors.New("réponse illisible")}
	}
	hits := make([]domain.RawHit, 0, len(doc.Web.Results))
	for _, r := range doc.Web.Results {
		if r.Title == "" {
			continue
		}
		u := r.URL
		raw, _ := json.Marshal(r)
		hits = append(hits, domain.RawHit{Title: r.Title, URL: &u, PayloadJSON: string(raw)})
	}
	return hits, nil
}

// ── connecteur de test contrôlé ─────────────────────────────────────────────

// Fake : comportement scripté pour les tests (succès, échecs, lenteur) ; compte les appels.
type Fake struct {
	mu     sync.Mutex
	Calls  int
	Script []func(ctx context.Context) ([]domain.RawHit, error) // consommé un à un ; au-delà, dernier élément répété
	Hits   []domain.RawHit
	Delay  time.Duration
}

func (f *Fake) Search(ctx context.Context, _ application.SearchParams) ([]domain.RawHit, error) {
	f.mu.Lock()
	i := f.Calls
	f.Calls++
	var step func(context.Context) ([]domain.RawHit, error)
	if len(f.Script) > 0 {
		if i >= len(f.Script) {
			i = len(f.Script) - 1
		}
		step = f.Script[i]
	}
	delay, hits := f.Delay, f.Hits
	f.mu.Unlock()
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if step != nil {
		return step(ctx)
	}
	return hits, nil
}

func (f *Fake) CallCount() int { f.mu.Lock(); defer f.mu.Unlock(); return f.Calls }
