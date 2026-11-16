package application_test

import (
	"context"
	"encoding/base32"
	"fmt"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// adminEnv : un super-admin inscrit et connecté.
func adminEnv(t *testing.T) (*env, application.Actor, *domain.Role) {
	t.Helper()
	e := newEnv(t, nil)
	bootstrap(t, e)
	_, _, s := enroll(t, e)
	super, err := e.store.RoleByName(context.Background(), domain.SuperAdminRoleName)
	if err != nil {
		t.Fatal(err)
	}
	return e, application.Actor{ID: s.Staff.ID, IP: ip1, TraceID: "trace"}, super
}

func mkRole(t *testing.T, e *env, a application.Actor, name string, perms ...string) *domain.Role {
	t.Helper()
	r, err := e.svc.CreateRole(context.Background(), a, name, "", perms)
	if err != nil {
		t.Fatalf("CreateRole %s : %v", name, err)
	}
	return r
}

// invitedToken renvoie le dernier jeton d'invitation publié pour ce membre.
func invitedToken(t *testing.T, e *env, staffID string) string {
	t.Helper()
	var tok string
	for _, ev := range e.events.OfType(application.EvStaffInvitationRequested) {
		if m := ev.Payload.(*eventsv1.StaffInvitationRequested); m.StaffId == staffID {
			tok = m.Token
		}
	}
	if tok == "" {
		t.Fatal("aucun jeton d'invitation publié")
	}
	return tok
}

// activate fait accepter l'invitation, inscrit le 2FA et renvoie le secret TOTP et la session.
func activate(t *testing.T, e *env, u *domain.StaffUser, password, ip string) ([]byte, *application.StaffSession) {
	t.Helper()
	ctx := context.Background()
	res, err := e.svc.AcceptStaffInvitation(ctx, invitedToken(t, e, u.ID), password, ip)
	if err != nil || res.Status != application.StaffTwoFactorEnroll {
		t.Fatalf("AcceptStaffInvitation : %v %+v", err, res)
	}
	enr, err := e.svc.StaffEnrollTwoFactor(ctx, res.ChallengeToken, ip)
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(enr.Secret)
	s, err := e.svc.StaffConfirmTwoFactor(ctx, enr.NextChallengeToken, totp(e, secret, 0), ip, "ua")
	if err != nil {
		t.Fatal(err)
	}
	return secret, s
}

// invite crée et active un membre avec les rôles donnés.
func invite(t *testing.T, e *env, a application.Actor, email string, roleIDs ...string) (*domain.StaffUser, application.Actor, *application.StaffSession) {
	t.Helper()
	u, err := e.svc.CreateStaff(context.Background(), a, application.StaffInput{Email: email, Name: "Membre " + email, RoleIDs: roleIDs})
	if err != nil {
		t.Fatalf("CreateStaff %s : %v", email, err)
	}
	_, s := activate(t, e, u, "un mot de passe long pour "+email, "10.0.0.9")
	e.clock.Advance(time.Minute)
	return u, application.Actor{ID: u.ID, IP: "10.0.0.9"}, s
}

func auditActions(t *testing.T, e *env) []string {
	t.Helper()
	items, _, err := e.svc.ListAudit(context.Background(), domain.AuditFilter{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for i := len(items) - 1; i >= 0; i-- {
		out = append(out, items[i].Action)
	}
	return out
}

// ── rôles et permissions ────────────────────────────────────────────────────

func testAdminRolesCRUD(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Catalogue", "catalog:write", "catalog:read")
	if r.System || len(r.Permissions) != 2 || r.Permissions[0] != "catalog:read" {
		t.Fatalf("rôle créé : %+v", r)
	}
	if _, err := e.svc.CreateRole(ctx, a, "catalogue", "", []string{"catalog:read"}); code(err) != apperr.CodeConflict {
		t.Fatalf("nom en double (insensible à la casse) : %v", err)
	}
	for _, bad := range [][]string{{"orders:fly"}, {"*"}} {
		if _, err := e.svc.CreateRole(ctx, a, "Mauvais", "", bad); code(err) != apperr.CodeValidation {
			t.Fatalf("permissions %v acceptées : %v", bad, err)
		}
	}
	if _, err := e.svc.CreateRole(ctx, a, "  ", "", []string{"catalog:read"}); code(err) != apperr.CodeValidation {
		t.Fatalf("nom vide accepté : %v", err)
	}
	up, err := e.svc.UpdateRole(ctx, a, r.ID, "Catalogue & stock", "Gère le catalogue", []string{"catalog:read", "inventory:read"})
	if err != nil || up.Name != "Catalogue & stock" || up.Description != "Gère le catalogue" || len(up.Permissions) != 2 {
		t.Fatalf("UpdateRole : %v %+v", err, up)
	}
	got, err := e.svc.GetRole(ctx, r.ID)
	if err != nil || got.Name != "Catalogue & stock" {
		t.Fatalf("GetRole : %v", err)
	}
	if _, err := e.svc.GetRole(ctx, "00000000-0000-0000-0000-000000000000"); code(err) != apperr.CodeNotFound {
		t.Fatalf("rôle inconnu : %v", err)
	}
	if err := e.svc.DeleteRole(ctx, a, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.GetRole(ctx, r.ID); code(err) != apperr.CodeNotFound {
		t.Fatalf("rôle supprimé encore présent : %v", err)
	}
}

func testAdminRolesSystemEtAttribues(t *testing.T) {
	e, a, super := adminEnv(t)
	ctx := context.Background()
	if _, err := e.svc.UpdateRole(ctx, a, super.ID, "Autre", "", []string{"catalog:read"}); code(err) != apperr.CodeConflict {
		t.Fatalf("rôle système modifiable : %v", err)
	}
	if err := e.svc.DeleteRole(ctx, a, super.ID); code(err) != apperr.CodeConflict {
		t.Fatalf("rôle système supprimable : %v", err)
	}
	r := mkRole(t, e, a, "Support", "support:read")
	if _, err := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: "s@ff.test", Name: "S", RoleIDs: []string{r.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.DeleteRole(ctx, a, r.ID); code(err) != apperr.CodeConflict {
		t.Fatalf("rôle encore attribué supprimable : %v", err)
	}
}

func testAdminRolesListeRechercheEtPagination(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		mkRole(t, e, a, fmt.Sprintf("Rôle %d", i), "catalog:read")
	}
	p, err := e.svc.ListRoles(ctx, "", 1, 4)
	if err != nil || p.Total != 6 || len(p.Items) != 4 || p.Pages != 2 || !p.Items[0].System {
		t.Fatalf("page 1 : %v %+v", err, p)
	}
	p2, _ := e.svc.ListRoles(ctx, "", 2, 4)
	if len(p2.Items) != 2 {
		t.Fatalf("page 2 : %d", len(p2.Items))
	}
	q, _ := e.svc.ListRoles(ctx, "rôle 3", 1, 20)
	if q.Total != 1 || q.Items[0].Name != "Rôle 3" {
		t.Fatalf("recherche : %+v", q)
	}
	if n := len(e.svc.ListPermissions()); n != len(domain.Permissions) || n < 30 {
		t.Fatalf("permissions : %d", n)
	}
}

func testAdminPasDElevationDePrivileges(t *testing.T) {
	e, a, super := adminEnv(t)
	ctx := context.Background()
	limited := mkRole(t, e, a, "Admin limité", "admin:read", "admin:write", "catalog:read")
	_, lim, _ := invite(t, e, a, "limite@ff.test", limited.ID)

	if _, err := e.svc.CreateRole(ctx, lim, "Remboursements", "", []string{"orders:refund"}); code(err) != apperr.CodeForbidden {
		t.Fatalf("un admin limité a accordé une permission qu'il n'a pas : %v", err)
	}
	if _, err := e.svc.CreateStaff(ctx, lim, application.StaffInput{Email: "x@ff.test", Name: "X", RoleIDs: []string{super.ID}}); code(err) != apperr.CodeForbidden {
		t.Fatalf("un admin limité a attribué le rôle super-admin : %v", err)
	}
	if _, err := e.svc.UpdateStaff(ctx, lim, lim.ID, application.StaffInput{Email: "limite@ff.test", Name: "Moi", RoleIDs: []string{super.ID}}); code(err) != apperr.CodeForbidden {
		t.Fatalf("auto-promotion en super-admin : %v", err)
	}
	if _, err := e.svc.CreateRole(ctx, lim, "Sous-ensemble", "", []string{"catalog:read"}); err != nil {
		t.Fatalf("un sous-ensemble de ses permissions doit passer : %v", err)
	}
}

// ── personnel ───────────────────────────────────────────────────────────────

func testAdminInvitationComplete(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Lecture", "admin:read")
	u, err := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: "Nouveau@FF.test", Name: "Nouveau", RoleIDs: []string{r.ID, r.ID}})
	if err != nil || u.Status != domain.StaffInvited || u.Email != "nouveau@ff.test" || len(u.RoleIDs) != 1 {
		t.Fatalf("CreateStaff : %v %+v", err, u)
	}
	if _, err := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: "nouveau@ff.test", Name: "Doublon", RoleIDs: []string{r.ID}}); code(err) != apperr.CodeConflict {
		t.Fatalf("adresse en double : %v", err)
	}
	for name, in := range map[string]application.StaffInput{
		"sans rôle": {Email: "a@ff.test", Name: "A"}, "rôle inconnu": {Email: "b@ff.test", Name: "B", RoleIDs: []string{"00000000-0000-0000-0000-000000000000"}},
		"sans nom": {Email: "c@ff.test", Name: " ", RoleIDs: []string{r.ID}}, "e-mail invalide": {Email: "pas-un-email", Name: "D", RoleIDs: []string{r.ID}},
	} {
		if _, err := e.svc.CreateStaff(ctx, a, in); code(err) != apperr.CodeValidation {
			t.Errorf("%s accepté : %v", name, err)
		}
	}
	// un invité ne peut pas se connecter avant d'avoir accepté
	if _, err := e.svc.StaffLogin(ctx, "nouveau@ff.test", "n'importe quoi", ip1); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("connexion d'un invité : %v", err)
	}
	tok := invitedToken(t, e, u.ID)
	if _, err := e.svc.AcceptStaffInvitation(ctx, tok, "court", ip1); code(err) != apperr.CodeValidation {
		t.Fatalf("mot de passe court : %v", err)
	}
	pw := "mot de passe du nouveau membre"
	res, err := e.svc.AcceptStaffInvitation(ctx, tok, pw, ip1)
	if err != nil || res.ChallengeToken == "" {
		t.Fatalf("AcceptStaffInvitation : %v", err)
	}
	if _, err := e.svc.AcceptStaffInvitation(ctx, tok, pw, ip1); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("jeton d'invitation réutilisé : %v", err)
	}
	got, _ := e.svc.GetStaffUser(ctx, u.ID)
	if got.Status != domain.StaffActive {
		t.Fatalf("statut après acceptation : %s", got.Status)
	}
	if r2, err := e.svc.StaffLogin(ctx, "nouveau@ff.test", pw, ip1); err != nil || r2.Status != application.StaffTwoFactorEnroll {
		t.Fatalf("connexion après acceptation : %v %+v", err, r2)
	}
}

func testAdminInvitationExpireEtRenvoi(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Lecture", "admin:read")
	u, _ := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: "inv@ff.test", Name: "Inv", RoleIDs: []string{r.ID}})
	old := invitedToken(t, e, u.ID)
	if err := e.svc.ResendStaffInvitation(ctx, a, u.ID); err != nil {
		t.Fatal(err)
	}
	fresh := invitedToken(t, e, u.ID)
	if fresh == old {
		t.Fatal("le renvoi doit produire un nouveau jeton")
	}
	if _, err := e.svc.AcceptStaffInvitation(ctx, old, "mot de passe suffisamment long", ip1); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("l'ancien jeton doit être invalidé par le renvoi : %v", err)
	}
	e.clock.Advance(73 * time.Hour)
	if _, err := e.svc.AcceptStaffInvitation(ctx, fresh, "mot de passe suffisamment long", ip1); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("invitation expirée (72 h) acceptée : %v", err)
	}
	if err := e.svc.ResendStaffInvitation(ctx, a, a.ID); code(err) != apperr.CodeConflict {
		t.Fatalf("renvoi à un membre déjà actif : %v", err)
	}
}

func testAdminGardeFousSoiMemeEtDernierSuperAdmin(t *testing.T) {
	e, a, super := adminEnv(t)
	ctx := context.Background()
	if _, err := e.svc.DeactivateStaff(ctx, a, a.ID); code(err) != apperr.CodeSelfActionForbidden {
		t.Fatalf("auto-désactivation : %v", err)
	}
	me, _ := e.svc.GetStaffUser(ctx, a.ID)
	lim := mkRole(t, e, a, "Lecture seule", "admin:read")
	if _, err := e.svc.UpdateStaff(ctx, a, a.ID, application.StaffInput{Email: me.Email, Name: me.Name, RoleIDs: []string{lim.ID}}); code(err) != apperr.CodeSelfActionForbidden {
		t.Fatalf("retrait de son propre rôle d'administration : %v", err)
	}
	// un second administrateur (non super-admin) tente de toucher au dernier super-admin
	adminRole := mkRole(t, e, a, "Administrateurs", "admin:read", "admin:write")
	_, other, _ := invite(t, e, a, "autre@ff.test", adminRole.ID)
	if _, err := e.svc.DeactivateStaff(ctx, other, a.ID); code(err) != apperr.CodeLastSuperAdmin {
		t.Fatalf("désactivation du dernier super-admin : %v", err)
	}
	if _, err := e.svc.UpdateStaff(ctx, other, a.ID, application.StaffInput{Email: me.Email, Name: me.Name, RoleIDs: []string{adminRole.ID}}); code(err) != apperr.CodeLastSuperAdmin {
		t.Fatalf("retrait du rôle du dernier super-admin : %v", err)
	}
	// avec deux super-admins, c'est permis
	two, _, _ := invite(t, e, a, "deux@ff.test", super.ID)
	if _, err := e.svc.DeactivateStaff(ctx, a, two.ID); err != nil {
		t.Fatalf("désactivation d'un super-admin quand il y en a deux : %v", err)
	}
	if _, err := e.svc.ReactivateStaff(ctx, a, two.ID); err != nil {
		t.Fatal(err)
	}
}

func testAdminDesactivationRevoqueLesSessions(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Lecture", "admin:read")
	u, _, s := invite(t, e, a, "partant@ff.test", r.ID)
	d, err := e.svc.DeactivateStaff(ctx, a, u.ID)
	if err != nil || d.Status != domain.StaffDeactivated {
		t.Fatalf("DeactivateStaff : %v", err)
	}
	if _, err := e.svc.StaffRefresh(ctx, s.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("session d'un membre désactivé encore valide : %v", err)
	}
	if _, err := e.svc.StaffLogin(ctx, "partant@ff.test", "un mot de passe long pour partant@ff.test", "10.0.0.9"); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("connexion d'un membre désactivé : %v", err)
	}
	if again, err := e.svc.DeactivateStaff(ctx, a, u.ID); err != nil || again.Status != domain.StaffDeactivated {
		t.Fatalf("la désactivation doit être idempotente : %v", err)
	}
	re, err := e.svc.ReactivateStaff(ctx, a, u.ID)
	if err != nil || re.Status != domain.StaffActive {
		t.Fatalf("ReactivateStaff : %v %+v", err, re)
	}
	// un invité désactivé puis réactivé reste invité (aucun mot de passe)
	inv, _ := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: "jamais@ff.test", Name: "Jamais", RoleIDs: []string{r.ID}})
	_, _ = e.svc.DeactivateStaff(ctx, a, inv.ID)
	if back, _ := e.svc.ReactivateStaff(ctx, a, inv.ID); back.Status != domain.StaffInvited {
		t.Fatalf("un invité réactivé doit rester invité : %s", back.Status)
	}
}

func testAdminModifierUnMembre(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r1, r2 := mkRole(t, e, a, "R1", "admin:read"), mkRole(t, e, a, "R2", "catalog:read")
	u, _, s := invite(t, e, a, "modif@ff.test", r1.ID)
	up, err := e.svc.UpdateStaff(ctx, a, u.ID, application.StaffInput{Email: "modif@ff.test", Name: "Nouveau nom", RoleIDs: []string{r2.ID}})
	if err != nil || up.Name != "Nouveau nom" || len(up.RoleIDs) != 1 || up.RoleIDs[0] != r2.ID {
		t.Fatalf("UpdateStaff : %v %+v", err, up)
	}
	got, _ := e.svc.GetStaffUser(ctx, u.ID)
	if got.RoleIDs[0] != r2.ID {
		t.Fatal("rôles non enregistrés")
	}
	if _, err := e.svc.StaffRefresh(ctx, s.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("après changement de rôles, ses anciennes sessions doivent être révoquées : %v", err)
	}
	if _, err := e.svc.UpdateStaff(ctx, a, u.ID, application.StaffInput{Email: "autre@ff.test", Name: "N", RoleIDs: []string{r2.ID}}); code(err) != apperr.CodeValidation {
		t.Fatalf("changement d'adresse accepté : %v", err)
	}
	if _, err := e.svc.UpdateStaff(ctx, a, u.ID, application.StaffInput{Email: "modif@ff.test", Name: "N", RoleIDs: []string{r2.ID}, Status: domain.StaffDeactivated}); code(err) != apperr.CodeValidation {
		t.Fatalf("changement de statut par la modification : %v", err)
	}
	if _, err := e.svc.UpdateStaff(ctx, a, "00000000-0000-0000-0000-000000000000", application.StaffInput{Email: "x@ff.test", Name: "N", RoleIDs: []string{r2.ID}}); code(err) != apperr.CodeNotFound {
		t.Fatalf("membre inconnu : %v", err)
	}
}

func testAdminReinitialiserLe2FA(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Lecture", "admin:read")
	u, _, s := invite(t, e, a, "deux@ff.test", r.ID)
	if err := e.svc.ResetStaffTwoFactor(ctx, a, u.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.svc.GetStaffUser(ctx, u.ID)
	if got.TwoFactorEnabled || len(got.TOTPSecret) != 0 {
		t.Fatalf("2FA non effacé : %+v", got)
	}
	if _, err := e.svc.StaffRefresh(ctx, s.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("sessions non révoquées : %v", err)
	}
	if res, err := e.svc.StaffLogin(ctx, "deux@ff.test", "un mot de passe long pour deux@ff.test", "10.0.0.9"); err != nil || res.Status != application.StaffTwoFactorEnroll {
		t.Fatalf("après réinitialisation, l'inscription du 2FA est exigée : %v %+v", err, res)
	}
	if n := len(e.events.OfType(application.EvStaffTwoFactorReset)); n != 1 {
		t.Fatalf("la personne doit être notifiée : %d événements", n)
	}
}

func testAdminListeDuPersonnel(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Lecture", "admin:read")
	for i := 0; i < 4; i++ {
		if _, err := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: fmt.Sprintf("m%d@ff.test", i), Name: fmt.Sprintf("Membre %d", i), RoleIDs: []string{r.ID}}); err != nil {
			t.Fatal(err)
		}
		e.clock.Advance(time.Second)
	}
	p, err := e.svc.ListStaff(ctx, "", 1, 3)
	if err != nil || p.Total != 5 || len(p.Items) != 3 || p.Pages != 2 {
		t.Fatalf("liste : %v %+v", err, p)
	}
	if p.Items[0].Email != "m3@ff.test" {
		t.Fatalf("tri du plus récent au plus ancien : %s", p.Items[0].Email)
	}
	if q, _ := e.svc.ListStaff(ctx, "MEMBRE 2", 1, 20); q.Total != 1 || q.Items[0].Email != "m2@ff.test" {
		t.Fatalf("recherche : %+v", q)
	}
	if q, _ := e.svc.ListStaff(ctx, "100%", 1, 20); q.Total != 0 {
		t.Fatalf("les jokers LIKE doivent être échappés : %+v", q)
	}
}

func testAdminChangerSonMotDePasse(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Lecture", "admin:read")
	u, _, s1 := invite(t, e, a, "pw@ff.test", r.ID)
	old := "un mot de passe long pour pw@ff.test"
	if err := e.svc.ChangeStaffPassword(ctx, u.ID, "faux", "nouveau mot de passe long", s1.FamilyID); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("mauvais mot de passe actuel : %v", err)
	}
	// une seconde session, qui doit être révoquée
	secret, _ := e.svc.Secrets.Open(mustStaff(t, e, u.ID).TOTPSecret, u.ID)
	e.clock.Advance(time.Minute)
	l, _ := e.svc.StaffLogin(ctx, "pw@ff.test", old, "10.0.0.9")
	s2, err := e.svc.StaffVerifyTwoFactor(ctx, l.ChallengeToken, totp(e, secret, 0), "10.0.0.9", "ua")
	if err != nil {
		t.Fatal(err)
	}
	e.clock.Advance(time.Minute)
	if err := e.svc.ChangeStaffPassword(ctx, u.ID, old, "court", s1.FamilyID); code(err) != apperr.CodeValidation {
		t.Fatalf("nouveau mot de passe court : %v", err)
	}
	if err := e.svc.ChangeStaffPassword(ctx, u.ID, old, "nouveau mot de passe long", s1.FamilyID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.StaffRefresh(ctx, s2.RefreshToken, "10.0.0.9", "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("les autres sessions doivent être révoquées : %v", err)
	}
	if _, err := e.svc.StaffRefresh(ctx, s1.RefreshToken, "10.0.0.9", "ua"); err != nil {
		t.Fatalf("la session courante doit être conservée : %v", err)
	}
	if _, err := e.svc.StaffLogin(ctx, "pw@ff.test", old, "10.0.0.9"); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("l'ancien mot de passe fonctionne encore : %v", err)
	}
}

func mustStaff(t *testing.T, e *env, id string) *domain.StaffUser {
	t.Helper()
	u, err := e.store.StaffByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// ── audit ───────────────────────────────────────────────────────────────────

func testAdminJournalDAudit(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	r := mkRole(t, e, a, "Lecture", "admin:read")
	u, _ := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: "audit@ff.test", Name: "A", RoleIDs: []string{r.ID}})
	e.clock.Advance(time.Second)
	_, _ = e.svc.UpdateStaff(ctx, a, u.ID, application.StaffInput{Email: "audit@ff.test", Name: "A2", RoleIDs: []string{r.ID}})
	e.clock.Advance(time.Second)
	_, _ = e.svc.DeactivateStaff(ctx, a, u.ID)
	e.clock.Advance(time.Second)
	_ = e.svc.DeleteRole(ctx, a, mkRole(t, e, a, "Éphémère", "admin:read").ID)

	got := auditActions(t, e)
	want := []string{"role.create", "staff.invite", "staff.update", "staff.deactivate", "role.create", "role.delete"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("actions auditées : %v\n attendu : %v", got, want)
	}
	items, _, _ := e.svc.ListAudit(ctx, domain.AuditFilter{EntityID: u.ID, Limit: 50})
	if len(items) != 3 {
		t.Fatalf("filtre par entité : %d", len(items))
	}
	upd := items[1] // du plus récent au plus ancien : deactivate, update, invite
	if upd.Action != "staff.update" || upd.ActorID != a.ID || upd.IP != ip1 || upd.TraceID != "trace" || upd.Before["name"] != "A" || upd.After["name"] != "A2" {
		t.Fatalf("avant/après de la modification : %+v", upd)
	}
	// pagination par curseur, sans doublon ni trou
	var seen []string
	cursor := ""
	for i := 0; i < 10; i++ {
		page, next, err := e.svc.ListAudit(ctx, domain.AuditFilter{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range page {
			seen = append(seen, it.ID)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if len(seen) != len(want) {
		t.Fatalf("pagination : %d entrées lues sur %d", len(seen), len(want))
	}
	if byAction, _, _ := e.svc.ListAudit(ctx, domain.AuditFilter{Action: "role.create", Limit: 50}); len(byAction) != 2 {
		t.Fatalf("filtre par action : %d", len(byAction))
	}
	if _, _, err := e.svc.ListAudit(ctx, domain.AuditFilter{Cursor: "n'importe quoi", Limit: 5}); err == nil {
		t.Fatal("curseur invalide accepté")
	}
}

// Un échec annule aussi l'audit : pas de trace d'une action qui n'a pas eu lieu.
func testAdminAuditEtActionSontAtomiques(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	before := len(auditActions(t, e))
	if _, err := e.svc.CreateStaff(ctx, a, application.StaffInput{Email: "x@ff.test", Name: "X", RoleIDs: []string{"00000000-0000-0000-0000-000000000000"}}); err == nil {
		t.Fatal("erreur attendue")
	}
	if after := len(auditActions(t, e)); after != before {
		t.Fatalf("une action refusée a laissé une trace d'audit (%d → %d)", before, after)
	}
}
