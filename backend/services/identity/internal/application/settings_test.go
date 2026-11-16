package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

func testReglagesParDefautEtValidation(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	got, err := e.svc.GetIdentitySettings(ctx)
	if err != nil || got != domain.DefaultIdentitySettings() {
		t.Fatalf("défauts du contrat attendus : %v %+v", err, got)
	}
	bad := got
	bad.AccessTokenMinutes, bad.InvitationHours = 0, -3
	bad.ThrottleBaseDelaySeconds, bad.ThrottleMaxDelaySeconds = 10, 5
	neg := -1
	bad.AuditLogDays = &neg
	_, err = e.svc.PutIdentitySettings(ctx, a, bad)
	var ae *apperr.Error
	if code(err) != apperr.CodeValidation || !asErr(err, &ae) || len(ae.Fields) != 4 {
		t.Fatalf("4 champs fautifs attendus : %v", err)
	}
	if again, _ := e.svc.GetIdentitySettings(ctx); again != got {
		t.Fatal("des réglages invalides ont été enregistrés")
	}
}

func testReglagesModifiesAppliquesEtAudites(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	in := domain.DefaultIdentitySettings()
	in.AccessTokenMinutes, in.StaffRefreshHours, in.ThrottleFreeAttempts = 5, 2, 1
	days := 400
	in.AuditLogDays = &days
	if _, err := e.svc.PutIdentitySettings(ctx, a, in); err != nil {
		t.Fatal(err)
	}
	got, _ := e.svc.GetIdentitySettings(ctx)
	if got.AccessTokenMinutes != 5 || got.AuditLogDays == nil || *got.AuditLogDays != 400 || got.PersonalDataDays != nil {
		t.Fatalf("relecture : %+v", got)
	}
	// effet réel : la durée du jeton d'accès suit le réglage
	r := mkRole(t, e, a, "Lecture", "admin:read")
	_, _, s := invite(t, e, a, "reglage@ff.test", r.ID)
	v, _ := jwtx.NewVerifierFromJWKS(e.keys[jwtx.AudStaff].JWKS(), jwtx.AudStaff, 0)
	if _, err := v.Verify(s.AccessToken, e.clock.Now().Add(3*time.Minute)); err != nil {
		t.Fatalf("jeton valide à 4 min (3 min après la création + 1 min d attente) : %v", err)
	}
	if _, err := v.Verify(s.AccessToken, e.clock.Now().Add(5*time.Minute)); err == nil {
		t.Fatal("jeton d'accès encore valide au-delà du réglage (5 min)")
	}
	if s.ExpiresIn != 5*time.Minute {
		t.Fatalf("expires_in : %v", s.ExpiresIn)
	}
	if n := len(auditActions(t, e)); n < 2 || auditActions(t, e)[n-1] == "" {
		t.Fatal("audit absent")
	}
	items, _, _ := e.svc.ListAudit(ctx, domain.AuditFilter{Action: "settings.update", Limit: 5})
	if len(items) != 1 || items[0].Before["accessTokenMinutes"] != float64(15) && items[0].Before["accessTokenMinutes"] != 15 || items[0].After == nil {
		t.Fatalf("audit avant/après : %+v", items)
	}
	_ = application.Actor{}
}

func asErr(err error, target **apperr.Error) bool {
	ae, ok := err.(*apperr.Error)
	if ok {
		*target = ae
	}
	return ok
}
