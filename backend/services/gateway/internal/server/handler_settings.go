package server

import (
	"context"

	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

func toSettings(s *identityv1.IdentitySettings) api.IdentitySettings {
	out := api.IdentitySettings{
		AccessTokenMinutes: s.GetAccessTokenMinutes(), CustomerRefreshDays: s.GetCustomerRefreshDays(), StaffRefreshHours: s.GetStaffRefreshHours(),
		PasswordResetCodeMinutes: s.GetPasswordResetCodeMinutes(), PasswordResetMaxAttempts: s.GetPasswordResetMaxAttempts(), StaffChallengeMinutes: s.GetStaffChallengeMinutes(),
		InvitationHours: s.GetInvitationHours(), EmailVerificationHours: s.GetEmailVerificationHours(), EmailChangeHours: s.GetEmailChangeHours(), RecentLoginMinutes: s.GetRecentLoginMinutes(),
		LoginAlertAfterFailures: s.GetLoginAlertAfterFailures(),
	}
	t := s.GetLoginThrottle()
	out.LoginThrottle = api.NewOptIdentitySettingsLoginThrottle(api.IdentitySettingsLoginThrottle{FreeAttempts: t.GetFreeAttempts(), BaseDelaySeconds: t.GetBaseDelaySeconds(), MaxDelaySeconds: t.GetMaxDelaySeconds()})
	ret := api.IdentitySettingsRetention{}
	if s.PersonalDataDays != nil {
		ret.PersonalDataDays = api.NewNilInt32(s.GetPersonalDataDays())
	} else {
		ret.PersonalDataDays.SetToNull()
	}
	if s.AuditLogDays != nil {
		ret.AuditLogDays = api.NewNilInt32(s.GetAuditLogDays())
	} else {
		ret.AuditLogDays.SetToNull()
	}
	out.Retention = api.NewOptIdentitySettingsRetention(ret)
	return out
}

func (h *handler) AdminIdentitySettingsGet(ctx context.Context) (api.AdminIdentitySettingsGetRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.GetSettings(ctx, &identityv1.GetSettingsRequest{})
	if err != nil {
		return nil, err
	}
	s := toSettings(res.GetSettings())
	return &s, nil
}

func (h *handler) AdminIdentitySettingsPut(ctx context.Context, req *api.IdentitySettings) (api.AdminIdentitySettingsPutRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	in := &identityv1.IdentitySettings{
		AccessTokenMinutes: req.AccessTokenMinutes, CustomerRefreshDays: req.CustomerRefreshDays, StaffRefreshHours: req.StaffRefreshHours,
		PasswordResetCodeMinutes: req.PasswordResetCodeMinutes, PasswordResetMaxAttempts: req.PasswordResetMaxAttempts, StaffChallengeMinutes: req.StaffChallengeMinutes,
		InvitationHours: req.InvitationHours, EmailVerificationHours: req.EmailVerificationHours, EmailChangeHours: req.EmailChangeHours, RecentLoginMinutes: req.RecentLoginMinutes,
		LoginAlertAfterFailures: req.LoginAlertAfterFailures,
	}
	if t, ok := req.LoginThrottle.Get(); ok {
		in.LoginThrottle = &identityv1.LoginThrottle{FreeAttempts: t.FreeAttempts, BaseDelaySeconds: t.BaseDelaySeconds, MaxDelaySeconds: t.MaxDelaySeconds}
	}
	if r, ok := req.Retention.Get(); ok {
		if v, ok := r.PersonalDataDays.Get(); ok {
			in.PersonalDataDays = &v
		}
		if v, ok := r.AuditLogDays.Get(); ok {
			in.AuditLogDays = &v
		}
		// Retention fournie avec deux null : on la transmet comme « présente » pour effacer les valeurs
		if in.PersonalDataDays == nil && in.AuditLogDays == nil {
			zero := int32(-1)
			in.PersonalDataDays, in.AuditLogDays = &zero, &zero // -1 = effacer (traduit par identity)
		}
	}
	res, err := h.id.PutSettings(ctx, &identityv1.PutSettingsRequest{Settings: in})
	if err != nil {
		return nil, err
	}
	s := toSettings(res.GetSettings())
	return &s, nil
}
