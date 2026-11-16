package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// metaIP lit l'adresse du client transmise par le gateway (les jetons de défi sont liés à cette adresse).
func metaIP(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-client-ip"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func staffTokens(s *application.StaffSession) *identityv1.StaffTokens {
	return &identityv1.StaffTokens{AccessToken: s.AccessToken, RefreshToken: s.RefreshToken, ExpiresInSeconds: int32(s.ExpiresIn.Seconds())}
}

func staffToProto(u *domain.StaffUser) *identityv1.StaffUser {
	st := identityv1.StaffStatus_STAFF_STATUS_UNSPECIFIED
	switch u.Status {
	case domain.StaffInvited:
		st = identityv1.StaffStatus_STAFF_STATUS_INVITED
	case domain.StaffActive:
		st = identityv1.StaffStatus_STAFF_STATUS_ACTIVE
	case domain.StaffDeactivated:
		st = identityv1.StaffStatus_STAFF_STATUS_DEACTIVATED
	}
	out := &identityv1.StaffUser{Id: u.ID, Email: u.Email, Name: u.Name, RoleIds: u.RoleIDs, Status: st, TwoFactorEnabled: u.TwoFactorEnabled, CreatedAt: timestamppb.New(u.CreatedAt), UpdatedAt: timestamppb.New(u.CreatedAt)}
	if u.LastLoginAt != nil {
		out.LastLoginAt = timestamppb.New(*u.LastLoginAt)
	}
	return out
}

func (s *Server) StaffLogin(ctx context.Context, r *identityv1.StaffLoginRequest) (*identityv1.StaffLoginResponse, error) {
	ip := r.GetClientIp()
	if ip == "" {
		ip = metaIP(ctx)
	}
	res, err := s.Svc.StaffLogin(ctx, r.GetEmail(), r.GetPassword(), ip)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.StaffLoginResponse{Status: res.Status, ChallengeToken: &res.ChallengeToken}, nil
}

func (s *Server) StaffEnrollTwoFactor(ctx context.Context, r *identityv1.StaffEnrollTwoFactorRequest) (*identityv1.StaffEnrollTwoFactorResponse, error) {
	e, err := s.Svc.StaffEnrollTwoFactor(ctx, r.GetChallengeToken(), metaIP(ctx))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.StaffEnrollTwoFactorResponse{Secret: e.Secret, OtpauthUri: e.OTPAuthURI, RecoveryCodes: e.RecoveryCodes, NextChallengeToken: e.NextChallengeToken}, nil
}

func (s *Server) StaffConfirmTwoFactor(ctx context.Context, r *identityv1.StaffConfirmTwoFactorRequest) (*identityv1.StaffConfirmTwoFactorResponse, error) {
	sess, err := s.Svc.StaffConfirmTwoFactor(ctx, r.GetChallengeToken(), r.GetCode(), metaIP(ctx), "")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.StaffConfirmTwoFactorResponse{Tokens: staffTokens(sess)}, nil
}

func (s *Server) StaffVerifyTwoFactor(ctx context.Context, r *identityv1.StaffVerifyTwoFactorRequest) (*identityv1.StaffVerifyTwoFactorResponse, error) {
	sess, err := s.Svc.StaffVerifyTwoFactor(ctx, r.GetChallengeToken(), r.GetCode(), metaIP(ctx), "")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.StaffVerifyTwoFactorResponse{Tokens: staffTokens(sess)}, nil
}

func (s *Server) StaffRefresh(ctx context.Context, r *identityv1.StaffRefreshRequest) (*identityv1.StaffRefreshResponse, error) {
	sess, err := s.Svc.StaffRefresh(ctx, r.GetRefreshToken(), metaIP(ctx), "")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.StaffRefreshResponse{Tokens: staffTokens(sess)}, nil
}

func (s *Server) StaffLogout(ctx context.Context, r *identityv1.StaffLogoutRequest) (*identityv1.StaffLogoutResponse, error) {
	return &identityv1.StaffLogoutResponse{}, apperr.ToGRPC(s.Svc.StaffLogout(ctx, r.GetRefreshToken()))
}

func (s *Server) StaffLogoutAll(ctx context.Context, r *identityv1.StaffLogoutAllRequest) (*identityv1.StaffLogoutAllResponse, error) {
	return &identityv1.StaffLogoutAllResponse{}, apperr.ToGRPC(s.Svc.StaffLogoutAll(ctx, r.GetStaffId()))
}

func (s *Server) GetStaffProfile(ctx context.Context, r *identityv1.GetStaffProfileRequest) (*identityv1.GetStaffProfileResponse, error) {
	u, roles, perms, err := s.Svc.StaffProfile(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetStaffProfileResponse{User: staffToProto(u), Roles: roles, Permissions: perms}, nil
}
