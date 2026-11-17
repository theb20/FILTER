package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	adminCalls  = map[string]int{}
	adminActors []string
)

func (f *fakeID) ListRoles(ctx context.Context, _ *identityv1.ListRolesRequest, _ ...grpc.CallOption) (*identityv1.ListRolesResponse, error) {
	adminCalls["ListRoles"]++
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		adminActors = append(adminActors, md.Get("x-actor-id")[0]+"|"+md.Get("x-actor-type")[0])
	}
	return &identityv1.ListRolesResponse{Items: []*identityv1.Role{{Id: uuid.NewString(), Name: "super-admin", Permissions: []string{"*"}, System: true}}, Page: &commonv1.PageResponse{Page: 1, PageSize: 100, Total: 1, TotalPages: 1}}, nil
}

func (f *fakeID) CreateRole(_ context.Context, in *identityv1.CreateRoleRequest, _ ...grpc.CallOption) (*identityv1.CreateRoleResponse, error) {
	adminCalls["CreateRole"]++
	return &identityv1.CreateRoleResponse{Role: &identityv1.Role{Id: uuid.NewString(), Name: in.GetRole().GetName(), Permissions: in.GetRole().GetPermissions()}}, nil
}

func (f *fakeID) GetStaffProfile(_ context.Context, in *identityv1.GetStaffProfileRequest, _ ...grpc.CallOption) (*identityv1.GetStaffProfileResponse, error) {
	adminCalls["GetStaffProfile"]++
	return &identityv1.GetStaffProfileResponse{User: &identityv1.StaffUser{Id: in.GetId(), Email: "a@b.ci", Name: "A", CreatedAt: nil}, Roles: []string{}, Permissions: []string{}}, nil
}

func staffToken(t *testing.T, r *rig, sub string, perms ...string) string {
	t.Helper()
	tok, err := r.keys[jwtx.AudStaff].Sign(jwtx.AudStaff, sub, time.Minute, time.Now(), jwtx.Claims{SID: "sid-1", Roles: []string{"x"}, Perms: perms, AMR: []string{"pwd", "otp"}})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func bearer(tok string) map[string]string { return map[string]string{"Authorization": "Bearer " + tok} }

// Chaque route du personnel exige la permission du contrat (x-required-permission), contrôlée AVANT d'atteindre identity.
func TestPermissionsDuPersonnelContrôléesParLeGateway(t *testing.T) {
	r := newRig(t, nil)
	sub := uuid.NewString()
	cases := []struct {
		name         string
		method, path string
		body         string
		token        string
		want         int
	}{
		{"sans jeton", "GET", "/v1/admin/roles", "", "", 401},
		{"jeton client", "GET", "/v1/admin/roles", "", customerToken(t, r, "s", time.Minute), 401},
		{"sans permission", "GET", "/v1/admin/roles", "", staffToken(t, r, sub), 403},
		{"mauvaise permission", "GET", "/v1/admin/roles", "", staffToken(t, r, sub, "catalog:read"), 403},
		{"admin:read suffit pour lire", "GET", "/v1/admin/roles", "", staffToken(t, r, sub, "admin:read"), 200},
		{"admin:read ne suffit pas pour écrire", "POST", "/v1/admin/roles", `{"name":"X","permissions":["catalog:read"]}`, staffToken(t, r, sub, "admin:read"), 403},
		{"admin:write écrit", "POST", "/v1/admin/roles", `{"name":"X","permissions":["catalog:read"]}`, staffToken(t, r, sub, "admin:write"), 201},
		{"joker", "GET", "/v1/admin/roles", "", staffToken(t, r, sub, "*"), 200},
		{"libre-service : /admin/me sans permission", "GET", "/v1/admin/me", "", staffToken(t, r, sub), 200},
	}
	for _, c := range cases {
		h := map[string]string{}
		if c.token != "" {
			h = bearer(c.token)
		}
		rec := r.do(req{c.method, c.path, c.body, h})
		if rec.Code != c.want {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
			continue
		}
		if c.want == 403 && problemOf(t, rec)["code"] != apperr.CodeForbidden {
			t.Errorf("%s : code d'erreur %v", c.name, problemOf(t, rec)["code"])
		}
	}
}

func TestLesRefusDePermissionNAtteignentJamaisIdentity(t *testing.T) {
	r := newRig(t, nil)
	before := adminCalls["CreateRole"]
	r.do(req{"POST", "/v1/admin/roles", `{"name":"X","permissions":["catalog:read"]}`, bearer(staffToken(t, r, uuid.NewString(), "admin:read"))})
	if adminCalls["CreateRole"] != before {
		t.Fatal("identity a été appelé malgré le refus de permission")
	}
}

func TestIdentiteDeLActeurTransmiseAIdentity(t *testing.T) {
	r := newRig(t, nil)
	sub := uuid.NewString()
	adminActors = nil
	r.do(req{"GET", "/v1/admin/roles", "", bearer(staffToken(t, r, sub, "admin:read"))})
	if len(adminActors) != 1 || adminActors[0] != sub+"|staff" {
		t.Fatalf("acteur transmis : %v", adminActors)
	}
}

func TestPermissionsDuContratTouteslesRoutesDuPersonnelSontCouvertes(t *testing.T) {
	// Une opération du personnel inconnue de la table est refusée (fail closed) : on vérifie sur une route réelle sans entrée.
	r := newRig(t, nil)
	rec := r.do(req{"GET", "/v1/admin/permissions", "", bearer(staffToken(t, r, uuid.NewString(), "catalog:write"))})
	if rec.Code != 403 {
		t.Fatalf("/admin/permissions exige admin:read : %d", rec.Code)
	}
}

func (f *fakeID) GetSettings(_ context.Context, _ *identityv1.GetSettingsRequest, _ ...grpc.CallOption) (*identityv1.GetSettingsResponse, error) {
	return &identityv1.GetSettingsResponse{Settings: &identityv1.IdentitySettings{AccessTokenMinutes: 15, LoginThrottle: &identityv1.LoginThrottle{}}}, nil
}

func TestReglagesIdentityExigentAdminReadEtWrite(t *testing.T) {
	r := newRig(t, nil)
	sub := uuid.NewString()
	if rec := r.do(req{"GET", "/v1/admin/identity-settings", "", bearer(staffToken(t, r, sub, "catalog:read"))}); rec.Code != 403 {
		t.Fatalf("sans admin:read : %d", rec.Code)
	}
	if rec := r.do(req{"GET", "/v1/admin/identity-settings", "", bearer(staffToken(t, r, sub, "admin:read"))}); rec.Code != 200 {
		t.Fatalf("admin:read : %d %s", rec.Code, rec.Body.String())
	}
	body := `{"accessTokenMinutes":15,"customerRefreshDays":30,"staffRefreshHours":8,"passwordResetCodeMinutes":15,"passwordResetMaxAttempts":5,"staffChallengeMinutes":5,"invitationHours":72,"emailVerificationHours":48,"emailChangeHours":24,"recentLoginMinutes":10,"loginAlertAfterFailures":5}`
	if rec := r.do(req{"PUT", "/v1/admin/identity-settings", body, bearer(staffToken(t, r, sub, "admin:read"))}); rec.Code != 403 {
		t.Fatalf("admin:read ne doit pas suffire pour écrire : %d", rec.Code)
	}
}
