package domain

import (
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
)

// IdentitySettings : réglages modifiables du service (contrat identity-settings). Les valeurs par défaut sont celles du contrat.
type IdentitySettings struct {
	AccessTokenMinutes       int
	CustomerRefreshDays      int
	StaffRefreshHours        int
	PasswordResetCodeMinutes int // réservé : la réinitialisation de mot de passe client n'est pas encore implémentée
	PasswordResetMaxAttempts int // idem
	StaffChallengeMinutes    int
	InvitationHours          int
	EmailVerificationHours   int
	EmailChangeHours         int
	RecentLoginMinutes       int
	ThrottleFreeAttempts     int
	ThrottleBaseDelaySeconds int
	ThrottleMaxDelaySeconds  int
	LoginAlertAfterFailures  int
	PersonalDataDays         *int // nil = non renseigné : aucune purge automatique
	AuditLogDays             *int
}

// DefaultIdentitySettings renvoie les défauts du contrat.
func DefaultIdentitySettings() IdentitySettings {
	return IdentitySettings{
		AccessTokenMinutes: 15, CustomerRefreshDays: 30, StaffRefreshHours: 8, PasswordResetCodeMinutes: 15, PasswordResetMaxAttempts: 5,
		StaffChallengeMinutes: 5, InvitationHours: 72, EmailVerificationHours: 48, EmailChangeHours: 24, RecentLoginMinutes: 10,
		ThrottleFreeAttempts: 3, ThrottleBaseDelaySeconds: 1, ThrottleMaxDelaySeconds: 900, LoginAlertAfterFailures: 5,
	}
}

// Validate applique les bornes du contrat (minimums) et la cohérence du ralentissement. Chaque champ fautif est signalé.
func (s IdentitySettings) Validate() error {
	var f []apperr.Field
	min1 := func(ptr string, v int) {
		if v < 1 {
			f = append(f, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: "doit être supérieur ou égal à 1"})
		}
	}
	min1("/accessTokenMinutes", s.AccessTokenMinutes)
	min1("/customerRefreshDays", s.CustomerRefreshDays)
	min1("/staffRefreshHours", s.StaffRefreshHours)
	min1("/passwordResetCodeMinutes", s.PasswordResetCodeMinutes)
	min1("/passwordResetMaxAttempts", s.PasswordResetMaxAttempts)
	min1("/staffChallengeMinutes", s.StaffChallengeMinutes)
	min1("/invitationHours", s.InvitationHours)
	min1("/emailVerificationHours", s.EmailVerificationHours)
	min1("/emailChangeHours", s.EmailChangeHours)
	min1("/recentLoginMinutes", s.RecentLoginMinutes)
	min1("/loginAlertAfterFailures", s.LoginAlertAfterFailures)
	min1("/loginThrottle/baseDelaySeconds", s.ThrottleBaseDelaySeconds)
	min1("/loginThrottle/maxDelaySeconds", s.ThrottleMaxDelaySeconds)
	if s.ThrottleFreeAttempts < 0 {
		f = append(f, apperr.Field{Pointer: "/loginThrottle/freeAttempts", Code: apperr.CodeValidation, Message: "doit être supérieur ou égal à 0"})
	}
	if s.ThrottleMaxDelaySeconds < s.ThrottleBaseDelaySeconds {
		f = append(f, apperr.Field{Pointer: "/loginThrottle/maxDelaySeconds", Code: apperr.CodeValidation, Message: "le délai maximal doit être supérieur ou égal au délai de base"})
	}
	for ptr, v := range map[string]*int{"/retention/personalDataDays": s.PersonalDataDays, "/retention/auditLogDays": s.AuditLogDays} {
		if v != nil && *v < 0 {
			f = append(f, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: "doit être supérieur ou égal à 0"})
		}
	}
	if len(f) > 0 {
		return apperr.Validation("réglages invalides", f...)
	}
	return nil
}

// Apply superpose ces réglages aux réglages d'exécution (les paramètres que le contrat ne rend pas modifiables sont conservés).
func (s IdentitySettings) Apply(base Settings) Settings {
	base.AccessTokenTTL = time.Duration(s.AccessTokenMinutes) * time.Minute
	base.CustomerRefreshTTL = time.Duration(s.CustomerRefreshDays) * 24 * time.Hour
	base.StaffRefreshTTL = time.Duration(s.StaffRefreshHours) * time.Hour
	base.StaffChallengeTTL = time.Duration(s.StaffChallengeMinutes) * time.Minute
	base.InvitationTTL = time.Duration(s.InvitationHours) * time.Hour
	base.EmailVerificationTTL = time.Duration(s.EmailVerificationHours) * time.Hour
	base.EmailChangeTTL = time.Duration(s.EmailChangeHours) * time.Hour
	base.RecentLoginWindow = time.Duration(s.RecentLoginMinutes) * time.Minute
	base.Throttle.FreeAttempts = s.ThrottleFreeAttempts
	base.Throttle.BaseDelay = time.Duration(s.ThrottleBaseDelaySeconds) * time.Second
	base.Throttle.MaxDelay = time.Duration(s.ThrottleMaxDelaySeconds) * time.Second
	base.LoginAlertAfterFailure = s.LoginAlertAfterFailures
	return base
}

// Snapshot pour l'audit.
func (s IdentitySettings) Snapshot() map[string]any {
	m := map[string]any{
		"accessTokenMinutes": s.AccessTokenMinutes, "customerRefreshDays": s.CustomerRefreshDays, "staffRefreshHours": s.StaffRefreshHours,
		"passwordResetCodeMinutes": s.PasswordResetCodeMinutes, "passwordResetMaxAttempts": s.PasswordResetMaxAttempts, "staffChallengeMinutes": s.StaffChallengeMinutes,
		"invitationHours": s.InvitationHours, "emailVerificationHours": s.EmailVerificationHours, "emailChangeHours": s.EmailChangeHours, "recentLoginMinutes": s.RecentLoginMinutes,
		"loginThrottle":           map[string]any{"freeAttempts": s.ThrottleFreeAttempts, "baseDelaySeconds": s.ThrottleBaseDelaySeconds, "maxDelaySeconds": s.ThrottleMaxDelaySeconds},
		"loginAlertAfterFailures": s.LoginAlertAfterFailures,
	}
	ret := map[string]any{"personalDataDays": nil, "auditLogDays": nil}
	if s.PersonalDataDays != nil {
		ret["personalDataDays"] = *s.PersonalDataDays
	}
	if s.AuditLogDays != nil {
		ret["auditLogDays"] = *s.AuditLogDays
	}
	m["retention"] = ret
	return m
}
