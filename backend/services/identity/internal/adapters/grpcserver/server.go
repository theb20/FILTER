// Package grpcserver expose les cas d'usage d'identity en gRPC (contrat proto/identity/v1).
// Les RPC non encore implémentées répondent UNIMPLEMENTED (jamais une réponse trompeuse).
package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implémente identityv1.IdentityServiceServer pour le lot « inscription / session ».
type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	Svc *application.Service
}

func (s *Server) Register(ctx context.Context, r *identityv1.RegisterRequest) (*identityv1.RegisterResponse, error) {
	in := application.RegisterInput{
		Kind: kindFromProto(r.GetKind()), Email: r.GetEmail(), Password: r.GetPassword(), FirstName: r.GetFirstName(), LastName: r.GetLastName(),
		Locale: r.GetLocale(), ClientIP: r.GetClientIp(),
	}
	if c := r.GetCompany(); c != nil {
		in.Company = &application.CompanyInput{LegalName: c.GetLegalName(), TaxID: c.GetTaxId(), Country: c.GetCountry()}
	}
	if err := s.Svc.Register(ctx, in); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.RegisterResponse{Status: "accepted"}, nil
}

func (s *Server) Login(ctx context.Context, r *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	sess, err := s.Svc.Login(ctx, application.LoginInput{Email: r.GetEmail(), Password: r.GetPassword(), IP: r.GetClientIp(), UserAgent: r.GetUserAgent()})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.LoginResponse{User: userToProto(sess.User), Tokens: tokens(sess), RefreshMaxAgeSeconds: int32(sess.RefreshMaxAge.Seconds())}, nil
}

func (s *Server) RefreshTokens(ctx context.Context, r *identityv1.RefreshTokensRequest) (*identityv1.RefreshTokensResponse, error) {
	sess, err := s.Svc.Refresh(ctx, r.GetRefreshToken(), r.GetClientIp(), r.GetUserAgent())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.RefreshTokensResponse{Tokens: tokens(sess), RefreshMaxAgeSeconds: int32(sess.RefreshMaxAge.Seconds())}, nil
}

func (s *Server) Logout(ctx context.Context, r *identityv1.LogoutRequest) (*identityv1.LogoutResponse, error) {
	return &identityv1.LogoutResponse{}, apperr.ToGRPC(s.Svc.Logout(ctx, r.GetRefreshToken()))
}

func (s *Server) LogoutAll(ctx context.Context, r *identityv1.LogoutAllRequest) (*identityv1.LogoutAllResponse, error) {
	return &identityv1.LogoutAllResponse{}, apperr.ToGRPC(s.Svc.LogoutAll(ctx, r.GetUserId()))
}

func (s *Server) VerifyEmail(ctx context.Context, r *identityv1.VerifyEmailRequest) (*identityv1.VerifyEmailResponse, error) {
	return &identityv1.VerifyEmailResponse{}, apperr.ToGRPC(s.Svc.VerifyEmail(ctx, r.GetToken()))
}

func (s *Server) ResendEmailVerification(ctx context.Context, r *identityv1.ResendEmailVerificationRequest) (*identityv1.ResendEmailVerificationResponse, error) {
	return &identityv1.ResendEmailVerificationResponse{}, apperr.ToGRPC(s.Svc.ResendEmailVerification(ctx, r.GetUserId()))
}

func (s *Server) GetUser(ctx context.Context, r *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	u, err := s.Svc.GetUser(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetUserResponse{User: userToProto(u)}, nil
}

func (s *Server) ChangePassword(ctx context.Context, r *identityv1.ChangePasswordRequest) (*identityv1.ChangePasswordResponse, error) {
	err := s.Svc.ChangePassword(ctx, r.GetUserId(), r.GetCurrentPassword(), r.GetNewPassword(), r.GetCurrentSessionId())
	return &identityv1.ChangePasswordResponse{}, apperr.ToGRPC(err)
}

func (s *Server) GetJwks(_ context.Context, r *identityv1.GetJwksRequest) (*identityv1.GetJwksResponse, error) {
	b, err := s.Svc.JWKS(r.GetAudience())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetJwksResponse{JwksJson: string(b)}, nil
}

// ── conversions ─────────────────────────────────────────────────────────────

func kindFromProto(k identityv1.UserKind) string {
	switch k {
	case identityv1.UserKind_USER_KIND_COMPANY:
		return domain.KindCompany
	case identityv1.UserKind_USER_KIND_INDIVIDUAL:
		return domain.KindIndividual
	default:
		return ""
	}
}

func kindToProto(k string) identityv1.UserKind {
	if k == domain.KindCompany {
		return identityv1.UserKind_USER_KIND_COMPANY
	}
	return identityv1.UserKind_USER_KIND_INDIVIDUAL
}

func tokens(s *application.Session) *identityv1.TokenPair {
	return &identityv1.TokenPair{AccessToken: s.AccessToken, RefreshToken: s.RefreshToken, ExpiresInSeconds: int32(s.ExpiresIn.Seconds())}
}

func userToProto(u *domain.User) *identityv1.User {
	out := &identityv1.User{
		Id: u.ID, Email: u.Email, EmailVerified: u.EmailVerified, Kind: kindToProto(u.Kind), Locale: u.Locale, PreferredCurrency: u.PreferredCurrency,
		ProConditionsActive: u.ProActive(), Status: u.Status, CreatedAt: timestamppb.New(u.CreatedAt),
	}
	if u.FirstName != "" {
		out.FirstName = &u.FirstName
	}
	if u.LastName != "" {
		out.LastName = &u.LastName
	}
	if u.Company != nil {
		out.Company = companyToProto(u.Company)
	}
	return out
}

// ── mot de passe oublié ──

func (s *Server) ForgotPassword(ctx context.Context, r *identityv1.ForgotPasswordRequest) (*identityv1.ForgotPasswordResponse, error) {
	return &identityv1.ForgotPasswordResponse{}, apperr.ToGRPC(s.Svc.ForgotPassword(ctx, r.GetEmail(), r.GetLocale()))
}

func (s *Server) VerifyPasswordCode(ctx context.Context, r *identityv1.VerifyPasswordCodeRequest) (*identityv1.VerifyPasswordCodeResponse, error) {
	tok, ttl, err := s.Svc.VerifyPasswordCode(ctx, r.GetEmail(), r.GetCode())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.VerifyPasswordCodeResponse{ResetToken: tok, ExpiresInSeconds: int32(ttl.Seconds())}, nil
}

func (s *Server) ResetPassword(ctx context.Context, r *identityv1.ResetPasswordRequest) (*identityv1.ResetPasswordResponse, error) {
	return &identityv1.ResetPasswordResponse{}, apperr.ToGRPC(s.Svc.ResetPassword(ctx, r.GetResetToken(), r.GetNewPassword()))
}

// ── connexion via un fournisseur ──

func (s *Server) StartOAuth(ctx context.Context, r *identityv1.StartOAuthRequest) (*identityv1.StartOAuthResponse, error) {
	u, state, bind, ttl, err := s.Svc.OAuthStart(ctx, r.GetProvider(), r.GetRedirectUri())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.StartOAuthResponse{AuthorizationUrl: u, State: state, Binding: bind, TtlSeconds: int32(ttl.Seconds())}, nil
}

func (s *Server) CompleteOAuth(ctx context.Context, r *identityv1.CompleteOAuthRequest) (*identityv1.CompleteOAuthResponse, error) {
	res := s.Svc.OAuthComplete(ctx, r.GetProvider(), r.GetCode(), r.GetState(), r.GetBinding())
	return &identityv1.CompleteOAuthResponse{ExchangeCode: res.ExchangeCode, RedirectUri: res.RedirectURI, Failed: res.Failed}, nil
}

func (s *Server) ExchangeOAuthCode(ctx context.Context, r *identityv1.ExchangeOAuthCodeRequest) (*identityv1.ExchangeOAuthCodeResponse, error) {
	sess, err := s.Svc.ExchangeOAuthCode(ctx, r.GetExchangeCode(), "", "")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.ExchangeOAuthCodeResponse{User: userToProto(sess.User), Tokens: tokens(sess), RefreshMaxAgeSeconds: int32(sess.RefreshMaxAge.Seconds())}, nil
}
