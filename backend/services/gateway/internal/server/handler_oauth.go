package server

import (
	"context"
	"net/http"
	"net/url"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Mot de passe oublié et connexion Google : le gateway relaie vers identity ; les secrets (code, jeton) ne sont jamais journalisés.

func (h *handler) ForgotPassword(ctx context.Context, req *api.ForgotPasswordRequest) (api.ForgotPasswordRes, error) {
	info := infoFrom(ctx)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.ForgotPassword(ctx, &identityv1.ForgotPasswordRequest{Email: req.Email, Locale: info.Locale, ClientIp: info.IP}); err != nil {
		return nil, err
	}
	return &api.ForgotPasswordAccepted{}, nil
}

func (h *handler) VerifyPasswordCode(ctx context.Context, req *api.VerifyCodeRequest) (api.VerifyPasswordCodeRes, error) {
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.VerifyPasswordCode(ctx, &identityv1.VerifyPasswordCodeRequest{Email: req.Email, Code: req.Code})
	if err != nil {
		return nil, err
	}
	return &api.VerifyCodeResult{ResetToken: res.GetResetToken(), ExpiresIn: res.GetExpiresInSeconds()}, nil
}

func (h *handler) ResetPassword(ctx context.Context, req *api.ResetPasswordRequest) (api.ResetPasswordRes, error) {
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.ResetPassword(ctx, &identityv1.ResetPasswordRequest{ResetToken: req.ResetToken, NewPassword: req.NewPassword}); err != nil {
		return nil, err
	}
	return &api.ResetPasswordNoContent{}, nil
}

// oauthCookie : cookie de liaison (HttpOnly, SameSite=Lax pour survivre à la navigation depuis Google, limité à /v1/auth/oauth).
func (s *Server) oauthCookie(value string, maxAge int) string {
	return (&http.Cookie{Name: "ff_oauth", Value: value, Path: "/v1/auth/oauth", Domain: s.cfg.CookieDomain, MaxAge: maxAge, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode}).String()
}

func (h *handler) StartOAuth(ctx context.Context, p api.StartOAuthParams) (api.StartOAuthRes, error) {
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.StartOAuth(ctx, &identityv1.StartOAuthRequest{Provider: string(p.Provider), RedirectUri: p.RedirectUri.String()})
	if err != nil {
		return nil, err
	}
	return &api.StartOAuthFound{Location: api.NewOptString(res.GetAuthorizationUrl()), SetCookie: api.NewOptString(h.srv.oauthCookie(res.GetBinding(), int(res.GetTtlSeconds())))}, nil
}

// OauthCallback : toutes les issues sont des redirections vers le site (jamais de JSON dans un navigateur).
func (h *handler) OauthCallback(ctx context.Context, p api.OauthCallbackParams) (api.OauthCallbackRes, error) {
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	code := p.Code.Or("")
	if p.Error.Or("") != "" {
		code = "" // refus de l'utilisateur ou erreur du fournisseur
	}
	res, err := h.id.CompleteOAuth(ctx, &identityv1.CompleteOAuthRequest{Provider: string(p.Provider), Code: code, State: p.State, Binding: p.FfOAuth.Or("")})
	if err != nil {
		return nil, err
	}
	target, perr := url.Parse(res.GetRedirectUri())
	if perr != nil || target.Host == "" {
		return nil, apperr.Internal(perr)
	}
	q := target.Query()
	if res.GetFailed() || res.GetExchangeCode() == "" {
		q.Set("error", "oauth_failed")
	} else {
		q.Set("code", res.GetExchangeCode())
	}
	target.RawQuery = q.Encode()
	return &api.OauthCallbackFound{Location: api.NewOptString(target.String()), SetCookie: api.NewOptString(h.srv.oauthCookie("", -1))}, nil
}

func (h *handler) ExchangeOAuthCode(ctx context.Context, req *api.OAuthExchangeRequest) (api.ExchangeOAuthCodeRes, error) {
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.ExchangeOAuthCode(ctx, &identityv1.ExchangeOAuthCodeRequest{ExchangeCode: req.Code})
	if err != nil {
		return nil, err
	}
	return &api.AuthResultHeaders{
		SetCookie: api.NewOptString(h.srv.refreshCookie(res.GetTokens().GetRefreshToken(), int(res.GetRefreshMaxAgeSeconds()))),
		Response:  api.AuthResult{User: toMe(res.GetUser()), Tokens: toTokens(res.GetTokens())},
	}, nil
}
