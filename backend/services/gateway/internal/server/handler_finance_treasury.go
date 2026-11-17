package server

import (
	"context"

	pb "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Finance F3 : trésorerie. `treasury:read` consulte, `treasury:write` gère comptes et relevés, `treasury:reconcile` rapproche.

func toBankAccount(b *pb.BankAccount) api.FinanceBankAccount {
	return api.FinanceBankAccount{ID: uuidOf(b.GetId()), Name: b.GetName(), BankName: b.GetBankName(), Currency: b.GetCurrency(), AccountId: uuidOf(b.GetAccountId()), AccountCode: b.GetAccountCode(), AccountLabel: b.GetAccountLabel(),
		HasNumber: b.GetHasNumber(), NumberLast4: nilStr(b.NumberLast4), OpeningBalance: b.GetOpeningBalance(), OpeningDate: parseDate(b.GetOpeningDate()), Active: b.GetActive(), LedgerBalance: b.GetLedgerBalance(),
		StatementBalance: b.GetStatementBalance(), LastStatementDate: nilDate(b.LastStatementDate)}
}

func bankProto(id string, in *api.BankAccountInput) *pb.BankAccount {
	return &pb.BankAccount{Id: id, Name: in.Name, BankName: in.BankName, AccountId: in.AccountId.String(), OpeningBalance: in.OpeningBalance, OpeningDate: dateStr(in.OpeningDate), Active: in.Active.Or(true)}
}

func (h *handler) AdminFinanceBankAccountsList(ctx context.Context, p api.AdminFinanceBankAccountsListParams) (api.AdminFinanceBankAccountsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListBankAccountsRequest{}
	if v, ok := p.Active.Get(); ok {
		req.Active = &v
	}
	res, err := c.ListBankAccounts(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.BankAccountsList{Items: make([]api.FinanceBankAccount, 0, len(res.GetItems()))}
	for _, b := range res.GetItems() {
		out.Items = append(out.Items, toBankAccount(b))
	}
	return out, nil
}

func (h *handler) AdminFinanceBankAccountsCreate(ctx context.Context, req *api.BankAccountInput) (api.AdminFinanceBankAccountsCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateBankAccount(ctx, &pb.CreateBankAccountRequest{BankAccount: bankProto("", req), Number: nilStrIn(req.Number)})
	if err != nil {
		return nil, err
	}
	out := toBankAccount(res.GetBankAccount())
	return &out, nil
}

func (h *handler) AdminFinanceBankAccountsGet(ctx context.Context, p api.AdminFinanceBankAccountsGetParams) (api.AdminFinanceBankAccountsGetRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetBankAccount(ctx, &pb.GetBankAccountRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toBankAccount(res.GetBankAccount())
	return &out, nil
}

func (h *handler) AdminFinanceBankAccountsUpdate(ctx context.Context, req *api.BankAccountInput, p api.AdminFinanceBankAccountsUpdateParams) (api.AdminFinanceBankAccountsUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateBankAccount(ctx, &pb.UpdateBankAccountRequest{BankAccount: bankProto(p.ID.String(), req), Number: nilStrIn(req.Number)})
	if err != nil {
		return nil, err
	}
	out := toBankAccount(res.GetBankAccount())
	return &out, nil
}

func toStatement(x *pb.BankStatement) api.FinanceBankStatement {
	out := api.FinanceBankStatement{ID: uuidOf(x.GetId()), BankAccountId: uuidOf(x.GetBankAccountId()), Filename: x.GetFilename(), PeriodStart: parseDate(x.GetPeriodStart()), PeriodEnd: parseDate(x.GetPeriodEnd()),
		LineCount: x.GetLineCount(), InsertedCount: x.GetInsertedCount(), SkippedCount: x.GetSkippedCount(), ImportedBy: uuidOf(x.GetImportedBy()), ImportedAt: x.GetImportedAt().AsTime()}
	if x.ClosingBalance != nil {
		out.ClosingBalance = api.NewOptNilInt64(x.GetClosingBalance())
	} else {
		out.ClosingBalance.SetToNull()
	}
	return out
}

func (h *handler) AdminFinanceBankStatementsList(ctx context.Context, p api.AdminFinanceBankStatementsListParams) (api.AdminFinanceBankStatementsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListBankStatements(ctx, &pb.ListBankStatementsRequest{BankAccountId: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := &api.StatementsList{Items: make([]api.FinanceBankStatement, 0, len(res.GetItems()))}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toStatement(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceBankStatementsImport(ctx context.Context, req *api.StatementImportRequest) (api.AdminFinanceBankStatementsImportRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	r := &pb.ImportBankStatementRequest{BankAccountId: req.BankAccountId.String(), Filename: req.Filename.Or(""), Csv: req.Csv, Simulate: req.Simulate.Or(false)}
	if v, ok := req.ClosingBalance.Get(); ok {
		r.ClosingBalance = &v
	}
	res, err := c.ImportBankStatement(ctx, r)
	if err != nil {
		return nil, err
	}
	out := &api.StatementImportResult{Simulated: res.GetSimulated(), Total: res.GetTotal(), Inserted: res.GetInserted(), Skipped: res.GetSkipped(), SumAmount: res.GetSumAmount(),
		PeriodStart: nilDate(res.PeriodStart), PeriodEnd: nilDate(res.PeriodEnd), Errors: []api.ImportLineError{}}
	for _, e := range res.GetErrors() {
		out.Errors = append(out.Errors, api.ImportLineError{Line: e.GetLine(), Message: e.GetMessage()})
	}
	if res.Statement != nil {
		out.Statement = api.NewOptNilFinanceBankStatement(toStatement(res.GetStatement()))
	} else {
		out.Statement.SetToNull()
	}
	return out, nil
}

func toBankLine(l *pb.BankLine) api.BankLine {
	out := api.BankLine{ID: uuidOf(l.GetId()), StatementId: uuidOf(l.GetStatementId()), BankAccountId: uuidOf(l.GetBankAccountId()), LineDate: parseDate(l.GetLineDate()), Label: l.GetLabel(), Reference: nilStr(l.Reference),
		Amount: l.GetAmount(), Matched: l.GetMatched(), Remaining: l.GetRemaining(), Status: api.BankLineStatus(l.GetStatus()), Matches: []api.ReconMatch{}}
	for _, m := range l.GetMatches() {
		out.Matches = append(out.Matches, api.ReconMatch{ID: uuidOf(m.GetId()), EntryLineId: uuidOf(m.GetEntryLineId()), EntryId: uuidOf(m.GetEntryId()), EntryNumber: m.GetEntryNumber(), Amount: m.GetAmount(), MatchedAt: m.GetMatchedAt().AsTime(), MatchedBy: uuidOf(m.GetMatchedBy())})
	}
	return out
}

func toCandidate(c *pb.LedgerCandidate) api.LedgerCandidate {
	return api.LedgerCandidate{EntryLineId: uuidOf(c.GetEntryLineId()), EntryId: uuidOf(c.GetEntryId()), EntryNumber: c.GetEntryNumber(), AccountingDate: parseDate(c.GetAccountingDate()), Label: c.GetLabel(),
		Reference: nilStr(c.Reference), Amount: c.GetAmount(), Remaining: c.GetRemaining(), Score: c.GetScore(), ExactAmount: c.GetExactAmount()}
}

func (h *handler) AdminFinanceBankLinesList(ctx context.Context, p api.AdminFinanceBankLinesListParams) (api.AdminFinanceBankLinesListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListBankLinesRequest{Page: page(p.Page, p.PageSize), BankAccountId: p.ID.String(), Query: p.Q.Or(""), From: optDatePtr(p.From), To: optDatePtr(p.To)}
	if v, ok := p.Status.Get(); ok {
		s := string(v)
		req.Status = &s
	}
	res, err := c.ListBankLines(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.BankLinesPage{Items: make([]api.BankLine, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, l := range res.GetItems() {
		out.Items = append(out.Items, toBankLine(l))
	}
	return out, nil
}

func (h *handler) AdminFinanceLedgerLinesList(ctx context.Context, p api.AdminFinanceLedgerLinesListParams) (api.AdminFinanceLedgerLinesListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	status := "unmatched"
	if v, ok := p.Status.Get(); ok {
		status = string(v)
	}
	q := p.Q.Or("")
	res, err := c.ListLedgerLines(ctx, &pb.ListLedgerLinesRequest{Page: page(p.Page, p.PageSize), BankAccountId: p.ID.String(), Status: status, Query: &q})
	if err != nil {
		return nil, err
	}
	out := &api.LedgerLinesPage{Items: make([]api.LedgerCandidate, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toCandidate(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceBankLineCandidates(ctx context.Context, p api.AdminFinanceBankLineCandidatesParams) (api.AdminFinanceBankLineCandidatesRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListCandidates(ctx, &pb.ListCandidatesRequest{BankLineId: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := &api.CandidatesList{Items: make([]api.LedgerCandidate, 0, len(res.GetItems()))}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toCandidate(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceReconciliationsCreate(ctx context.Context, req *api.ReconciliationInput) (api.AdminFinanceReconciliationsCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateReconciliation(ctx, &pb.CreateReconciliationRequest{BankLineId: req.BankLineId.String(), EntryLineId: req.EntryLineId.String(), Amount: req.Amount.Or(0)})
	if err != nil {
		return nil, err
	}
	out := toBankLine(res.GetLine())
	return &out, nil
}

func (h *handler) AdminFinanceReconciliationsDelete(ctx context.Context, p api.AdminFinanceReconciliationsDeleteParams) (api.AdminFinanceReconciliationsDeleteRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.DeleteReconciliation(ctx, &pb.DeleteReconciliationRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toBankLine(res.GetLine())
	return &out, nil
}

func (h *handler) AdminFinanceReconciliationSummary(ctx context.Context, p api.AdminFinanceReconciliationSummaryParams) (api.AdminFinanceReconciliationSummaryRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetReconciliationSummary(ctx, &pb.GetReconciliationSummaryRequest{BankAccountId: p.ID.String()})
	if err != nil {
		return nil, err
	}
	s := res.GetSummary()
	return &api.ReconciliationSummary{BankAccountId: uuidOf(s.GetBankAccountId()), Currency: s.GetCurrency(), LedgerBalance: s.GetLedgerBalance(), StatementBalance: s.GetStatementBalance(), Difference: s.GetDifference(),
		UnmatchedStatement: s.GetUnmatchedStatement(), UnmatchedLedger: s.GetUnmatchedLedger(), UnmatchedStatementCount: s.GetUnmatchedStatementCount(), UnmatchedLedgerCount: s.GetUnmatchedLedgerCount(),
		OpeningGap: s.GetOpeningGap(), Unexplained: s.GetUnexplained(), LastStatementDate: nilDate(s.LastStatementDate)}, nil
}

func (h *handler) AdminFinanceCashForecast(ctx context.Context, p api.AdminFinanceCashForecastParams) (api.AdminFinanceCashForecastRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	days := int32(0)
	if v, ok := p.HorizonDays.Get(); ok {
		days = int32(v)
	}
	res, err := c.GetCashForecast(ctx, &pb.GetCashForecastRequest{HorizonDays: days})
	if err != nil {
		return nil, err
	}
	out := &api.CashForecast{Currency: res.GetCurrency(), AsOf: parseDate(res.GetAsOf()), HorizonDays: res.GetHorizonDays(), OpeningBalance: res.GetOpeningBalance(), OverdueInflows: res.GetOverdueInflows(),
		OverdueOutflows: res.GetOverdueOutflows(), UndatedInflows: res.GetUndatedInflows(), UndatedOutflows: res.GetUndatedOutflows(), ExcludedDrafts: res.GetExcludedDrafts(),
		ExcludedDraftsAmount: res.GetExcludedDraftsAmount(), BankAccounts: res.GetBankAccounts(), Assumptions: res.GetAssumptions(), Buckets: make([]api.ForecastBucket, 0, len(res.GetBuckets()))}
	if out.Assumptions == nil {
		out.Assumptions = []string{}
	}
	for _, b := range res.GetBuckets() {
		out.Buckets = append(out.Buckets, api.ForecastBucket{StartDate: parseDate(b.GetStartDate()), EndDate: parseDate(b.GetEndDate()), Inflows: b.GetInflows(), Outflows: b.GetOutflows(), ProjectedBalance: b.GetProjectedBalance()})
	}
	return out, nil
}
