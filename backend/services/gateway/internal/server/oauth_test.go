package server_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/theb20/FILTER/backend/services/gateway/internal/server"

	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"google.golang.org/grpc"
)

type oauthID struct {
	*fakeID
	complete *identityv1.CompleteOAuthResponse
	lastCode string
}

func (o *oauthID) StartOAuth(_ context.Context, in *identityv1.StartOAuthRequest, _ ...grpc.CallOption) (*identityv1.StartOAuthResponse, error) {
	return &identityv1.StartOAuthResponse{AuthorizationUrl: "https://accounts.google.com/o/oauth2/v2/auth?state=S&redirect=" + url.QueryEscape(in.GetRedirectUri()), State: "S", Binding: "SEALED", TtlSeconds: 600}, nil
}

func (o *oauthID) CompleteOAuth(_ context.Context, in *identityv1.CompleteOAuthRequest, _ ...grpc.CallOption) (*identityv1.CompleteOAuthResponse, error) {
	o.lastCode = in.GetBinding()
	return o.complete, nil
}

func (o *oauthID) ExchangeOAuthCode(_ context.Context, in *identityv1.ExchangeOAuthCodeRequest, _ ...grpc.CallOption) (*identityv1.ExchangeOAuthCodeResponse, error) {
	return &identityv1.ExchangeOAuthCodeResponse{User: testUser(), Tokens: &identityv1.TokenPair{AccessToken: "access.jwt.value", RefreshToken: refreshSecret, ExpiresInSeconds: 900}, RefreshMaxAgeSeconds: 2592000}, nil
}

func TestConnexionGoogle(t *testing.T) {
	id := &oauthID{fakeID: &fakeID{}}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Identity = id })
	site := url.QueryEscape("http://localhost:5173/oauth/callback")

	// start : redirection vers Google + cookie de liaison (HttpOnly, Lax, limité à /v1/auth/oauth)
	rec := r.do(req{"GET", "/v1/auth/oauth/google/start?redirectUri=" + site, "", nil})
	if rec.Code != 302 || !strings.HasPrefix(rec.Header().Get("Location"), "https://accounts.google.com/") {
		t.Fatalf("start : %d %v", rec.Code, rec.Header())
	}
	c := rec.Header().Get("Set-Cookie")
	for _, want := range []string{"ff_oauth=SEALED", "HttpOnly", "SameSite=Lax", "Path=/v1/auth/oauth", "Max-Age=600"} {
		if !strings.Contains(c, want) {
			t.Errorf("cookie de liaison sans %q : %s", want, c)
		}
	}

	// retour réussi : redirection vers le site avec le code d'échange ; le cookie de liaison est effacé et lu
	id.complete = &identityv1.CompleteOAuthResponse{ExchangeCode: "CODE1", RedirectUri: "http://localhost:5173/oauth/callback"}
	rec = r.do(req{"GET", "/v1/auth/oauth/google/callback?code=abc&state=S", "", map[string]string{"Cookie": "ff_oauth=SEALED"}})
	if rec.Code != 302 || rec.Header().Get("Location") != "http://localhost:5173/oauth/callback?code=CODE1" {
		t.Fatalf("callback : %d %v", rec.Code, rec.Header())
	}
	if id.lastCode != "SEALED" || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") && !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=-1") && !strings.Contains(rec.Header().Get("Set-Cookie"), "ff_oauth=;") {
		t.Errorf("cookie lu %q, effacé : %s", id.lastCode, rec.Header().Get("Set-Cookie"))
	}

	// échec : toujours une redirection, jamais du JSON
	id.complete = &identityv1.CompleteOAuthResponse{RedirectUri: "http://localhost:5173/oauth/callback", Failed: true}
	rec = r.do(req{"GET", "/v1/auth/oauth/google/callback?error=access_denied&state=S", "", nil})
	if rec.Code != 302 || rec.Header().Get("Location") != "http://localhost:5173/oauth/callback?error=oauth_failed" {
		t.Fatalf("échec : %d %v", rec.Code, rec.Header())
	}

	// échange : session (jeton d'accès dans le corps, cookie de rafraîchissement HttpOnly Strict)
	rec = r.do(req{"POST", "/v1/auth/oauth/exchange", `{"code":"CODE1"}`, nil})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "access.jwt.value") || strings.Contains(rec.Body.String(), refreshSecret) {
		t.Fatalf("exchange : %d %s", rec.Code, rec.Body.String())
	}
	if c := rec.Header().Get("Set-Cookie"); !strings.Contains(c, "HttpOnly") || !strings.Contains(c, "SameSite=Strict") {
		t.Errorf("cookie de rafraîchissement : %s", c)
	}
}
