package server_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/google/uuid"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	pb "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeFinance struct {
	pb.FinanceAdminServiceClient
	calls    map[string]int
	actors   []string
	entry    *pb.Entry
	op       *pb.Operation
	dup      bool
	uploaded []byte
	number   *string
	simulate bool
	horizon  int32
}

func (f *fakeFinance) note(name string) { f.calls[name]++ }

func fe(id string) *pb.Entry {
	return &pb.Entry{Id: id, JournalId: uuid.NewString(), JournalCode: "VT", FiscalPeriodId: uuid.NewString(), AccountingDate: "2026-02-01", Status: "draft", Label: "x", Currency: "XOF", Origin: "manual",
		CreatedBy: uuid.NewString(), CreatedAt: timestamppb.Now(), Lines: []*pb.EntryLine{{Id: uuid.NewString(), AccountId: uuid.NewString(), AccountCode: "A", AccountLabel: "A", Debit: 5}}, TotalDebit: 5}
}

func (f *fakeFinance) ListEntries(_ context.Context, _ *pb.ListEntriesRequest, _ ...grpc.CallOption) (*pb.ListEntriesResponse, error) {
	f.note("ListEntries")
	return &pb.ListEntriesResponse{Items: []*pb.Entry{fe(uuid.NewString())}, Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: 1, TotalPages: 1}}, nil
}
func (f *fakeFinance) CreateEntry(_ context.Context, in *pb.CreateEntryRequest, _ ...grpc.CallOption) (*pb.CreateEntryResponse, error) {
	f.note("CreateEntry")
	f.entry = in.GetEntry()
	return &pb.CreateEntryResponse{Entry: fe(uuid.NewString())}, nil
}
func (f *fakeFinance) PostEntry(_ context.Context, in *pb.PostEntryRequest, _ ...grpc.CallOption) (*pb.PostEntryResponse, error) {
	f.note("PostEntry")
	return &pb.PostEntryResponse{Entry: fe(in.GetId())}, nil
}
func (f *fakeFinance) ReverseEntry(_ context.Context, in *pb.ReverseEntryRequest, _ ...grpc.CallOption) (*pb.ReverseEntryResponse, error) {
	f.note("ReverseEntry")
	return &pb.ReverseEntryResponse{Entry: fe(uuid.NewString())}, nil
}
func (f *fakeFinance) CloseFiscalPeriod(_ context.Context, in *pb.CloseFiscalPeriodRequest, _ ...grpc.CallOption) (*pb.CloseFiscalPeriodResponse, error) {
	f.note("CloseFiscalPeriod")
	return &pb.CloseFiscalPeriodResponse{Period: &pb.FiscalPeriod{Id: in.GetId(), FiscalYearId: uuid.NewString(), Number: 1, StartDate: "2026-01-01", EndDate: "2026-01-31", Status: "closed"}}, nil
}
func (f *fakeFinance) ApproveApproval(_ context.Context, in *pb.ApproveApprovalRequest, _ ...grpc.CallOption) (*pb.ApproveApprovalResponse, error) {
	f.note("ApproveApproval")
	return &pb.ApproveApprovalResponse{Approval: &pb.Approval{Id: in.GetId(), Type: "reopen_period", SubjectId: uuid.NewString(), Status: "approved", RequestedBy: uuid.NewString(), RequestedAt: timestamppb.Now()}}, nil
}
func (f *fakeFinance) CreateAccount(_ context.Context, in *pb.CreateAccountRequest, _ ...grpc.CallOption) (*pb.CreateAccountResponse, error) {
	f.note("CreateAccount")
	a := in.GetAccount()
	a.Id = uuid.NewString()
	return &pb.CreateAccountResponse{Account: a}, nil
}
func (f *fakeFinance) PutSettings(_ context.Context, in *pb.PutSettingsRequest, _ ...grpc.CallOption) (*pb.PutSettingsResponse, error) {
	f.note("PutSettings")
	return &pb.PutSettingsResponse{Settings: in.GetSettings()}, nil
}
func (f *fakeFinance) ExportTrialBalance(_ context.Context, _ *pb.ExportTrialBalanceRequest, _ ...grpc.CallOption) (*pb.ExportTrialBalanceResponse, error) {
	f.note("ExportTrialBalance")
	return &pb.ExportTrialBalanceResponse{Csv: []byte("compte;libelle\n")}, nil
}

// Chaque droit est distinct : lecture, saisie, validation, clôture, administration, export.
func TestPermissionsFinance(t *testing.T) {
	f := &fakeFinance{calls: map[string]int{}}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Finance = f })
	sub, id := uuid.NewString(), uuid.NewString()
	jid, aid := uuid.NewString(), uuid.NewString()
	entry := `{"journalId":"` + jid + `","accountingDate":"2026-02-01","label":"Loyer","lines":[{"accountId":"` + aid + `","debit":5000,"credit":0},{"accountId":"` + aid + `","debit":0,"credit":5000}]}`
	for _, c := range []struct {
		name, method, path, body string
		perms                    []string
		want                     int
		call                     string
	}{
		{"liste sans permission", "GET", "/v1/admin/finance/entries", "", nil, 403, ""},
		{"liste avec une permission d'une autre famille", "GET", "/v1/admin/finance/entries", "", []string{"catalog:read"}, 403, ""},
		{"liste avec finance:read", "GET", "/v1/admin/finance/entries", "", []string{"finance:read"}, 200, "ListEntries"},
		{"saisie avec finance:read seulement", "POST", "/v1/admin/finance/entries", entry, []string{"finance:read"}, 403, ""},
		{"saisie avec finance:write", "POST", "/v1/admin/finance/entries", entry, []string{"finance:write"}, 201, "CreateEntry"},
		{"validation avec finance:write seulement", "POST", "/v1/admin/finance/entries/" + id + "/post", "", []string{"finance:write"}, 403, ""},
		{"validation avec finance:approve", "POST", "/v1/admin/finance/entries/" + id + "/post", "", []string{"finance:approve"}, 200, "PostEntry"},
		{"contre-passation avec finance:approve seulement", "POST", "/v1/admin/finance/entries/" + id + "/reverse", `{"reason":"erreur de saisie"}`, []string{"finance:approve"}, 403, ""},
		{"contre-passation avec finance:write", "POST", "/v1/admin/finance/entries/" + id + "/reverse", `{"reason":"erreur de saisie"}`, []string{"finance:write"}, 201, "ReverseEntry"},
		{"clôture avec finance:write", "POST", "/v1/admin/finance/fiscal-periods/" + id + "/close", "", []string{"finance:write", "finance:approve"}, 403, ""},
		{"clôture avec finance:close", "POST", "/v1/admin/finance/fiscal-periods/" + id + "/close", "", []string{"finance:close"}, 200, "CloseFiscalPeriod"},
		{"approbation avec finance:approve seulement", "POST", "/v1/admin/finance/approvals/" + id + "/approve", `{}`, []string{"finance:approve"}, 403, ""},
		{"approbation avec finance:close", "POST", "/v1/admin/finance/approvals/" + id + "/approve", `{}`, []string{"finance:close"}, 200, "ApproveApproval"},
		{"plan comptable avec finance:write", "POST", "/v1/admin/finance/accounts", `{"code":"512","label":"Banque","kind":"asset"}`, []string{"finance:write"}, 403, ""},
		{"plan comptable avec finance:admin", "POST", "/v1/admin/finance/accounts", `{"code":"512","label":"Banque","kind":"asset"}`, []string{"finance:admin"}, 201, "CreateAccount"},
		{"réglages avec finance:write", "PUT", "/v1/admin/finance/settings", `{"baseCurrency":"XOF","fourEyesMinAmount":0}`, []string{"finance:write"}, 403, ""},
		{"réglages avec finance:admin", "PUT", "/v1/admin/finance/settings", `{"baseCurrency":"XOF","fourEyesMinAmount":0}`, []string{"finance:admin"}, 200, "PutSettings"},
		{"export avec finance:read seulement", "GET", "/v1/admin/finance/reports/trial-balance/export?from=2026-01-01&to=2026-12-31", "", []string{"finance:read"}, 403, ""},
		{"export avec finance:export", "GET", "/v1/admin/finance/reports/trial-balance/export?from=2026-01-01&to=2026-12-31", "", []string{"finance:export"}, 200, "ExportTrialBalance"},
	} {
		before := f.calls[c.call]
		rec := r.do(req{c.method, c.path, c.body, bearer(staffToken(t, r, sub, c.perms...))})
		if rec.Code != c.want {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
			continue
		}
		if c.call != "" && f.calls[c.call] != before+1 {
			t.Errorf("%s : %s non appelé", c.name, c.call)
		}
	}
	// aucun appel relayé pour les refus
	if f.calls["PostEntry"] != 1 || f.calls["CreateAccount"] != 1 || f.calls["PutSettings"] != 1 || f.calls["CloseFiscalPeriod"] != 1 || f.calls["ApproveApproval"] != 1 || f.calls["ExportTrialBalance"] != 1 {
		t.Fatalf("un refus a été relayé : %v", f.calls)
	}
	// le joker donne tous les droits ; sans jeton : 401 ; service absent : 501
	if rec := r.do(req{"GET", "/v1/admin/finance/entries", "", bearer(staffToken(t, r, sub, "*"))}); rec.Code != 200 {
		t.Fatalf("joker : %d", rec.Code)
	}
	if rec := r.do(req{"GET", "/v1/admin/finance/entries", "", nil}); rec.Code != 401 {
		t.Fatalf("sans jeton : %d", rec.Code)
	}
	r2 := newRigWith(t, nil, nil)
	if rec := r2.do(req{"GET", "/v1/admin/finance/entries", "", bearer(staffToken(t, r2, sub, "finance:read"))}); rec.Code != 501 {
		t.Fatalf("service absent : %d", rec.Code)
	}
}

func TestFinanceRejetteLesMontantsInvalidesAvantRelais(t *testing.T) {
	f := &fakeFinance{calls: map[string]int{}}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Finance = f })
	tok := bearer(staffToken(t, r, uuid.NewString(), "finance:write"))
	jid, aid := uuid.NewString(), uuid.NewString()
	for name, body := range map[string]string{
		"une seule ligne": `{"journalId":"` + jid + `","accountingDate":"2026-02-01","label":"x","lines":[{"accountId":"` + aid + `","debit":5,"credit":0}]}`,
		"montant négatif": `{"journalId":"` + jid + `","accountingDate":"2026-02-01","label":"x","lines":[{"accountId":"` + aid + `","debit":-5,"credit":0},{"accountId":"` + aid + `","debit":0,"credit":5}]}`,
		"montant décimal": `{"journalId":"` + jid + `","accountingDate":"2026-02-01","label":"x","lines":[{"accountId":"` + aid + `","debit":5.5,"credit":0},{"accountId":"` + aid + `","debit":0,"credit":5}]}`,
		"date invalide":   `{"journalId":"` + jid + `","accountingDate":"01/02/2026","label":"x","lines":[{"accountId":"` + aid + `","debit":5,"credit":0},{"accountId":"` + aid + `","debit":0,"credit":5}]}`,
		"libellé vide":    `{"journalId":"` + jid + `","accountingDate":"2026-02-01","label":"","lines":[{"accountId":"` + aid + `","debit":5,"credit":0},{"accountId":"` + aid + `","debit":0,"credit":5}]}`,
	} {
		n := f.calls["CreateEntry"]
		rec := r.do(req{"POST", "/v1/admin/finance/entries", body, tok})
		if (rec.Code != 400 && rec.Code != 422) || f.calls["CreateEntry"] != n {
			t.Errorf("%s : %d (relayé : %v)", name, rec.Code, f.calls["CreateEntry"] != n)
		}
	}
}

func fop(id string) *pb.Operation {
	return &pb.Operation{Id: id, Kind: "expense", OperationDate: "2026-03-10", CategoryId: uuid.NewString(), CategoryLabel: "Loyer", Label: "x", AmountNet: 100, AmountGross: 100, CounterpartyAccountId: uuid.NewString(),
		CounterpartyAccountCode: "401", SettlementMode: "on_account", Currency: "XOF", Status: "draft", CreatedBy: uuid.NewString(), CreatedAt: timestamppb.Now(), Payments: []*pb.OperationPayment{}}
}

func (f *fakeFinance) ListOperations(_ context.Context, _ *pb.ListOperationsRequest, _ ...grpc.CallOption) (*pb.ListOperationsResponse, error) {
	f.note("ListOperations")
	return &pb.ListOperationsResponse{Items: []*pb.Operation{fop(uuid.NewString())}, Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: 1, TotalPages: 1}, TotalNet: 100, TotalGross: 100, Currency: "XOF"}, nil
}
func (f *fakeFinance) CreateOperation(_ context.Context, in *pb.CreateOperationRequest, _ ...grpc.CallOption) (*pb.CreateOperationResponse, error) {
	f.note("CreateOperation")
	f.op, f.dup = in.GetOperation(), in.GetDuplicateConfirmed()
	return &pb.CreateOperationResponse{Operation: fop(uuid.NewString())}, nil
}
func (f *fakeFinance) GenerateOperationEntry(_ context.Context, in *pb.GenerateOperationEntryRequest, _ ...grpc.CallOption) (*pb.GenerateOperationEntryResponse, error) {
	f.note("GenerateOperationEntry")
	return &pb.GenerateOperationEntryResponse{Operation: fop(in.GetId())}, nil
}
func (f *fakeFinance) UploadAttachment(_ context.Context, in *pb.UploadAttachmentRequest, _ ...grpc.CallOption) (*pb.UploadAttachmentResponse, error) {
	f.note("UploadAttachment")
	f.uploaded = in.GetData()
	return &pb.UploadAttachmentResponse{Attachment: &pb.Attachment{Id: uuid.NewString(), OperationId: in.GetOperationId(), Filename: in.GetFilename(), ContentType: "application/pdf", Size: int64(len(in.GetData())), Sha256: "0", CreatedAt: timestamppb.Now()}}, nil
}
func (f *fakeFinance) ListRecurringTemplates(_ context.Context, _ *pb.ListRecurringTemplatesRequest, _ ...grpc.CallOption) (*pb.ListRecurringTemplatesResponse, error) {
	f.note("ListRecurringTemplates")
	return &pb.ListRecurringTemplatesResponse{}, nil
}

func TestPermissionsFinanceOperations(t *testing.T) {
	f := &fakeFinance{calls: map[string]int{}}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Finance = f })
	sub, id := uuid.NewString(), uuid.NewString()
	op := `{"kind":"expense","operationDate":"2026-03-10","categoryId":"` + uuid.NewString() + `","label":"Loyer","amountNet":100000,"taxAmount":0,"counterpartyAccountId":"` + uuid.NewString() + `","settlementMode":"on_account","duplicateConfirmed":true}`
	for _, c := range []struct {
		name, method, path, body string
		perms                    []string
		want                     int
		call                     string
	}{
		{"liste sans permission", "GET", "/v1/admin/finance/operations", "", nil, 403, ""},
		{"liste avec finance:read", "GET", "/v1/admin/finance/operations", "", []string{"finance:read"}, 200, "ListOperations"},
		{"saisie avec finance:read", "POST", "/v1/admin/finance/operations", op, []string{"finance:read"}, 403, ""},
		{"saisie avec finance:write", "POST", "/v1/admin/finance/operations", op, []string{"finance:write"}, 201, "CreateOperation"},
		{"génération de l'écriture avec finance:read", "POST", "/v1/admin/finance/operations/" + id + "/entry", `{"journalId":"` + uuid.NewString() + `"}`, []string{"finance:read"}, 403, ""},
		{"génération de l'écriture avec finance:write", "POST", "/v1/admin/finance/operations/" + id + "/entry", `{"journalId":"` + uuid.NewString() + `"}`, []string{"finance:write"}, 200, "GenerateOperationEntry"},
		{"modèles récurrents en lecture", "GET", "/v1/admin/finance/recurring-templates", "", []string{"finance:read"}, 200, "ListRecurringTemplates"},
		{"catégorie avec finance:write seulement", "POST", "/v1/admin/finance/operation-categories", `{"kind":"expense","label":"x","accountId":"` + uuid.NewString() + `"}`, []string{"finance:write"}, 403, ""},
	} {
		before := f.calls[c.call]
		rec := r.do(req{c.method, c.path, c.body, bearer(staffToken(t, r, sub, c.perms...))})
		if rec.Code != c.want {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, rec.Code, c.want, rec.Body.String())
			continue
		}
		if c.call != "" && f.calls[c.call] != before+1 {
			t.Errorf("%s : %s non appelé", c.name, c.call)
		}
	}
	if !f.dup || f.op.GetAmountNet() != 100000 || f.op.GetSettlementMode() != "on_account" {
		t.Fatalf("relayé : dup=%v %+v", f.dup, f.op)
	}
	if f.calls["CreateOperation"] != 1 || f.calls["GenerateOperationEntry"] != 1 {
		t.Fatalf("un refus a été relayé : %v", f.calls)
	}
}

// L'envoi d'une pièce accepte un corps plus gros (5 Mo en base64), mais SEULEMENT sur cette route ; le contenu invalide est refusé avant le service.
func TestPiecesJustificativesLimiteDeCorps(t *testing.T) {
	f := &fakeFinance{calls: map[string]int{}}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Finance = f })
	tok := bearer(staffToken(t, r, uuid.NewString(), "finance:write"))
	id := uuid.NewString()
	big := make([]byte, 3<<20) // 3 Mio → ~4 Mio en base64, au-delà de la limite générale de 1 Mio
	copy(big, "%PDF-1.4")
	enc := base64.StdEncoding.EncodeToString(big)
	rec := r.do(req{"POST", "/v1/admin/finance/operations/" + id + "/attachments", `{"filename":"facture.pdf","dataBase64":"` + enc + `"}`, tok})
	if rec.Code != 201 || len(f.uploaded) != len(big) {
		t.Fatalf("gros fichier : %d (%d octets reçus par le service)", rec.Code, len(f.uploaded))
	}
	rec = r.do(req{"POST", "/v1/admin/finance/operations/" + id + "/attachments", `{"filename":"x.pdf","dataBase64":"pas du base64 !!"}`, tok})
	if rec.Code != 400 && rec.Code != 422 {
		t.Fatalf("base64 invalide : %d", rec.Code)
	}
	n := f.calls["UploadAttachment"]
	rec = r.do(req{"POST", "/v1/admin/finance/entries", `{"journalId":"` + uuid.NewString() + `","accountingDate":"2026-02-01","label":"` + strings.Repeat("a", 2<<20) + `","lines":[]}`, tok})
	if rec.Code < 400 || rec.Code >= 500 || f.calls["CreateEntry"] != 0 {
		t.Fatalf("corps de 2 Mio sur une autre route : %d", rec.Code)
	}
	if f.calls["UploadAttachment"] != n {
		t.Fatal("un envoi invalide a atteint le service")
	}
}

func (f *fakeFinance) ListBankAccounts(_ context.Context, _ *pb.ListBankAccountsRequest, _ ...grpc.CallOption) (*pb.ListBankAccountsResponse, error) {
	f.note("ListBankAccounts")
	l4 := "1234"
	return &pb.ListBankAccountsResponse{Items: []*pb.BankAccount{{Id: uuid.NewString(), Name: "Courant", BankName: "Banque", Currency: "XOF", AccountId: uuid.NewString(), AccountCode: "521", AccountLabel: "Banque", HasNumber: true, NumberLast4: &l4, OpeningDate: "2026-01-01", Active: true}}}, nil
}
func (f *fakeFinance) CreateBankAccount(_ context.Context, in *pb.CreateBankAccountRequest, _ ...grpc.CallOption) (*pb.CreateBankAccountResponse, error) {
	f.note("CreateBankAccount")
	f.number = in.Number
	b := in.GetBankAccount()
	b.Id, b.Currency, b.AccountCode, b.AccountLabel = uuid.NewString(), "XOF", "521", "Banque"
	return &pb.CreateBankAccountResponse{BankAccount: b}, nil
}
func (f *fakeFinance) ImportBankStatement(_ context.Context, in *pb.ImportBankStatementRequest, _ ...grpc.CallOption) (*pb.ImportBankStatementResponse, error) {
	f.note("ImportBankStatement")
	f.simulate = in.GetSimulate()
	return &pb.ImportBankStatementResponse{Simulated: in.GetSimulate(), Total: 2, Inserted: 2}, nil
}
func (f *fakeFinance) CreateReconciliation(_ context.Context, in *pb.CreateReconciliationRequest, _ ...grpc.CallOption) (*pb.CreateReconciliationResponse, error) {
	f.note("CreateReconciliation")
	return &pb.CreateReconciliationResponse{Line: &pb.BankLine{Id: in.GetBankLineId(), StatementId: uuid.NewString(), BankAccountId: uuid.NewString(), LineDate: "2026-02-03", Label: "V", Amount: 1000, Matched: 1000, Status: "matched"}}, nil
}
func (f *fakeFinance) GetCashForecast(_ context.Context, in *pb.GetCashForecastRequest, _ ...grpc.CallOption) (*pb.GetCashForecastResponse, error) {
	f.note("GetCashForecast")
	f.horizon = in.GetHorizonDays()
	return &pb.GetCashForecastResponse{Currency: "XOF", AsOf: "2026-02-01", HorizonDays: 90}, nil
}

func TestPermissionsTresorerie(t *testing.T) {
	f := &fakeFinance{calls: map[string]int{}}
	r := newRigDeps(t, nil, func(d *server.Deps) { d.Finance = f })
	sub, id := uuid.NewString(), uuid.NewString()
	acct := `{"name":"Courant","bankName":"Banque","accountId":"` + uuid.NewString() + `","openingBalance":0,"openingDate":"2026-01-01","number":"CI93 CI008 01111 123456789012 34"}`
	imp := `{"bankAccountId":"` + id + `","csv":"2026-02-03;Virement;R1;1000","simulate":true}`
	rec := `{"bankLineId":"` + uuid.NewString() + `","entryLineId":"` + uuid.NewString() + `"}`
	for _, c := range []struct {
		name, method, path, body string
		perms                    []string
		want                     int
		call                     string
	}{
		{"liste sans permission", "GET", "/v1/admin/finance/bank-accounts", "", nil, 403, ""},
		{"liste avec finance:read (autre famille)", "GET", "/v1/admin/finance/bank-accounts", "", []string{"finance:read"}, 403, ""},
		{"liste avec treasury:read", "GET", "/v1/admin/finance/bank-accounts", "", []string{"treasury:read"}, 200, "ListBankAccounts"},
		{"création avec treasury:read", "POST", "/v1/admin/finance/bank-accounts", acct, []string{"treasury:read"}, 403, ""},
		{"création avec treasury:write", "POST", "/v1/admin/finance/bank-accounts", acct, []string{"treasury:write"}, 201, "CreateBankAccount"},
		{"import avec treasury:reconcile seulement", "POST", "/v1/admin/finance/bank-statements/import", imp, []string{"treasury:reconcile"}, 403, ""},
		{"import avec treasury:write", "POST", "/v1/admin/finance/bank-statements/import", imp, []string{"treasury:write"}, 200, "ImportBankStatement"},
		{"rapprochement avec treasury:write seulement", "POST", "/v1/admin/finance/reconciliations", rec, []string{"treasury:write"}, 403, ""},
		{"rapprochement avec treasury:reconcile", "POST", "/v1/admin/finance/reconciliations", rec, []string{"treasury:reconcile"}, 201, "CreateReconciliation"},
		{"prévision avec treasury:read", "GET", "/v1/admin/finance/cash-forecast?horizonDays=90", "", []string{"treasury:read"}, 200, "GetCashForecast"},
		{"prévision : horizon hors liste", "GET", "/v1/admin/finance/cash-forecast?horizonDays=45", "", []string{"treasury:read"}, 400, ""},
	} {
		before := f.calls[c.call]
		resp := r.do(req{c.method, c.path, c.body, bearer(staffToken(t, r, sub, c.perms...))})
		if resp.Code != c.want && !(c.want == 400 && resp.Code == 422) {
			t.Errorf("%s : %d, attendu %d (%s)", c.name, resp.Code, c.want, resp.Body.String())
			continue
		}
		if c.call != "" && f.calls[c.call] != before+1 {
			t.Errorf("%s : %s non appelé", c.name, c.call)
		}
	}
	if f.calls["CreateBankAccount"] != 1 || f.calls["ImportBankStatement"] != 1 || f.calls["CreateReconciliation"] != 1 || f.calls["GetCashForecast"] != 1 {
		t.Fatalf("un refus a été relayé : %v", f.calls)
	}
	if f.number == nil || !strings.Contains(*f.number, "123456789012") || !f.simulate || f.horizon != 90 {
		t.Fatalf("relayé : number=%v simulate=%v horizon=%d", f.number, f.simulate, f.horizon)
	}
	// le numéro complet n'est JAMAIS dans une réponse : seuls les 4 derniers caractères
	resp := r.do(req{"GET", "/v1/admin/finance/bank-accounts", "", bearer(staffToken(t, r, sub, "treasury:read"))})
	if strings.Contains(resp.Body.String(), "123456789012") || !strings.Contains(resp.Body.String(), `"numberLast4":"1234"`) {
		t.Fatalf("réponse : %s", resp.Body.String())
	}
}
