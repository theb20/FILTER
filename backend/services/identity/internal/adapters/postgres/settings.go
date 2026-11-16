package postgres

import (
	"context"
	"time"

	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var _ application.SettingsRepo = (*Store)(nil)

func (s *Store) LoadIdentitySettings(ctx context.Context) (domain.IdentitySettings, error) {
	var v domain.IdentitySettings
	err := s.q(ctx).QueryRow(ctx, `SELECT access_token_minutes, customer_refresh_days, staff_refresh_hours, password_reset_code_minutes, password_reset_max_attempts,
		staff_challenge_minutes, invitation_hours, email_verification_hours, email_change_hours, recent_login_minutes,
		throttle_free_attempts, throttle_base_delay_seconds, throttle_max_delay_seconds, login_alert_after_failures, personal_data_days, audit_log_days
		FROM identity_settings WHERE id = 1`).Scan(&v.AccessTokenMinutes, &v.CustomerRefreshDays, &v.StaffRefreshHours, &v.PasswordResetCodeMinutes, &v.PasswordResetMaxAttempts,
		&v.StaffChallengeMinutes, &v.InvitationHours, &v.EmailVerificationHours, &v.EmailChangeHours, &v.RecentLoginMinutes,
		&v.ThrottleFreeAttempts, &v.ThrottleBaseDelaySeconds, &v.ThrottleMaxDelaySeconds, &v.LoginAlertAfterFailures, &v.PersonalDataDays, &v.AuditLogDays)
	return v, err
}

func (s *Store) SaveIdentitySettings(ctx context.Context, v domain.IdentitySettings, updatedBy string, at time.Time) error {
	_, err := s.q(ctx).Exec(ctx, `UPDATE identity_settings SET access_token_minutes=$1, customer_refresh_days=$2, staff_refresh_hours=$3, password_reset_code_minutes=$4, password_reset_max_attempts=$5,
		staff_challenge_minutes=$6, invitation_hours=$7, email_verification_hours=$8, email_change_hours=$9, recent_login_minutes=$10,
		throttle_free_attempts=$11, throttle_base_delay_seconds=$12, throttle_max_delay_seconds=$13, login_alert_after_failures=$14,
		personal_data_days=$15, audit_log_days=$16, updated_at=$17, updated_by=$18 WHERE id = 1`,
		v.AccessTokenMinutes, v.CustomerRefreshDays, v.StaffRefreshHours, v.PasswordResetCodeMinutes, v.PasswordResetMaxAttempts,
		v.StaffChallengeMinutes, v.InvitationHours, v.EmailVerificationHours, v.EmailChangeHours, v.RecentLoginMinutes,
		v.ThrottleFreeAttempts, v.ThrottleBaseDelaySeconds, v.ThrottleMaxDelaySeconds, v.LoginAlertAfterFailures, v.PersonalDataDays, v.AuditLogDays, at, updatedBy)
	return err
}
