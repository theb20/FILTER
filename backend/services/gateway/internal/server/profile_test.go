package server_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Dernières requêtes reçues par le faux service identity (un seul test s'exécute à la fois dans ce fichier).
var (
	lastUpdateUser   *identityv1.UpdateUserRequest
	lastEmailChange  *identityv1.RequestEmailChangeRequest
	lastEmailConfirm *identityv1.ConfirmEmailChangeRequest
	lastGetCompany   *identityv1.GetCompanyRequest
	lastPutCompany   *identityv1.PutCompanyRequest
	lastConsent      *identityv1.RecordConsentRequest
	lastGetConsents  *identityv1.GetConsentsRequest
	profileErr       error
)

func (f *fakeID) UpdateUser(_ context.Context, in *identityv1.UpdateUserRequest, _ ...grpc.CallOption) (*identityv1.UpdateUserResponse, error) {
	lastUpdateUser = in
	if profileErr != nil {
		return nil, profileErr
	}
	u := testUser()
	u.Id = in.GetId()
	if in.PhoneE164 != nil {
		u.PhoneE164 = in.PhoneE164
	}
	return &identityv1.UpdateUserResponse{User: u}, nil
}

func (f *fakeID) RequestEmailChange(_ context.Context, in *identityv1.RequestEmailChangeRequest, _ ...grpc.CallOption) (*identityv1.RequestEmailChangeResponse, error) {
	lastEmailChange = in
	if profileErr != nil {
		return nil, profileErr
	}
	return &identityv1.RequestEmailChangeResponse{Status: "accepted"}, nil
}

func (f *fakeID) ConfirmEmailChange(_ context.Context, in *identityv1.ConfirmEmailChangeRequest, _ ...grpc.CallOption) (*identityv1.ConfirmEmailChangeResponse, error) {
	lastEmailConfirm = in
	if profileErr != nil {
		return nil, profileErr
	}
	return &identityv1.ConfirmEmailChangeResponse{}, nil
}

func testCompany() *identityv1.Company {
	job := "Gérant"
	return &identityv1.Company{Id: uuid.NewString(), LegalName: "Pièces Plus", TaxId: "CI-123", Country: "CI", Status: identityv1.CompanyStatus_COMPANY_STATUS_PENDING, JobTitle: &job}
}

func (f *fakeID) GetCompany(_ context.Context, in *identityv1.GetCompanyRequest, _ ...grpc.CallOption) (*identityv1.GetCompanyResponse, error) {
	lastGetCompany = in
	if profileErr != nil {
		return nil, profileErr
	}
	return &identityv1.GetCompanyResponse{Company: testCompany()}, nil
}

func (f *fakeID) PutCompany(_ context.Context, in *identityv1.PutCompanyRequest, _ ...grpc.CallOption) (*identityv1.PutCompanyResponse, error) {
	lastPutCompany = in
	if profileErr != nil {
		return nil, profileErr
	}
	c := testCompany()
	c.LegalName, c.TaxId, c.Country = in.GetCompany().GetLegalName(), in.GetCompany().GetTaxId(), in.GetCompany().GetCountry()
	return &identityv1.PutCompanyResponse{Company: c}, nil
}

func (f *fakeID) RecordConsent(_ context.Context, in *identityv1.RecordConsentRequest, _ ...grpc.CallOption) (*identityv1.RecordConsentResponse, error) {
	lastConsent = in
	if profileErr != nil {
		return nil, profileErr
	}
	return &identityv1.RecordConsentResponse{Record: &identityv1.ConsentRecord{Id: uuid.NewString(), Purposes: in.GetPurposes(), PolicyVersion: in.GetPolicyVersion(), RecordedAt: timestamppb.Now()}}, nil
}

func (f *fakeID) GetConsents(_ context.Context, in *identityv1.GetConsentsRequest, _ ...grpc.CallOption) (*identityv1.GetConsentsResponse, error) {
	lastGetConsents = in
	if profileErr != nil {
		return nil, profileErr
	}
	return &identityv1.GetConsentsResponse{Record: &identityv1.ConsentRecord{Id: uuid.NewString(), PolicyVersion: "2026-01", RecordedAt: timestamppb.Now(), Purposes: &identityv1.ConsentPurposes{Necessary: true, Analytics: true}}}, nil
}

func resetProfile() {
	lastUpdateUser, lastEmailChange, lastEmailConfirm, lastGetCompany, lastPutCompany, lastConsent, lastGetConsents, profileErr = nil, nil, nil, nil, nil, nil, nil, nil
}

func jsonBody(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("JSON invalide : %v (%s)", err, body)
	}
	return m
}

func TestModifierMonProfil(t *testing.T) {
	resetProfile()
	t.Cleanup(resetProfile)
	r := newRig(t, nil)
	sub := uuid.NewString()
	tok := customerTokenFor(t, r, sub)

	if rec := r.do(req{"PATCH", "/v1/me", `{"firstName":"Awa"}`, nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	if rec := r.do(req{"PATCH", "/v1/me", `{"firstName":"Awa"}`, bearer(staffToken(t, r, sub, "*"))}); rec.Code != 401 {
		t.Fatalf("jeton du personnel : %d", rec.Code)
	}
	// champ inconnu, dont `id` ou `email` : refusé, rien n'est transmis
	for _, body := range []string{`{"id":"` + uuid.NewString() + `"}`, `{"email":"x@y.ci"}`, `{"status":"active"}`} {
		if rec := r.do(req{"PATCH", "/v1/me", body, bearer(tok)}); rec.Code != 400 || lastUpdateUser != nil {
			t.Fatalf("%s : %d (transmis=%v)", body, rec.Code, lastUpdateUser != nil)
		}
	}
	rec := r.do(req{"PATCH", "/v1/me", `{"firstName":"Awa","phone":"+2250747225087","locale":"en"}`, bearer(tok)})
	if rec.Code != 200 {
		t.Fatalf("modification : %d %s", rec.Code, rec.Body.String())
	}
	in := lastUpdateUser
	if in.GetId() != sub || in.GetFirstName() != "Awa" || in.GetPhoneE164() != "+2250747225087" || in.GetLocale() != "en" || in.LastName != nil || in.PreferredCurrency != nil {
		t.Fatalf("seuls les champs envoyés doivent être transmis, pour le compte du jeton : %+v", in)
	}
	if got := jsonBody(t, rec.Body.String()); got["phone"] != "+2250747225087" {
		t.Fatalf("le téléphone doit apparaître dans la réponse : %s", rec.Body.String())
	}
	// téléphone invalide : refusé dès le contrat
	if rec := r.do(req{"PATCH", "/v1/me", `{"phone":"0747225087"}`, bearer(tok)}); rec.Code != 400 && rec.Code != 422 {
		t.Fatalf("téléphone invalide : %d", rec.Code)
	}
	// une erreur de validation d'identity devient 422
	profileErr = apperr.ToGRPC(apperr.Validation("données invalides", apperr.Field{Pointer: "/firstName", Code: apperr.CodeValidation, Message: "trop long"}))
	if rec := r.do(req{"PATCH", "/v1/me", `{"firstName":"x"}`, bearer(tok)}); rec.Code != 422 {
		t.Fatalf("validation métier : %d %s", rec.Code, rec.Body.String())
	}
}

func TestChangerMonAdresseEmail(t *testing.T) {
	resetProfile()
	t.Cleanup(resetProfile)
	r := newRig(t, nil)
	sub := uuid.NewString()
	tok := customerTokenFor(t, r, sub)

	if rec := r.do(req{"POST", "/v1/me/email", `{"newEmail":"n@exemple.ci","password":"x"}`, nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	rec := r.do(req{"POST", "/v1/me/email", `{"newEmail":"n@exemple.ci","password":"mon-mot-de-passe"}`, bearer(tok)})
	if rec.Code != 202 || jsonBody(t, rec.Body.String())["status"] != "accepted" {
		t.Fatalf("demande : %d %s", rec.Code, rec.Body.String())
	}
	if lastEmailChange.GetUserId() != sub || lastEmailChange.GetNewEmail() != "n@exemple.ci" || lastEmailChange.GetPassword() != "mon-mot-de-passe" {
		t.Fatalf("requête transmise : %+v", lastEmailChange)
	}
	if strings.Contains(rec.Body.String(), "mon-mot-de-passe") {
		t.Fatal("le mot de passe ne doit jamais être renvoyé")
	}
	// mot de passe incorrect → 401 INVALID_CREDENTIALS ; trop d'essais → 429 ; compte sans mot de passe → 403
	for code, want := range map[error]int{
		apperr.ToGRPC(apperr.Unauthenticated(apperr.CodeInvalidCredentials, "mot de passe incorrect")): 401,
		apperr.ToGRPC(apperr.RateLimited(5e9)):                                                         429,
		apperr.ToGRPC(apperr.PermissionDenied(apperr.CodeForbidden, "connexion récente requise")):      403,
	} {
		profileErr = code
		if rec := r.do(req{"POST", "/v1/me/email", `{"newEmail":"n@exemple.ci","password":"x"}`, bearer(tok)}); rec.Code != want {
			t.Fatalf("attendu %d, obtenu %d : %s", want, rec.Code, rec.Body.String())
		}
	}
	profileErr = nil
	if rec := r.do(req{"POST", "/v1/me/email", `{"newEmail":"pas-un-email","password":"x"}`, bearer(tok)}); rec.Code != 400 && rec.Code != 422 {
		t.Fatalf("adresse invalide : %d", rec.Code)
	}
}

func TestConfirmerLeChangementDAdresse(t *testing.T) {
	resetProfile()
	t.Cleanup(resetProfile)
	r := newRig(t, nil)
	// route publique (le lien s'ouvre depuis la boîte mail, sans session)
	rec := r.do(req{"POST", "/v1/me/email/confirm", `{"token":"jeton-recu-par-mail"}`, nil})
	if rec.Code != 204 || lastEmailConfirm.GetToken() != "jeton-recu-par-mail" {
		t.Fatalf("confirmation : %d %s", rec.Code, rec.Body.String())
	}
	profileErr = apperr.ToGRPC(apperr.New(3, apperr.CodeInvalidCode, "lien invalide ou expiré"))
	if rec := r.do(req{"POST", "/v1/me/email/confirm", `{"token":"mauvais"}`, nil}); rec.Code != 422 || !strings.Contains(rec.Body.String(), "INVALID_CODE") { // même correspondance que /auth/verify-email
		t.Fatalf("lien invalide : %d %s", rec.Code, rec.Body.String())
	}
	if rec := r.do(req{"POST", "/v1/me/email/confirm", `{}`, nil}); rec.Code != 400 && rec.Code != 422 {
		t.Fatalf("jeton absent : %d", rec.Code)
	}
}

func TestMonEntreprise(t *testing.T) {
	resetProfile()
	t.Cleanup(resetProfile)
	r := newRig(t, nil)
	sub := uuid.NewString()
	tok := customerTokenFor(t, r, sub)

	if rec := r.do(req{"GET", "/v1/me/company", "", nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	rec := r.do(req{"GET", "/v1/me/company", "", bearer(tok)})
	if rec.Code != 200 || lastGetCompany.GetUserId() != sub {
		t.Fatalf("lecture : %d %s", rec.Code, rec.Body.String())
	}
	got := jsonBody(t, rec.Body.String())
	if got["status"] != "pending" || got["legalName"] != "Pièces Plus" || got["rejectionReason"] != nil || got["decidedAt"] != nil || got["taxIdVerified"] != false {
		t.Fatalf("représentation : %s", rec.Body.String())
	}
	rec = r.do(req{"PUT", "/v1/me/company", `{"legalName":"Pièces Plus SARL","taxId":"CI-123","country":"CI","jobTitle":"Gérant"}`, bearer(tok)})
	if rec.Code != 200 {
		t.Fatalf("écriture : %d %s", rec.Code, rec.Body.String())
	}
	c := lastPutCompany.GetCompany()
	if lastPutCompany.GetUserId() != sub || c.GetLegalName() != "Pièces Plus SARL" || c.GetCountry() != "CI" || c.GetJobTitle() != "Gérant" {
		t.Fatalf("requête transmise : %+v", lastPutCompany)
	}
	// le statut ne se choisit jamais : champ refusé
	if rec := r.do(req{"PUT", "/v1/me/company", `{"legalName":"A","taxId":"B","country":"CI","status":"approved"}`, bearer(tok)}); rec.Code != 400 {
		t.Fatalf("statut imposé par le client : %d", rec.Code)
	}
	// compte particulier : 404 en lecture, 409 en écriture
	profileErr = apperr.ToGRPC(apperr.NotFound("aucune entreprise sur ce compte"))
	if rec := r.do(req{"GET", "/v1/me/company", "", bearer(tok)}); rec.Code != 404 {
		t.Fatalf("particulier, lecture : %d", rec.Code)
	}
	profileErr = apperr.ToGRPC(apperr.New(6, apperr.CodeConflict, "un compte particulier n'a pas d'entreprise"))
	if rec := r.do(req{"PUT", "/v1/me/company", `{"legalName":"A","taxId":"B","country":"CI"}`, bearer(tok)}); rec.Code != 409 {
		t.Fatalf("particulier, écriture : %d", rec.Code)
	}
}

const consentBody = `{"purposes":{"necessary":true,"analytics":true},"policyVersion":"2026-01","anonymousId":"visiteur-1234"}`

func TestConsentementVisiteur(t *testing.T) {
	resetProfile()
	t.Cleanup(resetProfile)
	r := newRig(t, nil)
	rec := r.do(req{"POST", "/v1/consents", consentBody, nil}) // route publique : le bandeau cookies précède toute connexion
	if rec.Code != 201 {
		t.Fatalf("enregistrement : %d %s", rec.Code, rec.Body.String())
	}
	in := lastConsent
	if in.UserId != nil || in.GetAnonymousId() != "visiteur-1234" || in.GetClientIp() == "" || !in.GetPurposes().GetNecessary() || !in.GetPurposes().GetAnalytics() || in.GetPurposes().GetMarketing() {
		t.Fatalf("requête transmise : %+v", in)
	}
	got := jsonBody(t, rec.Body.String())
	if got["policyVersion"] != "2026-01" || got["purposes"].(map[string]any)["analytics"] != true {
		t.Fatalf("réponse : %s", rec.Body.String())
	}
	// même avec un jeton, la route visiteur ne rattache JAMAIS le choix à un compte
	lastConsent = nil
	r.do(req{"POST", "/v1/consents", consentBody, bearer(customerTokenFor(t, r, uuid.NewString()))})
	if lastConsent == nil || lastConsent.UserId != nil {
		t.Fatalf("la route visiteur ne doit pas utiliser le jeton : %+v", lastConsent)
	}
	for name, body := range map[string]string{
		"sans version":     `{"purposes":{"necessary":true}}`,
		"sans nécessaires": `{"purposes":{"analytics":true},"policyVersion":"1"}`,
		"champ inconnu":    `{"purposes":{"necessary":true,"ads":true},"policyVersion":"1"}`,
	} {
		if rec := r.do(req{"POST", "/v1/consents", body, nil}); rec.Code != 400 && rec.Code != 422 {
			t.Fatalf("%s : %d", name, rec.Code)
		}
	}
	profileErr = apperr.ToGRPC(apperr.Validation("données invalides", apperr.Field{Pointer: "/purposes/necessary", Code: apperr.CodeValidation, Message: "non refusable"}))
	if rec := r.do(req{"POST", "/v1/consents", `{"purposes":{"necessary":false},"policyVersion":"1"}`, nil}); rec.Code != 422 {
		t.Fatalf("nécessaires refusés : %d", rec.Code)
	}
}

func TestMesConsentements(t *testing.T) {
	resetProfile()
	t.Cleanup(resetProfile)
	r := newRig(t, nil)
	sub := uuid.NewString()
	tok := customerTokenFor(t, r, sub)
	if rec := r.do(req{"PUT", "/v1/me/consents", consentBody, nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	rec := r.do(req{"PUT", "/v1/me/consents", consentBody, bearer(tok)})
	if rec.Code != 200 || lastConsent.GetUserId() != sub {
		t.Fatalf("écriture : %d %s (%+v)", rec.Code, rec.Body.String(), lastConsent)
	}
	rec = r.do(req{"GET", "/v1/me/consents", "", bearer(tok)})
	if rec.Code != 200 || lastGetConsents.GetUserId() != sub {
		t.Fatalf("lecture : %d %s", rec.Code, rec.Body.String())
	}
	// jamais consenti : 404, aucune donnée inventée
	profileErr = apperr.ToGRPC(apperr.NotFound("aucun consentement enregistré"))
	if rec := r.do(req{"GET", "/v1/me/consents", "", bearer(tok)}); rec.Code != 404 {
		t.Fatalf("aucun consentement : %d", rec.Code)
	}
}
