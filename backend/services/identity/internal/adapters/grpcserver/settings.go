package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

func settingsToProto(s domain.IdentitySettings) *identityv1.IdentitySettings {
	return &identityv1.IdentitySettings{
		AccessTokenMinutes: int32(s.AccessTokenMinutes), CustomerRefreshDays: int32(s.CustomerRefreshDays), StaffRefreshHours: int32(s.StaffRefreshHours),
		PasswordResetCodeMinutes: int32(s.PasswordResetCodeMinutes), PasswordResetMaxAttempts: int32(s.PasswordResetMaxAttempts), StaffChallengeMinutes: int32(s.StaffChallengeMinutes),
		InvitationHours: int32(s.InvitationHours), EmailVerificationHours: int32(s.EmailVerificationHours), EmailChangeHours: int32(s.EmailChangeHours), RecentLoginMinutes: int32(s.RecentLoginMinutes),
		LoginThrottle:           &identityv1.LoginThrottle{FreeAttempts: int32(s.ThrottleFreeAttempts), BaseDelaySeconds: int32(s.ThrottleBaseDelaySeconds), MaxDelaySeconds: int32(s.ThrottleMaxDelaySeconds)},
		LoginAlertAfterFailures: int32(s.LoginAlertAfterFailures), PersonalDataDays: optInt(s.PersonalDataDays), AuditLogDays: optInt(s.AuditLogDays),
	}
}

func optInt(v *int) *int32 {
	if v == nil {
		return nil
	}
	x := int32(*v)
	return &x
}

func intPtr(v *int32) *int {
	if v == nil || *v < 0 { // -1 : « effacer » (rétention fournie avec des null)
		return nil
	}
	x := int(*v)
	return &x
}

func (s *Server) GetSettings(ctx context.Context, _ *identityv1.GetSettingsRequest) (*identityv1.GetSettingsResponse, error) {
	v, err := s.Svc.GetIdentitySettings(ctx)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetSettingsResponse{Settings: settingsToProto(v)}, nil
}

// PutSettings : un sous-message absent (ralentissement, rétention) laisse les valeurs actuelles inchangées.
func (s *Server) PutSettings(ctx context.Context, r *identityv1.PutSettingsRequest) (*identityv1.PutSettingsResponse, error) {
	cur, err := s.Svc.GetIdentitySettings(ctx)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	in := r.GetSettings()
	next := domain.IdentitySettings{
		AccessTokenMinutes: int(in.GetAccessTokenMinutes()), CustomerRefreshDays: int(in.GetCustomerRefreshDays()), StaffRefreshHours: int(in.GetStaffRefreshHours()),
		PasswordResetCodeMinutes: int(in.GetPasswordResetCodeMinutes()), PasswordResetMaxAttempts: int(in.GetPasswordResetMaxAttempts()), StaffChallengeMinutes: int(in.GetStaffChallengeMinutes()),
		InvitationHours: int(in.GetInvitationHours()), EmailVerificationHours: int(in.GetEmailVerificationHours()), EmailChangeHours: int(in.GetEmailChangeHours()), RecentLoginMinutes: int(in.GetRecentLoginMinutes()),
		ThrottleFreeAttempts: cur.ThrottleFreeAttempts, ThrottleBaseDelaySeconds: cur.ThrottleBaseDelaySeconds, ThrottleMaxDelaySeconds: cur.ThrottleMaxDelaySeconds,
		LoginAlertAfterFailures: int(in.GetLoginAlertAfterFailures()), PersonalDataDays: cur.PersonalDataDays, AuditLogDays: cur.AuditLogDays,
	}
	if t := in.GetLoginThrottle(); t != nil {
		next.ThrottleFreeAttempts, next.ThrottleBaseDelaySeconds, next.ThrottleMaxDelaySeconds = int(t.GetFreeAttempts()), int(t.GetBaseDelaySeconds()), int(t.GetMaxDelaySeconds())
	}
	if in.PersonalDataDays != nil || in.AuditLogDays != nil {
		next.PersonalDataDays, next.AuditLogDays = intPtr(in.PersonalDataDays), intPtr(in.AuditLogDays)
	}
	out, err := s.Svc.PutIdentitySettings(ctx, actor(ctx), next)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.PutSettingsResponse{Settings: settingsToProto(out)}, nil
}
