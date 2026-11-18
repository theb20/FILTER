package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	catalogadminv1 "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	contentadminv1 "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	financeadminv1 "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	sourcingv1 "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// handler implémente api.Handler pour les routes de ce lot (inscription et sessions des clients). Toutes les autres
// routes du périmètre validé répondent 501 (api.UnimplementedHandler) tant que leur RPC identity n'existe pas.
type handler struct {
	api.UnimplementedHandler
	srv *Server
	id  identityv1.IdentityServiceClient
	fin financeadminv1.FinanceAdminServiceClient // nil tant que le service finance n'est pas configuré (501)
	cat catalogadminv1.CatalogAdminServiceClient // nil tant que le service catalog n'est pas configuré (501)
	cms contentadminv1.ContentAdminServiceClient // nil tant que le service content n'est pas configuré (501)
	src sourcingv1.SourcingServiceClient         // nil tant que le service sourcing n'est pas configuré (les routes répondent alors 501)
}

var _ api.Handler = (*handler)(nil)

// rpc prépare le contexte gRPC : délai et métadonnées du gateway (G9).
func (h *handler) rpc(ctx context.Context) (context.Context, context.CancelFunc) {
	return h.rpcAs(ctx, "customer")
}

// rpcAs : comme rpc, en indiquant le type d'acteur (customer | staff).
func (h *handler) rpcAs(ctx context.Context, actor string) (context.Context, context.CancelFunc) {
	info := infoFrom(ctx)
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", info.TraceID, "x-locale", info.Locale, "x-client-ip", info.IP, "x-actor-type", actor)
	if c := claimsFrom(ctx); c != nil {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-actor-id", c.Subject, "x-session-id", c.SID)
	}
	return context.WithTimeout(ctx, h.srv.cfg.RequestTimeoutRPC)
}

// ── sondes ──────────────────────────────────────────────────────────────────

func (h *handler) Healthz(context.Context) (api.HealthzRes, error) {
	return &api.HealthzResponse{Status: api.HealthzResponseStatusOk}, nil
}

func (h *handler) Readyz(ctx context.Context) (api.ReadyzRes, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := h.srv.ready(ctx); err != nil {
		return nil, apperr.New(codes.Unavailable, apperr.CodeInternal, "service indisponible")
	}
	return &api.ReadyzResponse{Status: api.ReadyzResponseStatusReady}, nil
}

// ── inscription ─────────────────────────────────────────────────────────────

func (h *handler) Register(ctx context.Context, req *api.RegisterRequest) (api.RegisterRes, error) {
	if !req.AcceptTerms {
		return nil, apperr.Validation("conditions non acceptées", apperr.Field{Pointer: "/acceptTerms", Code: apperr.CodeValidation, Message: "l'acceptation des conditions est obligatoire"})
	}
	info := infoFrom(ctx)
	in := &identityv1.RegisterRequest{
		Kind: kindToProto(string(req.Kind)), Email: req.Email, Password: req.Password, FirstName: req.FirstName.Or(""), LastName: req.LastName.Or(""),
		Locale: string(req.Locale.Or(api.Locale(info.Locale))), ClientIp: info.IP,
	}
	if c, ok := req.Company.Get(); ok {
		in.Company = &identityv1.CompanyInput{LegalName: c.LegalName, TaxId: c.TaxId, Country: string(c.Country)}
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	// Réponse UNIFORME (202) : aucune fuite d'existence. Les erreurs de forme (422) viennent d'identity.
	if _, err := h.id.Register(ctx, in); err != nil {
		return nil, err
	}
	return &api.Accepted{Status: api.AcceptedStatusAccepted}, nil
}

// ── sessions ────────────────────────────────────────────────────────────────

func (h *handler) Login(ctx context.Context, req *api.LoginRequest) (api.LoginRes, error) {
	info := infoFrom(ctx)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.Login(ctx, &identityv1.LoginRequest{Email: req.Email, Password: req.Password, ClientIp: info.IP, UserAgent: info.UA})
	if err != nil {
		return nil, err
	}
	return &api.AuthResultHeaders{
		SetCookie: api.NewOptString(h.srv.refreshCookie(res.GetTokens().GetRefreshToken(), int(res.GetRefreshMaxAgeSeconds()))),
		Response:  api.AuthResult{User: toMe(res.GetUser()), Tokens: toTokens(res.GetTokens())},
	}, nil
}

func (h *handler) RefreshTokens(ctx context.Context, _ api.RefreshTokensParams) (api.RefreshTokensRes, error) {
	info := infoFrom(ctx)
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.RefreshTokens(ctx2, &identityv1.RefreshTokensRequest{RefreshToken: refreshCookieFrom(ctx), ClientIp: info.IP, UserAgent: info.UA})
	if err != nil {
		return nil, err
	}
	return &api.AuthTokensHeaders{
		SetCookie: api.NewOptString(h.srv.refreshCookie(res.GetTokens().GetRefreshToken(), int(res.GetRefreshMaxAgeSeconds()))),
		Response:  toTokens(res.GetTokens()),
	}, nil
}

func (h *handler) Logout(ctx context.Context, _ api.LogoutParams) (api.LogoutRes, error) {
	if tok := refreshCookieFrom(ctx); tok != "" { // cookie absent : rien à révoquer, mais la réponse reste 204
		ctx2, cancel := h.rpc(ctx)
		defer cancel()
		if _, err := h.id.Logout(ctx2, &identityv1.LogoutRequest{RefreshToken: tok}); err != nil {
			return nil, err
		}
	}
	return &api.LogoutNoContent{SetCookie: api.NewOptString(h.srv.clearCookie())}, nil
}

func (h *handler) LogoutAll(ctx context.Context) (api.LogoutAllRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.LogoutAll(ctx2, &identityv1.LogoutAllRequest{UserId: claimsFrom(ctx).Subject}); err != nil {
		return nil, err
	}
	return &api.LogoutAllNoContent{SetCookie: api.NewOptString(h.srv.clearCookie())}, nil
}

// ── compte ──────────────────────────────────────────────────────────────────

func (h *handler) GetMe(ctx context.Context) (api.GetMeRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	res, err := h.id.GetUser(ctx2, &identityv1.GetUserRequest{Id: claimsFrom(ctx).Subject})
	if err != nil {
		return nil, err
	}
	me := toMe(res.GetUser())
	return &me, nil
}

func (h *handler) VerifyEmail(ctx context.Context, req *api.VerifyEmailRequest) (api.VerifyEmailRes, error) {
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.VerifyEmail(ctx, &identityv1.VerifyEmailRequest{Token: req.Token}); err != nil {
		return nil, err
	}
	return &api.VerifyEmailNoContent{}, nil
}

func (h *handler) ResendEmailVerification(ctx context.Context) (api.ResendEmailVerificationRes, error) {
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.ResendEmailVerification(ctx2, &identityv1.ResendEmailVerificationRequest{UserId: claimsFrom(ctx).Subject}); err != nil {
		return nil, err
	}
	return &api.Accepted{Status: api.AcceptedStatusAccepted}, nil
}

func (h *handler) ChangeMyPassword(ctx context.Context, req *api.ChangePasswordRequest) (api.ChangeMyPasswordRes, error) {
	c := claimsFrom(ctx)
	in := &identityv1.ChangePasswordRequest{UserId: c.Subject, CurrentPassword: req.CurrentPassword, NewPassword: req.NewPassword}
	if c.SID != "" {
		in.CurrentSessionId = &c.SID
	}
	ctx2, cancel := h.rpc(ctx)
	defer cancel()
	if _, err := h.id.ChangePassword(ctx2, in); err != nil {
		return nil, err
	}
	return &api.ChangeMyPasswordNoContent{}, nil
}

// ── cookie de rafraîchissement (G2, G16) ────────────────────────────────────

// refreshCookie : HttpOnly, Secure, SameSite=Strict, limité à /v1/auth ; jamais dans un corps de réponse.
func (s *Server) refreshCookie(value string, maxAgeSeconds int) string {
	return (&http.Cookie{Name: s.cfg.CookieName, Value: value, Path: s.cfg.CookiePath, Domain: s.cfg.CookieDomain, MaxAge: maxAgeSeconds, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode}).String()
}

func (s *Server) clearCookie() string {
	return (&http.Cookie{Name: s.cfg.CookieName, Value: "", Path: s.cfg.CookiePath, Domain: s.cfg.CookieDomain, MaxAge: -1, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode}).String()
}

// ── conversions ─────────────────────────────────────────────────────────────

func kindToProto(k string) identityv1.UserKind {
	if k == "company" {
		return identityv1.UserKind_USER_KIND_COMPANY
	}
	return identityv1.UserKind_USER_KIND_INDIVIDUAL
}

func toTokens(t *identityv1.TokenPair) api.AuthTokens {
	// Le jeton de rafraîchissement n'est PAS copié : il part uniquement dans le cookie.
	return api.AuthTokens{AccessToken: t.GetAccessToken(), TokenType: api.AuthTokensTokenTypeBearer, ExpiresIn: t.GetExpiresInSeconds()}
}

func toMe(u *identityv1.User) api.Me {
	id, _ := uuid.Parse(u.GetId())
	me := api.Me{
		ID: id, Email: u.GetEmail(), EmailVerified: u.GetEmailVerified(), Kind: api.MeKindIndividual, Locale: api.Locale(u.GetLocale()),
		PreferredCurrency: u.GetPreferredCurrency(), ProConditionsActive: u.GetProConditionsActive(), CreatedAt: u.GetCreatedAt().AsTime(),
	}
	if u.GetKind() == identityv1.UserKind_USER_KIND_COMPANY {
		me.Kind = api.MeKindCompany
	}
	if u.FirstName != nil {
		me.FirstName = api.NewNilString(u.GetFirstName())
	} else {
		me.FirstName.SetToNull()
	}
	if u.LastName != nil {
		me.LastName = api.NewNilString(u.GetLastName())
	} else {
		me.LastName.SetToNull()
	}
	if u.PhoneE164 != nil {
		me.Phone = api.NewNilPhoneE164(api.PhoneE164(u.GetPhoneE164()))
	} else {
		me.Phone.SetToNull()
	}
	if u.Company != nil {
		me.Company = api.NewNilCompany(toCompany(u.GetCompany()))
	} else {
		me.Company.SetToNull()
	}
	if u.AvatarUpdatedAt != nil {
		me.AvatarUpdatedAt = api.NewOptNilDateTime(u.GetAvatarUpdatedAt().AsTime())
	} else {
		me.AvatarUpdatedAt.SetToNull()
	}
	if u.CoverUpdatedAt != nil {
		me.CoverUpdatedAt = api.NewOptNilDateTime(u.GetCoverUpdatedAt().AsTime())
	} else {
		me.CoverUpdatedAt.SetToNull()
	}
	return me
}
