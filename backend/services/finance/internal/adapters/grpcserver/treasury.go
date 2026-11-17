package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func bankTo(b *domain.BankAccount) *pb.BankAccount {
	return &pb.BankAccount{Id: b.ID, Name: b.Name, BankName: b.BankName, Currency: b.Currency, AccountId: b.AccountID, AccountCode: b.AccountCode, AccountLabel: b.AccountLabel, HasNumber: b.HasNumber,
		NumberLast4: b.NumberLast4, OpeningBalance: b.OpeningBalance, OpeningDate: date(b.OpeningDate), Active: b.Active, LedgerBalance: b.LedgerBalance, StatementBalance: b.StatementBalance, LastStatementDate: optT(b.LastStatementDate)}
}

func bankFrom(b *pb.BankAccount) (domain.BankAccount, error) {
	od, err := reqDate(b.GetOpeningDate(), "/openingDate")
	if err != nil {
		return domain.BankAccount{}, err
	}
	return domain.BankAccount{Name: b.GetName(), BankName: b.GetBankName(), AccountID: b.GetAccountId(), OpeningBalance: b.GetOpeningBalance(), OpeningDate: od}, nil
}

func (s *Server) ListBankAccounts(ctx context.Context, r *pb.ListBankAccountsRequest) (*pb.ListBankAccountsResponse, error) {
	v, err := s.Svc.ListBankAccounts(ctx, r.Active)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListBankAccountsResponse{Items: []*pb.BankAccount{}}
	for _, b := range v {
		out.Items = append(out.Items, bankTo(b))
	}
	return out, nil
}

func (s *Server) GetBankAccount(ctx context.Context, r *pb.GetBankAccountRequest) (*pb.GetBankAccountResponse, error) {
	v, err := s.Svc.GetBankAccount(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetBankAccountResponse{BankAccount: bankTo(v)}, nil
}

func (s *Server) CreateBankAccount(ctx context.Context, r *pb.CreateBankAccountRequest) (*pb.CreateBankAccountResponse, error) {
	in, err := bankFrom(r.GetBankAccount())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.CreateBankAccount(ctx, actor(ctx), in, r.Number)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateBankAccountResponse{BankAccount: bankTo(v)}, nil
}

func (s *Server) UpdateBankAccount(ctx context.Context, r *pb.UpdateBankAccountRequest) (*pb.UpdateBankAccountResponse, error) {
	in, err := bankFrom(r.GetBankAccount())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.UpdateBankAccount(ctx, actor(ctx), r.GetBankAccount().GetId(), in, r.GetBankAccount().GetActive(), r.Number)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateBankAccountResponse{BankAccount: bankTo(v)}, nil
}

func stTo(x *domain.Statement) *pb.BankStatement {
	return &pb.BankStatement{Id: x.ID, BankAccountId: x.BankAccountID, Filename: x.Filename, PeriodStart: date(x.PeriodStart), PeriodEnd: date(x.PeriodEnd), LineCount: int32(x.LineCount),
		InsertedCount: int32(x.InsertedCount), SkippedCount: int32(x.SkippedCount), ClosingBalance: x.ClosingBalance, ImportedBy: x.ImportedBy, ImportedAt: timestamppb.New(x.ImportedAt)}
}

func (s *Server) ListBankStatements(ctx context.Context, r *pb.ListBankStatementsRequest) (*pb.ListBankStatementsResponse, error) {
	v, err := s.Svc.ListStatements(ctx, r.GetBankAccountId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListBankStatementsResponse{Items: []*pb.BankStatement{}}
	for _, x := range v {
		out.Items = append(out.Items, stTo(x))
	}
	return out, nil
}

func (s *Server) ImportBankStatement(ctx context.Context, r *pb.ImportBankStatementRequest) (*pb.ImportBankStatementResponse, error) {
	res, err := s.Svc.ImportBankStatement(ctx, actor(ctx), r.GetBankAccountId(), r.GetFilename(), r.GetCsv(), r.ClosingBalance, r.GetSimulate())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ImportBankStatementResponse{Simulated: res.Simulated, Total: int32(res.Total), Inserted: int32(res.Inserted), Skipped: int32(res.Skipped), SumAmount: res.SumAmount,
		PeriodStart: optT(res.PeriodStart), PeriodEnd: optT(res.PeriodEnd), Errors: []*pb.ImportError{}}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, &pb.ImportError{Line: int32(e.Line), Message: e.Message})
	}
	if res.Statement != nil {
		out.Statement = stTo(res.Statement)
	}
	return out, nil
}

func lineTo(l *domain.BankLine) *pb.BankLine {
	out := &pb.BankLine{Id: l.ID, StatementId: l.StatementID, BankAccountId: l.BankAccountID, LineDate: date(l.LineDate), Label: l.Label, Reference: l.Reference, Amount: l.Amount,
		Matched: l.Matched, Remaining: l.Remaining, Status: l.Status, Matches: []*pb.ReconMatch{}}
	for _, m := range l.Matches {
		out.Matches = append(out.Matches, &pb.ReconMatch{Id: m.ID, EntryLineId: m.EntryLineID, EntryId: m.EntryID, EntryNumber: m.EntryNumber, Amount: m.Amount, MatchedAt: timestamppb.New(m.MatchedAt), MatchedBy: m.MatchedBy})
	}
	return out
}

func candTo(c domain.LedgerCandidate) *pb.LedgerCandidate {
	return &pb.LedgerCandidate{EntryLineId: c.EntryLineID, EntryId: c.EntryID, EntryNumber: c.EntryNumber, AccountingDate: date(c.AccountingDate), Label: c.Label, Reference: c.Reference,
		Amount: c.Amount, Remaining: c.Remaining, Score: int32(c.Score), ExactAmount: c.ExactAmount}
}

func (s *Server) ListBankLines(ctx context.Context, r *pb.ListBankLinesRequest) (*pb.ListBankLinesResponse, error) {
	from, err := optDate(r.From, "/from")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	to, err := optDate(r.To, "/to")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	p, err := s.Svc.ListBankLines(ctx, application.BankLineFilter{BankAccountID: r.GetBankAccountId(), Query: r.GetQuery(), Status: r.Status, From: from, To: to}, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListBankLinesResponse{Items: []*pb.BankLine{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, lineTo(x))
	}
	return out, nil
}

func (s *Server) ListLedgerLines(ctx context.Context, r *pb.ListLedgerLinesRequest) (*pb.ListLedgerLinesResponse, error) {
	p, err := s.Svc.ListLedgerLines(ctx, r.GetBankAccountId(), r.GetStatus() != "all", r.GetQuery(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListLedgerLinesResponse{Items: []*pb.LedgerCandidate{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, candTo(x))
	}
	return out, nil
}

func (s *Server) ListCandidates(ctx context.Context, r *pb.ListCandidatesRequest) (*pb.ListCandidatesResponse, error) {
	v, err := s.Svc.ListCandidates(ctx, r.GetBankLineId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListCandidatesResponse{Items: []*pb.LedgerCandidate{}}
	for _, x := range v {
		out.Items = append(out.Items, candTo(x))
	}
	return out, nil
}

func (s *Server) CreateReconciliation(ctx context.Context, r *pb.CreateReconciliationRequest) (*pb.CreateReconciliationResponse, error) {
	l, err := s.Svc.CreateReconciliation(ctx, actor(ctx), r.GetBankLineId(), r.GetEntryLineId(), r.GetAmount())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateReconciliationResponse{Line: lineTo(l)}, nil
}

func (s *Server) DeleteReconciliation(ctx context.Context, r *pb.DeleteReconciliationRequest) (*pb.DeleteReconciliationResponse, error) {
	l, err := s.Svc.DeleteReconciliation(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteReconciliationResponse{Line: lineTo(l)}, nil
}

func (s *Server) GetReconciliationSummary(ctx context.Context, r *pb.GetReconciliationSummaryRequest) (*pb.GetReconciliationSummaryResponse, error) {
	v, err := s.Svc.GetReconciliationSummary(ctx, r.GetBankAccountId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetReconciliationSummaryResponse{Summary: &pb.ReconciliationSummary{BankAccountId: v.BankAccountID, Currency: v.Currency, LedgerBalance: v.LedgerBalance, StatementBalance: v.StatementBalance,
		Difference: v.Difference, UnmatchedStatement: v.UnmatchedStatement, UnmatchedLedger: v.UnmatchedLedger, UnmatchedStatementCount: int32(v.UnmatchedStatementCount),
		UnmatchedLedgerCount: int32(v.UnmatchedLedgerCount), Unexplained: v.Unexplained, LastStatementDate: optT(v.LastStatementDate), OpeningGap: v.OpeningGap}}, nil
}

func (s *Server) GetCashForecast(ctx context.Context, r *pb.GetCashForecastRequest) (*pb.GetCashForecastResponse, error) {
	f, err := s.Svc.GetCashForecast(ctx, int(r.GetHorizonDays()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.GetCashForecastResponse{Currency: f.Currency, AsOf: date(f.AsOf), HorizonDays: int32(f.HorizonDays), OpeningBalance: f.OpeningBalance, OverdueInflows: f.OverdueInflows,
		OverdueOutflows: f.OverdueOutflows, UndatedInflows: f.UndatedInflows, UndatedOutflows: f.UndatedOutflows, ExcludedDrafts: int32(f.ExcludedDrafts), ExcludedDraftsAmount: f.ExcludedDraftsAmount,
		BankAccounts: int32(f.BankAccounts), Assumptions: f.Assumptions, Buckets: []*pb.ForecastBucket{}}
	for _, b := range f.Buckets {
		out.Buckets = append(out.Buckets, &pb.ForecastBucket{StartDate: date(b.Start), EndDate: date(b.End), Inflows: b.Inflows, Outflows: b.Outflows, ProjectedBalance: b.Balance})
	}
	return out, nil
}
