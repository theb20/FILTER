package application_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
)

// ── mot de passe oublié ─────────────────────────────────────────────────────

func resetCode(t *testing.T, e *env) string {
	t.Helper()
	ev := e.events.OfType(application.EvPasswordResetRequested)
	if len(ev) == 0 {
		t.Fatal("aucun code envoyé")
	}
	return ev[len(ev)-1].Payload.(*eventsv1.PasswordResetRequested).Code
}

func testMotDePasseOublieParcoursComplet(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "a@b.ci", pw)
	old, _ := login(e, "a@b.ci", pw, "1.1.1.1")
	if err := e.svc.ForgotPassword(ctx, "A@B.ci", "fr"); err != nil {
		t.Fatal(err)
	}
	c := resetCode(t, e)
	if len(c) != 6 {
		t.Fatalf("code à 6 chiffres attendu : %q", c)
	}
	if _, _, err := e.svc.VerifyPasswordCode(ctx, "a@b.ci", "000000"); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("code faux : %v", err)
	}
	tok, _, err := e.svc.VerifyPasswordCode(ctx, "a@b.ci", c)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.svc.VerifyPasswordCode(ctx, "a@b.ci", c); err == nil {
		t.Fatal("un code ne sert qu'une fois")
	}
	if err := e.svc.ResetPassword(ctx, tok, "court"); err == nil {
		t.Fatal("mot de passe trop court refusé")
	}
	if err := e.svc.ResetPassword(ctx, tok, "un tout nouveau mot de passe"); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.ResetPassword(ctx, tok, "encore un autre mot de passe"); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("le jeton ne sert qu'une fois : %v", err)
	}
	if _, err := login(e, "a@b.ci", pw, "2.2.2.2"); err == nil {
		t.Fatal("l'ancien mot de passe doit être refusé")
	}
	if _, err := login(e, "a@b.ci", "un tout nouveau mot de passe", "2.2.2.2"); err != nil {
		t.Fatalf("le nouveau mot de passe doit fonctionner : %v", err)
	}
	if _, err := e.svc.Refresh(ctx, old.RefreshToken, "", ""); err == nil {
		t.Fatal("tous les appareils doivent être déconnectés")
	}
	u, _ := e.store.ByEmail(ctx, "a@b.ci")
	if !u.EmailVerified {
		t.Fatal("le code reçu par e-mail prouve la possession de l'adresse")
	}
}

func testMotDePasseOublieAucuneFuiteEtEssaisRalentis(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "a@b.ci", pw)
	if err := e.svc.ForgotPassword(ctx, "inconnu@b.ci", "fr"); err != nil {
		t.Fatalf("compte inconnu : réponse identique attendue : %v", err)
	}
	if n := len(e.events.OfType(application.EvPasswordResetRequested)); n != 0 {
		t.Fatalf("aucun envoi pour un compte inconnu : %d", n)
	}
	if err := e.svc.ForgotPassword(ctx, "pas-un-email", "fr"); code(err) != apperr.CodeValidation {
		t.Fatalf("la forme invalide est signalée : %v", err)
	}
	// deviner le code est ralenti : après plusieurs essais faux, la demande est refusée (429) même avec le bon code
	_ = e.svc.ForgotPassword(ctx, "a@b.ci", "fr")
	good := resetCode(t, e)
	limited := false
	for i := 0; i < 8; i++ {
		if _, _, err := e.svc.VerifyPasswordCode(ctx, "a@b.ci", "11111"+string(rune('0'+i))); code(err) == apperr.CodeRateLimited {
			limited = true
		}
	}
	if !limited {
		t.Fatal("les essais répétés doivent être ralentis")
	}
	if _, _, err := e.svc.VerifyPasswordCode(ctx, "a@b.ci", good); code(err) != apperr.CodeRateLimited {
		t.Fatalf("même le bon code attend la fin du ralentissement : %v", err)
	}
}

// ── Google ──────────────────────────────────────────────────────────────────

type google struct {
	srv    *httptest.Server
	key    *rsa.PrivateKey
	claims map[string]any // jeton renvoyé au prochain échange
}

func newGoogle(t *testing.T, e *env) *google {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	g := &google{key: key}
	mux := http.NewServeMux()
	mux.HandleFunc("/certs", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kid": "k1", "kty": "RSA", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "bon-code" || r.Form.Get("code_verifier") == "" || r.Form.Get("client_secret") != "secret" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": g.sign(t, g.claims)})
	})
	g.srv = httptest.NewServer(mux)
	t.Cleanup(g.srv.Close)
	e.svc.OAuthCfg = application.OAuthConfig{
		Google:           &application.GoogleConfig{ClientID: "cid", ClientSecret: "secret", CallbackURL: "http://site.test/v1/auth/oauth/google/callback"},
		AllowedRedirects: []string{"http://site.test/oauth/callback"}, HTTP: g.srv.Client(), AuthURL: g.srv.URL + "/auth", TokenURL: g.srv.URL + "/token", JWKSURL: g.srv.URL + "/certs",
	}
	return g
}

func (g *google) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
	in := enc(map[string]string{"alg": "RS256", "kid": "k1", "typ": "JWT"}) + "." + enc(claims)
	sum := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, g.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return in + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// start déroule `start` et renvoie (state, binding, nonce) ; le nonce se lit dans l'adresse d'autorisation.
func (g *google) start(t *testing.T, e *env) (state, binding, nonce string) {
	t.Helper()
	u, state, binding, _, err := e.svc.OAuthStart(context.Background(), "google", "http://site.test/oauth/callback")
	if err != nil {
		t.Fatal(err)
	}
	q, _ := url.Parse(u)
	v := q.Query()
	if v.Get("code_challenge_method") != "S256" || v.Get("code_challenge") == "" || v.Get("state") != state || v.Get("nonce") == "" || v.Get("client_id") != "cid" || v.Get("response_type") != "code" {
		t.Fatalf("paramètres d'autorisation : %v", v)
	}
	return state, binding, v.Get("nonce")
}

func (g *google) setClaims(e *env, nonce, sub, email string, verified bool) {
	g.claims = map[string]any{"iss": "https://accounts.google.com", "aud": "cid", "sub": sub, "email": email, "email_verified": verified, "nonce": nonce,
		"exp": float64(e.clock.Now().Add(time.Hour).Unix()), "given_name": "Awa", "family_name": "Koné", "locale": "fr"}
}

func testOAuthNouveauCompteEtReconnexion(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	g := newGoogle(t, e)
	state, bind, nonce := g.start(t, e)
	g.setClaims(e, nonce, "sub-1", "Awa@Gmail.com", true)
	res := e.svc.OAuthComplete(ctx, "google", "bon-code", state, bind)
	if res.Failed || res.ExchangeCode == "" || res.RedirectURI != "http://site.test/oauth/callback" {
		t.Fatalf("résultat : %+v", res)
	}
	s, err := e.svc.ExchangeOAuthCode(ctx, res.ExchangeCode, "1.1.1.1", "ua")
	if err != nil {
		t.Fatal(err)
	}
	if s.User.Email != "awa@gmail.com" || !s.User.EmailVerified || s.User.FirstName != "Awa" || s.AccessToken == "" || s.RefreshToken == "" {
		t.Fatalf("compte créé : %+v", s.User)
	}
	if _, err := e.svc.ExchangeOAuthCode(ctx, res.ExchangeCode, "", ""); err == nil {
		t.Fatal("le code d'échange ne sert qu'une fois")
	}
	// un compte créé par Google n'a pas de mot de passe : la connexion par mot de passe est impossible
	if _, err := login(e, "awa@gmail.com", "", "1.1.1.1"); err == nil {
		t.Fatal("pas de connexion par mot de passe sans mot de passe")
	}
	// reconnexion : même identité (sujet), même compte, même si l'adresse a changé chez Google
	state, bind, nonce = g.start(t, e)
	g.setClaims(e, nonce, "sub-1", "autre@gmail.com", true)
	res = e.svc.OAuthComplete(ctx, "google", "bon-code", state, bind)
	s2, err := e.svc.ExchangeOAuthCode(ctx, res.ExchangeCode, "", "")
	if err != nil || s2.User.ID != s.User.ID {
		t.Fatalf("reconnexion : %v", err)
	}
	// le code d'échange expire en 60 s
	state, bind, nonce = g.start(t, e)
	g.setClaims(e, nonce, "sub-1", "awa@gmail.com", true)
	res = e.svc.OAuthComplete(ctx, "google", "bon-code", state, bind)
	e.clock.Advance(61 * time.Second)
	if _, err := e.svc.ExchangeOAuthCode(ctx, res.ExchangeCode, "", ""); err == nil {
		t.Fatal("code d'échange expiré")
	}
}

func testOAuthLiaisonSeulementSiAdresseVerifiee(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	g := newGoogle(t, e)
	register(t, e, "a@b.ci", pw) // adresse NON vérifiée
	state, bind, nonce := g.start(t, e)
	g.setClaims(e, nonce, "sub-9", "a@b.ci", true)
	if res := e.svc.OAuthComplete(ctx, "google", "bon-code", state, bind); !res.Failed {
		t.Fatal("un compte local non vérifié ne doit pas être repris par Google (prise de contrôle)")
	}
	if err := e.svc.VerifyEmail(ctx, verificationToken(t, e)); err != nil {
		t.Fatal(err)
	}
	state, bind, nonce = g.start(t, e)
	g.setClaims(e, nonce, "sub-9", "a@b.ci", true)
	res := e.svc.OAuthComplete(ctx, "google", "bon-code", state, bind)
	if res.Failed {
		t.Fatal("adresse vérifiée : la liaison est permise")
	}
	s, err := e.svc.ExchangeOAuthCode(ctx, res.ExchangeCode, "", "")
	local, _ := e.store.ByEmail(ctx, "a@b.ci")
	if err != nil || s.User.ID != local.ID {
		t.Fatalf("le compte existant doit être réutilisé : %v", err)
	}
}

func testOAuthRefusDesFalsifications(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	g := newGoogle(t, e)
	good := func(state, bind, nonce string) { g.setClaims(e, nonce, "sub-x", "x@gmail.com", true) }
	failed := func(name, code, state, bind string) {
		t.Helper()
		if res := e.svc.OAuthComplete(ctx, "google", code, state, bind); !res.Failed || res.ExchangeCode != "" {
			t.Errorf("%s : refus attendu, obtenu %+v", name, res)
		}
	}
	state, bind, nonce := g.start(t, e)
	good(state, bind, nonce)
	failed("state falsifié", "bon-code", "autre", bind)
	failed("sans cookie de liaison", "bon-code", state, "")
	failed("liaison corrompue", "bon-code", state, bind[:len(bind)-4]+"AAAA")
	failed("refus de l'utilisateur (pas de code)", "", state, bind)
	failed("code refusé par Google", "mauvais-code", state, bind)
	// un jeton d'identité falsifié
	for name, mut := range map[string]func(){
		"nonce différent":                 func() { g.claims["nonce"] = "autre" },
		"audience étrangère":              func() { g.claims["aud"] = "autre-appli" },
		"émetteur étranger":               func() { g.claims["iss"] = "https://evil.example" },
		"jeton expiré":                    func() { g.claims["exp"] = float64(e.clock.Now().Add(-time.Hour).Unix()) },
		"adresse non vérifiée par Google": func() { g.claims["email_verified"] = false },
	} {
		state, bind, nonce = g.start(t, e)
		good(state, bind, nonce)
		mut()
		failed(name, "bon-code", state, bind)
	}
	// signature d'une autre clé
	state, bind, nonce = g.start(t, e)
	good(state, bind, nonce)
	g.key, _ = rsa.GenerateKey(rand.Reader, 2048) // la clé publiée n'est plus celle qui signe
	failed("signature inconnue", "bon-code", state, bind)
	// liaison expirée
	g2 := newGoogle(t, e)
	state, bind, nonce = g2.start(t, e)
	g2.setClaims(e, nonce, "sub-y", "y@gmail.com", true)
	e.clock.Advance(11 * time.Minute)
	if res := e.svc.OAuthComplete(ctx, "google", "bon-code", state, bind); !res.Failed {
		t.Error("liaison expirée : refus attendu")
	}
	// adresse de retour hors liste blanche, fournisseur inconnu
	if _, _, _, _, err := e.svc.OAuthStart(ctx, "google", "http://evil.example/cb"); code(err) != apperr.CodeValidation {
		t.Errorf("redirectUri hors liste blanche : %v", err)
	}
	if _, _, _, _, err := e.svc.OAuthStart(ctx, "facebook", "http://site.test/oauth/callback"); code(err) != apperr.CodeNotFound {
		t.Errorf("fournisseur inconnu : %v", err)
	}
	e.svc.OAuthCfg.Google = nil
	if _, _, _, _, err := e.svc.OAuthStart(ctx, "google", "http://site.test/oauth/callback"); code(err) != apperr.CodeNotFound {
		t.Errorf("Google non configuré : %v", err)
	}
}
