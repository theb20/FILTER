package breach

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func sha1Upper(p string) string {
	s := sha1.Sum([]byte(p)) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(s[:]))
}

// rangeServer imite l'API par plage : renvoie les suffixes connus pour le préfixe demandé + des lignes de remplissage.
func rangeServer(t *testing.T, leaked ...string) (*httptest.Server, *atomic.Value) {
	t.Helper()
	var lastPath atomic.Value
	var lastHeader atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath.Store(r.URL.Path)
		lastHeader.Store(r.Header.Clone())
		prefix := strings.TrimPrefix(r.URL.Path, "/")
		var b strings.Builder
		for _, p := range leaked {
			if h := sha1Upper(p); strings.HasPrefix(h, prefix) {
				fmt.Fprintf(&b, "%s:%d\r\n", h[5:], 42)
			}
		}
		b.WriteString("0018A45C4D1DEF81644B54AB7F969B88D65:0\r\n") // remplissage : compteur à 0
		fmt.Fprint(w, b.String())
	}))
	t.Cleanup(srv.Close)
	return srv, &lastPath
}

func TestCompromisViaPrefixeDeCondensat(t *testing.T) {
	srv, lastPath := rangeServer(t, "correct horse battery staple")
	c := NewChecker(srv.URL+"/", time.Second, NewLocal(), nil)

	bad, err := c.Compromised(context.Background(), "correct horse battery staple")
	if err != nil || !bad {
		t.Fatalf("un mot de passe fuité doit être refusé : %v %v", bad, err)
	}
	// k-anonymat : le chemin demandé ne contient que le PRÉFIXE de 5 caractères
	full := sha1Upper("correct horse battery staple")
	if got := lastPath.Load().(string); got != "/"+full[:5] {
		t.Fatalf("seuls 5 caractères du condensat peuvent quitter le serveur : %q", got)
	}
	if strings.Contains(lastPath.Load().(string), full[5:]) {
		t.Fatal("le suffixe ne doit JAMAIS être envoyé")
	}
	ok, _ := c.Compromised(context.Background(), "un mot de passe jamais vu 7f3a9c")
	if ok {
		t.Fatal("un mot de passe absent de la plage ne doit pas être refusé")
	}
}

func TestLignesDeRemplissageIgnorees(t *testing.T) {
	// la ligne de remplissage a un compteur à 0 : même si son suffixe correspond, ce n'est PAS une fuite
	pw := "un mot de passe quelconque 1234"
	h := sha1Upper(pw)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "%s:0\r\n", h[5:]) }))
	defer srv.Close()
	if bad, _ := NewChecker(srv.URL+"/", time.Second, nil, nil).Compromised(context.Background(), pw); bad {
		t.Fatal("un compteur à 0 (remplissage) ne signale aucune fuite")
	}
}

func TestEnTeteDeRemplissageEtUserAgentNeutre(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.Header.Clone() }))
	defer srv.Close()
	_, _ = NewChecker(srv.URL+"/", time.Second, nil, nil).Compromised(context.Background(), "whatever password")
	if got.Get("Add-Padding") != "true" || got.Get("User-Agent") != "filterfinder-identity" {
		t.Fatalf("en-têtes inattendus : %v", got)
	}
	if got.Get("Authorization") != "" || got.Get("Cookie") != "" {
		t.Fatal("aucun identifiant ne doit partir vers le service externe")
	}
}

func TestRepliLocalSiLeServiceNeRepondPas(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) }))
	defer slow.Close()
	c := NewChecker(slow.URL+"/", 80*time.Millisecond, NewLocal(), nil) // délai court

	start := time.Now()
	bad, err := c.Compromised(context.Background(), "password123")
	if err != nil {
		t.Fatalf("une panne du service ne doit JAMAIS produire d'erreur (inscription non bloquée) : %v", err)
	}
	if !bad {
		t.Fatal("le repli local doit refuser un mot de passe très courant")
	}
	if time.Since(start) > 300*time.Millisecond {
		t.Fatalf("le délai court n'est pas respecté : %v", time.Since(start))
	}
	// un mot de passe inconnu de la liste locale passe : le repli n'est qu'un filet, il ne bloque pas
	if bad, err := c.Compromised(context.Background(), "mot de passe long et unique 9x!"); bad || err != nil {
		t.Fatalf("hors liste locale : accepté, sans erreur : %v %v", bad, err)
	}
}

func TestRepliLocalSurErreurHTTPEtServiceCoupe(t *testing.T) {
	e500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", 500) }))
	defer e500.Close()
	if bad, err := NewChecker(e500.URL+"/", time.Second, NewLocal(), nil).Compromised(context.Background(), "azertyuiop"); !bad || err != nil {
		t.Fatalf("HTTP 500 : repli local attendu : %v %v", bad, err)
	}
	down := httptest.NewServer(http.NotFoundHandler())
	url := down.URL + "/"
	down.Close() // connexion refusée
	if bad, err := NewChecker(url, time.Second, NewLocal(), nil).Compromised(context.Background(), "azertyuiop"); !bad || err != nil {
		t.Fatalf("service coupé : repli local attendu : %v %v", bad, err)
	}
	// redirection : non suivie, donc traitée comme une panne
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "http://evil.example/", 302) }))
	defer redir.Close()
	if _, err := NewChecker(redir.URL+"/", time.Second, NewLocal(), nil).Compromised(context.Background(), "x y z 12345"); err != nil {
		t.Fatalf("redirection : aucune erreur attendue : %v", err)
	}
}

func TestAucunSecretDansLesLogs(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	down := httptest.NewServer(http.NotFoundHandler())
	url := down.URL + "/"
	down.Close()
	pw := "mon-mot-de-passe-secret-42"
	_, _ = NewChecker(url, 100*time.Millisecond, NewLocal(), log).Compromised(context.Background(), pw)
	out := strings.ToUpper(buf.String())
	full := sha1Upper(pw)
	for _, leak := range []string{strings.ToUpper(pw), full, full[:5], full[5:]} {
		if strings.Contains(out, leak) {
			t.Fatalf("fuite dans les logs (%q) : %s", leak, buf.String())
		}
	}
	if !strings.Contains(buf.String(), "repli") {
		t.Fatalf("le repli doit être journalisé (sans secret) : %s", buf.String())
	}
}

func TestListeLocale(t *testing.T) {
	l := NewLocal("Mon Mot De Passe Maison")
	for _, p := range []string{"PASSWORD123", "  azertyuiop ", "mon mot de passe maison"} {
		if !l.Contains(p) {
			t.Errorf("%q doit être reconnu (insensible à la casse et aux espaces)", p)
		}
	}
	if l.Contains("un mot de passe robuste et long") {
		t.Error("faux positif")
	}
}
