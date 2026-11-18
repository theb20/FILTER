package application_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// ── outillage ───────────────────────────────────────────────────────────────

type clock struct{ t time.Time }

func (c *clock) Now() time.Time          { return c.t }
func (c *clock) Advance(d time.Duration) { c.t = c.t.Add(d) }

type countingHasher struct {
	domain.Hasher
	hashes atomic.Int32
}

func (h *countingHasher) Hash(p string) (string, error) { h.hashes.Add(1); return h.Hasher.Hash(p) }

type fakeBreach map[string]bool

func (f fakeBreach) Compromised(_ context.Context, p string) (bool, error) { return f[p], nil }

type env struct {
	svc    *application.Service
	store  store
	events eventReader
	clock  *clock
	hasher *countingHasher
	keys   map[string]*jwtx.KeySet
}

func newKey(t *testing.T) *jwtx.KeySet {
	t.Helper()
	k, err := jwtx.NewKeySet()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func newEnv(t *testing.T, breach application.BreachChecker) *env {
	t.Helper()
	k, err := jwtx.NewKeySet()
	if err != nil {
		t.Fatal(err)
	}
	w := current.build(t)
	e := &env{store: w.store, events: w.events, clock: &clock{t: time.Unix(1_800_000_000, 0)}, keys: map[string]*jwtx.KeySet{jwtx.AudCustomer: k, jwtx.AudStaff: newKey(t), jwtx.AudStaffChallenge: newKey(t)}}
	e.hasher = &countingHasher{Hasher: domain.Hasher{P: domain.Argon2idParams{MemoryKiB: 64, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}}} // léger : tests rapides
	box, err := domain.NewSecretBox(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	e.svc = application.New(application.Service{
		Users: w.store, Sessions: w.store, Tokens: w.store, Failures: w.store, UoW: w.uow, Staff: w.store, StaffSessions: w.store.StaffSessions(), Challenges: w.store, StaffAdmin: w.store, Audit: w.store, Customers: w.store, OAuth: w.store, Secrets: box, Hasher: e.hasher, Breach: breach, Events: w.publisher,
		Keys: e.keys, Settings: &application.DBSettings{Repo: w.store, Base: domain.DefaultSettings()}, SettingsStore: w.store, Preferences: w.store, Consents: w.store, Security: w.store, Now: e.clock.Now,
	})
	return e
}

func register(t *testing.T, e *env, email, pw string) {
	t.Helper()
	if err := e.svc.Register(context.Background(), application.RegisterInput{Kind: "individual", Email: email, Password: pw, FirstName: "A", LastName: "B", Locale: "fr", ClientIP: "1.1.1.1"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func code(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return "?"
}

const pw = "correct horse battery"

// ── inscription : aucune fuite d'existence ──────────────────────────────────

func testInscriptionUniforme(t *testing.T) {
	e := newEnv(t, nil)
	register(t, e, "Awa@Exemple.CI", pw)
	h1 := e.hasher.hashes.Load()
	register(t, e, "awa@exemple.ci", pw) // même adresse (casse différente) : aucune différence visible
	h2 := e.hasher.hashes.Load()

	if h1 != 1 || h2 != 2 {
		t.Fatalf("le mot de passe doit être haché DANS LES DEUX cas (égalisation du travail) : %d puis %d", h1, h2)
	}
	if n := len(e.events.OfType(application.EvEmailVerificationRequested)); n != 1 {
		t.Fatalf("1 e-mail de vérification attendu pour la nouvelle adresse, obtenu %d", n)
	}
	ev := e.events.OfType(application.EvRegistrationOnExisting)
	if len(ev) != 1 || ev[0].Payload.(*eventsv1.RegistrationAttemptOnExistingAccount).Email != "awa@exemple.ci" {
		t.Fatalf("le titulaire doit être prévenu par e-mail : %+v", ev)
	}
}

func testInscriptionErreursDeFormeSeulement(t *testing.T) {
	e := newEnv(t, fakeBreach{"password123": true})
	ctx := context.Background()
	cases := []struct {
		name string
		in   application.RegisterInput
		want string
	}{
		{"e-mail invalide", application.RegisterInput{Kind: "individual", Email: "pas-un-email", Password: pw}, apperr.CodeValidation},
		{"mot de passe trop court", application.RegisterInput{Kind: "individual", Email: "a@b.ci", Password: "court"}, apperr.CodeValidation},
		{"mot de passe compromis", application.RegisterInput{Kind: "individual", Email: "a@b.ci", Password: "password123"}, apperr.CodeValidation},
		{"entreprise sans société", application.RegisterInput{Kind: "company", Email: "a@b.ci", Password: pw}, apperr.CodeValidation},
	}
	for _, c := range cases {
		err := e.svc.Register(ctx, c.in)
		if code(err) != c.want {
			t.Errorf("%s : code %q, attendu %q (%v)", c.name, code(err), c.want, err)
		}
	}
	// le détail du refus « compromis » est porté par le champ
	err := e.svc.Register(ctx, application.RegisterInput{Kind: "individual", Email: "a@b.ci", Password: "password123"})
	var ae *apperr.Error
	if !errors.As(err, &ae) || len(ae.Fields) == 0 || ae.Fields[0].Code != apperr.CodePasswordCompromised {
		t.Fatalf("PASSWORD_COMPROMISED attendu sur le champ : %+v", err)
	}
}

func testPasswordPolitique8a128SansComplexite(t *testing.T) {
	e := newEnv(t, nil)
	for _, ok := range []string{"aaaaaaaa", "12345678", "mot de passe tout en minuscules sans chiffre"} {
		if err := e.svc.Register(context.Background(), application.RegisterInput{Kind: "individual", Email: "x" + ok[:2] + "@b.ci", Password: ok}); err != nil {
			t.Errorf("%q doit être accepté (aucune règle de complexité) : %v", ok, err)
		}
	}
}

// ── vérification de l'e-mail ────────────────────────────────────────────────

func verificationToken(t *testing.T, e *env) string {
	t.Helper()
	ev := e.events.OfType(application.EvEmailVerificationRequested)
	if len(ev) == 0 {
		t.Fatal("aucun événement de vérification")
	}
	return ev[len(ev)-1].Payload.(*eventsv1.EmailVerificationRequested).Token
}

func testVerificationEmailUsageUniqueEtExpiration(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "a@b.ci", pw)
	tok := verificationToken(t, e)
	if err := e.svc.VerifyEmail(ctx, tok); err != nil {
		t.Fatal(err)
	}
	u, _ := e.store.ByEmail(ctx, "a@b.ci")
	if !u.EmailVerified {
		t.Fatal("l'e-mail doit être vérifié")
	}
	if err := e.svc.VerifyEmail(ctx, tok); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("un jeton déjà utilisé doit être refusé : %v", err)
	}
	// expiration (48 h)
	register(t, e, "c@d.ci", pw)
	tok2 := verificationToken(t, e)
	e.clock.Advance(49 * time.Hour)
	if err := e.svc.VerifyEmail(ctx, tok2); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("un jeton expiré doit être refusé : %v", err)
	}
}

// ── connexion ───────────────────────────────────────────────────────────────

func login(e *env, email, pass, ip string) (*application.Session, error) {
	return e.svc.Login(context.Background(), application.LoginInput{Email: email, Password: pass, IP: ip, UserAgent: "test"})
}

func testConnexionSansEmailVerifie(t *testing.T) {
	e := newEnv(t, nil)
	register(t, e, "a@b.ci", pw)
	s, err := login(e, "A@B.ci", pw, "1.1.1.1") // e-mail NON vérifié : la connexion reste permise
	if err != nil {
		t.Fatal(err)
	}
	// le jeton d'accès est vérifiable avec le JWKS publié, audience `customer`, claim `sid`
	jwks, _ := e.svc.JWKS(jwtx.AudCustomer)
	v, _ := jwtx.NewVerifierFromJWKS(jwks, jwtx.AudCustomer, 30*time.Second)
	c, err := v.Verify(s.AccessToken, e.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if c.Subject != s.User.ID || c.SID != s.FamilyID || c.EmailVerified {
		t.Fatalf("claims inattendues : %+v", c)
	}
	if s.ExpiresIn != 15*time.Minute || s.RefreshMaxAge != 30*24*time.Hour {
		t.Fatalf("durées par défaut = %v / %v", s.ExpiresIn, s.RefreshMaxAge)
	}
}

func testIdentifiantsInvalidesIndistinguables(t *testing.T) {
	e := newEnv(t, nil)
	register(t, e, "a@b.ci", pw)
	_, e1 := login(e, "a@b.ci", "mauvais mot de passe", "1.1.1.1")
	_, e2 := login(e, "inconnu@b.ci", "mauvais mot de passe", "2.2.2.2")
	if code(e1) != apperr.CodeInvalidCredentials || code(e2) != apperr.CodeInvalidCredentials {
		t.Fatalf("même erreur attendue : %v / %v", e1, e2)
	}
	if e1.Error() != e2.Error() {
		t.Fatalf("les messages doivent être identiques (pas d'oracle) : %q / %q", e1, e2)
	}
}

func testRalentissementProgressifSansVerrouillage(t *testing.T) {
	e := newEnv(t, nil)
	register(t, e, "a@b.ci", pw)
	ip := "9.9.9.9"
	for i := 0; i < 3; i++ { // 3 échecs gratuits
		if _, err := login(e, "a@b.ci", "faux", ip); code(err) != apperr.CodeInvalidCredentials {
			t.Fatalf("échec %d : %v", i+1, err)
		}
	}
	_, err := login(e, "a@b.ci", "faux", ip) // 4e échec : accepté (3 gratuits) puis ralenti
	if code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("4e échec : %v", err)
	}
	// Immédiatement après : ralenti, MÊME avec le bon mot de passe (sinon le ralentissement ne protégerait pas)
	_, err = login(e, "a@b.ci", pw, ip)
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.CodeRateLimited || ae.RetryAfter != time.Second {
		t.Fatalf("429 + Retry-After 1 s attendus : %+v", err)
	}
	// AUCUN verrouillage : une fois le délai écoulé, le bon mot de passe fonctionne
	e.clock.Advance(1500 * time.Millisecond)
	if _, err := login(e, "a@b.ci", pw, ip); err != nil {
		t.Fatalf("le compte ne doit jamais être verrouillé : %v", err)
	}
	// et le compteur est remis à zéro
	if _, err := login(e, "a@b.ci", "faux", ip); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("après un succès, l'historique est oublié : %v", err)
	}
}

func testUnTiersNePeutPasBloquerLeClient(t *testing.T) {
	e := newEnv(t, nil)
	register(t, e, "a@b.ci", pw)
	for i := 0; i < 50; i++ { // un attaquant martèle le compte depuis SON adresse IP
		_, _ = login(e, "a@b.ci", "faux", "6.6.6.6")
		e.clock.Advance(time.Second) // sans jamais attendre assez
	}
	// le titulaire, depuis une autre IP, se connecte normalement
	if _, err := login(e, "a@b.ci", pw, "1.1.1.1"); err != nil {
		t.Fatalf("le ralentissement est par couple (compte, IP) : le titulaire ne doit pas être bloqué : %v", err)
	}
}

func testAlerteAuTitulaireApresPlusieursEchecs(t *testing.T) {
	e := newEnv(t, nil)
	register(t, e, "a@b.ci", pw)
	for i := 0; i < 8; i++ { // IP différentes : aucun ralentissement par couple, mais l'alerte se déclenche
		_, _ = login(e, "a@b.ci", "faux", "10.0.0."+string(rune('1'+i)))
	}
	if n := len(e.events.OfType(application.EvSuspiciousLogin)); n != 1 {
		t.Fatalf("une seule alerte attendue par fenêtre (pas de spam), obtenu %d", n)
	}
	// compte inexistant : aucune alerte (rien à protéger, rien à révéler)
	for i := 0; i < 8; i++ {
		_, _ = login(e, "fantome@b.ci", "faux", "20.0.0."+string(rune('1'+i)))
	}
	if n := len(e.events.OfType(application.EvSuspiciousLogin)); n != 1 {
		t.Fatalf("pas d'alerte pour un compte inexistant, obtenu %d", n)
	}
}

// ── rafraîchissement : rotation et détection de rejeu ───────────────────────

func testRotationEtRejeuRevoqueLaFamille(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "a@b.ci", pw)
	s0, _ := login(e, "a@b.ci", pw, "1.1.1.1")

	s1, err := e.svc.Refresh(ctx, s0.RefreshToken, "1.1.1.1", "ua")
	if err != nil {
		t.Fatal(err)
	}
	if s1.RefreshToken == s0.RefreshToken || s1.FamilyID != s0.FamilyID {
		t.Fatal("rotation : nouveau jeton, même famille")
	}
	// REJEU de l'ancien jeton : refusé ET toute la famille est révoquée
	if _, err := e.svc.Refresh(ctx, s0.RefreshToken, "6.6.6.6", "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("le rejeu doit être refusé : %v", err)
	}
	if _, err := e.svc.Refresh(ctx, s1.RefreshToken, "1.1.1.1", "ua"); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("après un rejeu, le jeton légitime de la même famille doit aussi être révoqué : %v", err)
	}
}

func testRefreshExpireEtInconnu(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "a@b.ci", pw)
	s, _ := login(e, "a@b.ci", pw, "1.1.1.1")
	if _, err := e.svc.Refresh(ctx, "jeton-inconnu", "", ""); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton inconnu : %v", err)
	}
	e.clock.Advance(31 * 24 * time.Hour)
	if _, err := e.svc.Refresh(ctx, s.RefreshToken, "", ""); code(err) != apperr.CodeUnauthenticated {
		t.Fatalf("jeton expiré (30 jours) : %v", err)
	}
}

func testRefreshConcurrentUnSeulGagnant(t *testing.T) {
	e := newEnv(t, nil)
	register(t, e, "a@b.ci", pw)
	s, _ := login(e, "a@b.ci", pw, "1.1.1.1")
	var wg sync.WaitGroup
	var ok atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.svc.Refresh(context.Background(), s.RefreshToken, "", ""); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("exactement 1 rafraîchissement doit réussir sur le même jeton, obtenu %d", ok.Load())
	}
}

// ── déconnexion ─────────────────────────────────────────────────────────────

func testLogoutIdempotentEtLogoutAll(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "a@b.ci", pw)
	a, _ := login(e, "a@b.ci", pw, "1.1.1.1")
	b, _ := login(e, "a@b.ci", pw, "2.2.2.2")

	if err := e.svc.Logout(ctx, a.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Logout(ctx, a.RefreshToken); err != nil { // idempotent
		t.Fatalf("une 2e déconnexion doit réussir : %v", err)
	}
	if err := e.svc.Logout(ctx, "inconnu"); err != nil {
		t.Fatalf("jeton inconnu : succès attendu (idempotent) : %v", err)
	}
	if _, err := e.svc.Refresh(ctx, a.RefreshToken, "", ""); err == nil {
		t.Fatal("la session déconnectée ne doit plus se rafraîchir")
	}
	if err := e.svc.LogoutAll(ctx, a.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Refresh(ctx, b.RefreshToken, "", ""); err == nil {
		t.Fatal("LogoutAll doit révoquer TOUTES les sessions")
	}
}

// ── changement de mot de passe ──────────────────────────────────────────────

func testChangementMotDePasse(t *testing.T) {
	e := newEnv(t, fakeBreach{"mot de passe compromis!": true})
	ctx := context.Background()
	register(t, e, "a@b.ci", pw)
	cur, _ := login(e, "a@b.ci", pw, "1.1.1.1")
	other, _ := login(e, "a@b.ci", pw, "2.2.2.2")
	uid := cur.User.ID

	if err := e.svc.ChangePassword(ctx, uid, "faux", "nouveau mot de passe", cur.FamilyID); code(err) != apperr.CodeInvalidCredentials {
		t.Fatalf("mot de passe actuel faux : %v", err)
	}
	if err := e.svc.ChangePassword(ctx, uid, pw, "mot de passe compromis!", cur.FamilyID); err == nil {
		t.Fatal("un mot de passe compromis doit être refusé")
	}
	if err := e.svc.ChangePassword(ctx, uid, pw, "nouveau mot de passe", cur.FamilyID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Refresh(ctx, other.RefreshToken, "", ""); err == nil {
		t.Fatal("les AUTRES sessions doivent être révoquées")
	}
	if _, err := e.svc.Refresh(ctx, cur.RefreshToken, "", ""); err != nil {
		t.Fatalf("la session courante doit être conservée : %v", err)
	}
	if _, err := login(e, "a@b.ci", "nouveau mot de passe", "3.3.3.3"); err != nil {
		t.Fatalf("le nouveau mot de passe doit fonctionner : %v", err)
	}
}

func testJWKSAudienceInconnue(t *testing.T) {
	e := newEnv(t, nil)
	if _, err := e.svc.JWKS("nimporte"); err == nil {
		t.Fatal("audience inconnue : erreur attendue")
	}
}
