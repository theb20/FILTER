package server

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/grpc/codes"
)

// Finance : le gateway relaie vers le service finance ; la permission de chaque route est contrôlée AVANT le relais
// (lecture, saisie, validation, clôture, administration et export sont des droits distincts). Aucun calcul ici.

func (h *handler) finance() (pb.FinanceAdminServiceClient, error) {
	if h.fin == nil {
		return nil, apperr.New(codes.Unimplemented, apperr.CodeNotImplemented, "le service finance n'est pas configuré")
	}
	return h.fin, nil
}

const dateLayout = "2006-01-02"

func dateStr(t time.Time) string { return t.Format(dateLayout) }

func optDatePtr(d api.OptDate) *string {
	if v, ok := d.Get(); ok {
		s := dateStr(v)
		return &s
	}
	return nil
}

func parseDate(s string) time.Time {
	t, _ := time.Parse(dateLayout, s)
	return t
}

func nilDateTime(t *pb.FiscalPeriod) api.OptNilDateTime {
	var o api.OptNilDateTime
	if t.ClosedAt == nil {
		o.SetToNull()
		return o
	}
	return api.NewOptNilDateTime(t.GetClosedAt().AsTime())
}

func nilUUIDStr(s *string) api.OptNilUUID {
	var o api.OptNilUUID
	if s == nil {
		o.SetToNull()
		return o
	}
	return api.NewOptNilUUID(uuidOf(*s))
}

// ── réglages ──

func toFinSettings(s *pb.Settings) api.FinanceSettings {
	return api.FinanceSettings{BaseCurrency: s.GetBaseCurrency(), FourEyesMinAmount: s.GetFourEyesMinAmount()}
}

func (h *handler) AdminFinanceSettingsGet(ctx context.Context) (api.AdminFinanceSettingsGetRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetSettings(ctx, &pb.GetSettingsRequest{})
	if err != nil {
		return nil, err
	}
	out := toFinSettings(res.GetSettings())
	return &out, nil
}

func (h *handler) AdminFinanceSettingsPut(ctx context.Context, req *api.FinanceSettings) (api.AdminFinanceSettingsPutRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.PutSettings(ctx, &pb.PutSettingsRequest{Settings: &pb.Settings{BaseCurrency: req.BaseCurrency, FourEyesMinAmount: req.FourEyesMinAmount}})
	if err != nil {
		return nil, err
	}
	out := toFinSettings(res.GetSettings())
	return &out, nil
}

// ── plan comptable ──

func toFinAccount(a *pb.Account) api.FinanceAccount {
	return api.FinanceAccount{ID: uuidOf(a.GetId()), Code: a.GetCode(), Label: a.GetLabel(), Kind: api.FinanceAccountKind(a.GetKind()), Active: a.GetActive()}
}

func (h *handler) AdminFinanceAccountsList(ctx context.Context, p api.AdminFinanceAccountsListParams) (api.AdminFinanceAccountsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListAccountsRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")}
	if v, ok := p.Kind.Get(); ok {
		k := string(v)
		req.Kind = &k
	}
	if v, ok := p.Active.Get(); ok {
		req.Active = &v
	}
	res, err := c.ListAccounts(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.AccountsPage{Items: make([]api.FinanceAccount, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toFinAccount(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceAccountsCreate(ctx context.Context, req *api.AccountInput) (api.AdminFinanceAccountsCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateAccount(ctx, &pb.CreateAccountRequest{Account: &pb.Account{Code: req.Code, Label: req.Label, Kind: string(req.Kind)}})
	if err != nil {
		return nil, err
	}
	out := toFinAccount(res.GetAccount())
	return &out, nil
}

func (h *handler) AdminFinanceAccountsGet(ctx context.Context, p api.AdminFinanceAccountsGetParams) (api.AdminFinanceAccountsGetRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetAccount(ctx, &pb.GetAccountRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toFinAccount(res.GetAccount())
	return &out, nil
}

func (h *handler) AdminFinanceAccountsUpdate(ctx context.Context, req *api.AccountInput, p api.AdminFinanceAccountsUpdateParams) (api.AdminFinanceAccountsUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateAccount(ctx, &pb.UpdateAccountRequest{Account: &pb.Account{Id: p.ID.String(), Code: req.Code, Label: req.Label, Kind: string(req.Kind), Active: req.Active.Or(true)}})
	if err != nil {
		return nil, err
	}
	out := toFinAccount(res.GetAccount())
	return &out, nil
}

func (h *handler) AdminFinanceAccountsImport(ctx context.Context, req *api.AccountImportRequest) (api.AdminFinanceAccountsImportRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ImportAccounts(ctx, &pb.ImportAccountsRequest{Csv: req.Csv, Simulate: req.Simulate.Or(false)})
	if err != nil {
		return nil, err
	}
	out := &api.AccountImportResult{Simulated: res.GetSimulated(), Total: res.GetTotal(), Created: res.GetCreated(), Skipped: res.GetSkipped(), Errors: []api.ImportLineError{}}
	for _, e := range res.GetErrors() {
		out.Errors = append(out.Errors, api.ImportLineError{Line: e.GetLine(), Message: e.GetMessage()})
	}
	return out, nil
}

// ── journaux ──

func toFinJournal(j *pb.Journal) api.FinanceJournal {
	return api.FinanceJournal{ID: uuidOf(j.GetId()), Code: j.GetCode(), Label: j.GetLabel(), Kind: api.FinanceJournalKind(j.GetKind()), Active: j.GetActive()}
}

func (h *handler) AdminFinanceJournalsList(ctx context.Context, p api.AdminFinanceJournalsListParams) (api.AdminFinanceJournalsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListJournalsRequest{}
	if v, ok := p.Active.Get(); ok {
		req.Active = &v
	}
	res, err := c.ListJournals(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.JournalsList{Items: make([]api.FinanceJournal, 0, len(res.GetItems()))}
	for _, j := range res.GetItems() {
		out.Items = append(out.Items, toFinJournal(j))
	}
	return out, nil
}

func (h *handler) AdminFinanceJournalsCreate(ctx context.Context, req *api.JournalInput) (api.AdminFinanceJournalsCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateJournal(ctx, &pb.CreateJournalRequest{Journal: &pb.Journal{Code: req.Code, Label: req.Label, Kind: string(req.Kind)}})
	if err != nil {
		return nil, err
	}
	out := toFinJournal(res.GetJournal())
	return &out, nil
}

func (h *handler) AdminFinanceJournalsUpdate(ctx context.Context, req *api.JournalInput, p api.AdminFinanceJournalsUpdateParams) (api.AdminFinanceJournalsUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateJournal(ctx, &pb.UpdateJournalRequest{Journal: &pb.Journal{Id: p.ID.String(), Code: req.Code, Label: req.Label, Kind: string(req.Kind), Active: req.Active.Or(true)}})
	if err != nil {
		return nil, err
	}
	out := toFinJournal(res.GetJournal())
	return &out, nil
}

// ── exercices ──

func toFiscalPeriod(p *pb.FiscalPeriod) api.FiscalPeriod {
	return api.FiscalPeriod{ID: uuidOf(p.GetId()), FiscalYearId: uuidOf(p.GetFiscalYearId()), Number: p.GetNumber(), StartDate: parseDate(p.GetStartDate()), EndDate: parseDate(p.GetEndDate()),
		Status: api.FiscalPeriodStatus(p.GetStatus()), ClosedAt: nilDateTime(p), ClosedBy: nilUUIDStr(p.ClosedBy)}
}

func toFiscalYear(y *pb.FiscalYear) api.FiscalYear {
	out := api.FiscalYear{ID: uuidOf(y.GetId()), Label: y.GetLabel(), StartDate: parseDate(y.GetStartDate()), EndDate: parseDate(y.GetEndDate()), Status: api.FiscalYearStatus(y.GetStatus()), Periods: []api.FiscalPeriod{}}
	for _, p := range y.GetPeriods() {
		out.Periods = append(out.Periods, toFiscalPeriod(p))
	}
	return out
}

func (h *handler) AdminFinanceFiscalYearsList(ctx context.Context) (api.AdminFinanceFiscalYearsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListFiscalYears(ctx, &pb.ListFiscalYearsRequest{})
	if err != nil {
		return nil, err
	}
	out := &api.FiscalYearsList{Items: make([]api.FiscalYear, 0, len(res.GetItems()))}
	for _, y := range res.GetItems() {
		out.Items = append(out.Items, toFiscalYear(y))
	}
	return out, nil
}

func (h *handler) AdminFinanceFiscalYearsCreate(ctx context.Context, req *api.FiscalYearInput) (api.AdminFinanceFiscalYearsCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateFiscalYear(ctx, &pb.CreateFiscalYearRequest{FiscalYear: &pb.FiscalYear{Label: req.Label, StartDate: dateStr(req.StartDate), EndDate: dateStr(req.EndDate)}})
	if err != nil {
		return nil, err
	}
	out := toFiscalYear(res.GetFiscalYear())
	return &out, nil
}

func (h *handler) AdminFinanceFiscalYearsGet(ctx context.Context, p api.AdminFinanceFiscalYearsGetParams) (api.AdminFinanceFiscalYearsGetRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetFiscalYear(ctx, &pb.GetFiscalYearRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toFiscalYear(res.GetFiscalYear())
	return &out, nil
}

func (h *handler) AdminFinanceFiscalPeriodsClose(ctx context.Context, p api.AdminFinanceFiscalPeriodsCloseParams) (api.AdminFinanceFiscalPeriodsCloseRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CloseFiscalPeriod(ctx, &pb.CloseFiscalPeriodRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toFiscalPeriod(res.GetPeriod())
	return &out, nil
}

// ── approbations ──

func toFinApproval(a *pb.Approval) api.FinanceApproval {
	out := api.FinanceApproval{ID: uuidOf(a.GetId()), Type: api.FinanceApprovalType(a.GetType()), SubjectId: uuidOf(a.GetSubjectId()), SubjectLabel: a.GetSubjectLabel(), Reason: a.GetReason(),
		Status: api.FinanceApprovalStatus(a.GetStatus()), RequestedBy: uuidOf(a.GetRequestedBy()), RequestedAt: a.GetRequestedAt().AsTime(), DecidedBy: nilUUIDStr(a.DecidedBy), DecisionNote: nilStr(a.DecisionNote)}
	if a.DecidedAt != nil {
		out.DecidedAt = api.NewOptNilDateTime(a.GetDecidedAt().AsTime())
	} else {
		out.DecidedAt.SetToNull()
	}
	return out
}

func (h *handler) AdminFinanceFiscalPeriodsReopenRequest(ctx context.Context, req *api.ReopenRequestInput, p api.AdminFinanceFiscalPeriodsReopenRequestParams) (api.AdminFinanceFiscalPeriodsReopenRequestRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.RequestPeriodReopen(ctx, &pb.RequestPeriodReopenRequest{Id: p.ID.String(), Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	out := toFinApproval(res.GetApproval())
	return &out, nil
}

func (h *handler) AdminFinanceApprovalsList(ctx context.Context, p api.AdminFinanceApprovalsListParams) (api.AdminFinanceApprovalsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListApprovalsRequest{Page: page(p.Page, p.PageSize)}
	if v, ok := p.Status.Get(); ok {
		s := string(v)
		req.Status = &s
	}
	res, err := c.ListApprovals(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.ApprovalsPage{Items: make([]api.FinanceApproval, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toFinApproval(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceApprovalsApprove(ctx context.Context, req *api.ApprovalDecisionInput, p api.AdminFinanceApprovalsApproveParams) (api.AdminFinanceApprovalsApproveRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ApproveApproval(ctx, &pb.ApproveApprovalRequest{Id: p.ID.String(), Note: nilStrIn(req.Note)})
	if err != nil {
		return nil, err
	}
	out := toFinApproval(res.GetApproval())
	return &out, nil
}

func (h *handler) AdminFinanceApprovalsReject(ctx context.Context, req *api.ApprovalDecisionInput, p api.AdminFinanceApprovalsRejectParams) (api.AdminFinanceApprovalsRejectRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.RejectApproval(ctx, &pb.RejectApprovalRequest{Id: p.ID.String(), Note: nilStrIn(req.Note)})
	if err != nil {
		return nil, err
	}
	out := toFinApproval(res.GetApproval())
	return &out, nil
}

// ── écritures ──

func toFinEntry(e *pb.Entry) api.FinanceEntry {
	out := api.FinanceEntry{ID: uuidOf(e.GetId()), JournalId: uuidOf(e.GetJournalId()), JournalCode: e.GetJournalCode(), FiscalPeriodId: uuidOf(e.GetFiscalPeriodId()), AccountingDate: parseDate(e.GetAccountingDate()),
		Status: api.FinanceEntryStatus(e.GetStatus()), Label: e.GetLabel(), Reference: nilStr(e.Reference), Currency: e.GetCurrency(), Origin: api.FinanceEntryOrigin(e.GetOrigin()),
		ReversesEntryId: nilUUIDStr(e.ReversesEntryId), ReversedByEntryId: nilUUIDStr(e.ReversedByEntryId), ReversalReason: nilStr(e.ReversalReason), Number: nilStr(e.Number),
		TotalDebit: e.GetTotalDebit(), TotalCredit: e.GetTotalCredit(), CreatedBy: uuidOf(e.GetCreatedBy()), CreatedAt: e.GetCreatedAt().AsTime(), PostedBy: nilUUIDStr(e.PostedBy), Lines: []api.EntryLine{}}
	if e.EntryNumber != nil {
		out.EntryNumber = api.NewOptNilInt64(e.GetEntryNumber())
	} else {
		out.EntryNumber.SetToNull()
	}
	if e.PostedAt != nil {
		out.PostedAt = api.NewOptNilDateTime(e.GetPostedAt().AsTime())
	} else {
		out.PostedAt.SetToNull()
	}
	if e.Source != nil {
		out.Source = api.NewOptNilEntrySource(api.EntrySource{Type: e.Source.GetType(), ID: uuidOf(e.Source.GetId()), Kind: e.Source.GetKind()})
	} else {
		out.Source.SetToNull()
	}
	for _, l := range e.GetLines() {
		out.Lines = append(out.Lines, api.EntryLine{ID: uuidOf(l.GetId()), AccountId: uuidOf(l.GetAccountId()), AccountCode: l.GetAccountCode(), AccountLabel: l.GetAccountLabel(), Debit: l.GetDebit(), Credit: l.GetCredit(), Memo: nilStr(l.Memo)})
	}
	return out
}

func entryProto(id string, in *api.EntryInput) *pb.Entry {
	e := &pb.Entry{Id: id, JournalId: in.JournalId.String(), AccountingDate: dateStr(in.AccountingDate), Label: in.Label, Reference: nilStrIn(in.Reference)}
	for _, l := range in.Lines {
		e.Lines = append(e.Lines, &pb.EntryLine{AccountId: l.AccountId.String(), Debit: l.Debit, Credit: l.Credit, Memo: nilStrIn(l.Memo)})
	}
	return e
}

func (h *handler) AdminFinanceEntriesList(ctx context.Context, p api.AdminFinanceEntriesListParams) (api.AdminFinanceEntriesListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListEntriesRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or(""), JournalId: optUUIDPtr(p.JournalId), AccountId: optUUIDPtr(p.AccountId), From: optDatePtr(p.From), To: optDatePtr(p.To)}
	if v, ok := p.Status.Get(); ok {
		s := string(v)
		req.Status = &s
	}
	res, err := c.ListEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.EntriesPage{Items: make([]api.FinanceEntry, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toFinEntry(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceEntriesCreate(ctx context.Context, req *api.EntryInput) (api.AdminFinanceEntriesCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateEntry(ctx, &pb.CreateEntryRequest{Entry: entryProto("", req)})
	if err != nil {
		return nil, err
	}
	out := toFinEntry(res.GetEntry())
	return &out, nil
}

func (h *handler) AdminFinanceEntriesGet(ctx context.Context, p api.AdminFinanceEntriesGetParams) (api.AdminFinanceEntriesGetRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetEntry(ctx, &pb.GetEntryRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toFinEntry(res.GetEntry())
	return &out, nil
}

func (h *handler) AdminFinanceEntriesUpdate(ctx context.Context, req *api.EntryInput, p api.AdminFinanceEntriesUpdateParams) (api.AdminFinanceEntriesUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateEntry(ctx, &pb.UpdateEntryRequest{Entry: entryProto(p.ID.String(), req)})
	if err != nil {
		return nil, err
	}
	out := toFinEntry(res.GetEntry())
	return &out, nil
}

func (h *handler) AdminFinanceEntriesDelete(ctx context.Context, p api.AdminFinanceEntriesDeleteParams) (api.AdminFinanceEntriesDeleteRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteEntry(ctx, &pb.DeleteEntryRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminFinanceEntriesDeleteNoContent{}, nil
}

func (h *handler) AdminFinanceEntriesPost(ctx context.Context, p api.AdminFinanceEntriesPostParams) (api.AdminFinanceEntriesPostRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.PostEntry(ctx, &pb.PostEntryRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toFinEntry(res.GetEntry())
	return &out, nil
}

func (h *handler) AdminFinanceEntriesReverse(ctx context.Context, req *api.ReverseInput, p api.AdminFinanceEntriesReverseParams) (api.AdminFinanceEntriesReverseRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	r := &pb.ReverseEntryRequest{Id: p.ID.String(), Reason: req.Reason}
	if v, ok := req.AccountingDate.Get(); ok {
		s := dateStr(v)
		r.AccountingDate = &s
	}
	res, err := c.ReverseEntry(ctx, r)
	if err != nil {
		return nil, err
	}
	out := toFinEntry(res.GetEntry())
	return &out, nil
}

// ── états ──

func (h *handler) AdminFinanceGeneralLedger(ctx context.Context, p api.AdminFinanceGeneralLedgerParams) (api.AdminFinanceGeneralLedgerRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetGeneralLedger(ctx, &pb.GetGeneralLedgerRequest{Page: page(p.Page, p.PageSize), AccountId: p.AccountId.String(), From: optDatePtr(p.From), To: optDatePtr(p.To)})
	if err != nil {
		return nil, err
	}
	out := &api.GeneralLedger{Account: toFinAccount(res.GetAccount()), Currency: res.GetCurrency(), OpeningBalance: res.GetOpeningBalance(), ClosingBalance: res.GetClosingBalance(),
		Items: make([]api.LedgerLine, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, l := range res.GetItems() {
		out.Items = append(out.Items, api.LedgerLine{EntryId: uuidOf(l.GetEntryId()), EntryNumber: l.GetEntryNumber(), AccountingDate: parseDate(l.GetAccountingDate()), JournalCode: l.GetJournalCode(),
			Label: l.GetLabel(), Reference: nilStr(l.Reference), Memo: nilStr(l.Memo), Debit: l.GetDebit(), Credit: l.GetCredit(), Balance: l.GetBalance()})
	}
	return out, nil
}

func (h *handler) AdminFinanceTrialBalance(ctx context.Context, p api.AdminFinanceTrialBalanceParams) (api.AdminFinanceTrialBalanceRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetTrialBalance(ctx, &pb.GetTrialBalanceRequest{From: dateStr(p.From), To: dateStr(p.To)})
	if err != nil {
		return nil, err
	}
	out := &api.TrialBalance{From: parseDate(res.GetFrom()), To: parseDate(res.GetTo()), Currency: res.GetCurrency(), TotalDebit: res.GetTotalDebit(), TotalCredit: res.GetTotalCredit(), Balanced: res.GetBalanced(), Rows: make([]api.TrialBalanceRow, 0, len(res.GetRows()))}
	for _, r := range res.GetRows() {
		out.Rows = append(out.Rows, api.TrialBalanceRow{AccountId: uuidOf(r.GetAccountId()), Code: r.GetCode(), Label: r.GetLabel(), Kind: r.GetKind(), OpeningBalance: r.GetOpeningBalance(), Debit: r.GetDebit(), Credit: r.GetCredit(), ClosingBalance: r.GetClosingBalance()})
	}
	return out, nil
}

func (h *handler) AdminFinanceGeneralLedgerExport(ctx context.Context, p api.AdminFinanceGeneralLedgerExportParams) (api.AdminFinanceGeneralLedgerExportRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ExportGeneralLedger(ctx, &pb.ExportGeneralLedgerRequest{AccountId: p.AccountId.String(), From: optDatePtr(p.From), To: optDatePtr(p.To)})
	if err != nil {
		return nil, err
	}
	return &api.AdminFinanceGeneralLedgerExportOK{Data: bytes.NewReader(res.GetCsv())}, nil
}

func (h *handler) AdminFinanceTrialBalanceExport(ctx context.Context, p api.AdminFinanceTrialBalanceExportParams) (api.AdminFinanceTrialBalanceExportRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ExportTrialBalance(ctx, &pb.ExportTrialBalanceRequest{From: dateStr(p.From), To: dateStr(p.To)})
	if err != nil {
		return nil, err
	}
	return &api.AdminFinanceTrialBalanceExportOK{Data: bytes.NewReader(res.GetCsv())}, nil
}

var _ = uuid.Nil
