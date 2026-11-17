package application_test

import (
	"context"
	"encoding/base32"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

const (
	adminMail = "admin@filterfinder.test"
	adminPw   = "un mot de passe long et unique"
	ip1       = "10.0.0.1"
)

func bootstrap(t *testing.T, e *env) {
	t.Helper()
	if _, err := e.svc.BootstrapAdmin(context.Background(), adminMail, "Administrateur", adminPw); err != nil {
		t.Fatalf("BootstrapAdmin: %v", err)
	}
}

func staffLogin(t *testing.T, e *env, ip string) *application.StaffLoginResult {
	t.Helper()
	r, err := e.svc.StaffLogin(context.Background(), adminMail, adminPw, ip)
	if err != nil {
		t.Fatalf("StaffLogin: %v", err)
	}
	return r
}

func totp(e *env, secret []byte, offset int64) string {
	return domain.TOTPCode(secret, e.clock.Now().Unix()/30+offset)
}

// enroll fait le parcours complet de première connexion et renvoie le secret et les codes de secours.
func enroll(t *testing.T, e *env) ([]byte, []string, *application.StaffSession) {
	t.Helper()
	ctx := context.Background()
	r := staffLogin(t, e, ip1)
	if r.Status != application.StaffTwoFactorEnroll {
		t.Fatalf("statut %q, attendu inscription 2FA", r.Status)
	}
	enr, err := e.svc.StaffEnrollTwoFactor(ctx, r.ChallengeToken, ip1)
	if err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(enr.Secret)
	if err != nil || len(enr.RecoveryCodes) != 8 || enr.NextChallengeToken == "" {
		t.Fatalf("inscription incomplète : %v %+v", err, enr)
	}
	s, err := e.svc.StaffConfirmTwoFactor(ctx, enr.NextChallengeToken, totp(e, secret, 0), ip1, "ua")
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	return secret, enr.RecoveryCodes, s
}

func testStaffAmorcageUnSeulSuperAdmin(t *testing.T) {
	e := newEnv(t, fakeBreach{"mot de passe compromis!": true})
	ctx := context.Background()
	if _, err := e.svc.BootstrapAdmin(ctx, adminMail, "A", "court"); code(err) != apperr.CodeValidation {
		t.Fatalf("mot de passe trop court accepté : %v", err)
	}
	if _, err := e.svc.BootstrapAdmin(ctx, adminMail, "A", "onze chars!"); code(err) != apperr.CodeValidation {
		t.Fatalf("mot de passe de 11 caractères accepté pour le personnel (minimum 12) : %v", err)
	}
	if _, err := e.svc.BootstrapAdmin(ctx, adminMail, "A", "mot de passe compromis!"); code(err) != apperr.CodeValidation {
		t.Fatalf("mot de passe compromis accepté : %v", err)
	}
	if _, err := e.svc.BootstrapAdmin(ctx, "pas-un-email", "A", adminPw); code(err) != apperr.CodeValidation {
		t.Fatalf("e-mail invalide accepté : %v", err)
	}
	bootstrap(t, e)
	if _, err := e.svc.BootstrapAdmin(ctx, "autre@filterfinder.test", "Autre", adminPw); code(err) != apperr.CodeConflict {
		t.Fatalf("un 2e super-admin ne doit pas pouvoir être amorcé : %v", err)
	}
	u, _, _, err := func() (*domain.StaffUser, []string, []string, error) {
		u, err := e.store.StaffByEmail(ctx, adminMail)
		if err != nil {
			return nil, nil, nil, err
		}
		return e.svc.StaffProfile(ctx, u.ID)
	}()
	if err != nil || u.TwoFactorEnabled || u.Status != domain.StaffActive || u.PasswordHash == adminPw || u.PasswordHash == "" {
		t.Fatalf("compte amorcé incohérent : %+v %v", u, err)
	}
}

func testStaffPremiereConnexionInscritLe2FA(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	_, _, s := enroll(t, e)

	v, err := jwtx.NewVerifierFromJWKS(e.keys[jwtx.AudStaff].JWKS(), jwtx.AudStaff, 0)
	if err != nil {
		t.Fatal(err)
	}
	c, err := v.Verify(s.AccessToken, e.clock.Now())
	if err != nil || c.SID == "" || len(c.Roles) != 1 || c.Roles[0] != domain.SuperAdminRoleName || len(c.Perms) != 1 || c.Perms[0] != "*" || len(c.AMR) != 2 {
		t.Fatalf("jeton d'accès du personnel incohérent : %+v %v", c, err)
	}
	if s.RefreshToken == "" || s.ExpiresIn != 15*time.Minute {
		t.Fatalf("session : %+v", s)
	}
	u, roles, perms, err := e.svc.StaffProfile(ctx, s.Staff.ID)
	if err != nil || !u.TwoFactorEnabled || u.LastLoginAt == nil || roles[0] != "super-admin" || perms[0] != "*" {
		t.Fatalf("profil : %+v %v %v %v", u, roles, perms, err)
	}
	// le secret TOTP n'est jamais stocké en clair
	if len(u.TOTPSecret) == 0 {
		t.Fatal("secret absent")
	}
	// un jeton d'accès CLIENT n'ouvre pas la porte du personnel (audiences distinctes)
	cv, _ := jwtx.NewVerifierFromJWKS(e.keys[jwtx.AudCustomer].JWKS(), jwtx.AudCustomer, 0)
	if _, err := cv.Verify(s.AccessToken, e.clock.Now()); err == nil {
		t.Fatal("un jeton du personnel ne doit pas être valide avec la clé des clients")
	}
}

func testStaffConnexionAvecTOTPEtUsageUnique(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	secret, _, _ := enroll(t, e)

	e.clock.Advance(time.Minute) // nouveau pas TOTP (le précédent a servi à confirmer)
	r := staffLogin(t, e, ip1)
	if r.Status != application.StaffTwoFactor {
		t.Fatalf("statut %q", r.Status)
	}
	code := totp(e, secret, 0)
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, code, ip1, "ua"); err != nil {
		t.Fatalf("Verify : %v", err)
	}
	// le jeton est à USAGE UNIQUE
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, code, ip1, "ua"); code2(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton de défi réutilisé : %v", err)
	}
	// le CODE est à usage unique aussi (rejeu d'un code espionné)
	r2 := staffLogin(t, e, ip1)
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r2.ChallengeToken, code, ip1, "ua"); code2(err) != apperr.CodeInvalidCode {
		t.Fatalf("un code TOTP déjà utilisé a été accepté : %v", err)
	}
	// le pas suivant passe avec le même jeton de défi (l'échec n'invalide pas le jeton)
	e.clock.Advance(30 * time.Second)
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r2.ChallengeToken, totp(e, secret, 0), ip1, "ua"); err != nil {
		t.Fatalf("code du pas suivant refusé : %v", err)
	}
}

func code2(err error) string { return code(err) }

func testStaffCodeDeSecoursUneSeuleFois(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	_, recovery, _ := enroll(t, e)
	r := staffLogin(t, e, ip1)
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, recovery[0], ip1, "ua"); err != nil {
		t.Fatalf("code de secours refusé : %v", err)
	}
	r = staffLogin(t, e, ip1)
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, recovery[0], ip1, "ua"); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("code de secours réutilisé : %v", err)
	}
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, recovery[1], ip1, "ua"); err != nil {
		t.Fatalf("un autre code de secours doit passer : %v", err)
	}
}

func testStaffCinqCodesErronesInvalidentLeJeton(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	secret, _, _ := enroll(t, e)
	e.clock.Advance(time.Minute)
	r := staffLogin(t, e, ip1)
	for i := 1; i <= 5; i++ {
		_, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, "000000", ip1, "ua")
		want := apperr.CodeInvalidCode
		if i == 5 {
			want = apperr.CodeUnauthenticated // 5e échec : le jeton est mort
		}
		if code(err) != want {
			t.Fatalf("essai %d : %v (attendu %s)", i, err, want)
		}
	}
	// même le BON code ne ressuscite pas un jeton épuisé
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, totp(e, secret, 0), ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton épuisé accepté : %v", err)
	}
}

func testStaffJetonDeDefiLieAIPPorteeEtExpiration(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	secret, _, _ := enroll(t, e)
	e.clock.Advance(time.Minute)

	r := staffLogin(t, e, ip1)
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, totp(e, secret, 0), "10.9.9.9", "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton présenté depuis une autre IP accepté : %v", err)
	}
	// mauvaise portée : un jeton `verify` ne sert pas à inscrire un 2FA
	if _, err := e.svc.StaffEnrollTwoFactor(ctx, r.ChallengeToken, ip1); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton verify utilisé pour enroll : %v", err)
	}
	// 2FA déjà actif : on ne peut pas ré-inscrire avec un jeton enroll (il n'en existe pas : login émet verify)
	// expiration (5 min)
	e.clock.Advance(6 * time.Minute)
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, totp(e, secret, 0), ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton expiré accepté : %v", err)
	}
	// jeton falsifié / d'une autre audience
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, "n.importe.quoi", "123456", ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton falsifié : %v", err)
	}
	cust, _ := e.keys[jwtx.AudCustomer].Sign(jwtx.AudStaffChallenge, "x", time.Minute, e.clock.Now(), jwtx.Claims{Scope: "verify"})
	if _, err := e.svc.StaffVerifyTwoFactor(ctx, cust, "123456", ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton signé par une autre clé accepté : %v", err)
	}
}

func testStaffEnrollConsommeLeJeton(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	r := staffLogin(t, e, ip1)
	if _, err := e.svc.StaffEnrollTwoFactor(ctx, r.ChallengeToken, ip1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.StaffEnrollTwoFactor(ctx, r.ChallengeToken, ip1); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("le jeton enroll doit être consommé : %v", err)
	}
	// tant que la confirmation n'a pas eu lieu, le 2FA n'est pas actif : la connexion redemande l'inscription
	if r2 := staffLogin(t, e, ip1); r2.Status != application.StaffTwoFactorEnroll {
		t.Fatalf("statut %q : le 2FA ne doit pas être actif avant confirmation", r2.Status)
	}
}

func testStaffConnexionRefusee(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	_, e1 := e.svc.StaffLogin(ctx, adminMail, "mauvais mot de passe", ip1)
	_, e2 := e.svc.StaffLogin(ctx, "inconnu@filterfinder.test", "mauvais mot de passe", ip1)
	if code(e1) != apperr.CodeInvalidCredentials || e1.Error() != e2.Error() {
		t.Fatalf("compte connu / inconnu distinguables : %v / %v", e1, e2)
	}
	// ralentissement progressif (3 essais libres, le 4e échec impose 1 s) — jamais de verrouillage
	for i := 0; i < 4; i++ {
		_, _ = e.svc.StaffLogin(ctx, adminMail, "x", "10.1.1.1")
	}
	if _, err := e.svc.StaffLogin(ctx, adminMail, adminPw, "10.1.1.1"); code(err) != apperr.CodeRateLimited {
		t.Fatalf("ralentissement absent : %v", err)
	}
	e.clock.Advance(20 * time.Minute)
	if _, err := e.svc.StaffLogin(ctx, adminMail, adminPw, "10.1.1.1"); err != nil {
		t.Fatalf("la connexion doit redevenir possible : %v", err)
	}
}

func testStaffEtClientsSontDeuxMondes(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	register(t, e, "client@b.ci", pw)
	ctx := context.Background()
	if _, err := e.svc.StaffLogin(ctx, "client@b.ci", pw, ip1); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("un client ne peut pas ouvrir une session du personnel : %v", err)
	}
	if _, err := login(e, adminMail, adminPw, ip1); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("le personnel ne peut pas ouvrir une session client : %v", err)
	}
}

func testStaffRefreshRotationRejeuEtDeconnexion(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	_, _, s := enroll(t, e)

	s2, err := e.svc.StaffRefresh(ctx, s.RefreshToken, ip1, "ua")
	if err != nil || s2.RefreshToken == s.RefreshToken || s2.FamilyID != s.FamilyID {
		t.Fatalf("rotation : %v", err)
	}
	// rejeu de l'ancien jeton : la session entière est révoquée
	if _, err := e.svc.StaffRefresh(ctx, s.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("rejeu accepté : %v", err)
	}
	if _, err := e.svc.StaffRefresh(ctx, s2.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("la session aurait dû être révoquée après rejeu : %v", err)
	}
	// un jeton de rafraîchissement CLIENT n'est pas valable côté personnel
	register(t, e, "c@b.ci", pw)
	cs, _ := login(e, "c@b.ci", pw, ip1)
	if _, err := e.svc.StaffRefresh(ctx, cs.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton client accepté côté personnel : %v", err)
	}
	// durée de vie : 8 h
	e.clock.Advance(time.Minute)
	r := staffLogin(t, e, ip1)
	secret, _, _ := func() ([]byte, int, int) {
		u, _ := e.store.StaffByEmail(ctx, adminMail)
		b, _ := e.svc.Secrets.Open(u.TOTPSecret, u.ID)
		return b, 0, 0
	}()
	s3, err := e.svc.StaffVerifyTwoFactor(ctx, r.ChallengeToken, totp(e, secret, 0), ip1, "ua")
	if err != nil {
		t.Fatal(err)
	}
	e.clock.Advance(9 * time.Hour)
	if _, err := e.svc.StaffRefresh(ctx, s3.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("session de plus de 8 h acceptée : %v", err)
	}
	// déconnexion idempotente
	if err := e.svc.StaffLogout(ctx, s3.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.StaffLogout(ctx, "inconnu"); err != nil {
		t.Fatal(err)
	}
}

func testStaffUnCodeTOTPNePasseQuUneFoisEnConcurrence(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	secret, _, _ := enroll(t, e)
	e.clock.Advance(time.Minute)
	const n = 6
	tokens := make([]string, n)
	for i := range tokens {
		tokens[i] = staffLogin(t, e, ip1).ChallengeToken
	}
	code := totp(e, secret, 0)
	var ok atomic.Int32
	var wg sync.WaitGroup
	for _, tk := range tokens {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.svc.StaffVerifyTwoFactor(ctx, tk, code, ip1, "ua"); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("le même code TOTP doit ouvrir exactement 1 session, obtenu %d", ok.Load())
	}
}

func testStaffDeconnexionDeToutesLesSessions(t *testing.T) {
	e := newEnv(t, nil)
	bootstrap(t, e)
	ctx := context.Background()
	_, _, s := enroll(t, e)
	if err := e.svc.StaffLogoutAll(ctx, s.Staff.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.StaffRefresh(ctx, s.RefreshToken, ip1, "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("session non révoquée : %v", err)
	}
}
