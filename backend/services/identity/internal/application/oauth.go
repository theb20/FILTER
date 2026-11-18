package application

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
)

// Connexion via Google : flux « code d'autorisation » durci (state + PKCE S256 + nonce, cookie de liaison, redirection en
// liste blanche, jeton d'identité vérifié : signature, iss, aud, exp, nonce, email_verified).

const (
	providerGoogle   = "google"
	oauthBindingTTL  = 10 * time.Minute
	oauthExchangeTTL = 60 * time.Second
	failedParam      = "oauth_failed"
)

// GoogleConfig : identifiants de l'application Google (lus dans l'environnement du service).
type GoogleConfig struct{ ClientID, ClientSecret, CallbackURL string }

// OAuthConfig regroupe la configuration. Les adresses du fournisseur sont remplaçables (tests).
type OAuthConfig struct {
	Google                     *GoogleConfig
	AllowedRedirects           []string // pages du site autorisées à recevoir le retour (liste blanche exacte)
	HTTP                       *http.Client
	AuthURL, TokenURL, JWKSURL string

	jwks *jwksCache // créé à la demande
}

func (c *OAuthConfig) endpoints() (auth, token, jwks string) {
	d := func(v, def string) string {
		if v != "" {
			return v
		}
		return def
	}
	return d(c.AuthURL, "https://accounts.google.com/o/oauth2/v2/auth"), d(c.TokenURL, "https://oauth2.googleapis.com/token"), d(c.JWKSURL, "https://www.googleapis.com/oauth2/v3/certs")
}

func (c *OAuthConfig) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

type binding struct {
	State, Verifier, Nonce, Redirect string
	Exp                              int64
}

func randURL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) redirectAllowed(u string) bool {
	for _, a := range s.OAuthCfg.AllowedRedirects {
		if a == u {
			return true
		}
	}
	return false
}

// OAuthStart prépare la redirection vers Google. `binding` est un secret scellé que le gateway pose en cookie (ff_oauth).
func (s *Service) OAuthStart(_ context.Context, provider, redirectURI string) (authURL, state, bind string, ttl time.Duration, err error) {
	if provider != providerGoogle || s.OAuthCfg.Google == nil {
		return "", "", "", 0, apperr.NotFound("fournisseur de connexion indisponible")
	}
	if !s.redirectAllowed(redirectURI) {
		return "", "", "", 0, invalid("/redirectUri", "adresse de retour non autorisée")
	}
	if s.Secrets == nil {
		return "", "", "", 0, apperr.Internal(errors.New("clé de chiffrement absente"))
	}
	st, e1 := randURL(24)
	ver, e2 := randURL(48)
	nonce, e3 := randURL(24)
	if err := errors.Join(e1, e2, e3); err != nil {
		return "", "", "", 0, apperr.Internal(err)
	}
	raw, _ := json.Marshal(binding{State: st, Verifier: ver, Nonce: nonce, Redirect: redirectURI, Exp: s.Now().Add(oauthBindingTTL).Unix()})
	sealed, err := s.Secrets.Seal(raw, "oauth-binding")
	if err != nil {
		return "", "", "", 0, apperr.Internal(err)
	}
	sum := sha256.Sum256([]byte(ver))
	authEP, _, _ := s.OAuthCfg.endpoints()
	q := url.Values{
		"client_id": {s.OAuthCfg.Google.ClientID}, "redirect_uri": {s.OAuthCfg.Google.CallbackURL}, "response_type": {"code"}, "scope": {"openid email profile"},
		"state": {st}, "nonce": {nonce}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}, "prompt": {"select_account"},
	}
	return authEP + "?" + q.Encode(), st, base64.RawURLEncoding.EncodeToString(sealed), oauthBindingTTL, nil
}

// OAuthResult : où renvoyer le navigateur. Échec : `Failed` (le site affiche un message générique).
type OAuthResult struct {
	ExchangeCode string
	RedirectURI  string
	Failed       bool
}

// OAuthComplete traite le retour de Google. Un code vide (refus de l'utilisateur, erreur Google) est un échec.
func (s *Service) OAuthComplete(ctx context.Context, provider, code, state, bind string) *OAuthResult {
	fallback := ""
	if len(s.OAuthCfg.AllowedRedirects) > 0 {
		fallback = s.OAuthCfg.AllowedRedirects[0]
	}
	fail := func(redirect string) *OAuthResult {
		if redirect == "" {
			redirect = fallback
		}
		return &OAuthResult{RedirectURI: redirect, Failed: true}
	}
	if provider != providerGoogle || s.OAuthCfg.Google == nil || s.Secrets == nil {
		return fail("")
	}
	sealed, err := base64.RawURLEncoding.DecodeString(bind)
	if err != nil {
		return fail("")
	}
	raw, err := s.Secrets.Open(sealed, "oauth-binding")
	var b binding
	if err != nil || json.Unmarshal(raw, &b) != nil || !s.redirectAllowed(b.Redirect) {
		return fail("")
	}
	if s.Now().Unix() > b.Exp || state == "" || state != b.State || code == "" {
		return fail(b.Redirect)
	}
	claims, err := s.googleIdentity(ctx, code, b)
	if err != nil {
		return fail(b.Redirect)
	}
	userID, err := s.oauthUser(ctx, claims)
	if err != nil {
		return fail(b.Redirect)
	}
	plain, hash, err := domain.NewOpaqueToken()
	if err != nil {
		return fail(b.Redirect)
	}
	if err := s.Tokens.Put(ctx, &domain.OneTimeToken{Hash: hash, Purpose: domain.PurposeOAuthExchange, UserID: userID, ExpiresAt: s.Now().Add(oauthExchangeTTL)}); err != nil {
		return fail(b.Redirect)
	}
	return &OAuthResult{ExchangeCode: plain, RedirectURI: b.Redirect}
}

type googleClaims struct {
	Iss           string  `json:"iss"`
	Sub           string  `json:"sub"`
	Email         string  `json:"email"`
	Nonce         string  `json:"nonce"`
	GivenName     string  `json:"given_name"`
	FamilyName    string  `json:"family_name"`
	Locale        string  `json:"locale"`
	Aud           any     `json:"aud"`
	Exp           float64 `json:"exp"`
	EmailVerified any     `json:"email_verified"`
}

func (c googleClaims) verified() bool {
	switch v := c.EmailVerified.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}

func (c googleClaims) audience(id string) bool {
	switch a := c.Aud.(type) {
	case string:
		return a == id
	case []any:
		for _, x := range a {
			if x == id {
				return true
			}
		}
	}
	return false
}

// googleIdentity échange le code (avec le vérificateur PKCE) puis valide le jeton d'identité.
func (s *Service) googleIdentity(ctx context.Context, code string, b binding) (*googleClaims, error) {
	g := s.OAuthCfg.Google
	_, tokenEP, _ := s.OAuthCfg.endpoints()
	form := url.Values{"code": {code}, "client_id": {g.ClientID}, "client_secret": {g.ClientSecret}, "redirect_uri": {g.CallbackURL}, "grant_type": {"authorization_code"}, "code_verifier": {b.Verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEP, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := s.OAuthCfg.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var tok struct {
		IDToken string `json:"id_token"`
	}
	if res.StatusCode != http.StatusOK || json.Unmarshal(body, &tok) != nil || tok.IDToken == "" {
		return nil, errors.New("échange du code refusé")
	}
	c, err := s.verifyIDToken(ctx, tok.IDToken)
	if err != nil {
		return nil, err
	}
	switch {
	case c.Iss != "https://accounts.google.com" && c.Iss != "accounts.google.com":
		return nil, errors.New("émetteur invalide")
	case !c.audience(g.ClientID):
		return nil, errors.New("audience invalide")
	case float64(s.Now().Unix()) > c.Exp:
		return nil, errors.New("jeton expiré")
	case c.Nonce != b.Nonce:
		return nil, errors.New("nonce invalide")
	case !c.verified() || c.Sub == "" || c.Email == "":
		return nil, errors.New("adresse non vérifiée par le fournisseur")
	}
	return c, nil
}

// ── vérification de la signature (RS256) avec les clés publiques de Google ──

var oauthInit sync.Mutex

type jwksCache struct {
	mu   sync.Mutex
	keys map[string]*rsa.PublicKey
	at   time.Time
}

func (s *Service) googleKey(ctx context.Context, kid string, refresh bool) (*rsa.PublicKey, error) {
	oauthInit.Lock()
	if s.OAuthCfg.jwks == nil {
		s.OAuthCfg.jwks = &jwksCache{}
	}
	c := s.OAuthCfg.jwks
	oauthInit.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !refresh && c.keys != nil && time.Since(c.at) < time.Hour {
		if k := c.keys[kid]; k != nil {
			return k, nil
		}
	}
	_, _, jwksEP := s.OAuthCfg.endpoints()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksEP, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.OAuthCfg.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var set struct {
		Keys []struct{ Kid, Kty, N, E string } `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&set); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, e1 := base64.RawURLEncoding.DecodeString(k.N)
		e, e2 := base64.RawURLEncoding.DecodeString(k.E)
		if e1 != nil || e2 != nil {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	c.keys, c.at = keys, time.Now()
	if k := keys[kid]; k != nil {
		return k, nil
	}
	return nil, errors.New("clé inconnue")
}

func (s *Service) verifyIDToken(ctx context.Context, token string) (*googleClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("jeton mal formé")
	}
	hb, e1 := base64.RawURLEncoding.DecodeString(parts[0])
	pb, e2 := base64.RawURLEncoding.DecodeString(parts[1])
	sig, e3 := base64.RawURLEncoding.DecodeString(parts[2])
	if err := errors.Join(e1, e2, e3); err != nil {
		return nil, err
	}
	var h struct{ Alg, Kid string }
	if err := json.Unmarshal(hb, &h); err != nil || h.Alg != "RS256" {
		return nil, errors.New("algorithme refusé")
	}
	key, err := s.googleKey(ctx, h.Kid, false)
	if err != nil {
		if key, err = s.googleKey(ctx, h.Kid, true); err != nil { // rotation des clés : une seconde lecture
			return nil, err
		}
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig); err != nil {
		return nil, fmt.Errorf("signature invalide: %w", err)
	}
	var c googleClaims
	if err := json.Unmarshal(pb, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// oauthUser retrouve ou crée le compte local. Liaison à un compte existant UNIQUEMENT si son adresse est déjà vérifiée.
func (s *Service) oauthUser(ctx context.Context, c *googleClaims) (string, error) {
	email, err := domain.NormalizeEmail(c.Email)
	if err != nil {
		return "", err
	}
	var userID string
	err = s.tx(ctx, func(ctx context.Context) error {
		if id, found, err := s.OAuth.OAuthUser(ctx, providerGoogle, c.Sub); err != nil {
			return err
		} else if found {
			userID = id
			return nil
		}
		now := s.Now()
		u, err := s.Users.ByEmail(ctx, email)
		switch {
		case err == nil:
			if !u.EmailVerified {
				return errors.New("compte local non vérifié") // pas de prise de contrôle d'un compte par l'adresse
			}
		case errors.Is(err, ErrNotFound):
			locale := "fr"
			if l := strings.ToLower(c.Locale); len(l) >= 2 && (l[:2] == "en" || l[:2] == "es") {
				locale = l[:2]
			}
			u = &domain.User{ID: s.NewID(), Email: email, Kind: domain.KindIndividual, FirstName: c.GivenName, LastName: c.FamilyName, Locale: locale,
				PreferredCurrency: "XOF", Status: domain.StatusActive, EmailVerified: true, CreatedAt: now}
			if err := s.Users.Create(ctx, u); err != nil {
				return err
			}
		default:
			return err
		}
		if u.Status != domain.StatusActive {
			return errors.New("compte suspendu")
		}
		userID = u.ID
		return s.OAuth.LinkOAuth(ctx, providerGoogle, c.Sub, u.ID, now)
	})
	if err != nil {
		return "", err
	}
	// le compte lié peut avoir été suspendu depuis : on ne connecte jamais un compte suspendu
	if u, err := s.Users.ByID(ctx, userID); err != nil || u.Status != domain.StatusActive {
		return "", errors.New("compte indisponible")
	}
	return userID, nil
}

// ExchangeOAuthCode échange le code à usage unique (60 s) contre une session.
func (s *Service) ExchangeOAuthCode(ctx context.Context, code, ip, ua string) (*Session, error) {
	deny := apperr.New(codes.Unauthenticated, apperr.CodeInvalidCode, "code invalide ou expiré")
	var sess *Session
	err := s.tx(ctx, func(ctx context.Context) error {
		id, ok, err := s.Tokens.Consume(ctx, domain.PurposeOAuthExchange, domain.HashToken(code), s.Now())
		if err != nil {
			return apperr.Internal(err)
		}
		if !ok {
			return deny
		}
		u, err := s.Users.ByID(ctx, id)
		if err != nil || u.Status != domain.StatusActive {
			return deny
		}
		now := s.Now()
		if sess, err = s.issueSession(ctx, s.Settings.Get(ctx), u, "", ip, ua, now); err != nil {
			return err
		}
		return s.recordLogin(ctx, u, ip, ua, domain.LoginMethodGoogle, now)
	})
	return sess, err
}
