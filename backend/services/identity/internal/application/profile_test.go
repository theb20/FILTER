package application_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

func sp(s string) *string { return &s }

func testProfilModificationPartielleEtValidation(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	id := customerID(t, e, "awa@exemple.ci")

	u, err := e.svc.UpdateProfile(ctx, id, domain.ProfilePatch{FirstName: sp("  Awa "), Phone: sp("+2250747225087"), Currency: sp("EUR")})
	if err != nil || u.FirstName != "Awa" || u.Phone != "+2250747225087" || u.PreferredCurrency != "EUR" || u.LastName != "B" || u.Locale != "fr" {
		t.Fatalf("patch partiel : %v %+v", err, u)
	}
	got, _ := e.store.ByID(ctx, id)
	if got.FirstName != "Awa" || got.Phone != "+2250747225087" || got.PreferredCurrency != "EUR" || got.LastName != "B" {
		t.Fatalf("persistance : %+v", got)
	}
	// effacer le téléphone ; la langue change ; le reste ne bouge pas
	if u, err = e.svc.UpdateProfile(ctx, id, domain.ProfilePatch{Phone: sp(""), Locale: sp("en")}); err != nil || u.Phone != "" || u.Locale != "en" || u.FirstName != "Awa" {
		t.Fatalf("effacement : %v %+v", err, u)
	}
	// rien d'invalide n'est appliqué, même partiellement
	for name, p := range map[string]domain.ProfilePatch{
		"téléphone sans +":      {Phone: sp("0747225087")},
		"téléphone trop court":  {Phone: sp("+12345")},
		"langue invalide":       {Locale: sp("FR_fr")},
		"devise minuscule":      {Currency: sp("eur")},
		"prénom trop long":      {FirstName: sp(strings.Repeat("a", 101))},
		"caractère de contrôle": {LastName: sp("a\nb")},
	} {
		p2 := p
		p2.FirstName = sp("NePasAppliquer")
		if name == "prénom trop long" {
			p2.FirstName = p.FirstName
		}
		if _, err := e.svc.UpdateProfile(ctx, id, p2); code(err) != apperr.CodeValidation {
			t.Fatalf("%s : validation attendue, obtenu %v", name, err)
		}
	}
	if g, _ := e.store.ByID(ctx, id); g.FirstName != "Awa" || g.Phone != "" {
		t.Fatalf("une modification refusée a été appliquée : %+v", g)
	}
	if _, err := e.svc.UpdateProfile(ctx, "00000000-0000-0000-0000-000000000000", domain.ProfilePatch{}); code(err) != apperr.CodeNotFound {
		t.Fatalf("compte inconnu : %v", err)
	}
}

// changeLink demande le changement et renvoie le jeton envoyé à la NOUVELLE adresse.
func changeLink(t *testing.T, e *env, id, next string) string {
	t.Helper()
	before := len(e.events.OfType(application.EvEmailChangeRequested))
	if err := e.svc.RequestEmailChange(context.Background(), id, next, pw); err != nil {
		t.Fatalf("RequestEmailChange : %v", err)
	}
	ev := e.events.OfType(application.EvEmailChangeRequested)
	if len(ev) != before+1 {
		t.Fatalf("événement attendu (avant %d, après %d)", before, len(ev))
	}
	return ev[len(ev)-1].Payload.(*eventsv1.EmailChangeRequested).Token
}

func testEmailChangementParcoursComplet(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "awa@exemple.ci", pw)
	u, _ := e.store.ByEmail(ctx, "awa@exemple.ci")
	s1, err := login(e, "awa@exemple.ci", pw, "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}

	tok := changeLink(t, e, u.ID, "  Nouvelle@Exemple.CI ")
	ev := e.events.OfType(application.EvEmailChangeRequested)[0].Payload.(*eventsv1.EmailChangeRequested)
	if ev.OldEmail != "awa@exemple.ci" || ev.NewEmail != "nouvelle@exemple.ci" || ev.UserId != u.ID || tok == "" {
		t.Fatalf("événement de demande : %+v", ev)
	}
	// l'adresse NE bascule PAS avant la confirmation
	if g, _ := e.store.ByID(ctx, u.ID); g.Email != "awa@exemple.ci" {
		t.Fatalf("adresse changée trop tôt : %s", g.Email)
	}
	if err := e.svc.ConfirmEmailChange(ctx, tok); err != nil {
		t.Fatal(err)
	}
	g, _ := e.store.ByID(ctx, u.ID)
	if g.Email != "nouvelle@exemple.ci" || !g.EmailVerified {
		t.Fatalf("après confirmation : %+v", g)
	}
	if _, err := e.store.ByEmail(ctx, "awa@exemple.ci"); err == nil {
		t.Fatal("l'ancienne adresse ne doit plus désigner le compte")
	}
	done := e.events.OfType(application.EvEmailChangeConfirmed)
	if len(done) != 1 || done[0].Payload.(*eventsv1.EmailChangeConfirmed).OldEmail != "awa@exemple.ci" || done[0].Payload.(*eventsv1.EmailChangeConfirmed).NewEmail != "nouvelle@exemple.ci" {
		t.Fatalf("l'ancienne adresse doit être notifiée : %+v", done)
	}
	// toutes les sessions sont révoquées ; la connexion se fait avec la nouvelle adresse
	if _, err := e.svc.Refresh(ctx, s1.RefreshToken, "1.1.1.1", "ua"); err == nil {
		t.Fatal("la session ouverte avant le changement doit être révoquée")
	}
	if _, err := login(e, "awa@exemple.ci", pw, "1.1.1.1"); err == nil {
		t.Fatal("l'ancienne adresse ne doit plus permettre de se connecter")
	}
	if _, err := login(e, "nouvelle@exemple.ci", pw, "1.1.1.1"); err != nil {
		t.Fatalf("connexion avec la nouvelle adresse : %v", err)
	}
}

func testEmailChangementRefusEtUniformite(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "awa@exemple.ci", pw)
	register(t, e, "autre@exemple.ci", pw)
	u, _ := e.store.ByEmail(ctx, "awa@exemple.ci")

	// mauvais mot de passe : refusé ; après 3 échecs gratuits, ralentissement progressif ; rien n'est émis
	for i := 0; i < 4; i++ {
		if err := e.svc.RequestEmailChange(ctx, u.ID, "x@exemple.ci", "mauvais-mot-de-passe"); code(err) != apperr.CodeInvalidCredentials {
			t.Fatalf("mot de passe incorrect (essai %d) : %v", i+1, err)
		}
	}
	if err := e.svc.RequestEmailChange(ctx, u.ID, "x@exemple.ci", pw); code(err) != apperr.CodeRateLimited {
		t.Fatalf("après des échecs, le ralentissement progressif s'applique : %v", err)
	}
	e.clock.Advance(time.Hour)
	// forme invalide / identique
	if err := e.svc.RequestEmailChange(ctx, u.ID, "pas-une-adresse", pw); code(err) != apperr.CodeValidation {
		t.Fatalf("adresse invalide : %v", err)
	}
	if err := e.svc.RequestEmailChange(ctx, u.ID, "AWA@exemple.ci", pw); code(err) != apperr.CodeValidation {
		t.Fatalf("adresse identique : %v", err)
	}
	if n := len(e.events.OfType(application.EvEmailChangeRequested)); n != 0 {
		t.Fatalf("aucun e-mail ne doit partir pour ces refus : %d", n)
	}
	// adresse déjà prise : réponse identique (nil), mais AUCUN e-mail
	if err := e.svc.RequestEmailChange(ctx, u.ID, "Autre@Exemple.ci", pw); err != nil {
		t.Fatalf("la réponse doit être uniforme : %v", err)
	}
	if n := len(e.events.OfType(application.EvEmailChangeRequested)); n != 0 {
		t.Fatalf("une adresse déjà utilisée ne doit déclencher aucun e-mail : %d", n)
	}
	if err := e.svc.RequestEmailChange(ctx, "00000000-0000-0000-0000-000000000000", "x@exemple.ci", pw); code(err) != apperr.CodeNotFound {
		t.Fatalf("compte inconnu : %v", err)
	}
	// jeton invalide
	if err := e.svc.ConfirmEmailChange(ctx, "n'importe-quoi"); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("jeton inconnu : %v", err)
	}
}

func testEmailChangementUnSeulLienEtUsageUnique(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "awa@exemple.ci", pw)
	u, _ := e.store.ByEmail(ctx, "awa@exemple.ci")

	old := changeLink(t, e, u.ID, "premiere@exemple.ci")
	cur := changeLink(t, e, u.ID, "seconde@exemple.ci") // remplace le premier lien
	if err := e.svc.ConfirmEmailChange(ctx, old); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("le lien remplacé doit être inutilisable : %v", err)
	}
	if err := e.svc.ConfirmEmailChange(ctx, cur); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.ConfirmEmailChange(ctx, cur); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("usage unique : %v", err)
	}
	if g, _ := e.store.ByID(ctx, u.ID); g.Email != "seconde@exemple.ci" {
		t.Fatalf("adresse finale : %s", g.Email)
	}
	// expiration (24 h)
	exp := changeLink(t, e, u.ID, "troisieme@exemple.ci")
	e.clock.Advance(25 * time.Hour)
	if err := e.svc.ConfirmEmailChange(ctx, exp); code(err) != apperr.CodeInvalidCode {
		t.Fatalf("lien expiré : %v", err)
	}
}

// Deux comptes veulent la même adresse : un seul l'obtient, l'autre est refusé proprement (409), sans état incohérent.
func testEmailChangementCourseSurLAdresse(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	register(t, e, "a@exemple.ci", pw)
	register(t, e, "b@exemple.ci", pw)
	ua, _ := e.store.ByEmail(ctx, "a@exemple.ci")
	ub, _ := e.store.ByEmail(ctx, "b@exemple.ci")
	ta := changeLink(t, e, ua.ID, "convoitee@exemple.ci")
	tb := changeLink(t, e, ub.ID, "convoitee@exemple.ci")

	var wg sync.WaitGroup
	var ok, conflict atomic.Int32
	for _, tok := range []string{ta, tb} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch err := e.svc.ConfirmEmailChange(ctx, tok); {
			case err == nil:
				ok.Add(1)
			case code(err) == apperr.CodeConflict:
				conflict.Add(1)
			default:
				t.Errorf("erreur inattendue : %v", err)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 || conflict.Load() != 1 {
		t.Fatalf("ok=%d conflit=%d (attendu 1 et 1)", ok.Load(), conflict.Load())
	}
	owner, err := e.store.ByEmail(ctx, "convoitee@exemple.ci")
	if err != nil {
		t.Fatal(err)
	}
	loser := ua.ID
	if owner.ID == ua.ID {
		loser = ub.ID
	}
	if g, _ := e.store.ByID(ctx, loser); g.Email == "convoitee@exemple.ci" {
		t.Fatal("les deux comptes portent la même adresse")
	}
}

func testEntrepriseLectureEtRevalidation(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	registerCompany(t, e, "pro@exemple.ci", "Pièces Plus")
	if err := e.svc.VerifyEmail(ctx, verificationToken(t, e)); err != nil { // l'approbation exige une adresse vérifiée
		t.Fatal(err)
	}
	u, _ := e.store.ByEmail(ctx, "pro@exemple.ci")

	c, err := e.svc.MyCompany(ctx, u.ID)
	if err != nil || c.LegalName != "Pièces Plus" || c.State() != domain.CompanyPending {
		t.Fatalf("lecture : %v %+v", err, c)
	}
	if _, err := e.svc.ApproveCompany(ctx, a, c.ID); err != nil {
		t.Fatal(err)
	}
	// la fonction seule ne touche PAS à la validation
	c, err = e.svc.PutMyCompany(ctx, u.ID, domain.CompanyData{LegalName: "Pièces Plus", TaxID: "CI-123", Country: "CI", JobTitle: " Gérant "})
	if err != nil || c.State() != domain.CompanyApproved || c.JobTitle != "Gérant" {
		t.Fatalf("fonction seule : %v %+v", err, c)
	}
	// une donnée LÉGALE repasse en attente (et efface la décision)
	c, err = e.svc.PutMyCompany(ctx, u.ID, domain.CompanyData{LegalName: "Pièces Plus SARL", TaxID: "CI-123", Country: "CI", JobTitle: "Gérant"})
	if err != nil || c.State() != domain.CompanyPending || c.Approved || c.DecidedAt != nil || c.RejectionReason != "" {
		t.Fatalf("donnée légale modifiée : %v %+v", err, c)
	}
	g, _ := e.store.ByID(ctx, u.ID)
	if g.ProActive() || g.Company.LegalName != "Pièces Plus SARL" || g.Company.JobTitle != "Gérant" {
		t.Fatalf("persistance / conditions pro : %+v %+v", g, g.Company)
	}
	// un rejet laisse sa raison jusqu'à la prochaine modification légale
	if _, err := e.svc.RejectCompany(ctx, a, c.ID, "pièce illisible"); err != nil {
		t.Fatal(err)
	}
	if c, _ = e.svc.MyCompany(ctx, u.ID); c.State() != domain.CompanyRejected || c.RejectionReason != "pièce illisible" {
		t.Fatalf("rejet : %+v", c)
	}
	if c, err = e.svc.PutMyCompany(ctx, u.ID, domain.CompanyData{LegalName: "Pièces Plus SARL", TaxID: "CI-999", Country: "CI"}); err != nil || c.State() != domain.CompanyPending || c.RejectionReason != "" {
		t.Fatalf("correction après rejet : %v %+v", err, c)
	}
}

func testEntrepriseValidationEtParticulier(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	registerCompany(t, e, "pro@exemple.ci", "Pièces Plus")
	pro, _ := e.store.ByEmail(ctx, "pro@exemple.ci")
	for name, d := range map[string]domain.CompanyData{
		"raison sociale vide":   {LegalName: " ", TaxID: "X", Country: "CI"},
		"raison trop longue":    {LegalName: strings.Repeat("a", 201), TaxID: "X", Country: "CI"},
		"identifiant vide":      {LegalName: "A", TaxID: "", Country: "CI"},
		"identifiant trop long": {LegalName: "A", TaxID: strings.Repeat("1", 65), Country: "CI"},
		"pays minuscule":        {LegalName: "A", TaxID: "X", Country: "ci"},
		"pays à 3 lettres":      {LegalName: "A", TaxID: "X", Country: "CIV"},
		"fonction trop longue":  {LegalName: "A", TaxID: "X", Country: "CI", JobTitle: strings.Repeat("a", 101)},
	} {
		if _, err := e.svc.PutMyCompany(ctx, pro.ID, d); code(err) != apperr.CodeValidation {
			t.Fatalf("%s : validation attendue, obtenu %v", name, err)
		}
	}
	if c := companyOf(t, e, "pro@exemple.ci"); c.LegalName != "Pièces Plus" {
		t.Fatalf("une saisie invalide a été enregistrée : %+v", c)
	}
	// un particulier : lecture 404, écriture 409
	ind := customerID(t, e, "awa@exemple.ci")
	if _, err := e.svc.MyCompany(ctx, ind); code(err) != apperr.CodeNotFound {
		t.Fatalf("particulier, lecture : %v", err)
	}
	if _, err := e.svc.PutMyCompany(ctx, ind, domain.CompanyData{LegalName: "A", TaxID: "X", Country: "CI"}); code(err) != apperr.CodeConflict {
		t.Fatalf("particulier, écriture : %v", err)
	}
}

func allPurposes() domain.ConsentPurposes {
	return domain.ConsentPurposes{Necessary: true, Preferences: true, Functional: true, Analytics: true, Marketing: true}
}

func testConsentementsJournalAppendOnly(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	id := customerID(t, e, "awa@exemple.ci")
	if _, err := e.svc.MyConsent(ctx, id); code(err) != apperr.CodeNotFound {
		t.Fatalf("aucun consentement donné : rien ne doit être inventé : %v", err)
	}
	r1, err := e.svc.RecordConsent(ctx, application.ConsentInput{UserID: id, Purposes: allPurposes(), PolicyVersion: "2026-01", Locale: "fr", ClientIP: "203.0.113.77"})
	if err != nil || r1.ID == "" || r1.RecordedAt.IsZero() {
		t.Fatalf("enregistrement : %v %+v", err, r1)
	}
	if r1.IPNet != "203.0.113.0/24" {
		t.Fatalf("l'adresse IP doit être tronquée : %q", r1.IPNet)
	}
	e.clock.Advance(time.Minute)
	r2, err := e.svc.RecordConsent(ctx, application.ConsentInput{UserID: id, Purposes: domain.ConsentPurposes{Necessary: true}, PolicyVersion: "2026-02", ClientIP: "2001:db8:abcd:12::1"})
	if err != nil || r2.ID == r1.ID || r2.IPNet != "2001:db8:abcd::/48" {
		t.Fatalf("second choix : %v %+v", err, r2)
	}
	last, err := e.svc.MyConsent(ctx, id)
	if err != nil || last.ID != r2.ID || last.Purposes.Marketing || last.PolicyVersion != "2026-02" {
		t.Fatalf("le dernier choix doit l'emporter : %v %+v", err, last)
	}
	// un autre compte ne voit rien
	other := customerID(t, e, "autre@exemple.ci")
	if _, err := e.svc.MyConsent(ctx, other); code(err) != apperr.CodeNotFound {
		t.Fatalf("isolation entre comptes : %v", err)
	}
	// un compte inconnu ne peut pas recevoir de preuve
	if _, err := e.svc.RecordConsent(ctx, application.ConsentInput{UserID: "00000000-0000-0000-0000-000000000000", Purposes: allPurposes(), PolicyVersion: "2026-01"}); code(err) != apperr.CodeNotFound {
		t.Fatalf("compte inconnu : %v", err)
	}
}

func testConsentementsValidationEtVisiteur(t *testing.T) {
	e := newEnv(t, nil)
	ctx := context.Background()
	for name, in := range map[string]application.ConsentInput{
		"nécessaires refusés":    {Purposes: domain.ConsentPurposes{Analytics: true}, PolicyVersion: "2026-01"},
		"version vide":           {Purposes: allPurposes()},
		"version invalide":       {Purposes: allPurposes(), PolicyVersion: "v 1;DROP"},
		"identifiant trop court": {Purposes: allPurposes(), PolicyVersion: "1", AnonymousID: "abc"},
		"langue invalide":        {Purposes: allPurposes(), PolicyVersion: "1", Locale: "FR"},
	} {
		if _, err := e.svc.RecordConsent(ctx, in); code(err) != apperr.CodeValidation {
			t.Fatalf("%s : validation attendue, obtenu %v", name, err)
		}
	}
	// visiteur avec identifiant fourni, puis sans : un identifiant est créé pour garder la preuve rattachée
	r, err := e.svc.RecordConsent(ctx, application.ConsentInput{AnonymousID: "visiteur-1234", Purposes: allPurposes(), PolicyVersion: "2026-01", ClientIP: "pas-une-ip"})
	if err != nil || r.AnonymousID != "visiteur-1234" || r.UserID != "" || r.IPNet != "" {
		t.Fatalf("visiteur : %v %+v", err, r)
	}
	r, err = e.svc.RecordConsent(ctx, application.ConsentInput{Purposes: allPurposes(), PolicyVersion: "2026-01"})
	if err != nil || r.AnonymousID == "" {
		t.Fatalf("visiteur sans identifiant : %v %+v", err, r)
	}
}

func testVerifierUnePermissionLueEnBase(t *testing.T) {
	e, a, super := adminEnv(t)
	ctx := context.Background()
	if ok, err := e.svc.CheckPermission(ctx, a.ID, "customers:read"); err != nil || !ok {
		t.Fatalf("le super-admin a toutes les permissions : %v %v", ok, err)
	}
	limited := mkRole(t, e, a, "lecteur", "customers:read")
	u, _, _ := invite(t, e, a, "lecteur@exemple.ci", limited.ID)
	_ = super
	if ok, err := e.svc.CheckPermission(ctx, u.ID, "customers:read"); err != nil || !ok {
		t.Fatalf("permission accordée : %v %v", ok, err)
	}
	if ok, _ := e.svc.CheckPermission(ctx, u.ID, "roles:write"); ok {
		t.Fatal("permission non accordée")
	}
	if ok, err := e.svc.CheckPermission(ctx, "00000000-0000-0000-0000-000000000000", "customers:read"); err != nil || ok {
		t.Fatalf("membre inconnu : refus sans erreur : %v %v", ok, err)
	}
	if _, err := e.svc.CheckPermission(ctx, a.ID, "n'existe:pas"); code(err) != apperr.CodeValidation {
		t.Fatalf("permission hors catalogue : %v", err)
	}
	if _, err := e.svc.CheckPermission(ctx, a.ID, "*"); code(err) != apperr.CodeValidation {
		t.Fatalf("le joker ne se demande pas : %v", err)
	}
	// désactivé : refus immédiat (lu en base, pas dans un jeton)
	if _, err := e.svc.DeactivateStaff(ctx, a, u.ID); err != nil {
		t.Fatal(err)
	}
	if ok, _ := e.svc.CheckPermission(ctx, u.ID, "customers:read"); ok {
		t.Fatal("un membre désactivé ne doit plus rien pouvoir")
	}
}
