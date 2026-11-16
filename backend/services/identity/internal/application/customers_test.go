package application_test

import (
	"context"
	"testing"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

func registerCompany(t *testing.T, e *env, email, legal string) {
	t.Helper()
	err := e.svc.Register(context.Background(), application.RegisterInput{
		Kind: "company", Email: email, Password: pw, FirstName: "A", LastName: "B", Locale: "fr", ClientIP: "1.1.1.1",
		Company: &application.CompanyInput{LegalName: legal, TaxID: "CI-123", Country: "CI"},
	})
	if err != nil {
		t.Fatalf("Register entreprise : %v", err)
	}
}

func companyOf(t *testing.T, e *env, email string) *domain.Company {
	t.Helper()
	u, err := e.store.ByEmail(context.Background(), email)
	if err != nil || u.Company == nil {
		t.Fatalf("compte entreprise introuvable : %v", err)
	}
	return u.Company
}

func testAdminClientsListeEtSuspension(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	register(t, e, "awa@exemple.ci", pw)
	registerCompany(t, e, "pro@exemple.ci", "Pièces Plus")

	all, err := e.svc.ListCustomers(ctx, "", "", "", 1, 20)
	if err != nil || all.Total != 2 {
		t.Fatalf("ListCustomers : %v %+v", err, all)
	}
	if r, _ := e.svc.ListCustomers(ctx, "", domain.KindCompany, "", 1, 20); r.Total != 1 || r.Items[0].Email != "pro@exemple.ci" {
		t.Fatalf("filtre kind : %+v", r)
	}
	if r, _ := e.svc.ListCustomers(ctx, "plus", "", "", 1, 20); r.Total != 1 {
		t.Fatalf("recherche par raison sociale : %+v", r)
	}
	if _, err := e.svc.ListCustomers(ctx, "", "robot", "", 1, 20); code(err) != apperr.CodeValidation {
		t.Fatalf("kind inconnu : %v", err)
	}

	awa := all.Items[0]
	if awa.Email != "awa@exemple.ci" {
		awa = all.Items[1]
	}
	u, err := e.svc.SetCustomerStatus(ctx, a, awa.ID, domain.StatusSuspended)
	if err != nil || u.Status != domain.StatusSuspended {
		t.Fatalf("suspension : %v", err)
	}
	if r, _ := e.svc.ListCustomers(ctx, "", "", domain.StatusSuspended, 1, 20); r.Total != 1 {
		t.Fatalf("filtre status : %+v", r)
	}
	if _, err := e.svc.SetCustomerStatus(ctx, a, awa.ID, "banni"); code(err) != apperr.CodeValidation {
		t.Fatalf("statut invalide : %v", err)
	}
	if _, err := e.svc.GetCustomer(ctx, "00000000-0000-0000-0000-000000000000"); code(err) != apperr.CodeNotFound {
		t.Fatalf("client inconnu : %v", err)
	}
	if _, err := e.svc.SetCustomerStatus(ctx, a, awa.ID, domain.StatusActive); err != nil {
		t.Fatal(err)
	}
	au, _, err := e.svc.ListAudit(ctx, domain.AuditFilter{Entity: "customer", Limit: 10})
	if err != nil || len(au) != 2 {
		t.Fatalf("audit : %v %+v", err, au)
	}
}

func testAdminEntrepriseApprobation(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	registerCompany(t, e, "pro@exemple.ci", "Pièces Plus")
	c := companyOf(t, e, "pro@exemple.ci")
	if c.State() != domain.CompanyPending {
		t.Fatalf("une entreprise inscrite est en attente, obtenu %q", c.State())
	}
	if n, _ := e.svc.CountPendingCompanies(ctx); n != 1 {
		t.Fatalf("compteur : %d", n)
	}
	// e-mail du titulaire non vérifié : refus
	if _, err := e.svc.ApproveCompany(ctx, a, c.ID); code(err) != apperr.CodeEmailNotVerified {
		t.Fatalf("e-mail non vérifié : %v", err)
	}
	if err := e.svc.VerifyEmail(ctx, verificationToken(t, e)); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.ApproveCompany(ctx, a, c.ID)
	if err != nil || got.State() != domain.CompanyApproved || got.DecidedAt == nil {
		t.Fatalf("approbation : %v %+v", err, got)
	}
	u, _ := e.store.ByEmail(ctx, "pro@exemple.ci")
	if !u.ProActive() {
		t.Fatal("les conditions pro doivent s'ouvrir après approbation")
	}
	// décision déjà prise : conflit
	if _, err := e.svc.ApproveCompany(ctx, a, c.ID); code(err) != apperr.CodeConflict {
		t.Fatalf("double décision : %v", err)
	}
	if _, err := e.svc.RejectCompany(ctx, a, c.ID, "trop tard"); code(err) != apperr.CodeConflict {
		t.Fatalf("rejet après approbation : %v", err)
	}
	if n, _ := e.svc.CountPendingCompanies(ctx); n != 0 {
		t.Fatalf("compteur après décision : %d", n)
	}
	if r, _ := e.svc.ListCompanies(ctx, "", domain.CompanyApproved, 1, 20); r.Total != 1 {
		t.Fatalf("liste approuvées : %+v", r)
	}
}

func testAdminEntrepriseRejet(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	registerCompany(t, e, "pro@exemple.ci", "Pièces Plus")
	c := companyOf(t, e, "pro@exemple.ci")
	if _, err := e.svc.RejectCompany(ctx, a, c.ID, "  "); code(err) != apperr.CodeValidation {
		t.Fatalf("motif obligatoire : %v", err)
	}
	got, err := e.svc.RejectCompany(ctx, a, c.ID, "Numéro fiscal illisible")
	if err != nil || got.State() != domain.CompanyRejected || got.RejectionReason != "Numéro fiscal illisible" {
		t.Fatalf("rejet : %v %+v", err, got)
	}
	u, _ := e.store.ByEmail(ctx, "pro@exemple.ci")
	if u.ProActive() {
		t.Fatal("une entreprise rejetée n'a pas les conditions pro")
	}
	if _, err := e.svc.GetCompany(ctx, "00000000-0000-0000-0000-000000000000"); code(err) != apperr.CodeNotFound {
		t.Fatalf("entreprise inconnue : %v", err)
	}
}

func testAdminEntrepriseExigeLaPermissionEnBase(t *testing.T) {
	e, a, _ := adminEnv(t)
	ctx := context.Background()
	registerCompany(t, e, "pro@exemple.ci", "Pièces Plus")
	c := companyOf(t, e, "pro@exemple.ci")
	// un acteur inconnu (ou sans la permission) est refusé même si le gateway l'avait laissé passer
	ghost := application.Actor{ID: "00000000-0000-0000-0000-000000000000"}
	if _, err := e.svc.RejectCompany(ctx, ghost, c.ID, "non"); code(err) != apperr.CodeForbidden {
		t.Fatalf("acteur inconnu : %v", err)
	}
	_ = a
}
