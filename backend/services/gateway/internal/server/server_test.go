package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const origin = "https://app.exemple.ci"

// ── faux service identity ───────────────────────────────────────────────────

type fakeID struct {
	identityv1.IdentityServiceClient // les méthodes non surchargées paniquent : le test échoue (elles ne doivent pas être appelées)

	loginErr, registerErr, getUserErr error
	lastRegister                      *identityv1.RegisterRequest
	lastLogin                         *identityv1.LoginRequest
	lastRefresh                       *identityv1.RefreshTokensRequest
	lastChange                        *identityv1.ChangePasswordRequest
	lastGetUser                       *identityv1.GetUserRequest
	logouts, refreshes                atomic.Int32
}

const refreshSecret = "REFRESH-TOKEN-SECRET-VALUE"

func (f *fakeID) Register(_ context.Context, in *identityv1.RegisterRequest, _ ...grpc.CallOption) (*identityv1.RegisterResponse, error) {
	f.lastRegister = in
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	return &identityv1.RegisterResponse{Status: "accepted"}, nil
}

func (f *fakeID) Login(_ context.Context, in *identityv1.LoginRequest, _ ...grpc.CallOption) (*identityv1.LoginResponse, error) {
	f.lastLogin = in
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return &identityv1.LoginResponse{User: testUser(), Tokens: &identityv1.TokenPair{AccessToken: "access.jwt.value", RefreshToken: refreshSecret, ExpiresInSeconds: 900}, RefreshMaxAgeSeconds: 2592000}, nil
}

func (f *fakeID) RefreshTokens(_ context.Context, in *identityv1.RefreshTokensRequest, _ ...grpc.CallOption) (*identityv1.RefreshTokensResponse, error) {
	f.refreshes.Add(1)
	f.lastRefresh = in
	return &identityv1.RefreshTokensResponse{Tokens: &identityv1.TokenPair{AccessToken: "new.access.jwt", RefreshToken: "ROTATED-" + refreshSecret, ExpiresInSeconds: 900}, RefreshMaxAgeSeconds: 2592000}, nil
}

func (f *fakeID) Logout(context.Context, *identityv1.LogoutRequest, ...grpc.CallOption) (*identityv1.LogoutResponse, error) {
	f.logouts.Add(1)
	return &identityv1.LogoutResponse{}, nil
}

func (f *fakeID) GetUser(_ context.Context, in *identityv1.GetUserRequest, _ ...grpc.CallOption) (*identityv1.GetUserResponse, error) {
	f.lastGetUser = in
	if f.getUserErr != nil {
		return nil, f.getUserErr
	}
	return &identityv1.GetUserResponse{User: testUser()}, nil
}

func (f *fakeID) ChangePassword(_ context.Context, in *identityv1.ChangePasswordRequest, _ ...grpc.CallOption) (*identityv1.ChangePasswordResponse, error) {
	f.lastChange = in
	return &identityv1.ChangePasswordResponse{}, nil
}

func testUser() *identityv1.User {
	fn := "Awa"
	return &identityv1.User{Id: uuid.NewString(), Email: "awa@exemple.ci", Kind: identityv1.UserKind_USER_KIND_INDIVIDUAL, FirstName: &fn, Locale: "fr", PreferredCurrency: "XOF", Status: "active", CreatedAt: timestamppb.Now()}
}

// ── JWKS de test ────────────────────────────────────────────────────────────

type jwksSource struct{ keys map[string]*jwtx.KeySet }

func (j *jwksSource) Fetch(_ context.Context, aud string) ([]byte, error) {
	k, ok := j.keys[aud]
	if !ok {
		return nil, errors.New("audience inconnue")
	}
	return k.JWKS(), nil
}

// ── banc d'essai ────────────────────────────────────────────────────────────

type rig struct {
	h    http.Handler
	id   *fakeID
	keys map[string]*jwtx.KeySet
	src  *jwksSource
	cfg  server.Config
	lim  *server.MemoryLimiter
	now  *time.Time
}

func newRig(t *testing.T, mut func(*server.Config)) *rig { return newRigWith(t, mut, nil) }

func newRigWith(t *testing.T, mut func(*server.Config), sourcing sourcingv1.SourcingServiceClient) *rig {
	t.Helper()
	keys := map[string]*jwtx.KeySet{}
	for _, aud := range []string{jwtx.AudCustomer, jwtx.AudStaff, jwtx.AudStaffChallenge} {
		k, err := jwtx.NewKeySet()
		if err != nil {
			t.Fatal(err)
		}
		keys[aud] = k
	}
	cfg := server.DefaultConfig()
	cfg.AllowedOrigins = []string{origin}
	cfg.Env = "test"
	if mut != nil {
		mut(&cfg)
	}
	src := &jwksSource{keys: keys}
	id := &fakeID{}
	lim := server.NewMemoryLimiter()
	h, err := server.New(server.Deps{Config: cfg, Log: slog.New(slog.NewJSONHandler(io.Discard, nil)), Identity: id, Sourcing: sourcing, Keys: server.NewKeyCache(src, cfg.ClockLeeway, 0, nil), Limiter: lim})
	if err != nil {
		t.Fatal(err)
	}
	return &rig{h: h, id: id, keys: keys, src: src, cfg: cfg, lim: lim}
}

type req struct {
	method, path, body string
	headers            map[string]string
}

func (r *rig) do(q req) *httptest.ResponseRecorder {
	var body io.Reader
	if q.body != "" {
		body = strings.NewReader(q.body)
	}
	hr := httptest.NewRequest(q.method, q.path, body)
	hr.RemoteAddr = "203.0.113.7:4242"
	if q.body != "" {
		hr.Header.Set("Content-Type", "application/json")
	}
	for k, v := range q.headers {
		hr.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, hr)
	return rec
}

func csrfHeaders() map[string]string {
	return map[string]string{"Origin": origin, "X-Requested-With": "ff-web", "Cookie": "ff_refresh=" + refreshSecret}
}

func customerToken(t *testing.T, r *rig, sid string, ttl time.Duration) string {
	t.Helper()
	tok, err := r.keys[jwtx.AudCustomer].Sign(jwtx.AudCustomer, uuid.NewString(), ttl, time.Now(), jwtx.Claims{SID: sid, Kind: "individual"})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func problemOf(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("Content-Type = %q, attendu application/problem+json (corps : %s)", ct, rec.Body.String())
	}
	var p map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

const loginBody = `{"email":"awa@exemple.ci","password":"correct horse battery"}`

// ── cookie de rafraîchissement ──────────────────────────────────────────────

func TestLoginPoseLeCookieEtNeRenvoiePasLeRefreshDansLeCorps(t *testing.T) {
	r := newRig(t, nil)
	rec := r.do(req{"POST", "/v1/auth/login", loginBody, nil})
	if rec.Code != 200 {
		t.Fatalf("statut %d : %s", rec.Code, rec.Body.String())
	}
	c := rec.Header().Get("Set-Cookie")
	for _, want := range []string{"ff_refresh=" + refreshSecret, "HttpOnly", "Secure", "SameSite=Strict", "Path=/v1/auth", "Max-Age=2592000"} {
		if !strings.Contains(c, want) {
			t.Errorf("Set-Cookie doit contenir %q : %s", want, c)
		}
	}
	body := rec.Body.String()
	if strings.Contains(body, refreshSecret) || strings.Contains(strings.ToLower(body), "refresh") {
		t.Fatalf("le jeton de rafraîchissement ne doit JAMAIS apparaître dans le corps : %s", body)
	}
	var res struct {
		Tokens struct {
			AccessToken string `json:"accessToken"`
			TokenType   string `json:"tokenType"`
			ExpiresIn   int    `json:"expiresIn"`
		} `json:"tokens"`
		User struct{ Email string } `json:"user"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Tokens.AccessToken == "" || res.Tokens.TokenType != "Bearer" || res.Tokens.ExpiresIn != 900 || res.User.Email != "awa@exemple.ci" {
		t.Fatalf("réponse inattendue : %s", body)
	}
	if r.id.lastLogin.GetClientIp() != "203.0.113.7" {
		t.Fatalf("l'IP cliente doit être transmise à identity : %q", r.id.lastLogin.GetClientIp())
	}
}

func TestCookieSecureDesactivableEnDev(t *testing.T) {
	r := newRig(t, func(c *server.Config) { c.CookieSecure = false })
	c := r.do(req{"POST", "/v1/auth/login", loginBody, nil}).Header().Get("Set-Cookie")
	if strings.Contains(c, "Secure") || !strings.Contains(c, "HttpOnly") {
		t.Fatalf("HttpOnly toujours, Secure selon la config : %s", c)
	}
}

func TestRefreshLitLeCookieEtLeRote(t *testing.T) {
	r := newRig(t, nil)
	rec := r.do(req{"POST", "/v1/auth/refresh", "", csrfHeaders()})
	if rec.Code != 200 {
		t.Fatalf("statut %d : %s", rec.Code, rec.Body.String())
	}
	if r.id.lastRefresh.GetRefreshToken() != refreshSecret {
		t.Fatal("le gateway doit transmettre à identity la valeur du COOKIE")
	}
	if c := rec.Header().Get("Set-Cookie"); !strings.Contains(c, "ROTATED-") || !strings.Contains(c, "HttpOnly") {
		t.Fatalf("le cookie doit être remplacé (rotation) : %s", c)
	}
	if strings.Contains(rec.Body.String(), "ROTATED") || strings.Contains(rec.Body.String(), refreshSecret) {
		t.Fatalf("aucun jeton de rafraîchissement dans le corps : %s", rec.Body.String())
	}
}

// ── CSRF (G16) ──────────────────────────────────────────────────────────────

func TestCSRFRefusEtAcceptation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{"Origin absent", func(h map[string]string) { delete(h, "Origin") }},
		{"Origin hors liste blanche", func(h map[string]string) { h["Origin"] = "https://evil.example" }},
		{"Sec-Fetch-Site cross-site", func(h map[string]string) { h["Sec-Fetch-Site"] = "cross-site" }},
		{"X-Requested-With absent", func(h map[string]string) { delete(h, "X-Requested-With") }},
		{"X-Requested-With incorrect", func(h map[string]string) { h["X-Requested-With"] = "XMLHttpRequest" }},
	}
	for _, path := range []string{"/v1/auth/refresh", "/v1/auth/logout"} {
		for _, c := range cases {
			r := newRig(t, nil)
			h := csrfHeaders()
			c.mutate(h)
			rec := r.do(req{"POST", path, "", h})
			if rec.Code != 403 || problemOf(t, rec)["code"] != apperr.CodeCSRFFailed {
				t.Errorf("%s / %s : attendu 403 CSRF_FAILED, obtenu %d %s", path, c.name, rec.Code, rec.Body.String())
			}
			if r.id.refreshes.Load() != 0 || r.id.logouts.Load() != 0 {
				t.Errorf("%s / %s : identity ne doit JAMAIS être appelé sur une requête CSRF", path, c.name)
			}
		}
	}
	// formulaire HTML classique (corps non JSON) refusé
	r := newRig(t, nil)
	h := csrfHeaders()
	h["Content-Type"] = "application/x-www-form-urlencoded"
	hr := httptest.NewRequest("POST", "/v1/auth/refresh", strings.NewReader("a=b"))
	for k, v := range h {
		hr.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, hr)
	if rec.Code != 403 {
		t.Errorf("corps de formulaire : attendu 403, obtenu %d", rec.Code)
	}
	// Sec-Fetch-Site same-site accepté
	r = newRig(t, nil)
	h = csrfHeaders()
	h["Sec-Fetch-Site"] = "same-site"
	if rec := r.do(req{"POST", "/v1/auth/refresh", "", h}); rec.Code != 200 {
		t.Errorf("same-site doit être accepté : %d %s", rec.Code, rec.Body.String())
	}
}

func TestLogoutIdempotentSansCookie(t *testing.T) {
	r := newRig(t, nil)
	h := csrfHeaders()
	delete(h, "Cookie")
	rec := r.do(req{"POST", "/v1/auth/logout", "", h})
	if rec.Code != 204 {
		t.Fatalf("cookie absent : 204 attendu (idempotent), obtenu %d %s", rec.Code, rec.Body.String())
	}
	if c := rec.Header().Get("Set-Cookie"); !strings.Contains(c, "ff_refresh=") || !strings.Contains(c, "Max-Age=0") || !strings.Contains(c, "Path=/v1/auth") {
		t.Fatalf("le cookie doit être effacé : %s", c)
	}
	if r.id.logouts.Load() != 0 {
		t.Fatal("rien à révoquer : identity ne doit pas être appelé")
	}
	// avec cookie : révocation côté identity
	rec = r.do(req{"POST", "/v1/auth/logout", "", csrfHeaders()})
	if rec.Code != 204 || r.id.logouts.Load() != 1 {
		t.Fatalf("avec cookie : %d, appels=%d", rec.Code, r.id.logouts.Load())
	}
}

// ── CORS avec identifiants (G6) ─────────────────────────────────────────────

func TestCORSAvecIdentifiants(t *testing.T) {
	r := newRig(t, nil)
	rec := r.do(req{"POST", "/v1/auth/login", loginBody, map[string]string{"Origin": origin}})
	h := rec.Header()
	if h.Get("Access-Control-Allow-Origin") != origin || h.Get("Access-Control-Allow-Credentials") != "true" || !strings.Contains(h.Get("Vary"), "Origin") {
		t.Fatalf("CORS avec identifiants : en-têtes inattendus %v", h)
	}
	// origine inconnue : aucun en-tête CORS
	rec = r.do(req{"POST", "/v1/auth/login", loginBody, map[string]string{"Origin": "https://evil.example"}})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" || rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("une origine hors liste blanche ne doit recevoir aucun en-tête CORS")
	}
	// préflight
	rec = r.do(req{"OPTIONS", "/v1/auth/refresh", "", map[string]string{"Origin": origin, "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "x-requested-with"}})
	if rec.Code != 204 || !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "X-Requested-With") || rec.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("préflight : %d %v", rec.Code, rec.Header())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("jamais de joker avec des identifiants")
	}
	rec = r.do(req{"OPTIONS", "/v1/auth/refresh", "", map[string]string{"Origin": "https://evil.example", "Access-Control-Request-Method": "POST"}})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" || rec.Header().Get("Access-Control-Allow-Methods") != "" {
		t.Fatal("préflight d'une origine inconnue : aucun en-tête d'autorisation")
	}
}

// ── inscription : réponse uniforme, aucun jeton ─────────────────────────────

const registerBody = `{"kind":"individual","email":"awa@exemple.ci","password":"correct horse battery","acceptTerms":true,"firstName":"Awa"}`

func TestInscriptionReponseUniformeSansJeton(t *testing.T) {
	r := newRig(t, nil)
	rec := r.do(req{"POST", "/v1/auth/register", registerBody, nil})
	if rec.Code != 202 || strings.TrimSpace(rec.Body.String()) != `{"status":"accepted"}` {
		t.Fatalf("202 {status:accepted} attendu : %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Set-Cookie") != "" || strings.Contains(strings.ToLower(rec.Body.String()), "token") {
		t.Fatal("aucun jeton ni cookie à l'inscription")
	}
	if r.id.lastRegister.GetEmail() != "awa@exemple.ci" || r.id.lastRegister.GetClientIp() != "203.0.113.7" {
		t.Fatalf("requête mal transmise : %+v", r.id.lastRegister)
	}
}

func TestInscriptionConditionsNonAcceptees(t *testing.T) {
	r := newRig(t, nil)
	rec := r.do(req{"POST", "/v1/auth/register", strings.Replace(registerBody, `"acceptTerms":true`, `"acceptTerms":false`, 1), nil})
	if rec.Code != 422 && rec.Code != 400 {
		t.Fatalf("acceptTerms=false doit être refusé : %d %s", rec.Code, rec.Body.String())
	}
}

func TestErreursDeFormeRenvoyeesAvecLeurCode(t *testing.T) {
	r := newRig(t, nil)
	r.id.registerErr = apperr.ToGRPC(apperr.Validation("mot de passe compromis", apperr.Field{Pointer: "/password", Code: apperr.CodePasswordCompromised, Message: "ce mot de passe a fuité"}))
	rec := r.do(req{"POST", "/v1/auth/register", registerBody, nil})
	p := problemOf(t, rec)
	if rec.Code != 422 || p["code"] != apperr.CodeValidation {
		t.Fatalf("422 VALIDATION_FAILED attendu : %d %v", rec.Code, p)
	}
	errs, _ := p["errors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["code"] != apperr.CodePasswordCompromised || errs[0].(map[string]any)["field"] != "/password" {
		t.Fatalf("détail de champ manquant : %v", p)
	}
}

// ── validation du contrat (G10) ─────────────────────────────────────────────

func TestChampsInconnusEtJSONInvalideRefuses(t *testing.T) {
	r := newRig(t, nil)
	for name, body := range map[string]string{
		"champ inconnu":     `{"email":"a@b.ci","password":"correct horse battery","admin":true}`,
		"JSON invalide":     `{"email":`,
		"type erroné":       `{"email":123,"password":"x"}`,
		"champ obligatoire": `{"email":"a@b.ci"}`,
	} {
		rec := r.do(req{"POST", "/v1/auth/login", body, nil})
		if rec.Code != 400 && rec.Code != 422 {
			t.Errorf("%s : 400/422 attendu, obtenu %d %s", name, rec.Code, rec.Body.String())
			continue
		}
		problemOf(t, rec)
	}
	if r.id.lastLogin != nil {
		t.Fatal("identity ne doit jamais recevoir une requête invalide")
	}
}

func TestCorpsTropGrosRefuse(t *testing.T) {
	r := newRig(t, nil)
	big := `{"email":"a@b.ci","password":"` + strings.Repeat("x", 2<<20) + `"}`
	rec := r.do(req{"POST", "/v1/auth/login", big, nil})
	if rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("un corps > 1 Mio doit être refusé (4xx) : %d", rec.Code)
	}
	if r.id.lastLogin != nil {
		t.Fatal("identity ne doit pas être appelé")
	}
}

// ── erreurs (G8) ────────────────────────────────────────────────────────────

func TestIdentifiantsInvalides401(t *testing.T) {
	r := newRig(t, nil)
	r.id.loginErr = apperr.ToGRPC(apperr.Unauthenticated(apperr.CodeInvalidCredentials, "identifiants invalides"))
	rec := r.do(req{"POST", "/v1/auth/login", loginBody, nil})
	if rec.Code != 401 || problemOf(t, rec)["code"] != apperr.CodeInvalidCredentials || rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("401 INVALID_CREDENTIALS sans cookie attendu : %d %s", rec.Code, rec.Body.String())
	}
}

func TestRalentissementProgressifRenvoie429EtRetryAfter(t *testing.T) {
	r := newRig(t, nil)
	r.id.loginErr = apperr.ToGRPC(apperr.RateLimited(8 * time.Second))
	rec := r.do(req{"POST", "/v1/auth/login", loginBody, nil})
	if rec.Code != 429 || rec.Header().Get("Retry-After") != "8" || problemOf(t, rec)["code"] != apperr.CodeRateLimited {
		t.Fatalf("429 + Retry-After: 8 attendus : %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestErreurInterneNExposeAucunDetail(t *testing.T) {
	r := newRig(t, nil)
	r.id.loginErr = errors.New("dial postgres://user:SECRETPW@db:5432/ff: connection refused")
	rec := r.do(req{"POST", "/v1/auth/login", loginBody, nil})
	if rec.Code != 500 {
		t.Fatalf("statut %d", rec.Code)
	}
	b := rec.Body.String()
	if strings.Contains(b, "SECRETPW") || strings.Contains(b, "postgres") {
		t.Fatalf("détail interne exposé : %s", b)
	}
	p := problemOf(t, rec)
	if p["traceId"] == nil || p["traceId"] != rec.Header().Get("X-Request-Id") {
		t.Fatalf("traceId absent ou différent de X-Request-Id : %v / %q", p["traceId"], rec.Header().Get("X-Request-Id"))
	}
}

// ── authentification des jetons (G2, G3) ────────────────────────────────────

func TestMeExigeUnJetonValide(t *testing.T) {
	r := newRig(t, nil)
	if rec := r.do(req{"GET", "/v1/me", "", nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : 401 attendu, obtenu %d", rec.Code)
	}
	good := customerToken(t, r, "fam-1", 15*time.Minute)
	rec := r.do(req{"GET", "/v1/me", "", map[string]string{"Authorization": "Bearer " + good}})
	if rec.Code != 200 {
		t.Fatalf("jeton valide : %d %s", rec.Code, rec.Body.String())
	}
	var me map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if me["email"] != "awa@exemple.ci" || me["kind"] != "individual" {
		t.Fatalf("Me inattendu : %v", me)
	}
	// jeton d'une AUTRE audience (personnel) : refusé
	staff, _ := r.keys[jwtx.AudStaff].Sign(jwtx.AudStaff, uuid.NewString(), time.Minute, time.Now(), jwtx.Claims{})
	if rec := r.do(req{"GET", "/v1/me", "", map[string]string{"Authorization": "Bearer " + staff}}); rec.Code != 401 {
		t.Fatalf("un jeton du personnel ne doit pas ouvrir une route client : %d", rec.Code)
	}
	// jeton expiré
	old, _ := r.keys[jwtx.AudCustomer].Sign(jwtx.AudCustomer, uuid.NewString(), time.Minute, time.Now().Add(-time.Hour), jwtx.Claims{})
	if rec := r.do(req{"GET", "/v1/me", "", map[string]string{"Authorization": "Bearer " + old}}); rec.Code != 401 {
		t.Fatalf("jeton expiré : %d", rec.Code)
	}
	// jeton falsifié (signé par une autre clé)
	other, _ := jwtx.NewKeySet()
	forged, _ := other.Sign(jwtx.AudCustomer, uuid.NewString(), time.Minute, time.Now(), jwtx.Claims{})
	rec = r.do(req{"GET", "/v1/me", "", map[string]string{"Authorization": "Bearer " + forged}})
	if rec.Code != 401 || problemOf(t, rec)["code"] != apperr.CodeUnauthenticated {
		t.Fatalf("jeton falsifié : %d", rec.Code)
	}
}

func TestChangementMotDePasseTransmetLaSession(t *testing.T) {
	r := newRig(t, nil)
	tok := customerToken(t, r, "fam-42", 15*time.Minute)
	rec := r.do(req{"POST", "/v1/me/password", `{"currentPassword":"a","newPassword":"correct horse battery"}`, map[string]string{"Authorization": "Bearer " + tok}})
	if rec.Code != 204 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if r.id.lastChange.GetCurrentSessionId() != "fam-42" {
		t.Fatalf("le claim sid doit être transmis pour conserver la session courante : %q", r.id.lastChange.GetCurrentSessionId())
	}
}

func TestRotationDeCleJWKS(t *testing.T) {
	r := newRig(t, nil)
	old := customerToken(t, r, "s", time.Minute)
	if rec := r.do(req{"GET", "/v1/me", "", map[string]string{"Authorization": "Bearer " + old}}); rec.Code != 200 {
		t.Fatal("jeton initial")
	}
	// identity fait tourner sa clé : les nouveaux jetons portent un kid inconnu du cache
	nk, _ := jwtx.NewKeySet()
	r.src.keys[jwtx.AudCustomer] = nk
	fresh, _ := nk.Sign(jwtx.AudCustomer, uuid.NewString(), time.Minute, time.Now(), jwtx.Claims{})
	if rec := r.do(req{"GET", "/v1/me", "", map[string]string{"Authorization": "Bearer " + fresh}}); rec.Code != 200 {
		t.Fatalf("le gateway doit recharger le JWKS sur kid inconnu : %d %s", rec.Code, rec.Body.String())
	}
}

func TestRoutesNonImplementeesRepondent501(t *testing.T) {
	r := newRig(t, nil)
	tok := customerToken(t, r, "s", time.Minute)
	rec := r.do(req{"GET", "/v1/me/company", "", map[string]string{"Authorization": "Bearer " + tok}})
	if rec.Code != 501 || problemOf(t, rec)["code"] != apperr.CodeNotImplemented {
		t.Fatalf("une route sans RPC doit répondre 501 (jamais une fausse réponse) : %d %s", rec.Code, rec.Body.String())
	}
	if rec := r.do(req{"GET", "/v1/inexistante", "", nil}); rec.Code != 404 {
		t.Fatalf("route inconnue : %d", rec.Code)
	}
}

// ── limitation de débit (G5) ────────────────────────────────────────────────

func TestLimitationDebitConnexion(t *testing.T) {
	r := newRig(t, nil)
	for i := 1; i <= 10; i++ {
		if rec := r.do(req{"POST", "/v1/auth/login", loginBody, nil}); rec.Code != 200 {
			t.Fatalf("requête %d : %d", i, rec.Code)
		}
	}
	rec := r.do(req{"POST", "/v1/auth/login", loginBody, nil}) // 11e dans la minute
	if rec.Code != 429 || rec.Header().Get("Retry-After") == "" || rec.Header().Get("RateLimit-Limit") != "10" {
		t.Fatalf("11e connexion : 429 + Retry-After attendus : %d %v", rec.Code, rec.Header())
	}
	problemOf(t, rec)
}

func TestLimitationDebitInscription(t *testing.T) {
	r := newRig(t, nil)
	for i := 1; i <= 5; i++ {
		if rec := r.do(req{"POST", "/v1/auth/register", registerBody, nil}); rec.Code != 202 {
			t.Fatalf("inscription %d : %d", i, rec.Code)
		}
	}
	if rec := r.do(req{"POST", "/v1/auth/register", registerBody, nil}); rec.Code != 429 {
		t.Fatalf("6e inscription/heure : 429 attendu, obtenu %d", rec.Code)
	}
}

// ── sondes et en-têtes ──────────────────────────────────────────────────────

func TestSondes(t *testing.T) {
	r := newRig(t, nil)
	for _, p := range []string{"/healthz", "/readyz", "/v1/healthz", "/v1/readyz"} {
		if rec := r.do(req{"GET", p, "", nil}); rec.Code != 200 {
			t.Errorf("%s : %d %s", p, rec.Code, rec.Body.String())
		}
	}
	h, _ := server.New(server.Deps{Config: r.cfg, Log: slog.New(slog.NewJSONHandler(io.Discard, nil)), Identity: r.id, Keys: server.NewKeyCache(r.src, 0, 0, nil),
		Ready: func(context.Context) error { return errors.New("identity injoignable") }})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/readyz", nil))
	if rec.Code != 503 || strings.Contains(rec.Body.String(), "injoignable") {
		t.Fatalf("readyz sans identity : 503 sans détail interne attendu : %d %s", rec.Code, rec.Body.String())
	}
}

func TestEnTetesDeSecurite(t *testing.T) {
	r := newRig(t, nil)
	h := r.do(req{"POST", "/v1/auth/login", loginBody, nil}).Header()
	if h.Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(h.Get("Strict-Transport-Security"), "max-age=") || h.Get("Cache-Control") != "no-store" || h.Get("X-Request-Id") == "" {
		t.Fatalf("en-têtes de sécurité manquants : %v", h)
	}
	// traceparent entrant repris
	rec := r.do(req{"GET", "/healthz", "", map[string]string{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}})
	if rec.Header().Get("X-Request-Id") != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("traceparent non repris : %q", rec.Header().Get("X-Request-Id"))
	}
}

func TestDomaineDuCookieConfigurable(t *testing.T) {
	// par défaut : cookie « host-only » (aucun attribut Domain)
	if c := newRig(t, nil).do(req{"POST", "/v1/auth/login", loginBody, nil}).Header().Get("Set-Cookie"); strings.Contains(c, "Domain=") {
		t.Fatalf("sans configuration, aucun Domain : %s", c)
	}
	r := newRig(t, func(c *server.Config) { c.CookieDomain = "exemple.ci" })
	c := r.do(req{"POST", "/v1/auth/login", loginBody, nil}).Header().Get("Set-Cookie")
	if !strings.Contains(c, "Domain=exemple.ci") || !strings.Contains(c, "SameSite=Strict") || !strings.Contains(c, "HttpOnly") {
		t.Fatalf("Domain configuré attendu : %s", c)
	}
	// l'effacement doit porter le MÊME domaine, sinon le navigateur ne supprime pas le cookie
	clr := r.do(req{"POST", "/v1/auth/logout", "", csrfHeaders()}).Header().Get("Set-Cookie")
	if !strings.Contains(clr, "Domain=exemple.ci") || !strings.Contains(clr, "Max-Age=0") {
		t.Fatalf("effacement sans le même Domain : %s", clr)
	}
}
