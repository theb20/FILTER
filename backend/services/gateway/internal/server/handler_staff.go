package server

import (
	"context"

	"github.com/google/uuid"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Authentification du personnel. Appelée par le SERVEUR du backoffice (pas par un navigateur) : les jetons passent
// dans le corps, jamais dans un cookie du gateway ; aucune couche CSRF n'est donc nécessaire ici.

func (h *handler) AdminLogin(ctx context.Context, req *api.StaffLoginRequest) (api.AdminLoginRes, error) {
	info := infoFrom(ctx)
	ctx, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	res, err := h.id.StaffLogin(ctx, &identityv1.StaffLoginRequest{Email: req.Email, Password: req.Password, ClientIp: info.IP})
	if err != nil {
		return nil, err
	}
	out := &api.StaffLoginResult{Status: api.StaffLoginResultStatus(res.GetStatus())}
	out.ChallengeToken = api.NewOptNilString(res.GetChallengeToken())
	out.Tokens.SetToNull()
	return out, nil
}

func (h *handler) AdminEnrollTwoFactor(ctx context.Context) (api.AdminEnrollTwoFactorRes, error) {
	tok := bearerFrom(ctx)
	ctx2, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	res, err := h.id.StaffEnrollTwoFactor(ctx2, &identityv1.StaffEnrollTwoFactorRequest{ChallengeToken: tok})
	if err != nil {
		return nil, err
	}
	return &api.TwoFactorEnrollment{Secret: res.GetSecret(), OtpauthUri: res.GetOtpauthUri(), RecoveryCodes: res.GetRecoveryCodes(), ChallengeToken: res.GetNextChallengeToken()}, nil
}

func (h *handler) AdminConfirmTwoFactor(ctx context.Context, req *api.TwoFactorCodeRequest) (api.AdminConfirmTwoFactorRes, error) {
	tok := bearerFrom(ctx)
	ctx2, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	res, err := h.id.StaffConfirmTwoFactor(ctx2, &identityv1.StaffConfirmTwoFactorRequest{ChallengeToken: tok, Code: req.Code})
	if err != nil {
		return nil, err
	}
	t := toStaffTokens(res.GetTokens())
	return &t, nil
}

func (h *handler) AdminVerifyTwoFactor(ctx context.Context, req *api.TwoFactorCodeRequest) (api.AdminVerifyTwoFactorRes, error) {
	tok := bearerFrom(ctx)
	ctx2, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	res, err := h.id.StaffVerifyTwoFactor(ctx2, &identityv1.StaffVerifyTwoFactorRequest{ChallengeToken: tok, Code: req.Code})
	if err != nil {
		return nil, err
	}
	t := toStaffTokens(res.GetTokens())
	return &t, nil
}

func (h *handler) AdminRefresh(ctx context.Context, req *api.RefreshRequest) (api.AdminRefreshRes, error) {
	ctx, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	res, err := h.id.StaffRefresh(ctx, &identityv1.StaffRefreshRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		return nil, err
	}
	t := toStaffTokens(res.GetTokens())
	return &t, nil
}

func (h *handler) AdminLogout(ctx context.Context, req *api.LogoutRequest) (api.AdminLogoutRes, error) {
	ctx, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	if _, err := h.id.StaffLogout(ctx, &identityv1.StaffLogoutRequest{RefreshToken: req.RefreshToken}); err != nil {
		return nil, err
	}
	return &api.AdminLogoutNoContent{}, nil
}

func (h *handler) AdminLogoutAll(ctx context.Context) (api.AdminLogoutAllRes, error) {
	sub := claimsFrom(ctx).Subject
	ctx2, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	if _, err := h.id.StaffLogoutAll(ctx2, &identityv1.StaffLogoutAllRequest{StaffId: sub}); err != nil {
		return nil, err
	}
	return &api.AdminLogoutAllNoContent{}, nil
}

func (h *handler) AdminGetMe(ctx context.Context) (api.AdminGetMeRes, error) {
	sub := claimsFrom(ctx).Subject
	ctx2, cancel := h.rpcAs(ctx, "staff")
	defer cancel()
	res, err := h.id.GetStaffProfile(ctx2, &identityv1.GetStaffProfileRequest{Id: sub})
	if err != nil {
		return nil, err
	}
	id, _ := uuid.Parse(res.GetUser().GetId())
	roles, perms := res.GetRoles(), res.GetPermissions()
	if roles == nil {
		roles = []string{}
	}
	if perms == nil {
		perms = []string{}
	}
	return &api.StaffMe{ID: id, Email: res.GetUser().GetEmail(), Name: res.GetUser().GetName(), Roles: roles, Permissions: perms, TwoFactorEnabled: res.GetUser().GetTwoFactorEnabled()}, nil
}

func toStaffTokens(t *identityv1.StaffTokens) api.StaffTokens {
	return api.StaffTokens{AccessToken: t.GetAccessToken(), RefreshToken: t.GetRefreshToken(), ExpiresIn: t.GetExpiresInSeconds()}
}
