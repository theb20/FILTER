// Package grpcserver expose le service finance en gRPC (contrat proto/financeadmin/v1).
package grpcserver

import (
	"context"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	pb "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedFinanceAdminServiceServer
	Svc *application.Service
}

func meta(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func actor(ctx context.Context) application.Actor {
	return application.Actor{ID: meta(ctx, "x-actor-id"), IP: meta(ctx, "x-client-ip"), TraceID: meta(ctx, "x-request-id")}
}

func pageResp[T any](p *application.Page[T]) *commonv1.PageResponse {
	return &commonv1.PageResponse{Page: int32(p.Page), PageSize: int32(p.PageSize), Total: int64(p.Total), TotalPages: int32(p.Pages)}
}

func date(t time.Time) string { return t.Format(domain.DateLayout) }

func optDate(s *string, ptr string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := domain.ParseDate(*s)
	if err != nil {
		return nil, apperr.Validation("date invalide", apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: err.Error()})
	}
	return &t, nil
}

func reqDate(s, ptr string) (time.Time, error) {
	t, err := optDate(&s, ptr)
	if err != nil {
		return time.Time{}, err
	}
	if t == nil {
		return time.Time{}, apperr.Validation("date obligatoire", apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: "date obligatoire"})
	}
	return *t, nil
}

func ts(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// ── réglages ──

func settingsTo(s *domain.Settings) *pb.Settings {
	return &pb.Settings{BaseCurrency: s.BaseCurrency, FourEyesMinAmount: s.FourEyesMinAmount, UpdatedAt: timestamppb.New(s.UpdatedAt)}
}

func (s *Server) GetSettings(ctx context.Context, _ *pb.GetSettingsRequest) (*pb.GetSettingsResponse, error) {
	v, err := s.Svc.GetSettings(ctx)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetSettingsResponse{Settings: settingsTo(v)}, nil
}

func (s *Server) PutSettings(ctx context.Context, r *pb.PutSettingsRequest) (*pb.PutSettingsResponse, error) {
	v, err := s.Svc.PutSettings(ctx, actor(ctx), domain.Settings{BaseCurrency: r.GetSettings().GetBaseCurrency(), FourEyesMinAmount: r.GetSettings().GetFourEyesMinAmount()})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.PutSettingsResponse{Settings: settingsTo(v)}, nil
}

// ── comptes ──

func accountTo(a *domain.Account) *pb.Account {
	return &pb.Account{Id: a.ID, Code: a.Code, Label: a.Label, Kind: a.Kind, Active: a.Active, CreatedAt: timestamppb.New(a.CreatedAt)}
}

func (s *Server) ListAccounts(ctx context.Context, r *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	p, err := s.Svc.ListAccounts(ctx, application.AccountFilter{Query: r.GetQuery(), Kind: r.Kind, Active: r.Active}, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListAccountsResponse{Items: []*pb.Account{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, accountTo(x))
	}
	return out, nil
}

func (s *Server) GetAccount(ctx context.Context, r *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	v, err := s.Svc.GetAccount(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetAccountResponse{Account: accountTo(v)}, nil
}

func (s *Server) CreateAccount(ctx context.Context, r *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	in := r.GetAccount()
	v, err := s.Svc.CreateAccount(ctx, actor(ctx), domain.Account{Code: in.GetCode(), Label: in.GetLabel(), Kind: in.GetKind()})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateAccountResponse{Account: accountTo(v)}, nil
}

func (s *Server) UpdateAccount(ctx context.Context, r *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	in := r.GetAccount()
	v, err := s.Svc.UpdateAccount(ctx, actor(ctx), in.GetId(), domain.Account{Code: in.GetCode(), Label: in.GetLabel(), Kind: in.GetKind()}, in.GetActive())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateAccountResponse{Account: accountTo(v)}, nil
}

func (s *Server) ImportAccounts(ctx context.Context, r *pb.ImportAccountsRequest) (*pb.ImportAccountsResponse, error) {
	v, err := s.Svc.ImportAccounts(ctx, actor(ctx), r.GetCsv(), r.GetSimulate())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ImportAccountsResponse{Simulated: v.Simulated, Total: int32(v.Total), Created: int32(v.Created), Skipped: int32(v.Skipped), Errors: []*pb.ImportError{}}
	for _, e := range v.Errors {
		out.Errors = append(out.Errors, &pb.ImportError{Line: int32(e.Line), Message: e.Message})
	}
	return out, nil
}

// ── journaux ──

func journalTo(j *domain.Journal) *pb.Journal {
	return &pb.Journal{Id: j.ID, Code: j.Code, Label: j.Label, Kind: j.Kind, Active: j.Active}
}

func (s *Server) ListJournals(ctx context.Context, r *pb.ListJournalsRequest) (*pb.ListJournalsResponse, error) {
	v, err := s.Svc.ListJournals(ctx, r.Active)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListJournalsResponse{Items: []*pb.Journal{}}
	for _, j := range v {
		out.Items = append(out.Items, journalTo(j))
	}
	return out, nil
}

func (s *Server) CreateJournal(ctx context.Context, r *pb.CreateJournalRequest) (*pb.CreateJournalResponse, error) {
	in := r.GetJournal()
	v, err := s.Svc.CreateJournal(ctx, actor(ctx), domain.Journal{Code: in.GetCode(), Label: in.GetLabel(), Kind: in.GetKind()})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateJournalResponse{Journal: journalTo(v)}, nil
}

func (s *Server) UpdateJournal(ctx context.Context, r *pb.UpdateJournalRequest) (*pb.UpdateJournalResponse, error) {
	in := r.GetJournal()
	v, err := s.Svc.UpdateJournal(ctx, actor(ctx), in.GetId(), domain.Journal{Code: in.GetCode(), Label: in.GetLabel(), Kind: in.GetKind()}, in.GetActive())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateJournalResponse{Journal: journalTo(v)}, nil
}

// ── exercices ──

func periodTo(p domain.FiscalPeriod) *pb.FiscalPeriod {
	return &pb.FiscalPeriod{Id: p.ID, FiscalYearId: p.FiscalYearID, Number: int32(p.Number), StartDate: date(p.Start), EndDate: date(p.End), Status: p.Status, ClosedAt: ts(p.ClosedAt), ClosedBy: p.ClosedBy}
}

func yearTo(y *domain.FiscalYear) *pb.FiscalYear {
	out := &pb.FiscalYear{Id: y.ID, Label: y.Label, StartDate: date(y.Start), EndDate: date(y.End), Status: y.Status(), Periods: []*pb.FiscalPeriod{}}
	for _, p := range y.Periods {
		out.Periods = append(out.Periods, periodTo(p))
	}
	return out
}

func (s *Server) ListFiscalYears(ctx context.Context, _ *pb.ListFiscalYearsRequest) (*pb.ListFiscalYearsResponse, error) {
	v, err := s.Svc.ListFiscalYears(ctx)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListFiscalYearsResponse{Items: []*pb.FiscalYear{}}
	for _, y := range v {
		out.Items = append(out.Items, yearTo(y))
	}
	return out, nil
}

func (s *Server) GetFiscalYear(ctx context.Context, r *pb.GetFiscalYearRequest) (*pb.GetFiscalYearResponse, error) {
	v, err := s.Svc.GetFiscalYear(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetFiscalYearResponse{FiscalYear: yearTo(v)}, nil
}

func (s *Server) CreateFiscalYear(ctx context.Context, r *pb.CreateFiscalYearRequest) (*pb.CreateFiscalYearResponse, error) {
	in := r.GetFiscalYear()
	start, err := reqDate(in.GetStartDate(), "/startDate")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	end, err := reqDate(in.GetEndDate(), "/endDate")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.CreateFiscalYear(ctx, actor(ctx), domain.FiscalYear{Label: in.GetLabel(), Start: start, End: end})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateFiscalYearResponse{FiscalYear: yearTo(v)}, nil
}

func (s *Server) CloseFiscalPeriod(ctx context.Context, r *pb.CloseFiscalPeriodRequest) (*pb.CloseFiscalPeriodResponse, error) {
	v, err := s.Svc.ClosePeriod(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CloseFiscalPeriodResponse{Period: periodTo(*v)}, nil
}

// ── approbations ──

func approvalTo(a *domain.Approval) *pb.Approval {
	return &pb.Approval{Id: a.ID, Type: a.Type, SubjectId: a.SubjectID, SubjectLabel: a.SubjectLabel, Reason: a.Reason, Status: a.Status, RequestedBy: a.RequestedBy,
		RequestedAt: timestamppb.New(a.RequestedAt), DecidedBy: a.DecidedBy, DecidedAt: ts(a.DecidedAt), DecisionNote: a.DecisionNote}
}

func (s *Server) RequestPeriodReopen(ctx context.Context, r *pb.RequestPeriodReopenRequest) (*pb.RequestPeriodReopenResponse, error) {
	v, err := s.Svc.RequestPeriodReopen(ctx, actor(ctx), r.GetId(), r.GetReason())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.RequestPeriodReopenResponse{Approval: approvalTo(v)}, nil
}

func (s *Server) ListApprovals(ctx context.Context, r *pb.ListApprovalsRequest) (*pb.ListApprovalsResponse, error) {
	p, err := s.Svc.ListApprovals(ctx, r.Status, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListApprovalsResponse{Items: []*pb.Approval{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, approvalTo(x))
	}
	return out, nil
}

func (s *Server) ApproveApproval(ctx context.Context, r *pb.ApproveApprovalRequest) (*pb.ApproveApprovalResponse, error) {
	v, err := s.Svc.DecideApproval(ctx, actor(ctx), r.GetId(), true, r.Note)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.ApproveApprovalResponse{Approval: approvalTo(v)}, nil
}

func (s *Server) RejectApproval(ctx context.Context, r *pb.RejectApprovalRequest) (*pb.RejectApprovalResponse, error) {
	v, err := s.Svc.DecideApproval(ctx, actor(ctx), r.GetId(), false, r.Note)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.RejectApprovalResponse{Approval: approvalTo(v)}, nil
}

// ── écritures ──

func entryTo(e *domain.Entry) *pb.Entry {
	out := &pb.Entry{Id: e.ID, JournalId: e.JournalID, JournalCode: e.JournalCode, FiscalPeriodId: e.FiscalPeriodID, AccountingDate: date(e.AccountingDate), Status: e.Status,
		EntryNumber: e.EntryNumber, Number: e.Number(), Label: e.Label, Reference: e.Reference, Currency: e.Currency, Origin: e.Origin, ReversesEntryId: e.ReversesEntryID,
		ReversedByEntryId: e.ReversedByEntryID, ReversalReason: e.ReversalReason, TotalDebit: e.TotalDebit, TotalCredit: e.TotalCredit, CreatedBy: e.CreatedBy,
		CreatedAt: timestamppb.New(e.CreatedAt), PostedBy: e.PostedBy, PostedAt: ts(e.PostedAt), Lines: []*pb.EntryLine{}}
	if e.Source != nil {
		out.Source = &pb.EntrySource{Type: e.Source.Type, Id: e.Source.ID, Kind: e.Source.Kind}
	}
	for _, l := range e.Lines {
		out.Lines = append(out.Lines, &pb.EntryLine{Id: l.ID, AccountId: l.AccountID, AccountCode: l.AccountCode, AccountLabel: l.AccountLabel, Debit: l.Debit, Credit: l.Credit, Memo: l.Memo})
	}
	return out
}

func entryFrom(e *pb.Entry) (domain.Entry, error) {
	d, err := reqDate(e.GetAccountingDate(), "/accountingDate")
	if err != nil {
		return domain.Entry{}, err
	}
	out := domain.Entry{JournalID: e.GetJournalId(), AccountingDate: d, Label: e.GetLabel(), Reference: e.Reference}
	for _, l := range e.GetLines() {
		out.Lines = append(out.Lines, domain.EntryLine{AccountID: l.GetAccountId(), Debit: l.GetDebit(), Credit: l.GetCredit(), Memo: l.Memo})
	}
	return out, nil
}

func (s *Server) ListEntries(ctx context.Context, r *pb.ListEntriesRequest) (*pb.ListEntriesResponse, error) {
	from, err := optDate(r.From, "/from")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	to, err := optDate(r.To, "/to")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	p, err := s.Svc.ListEntries(ctx, application.EntryFilter{Query: r.GetQuery(), Status: r.Status, JournalID: r.JournalId, AccountID: r.AccountId, From: from, To: to}, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListEntriesResponse{Items: []*pb.Entry{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, entryTo(x))
	}
	return out, nil
}

func (s *Server) GetEntry(ctx context.Context, r *pb.GetEntryRequest) (*pb.GetEntryResponse, error) {
	v, err := s.Svc.GetEntry(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetEntryResponse{Entry: entryTo(v)}, nil
}

func (s *Server) CreateEntry(ctx context.Context, r *pb.CreateEntryRequest) (*pb.CreateEntryResponse, error) {
	in, err := entryFrom(r.GetEntry())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.CreateEntry(ctx, actor(ctx), in)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateEntryResponse{Entry: entryTo(v)}, nil
}

func (s *Server) UpdateEntry(ctx context.Context, r *pb.UpdateEntryRequest) (*pb.UpdateEntryResponse, error) {
	in, err := entryFrom(r.GetEntry())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.UpdateEntry(ctx, actor(ctx), r.GetEntry().GetId(), in)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateEntryResponse{Entry: entryTo(v)}, nil
}

func (s *Server) DeleteEntry(ctx context.Context, r *pb.DeleteEntryRequest) (*pb.DeleteEntryResponse, error) {
	if err := s.Svc.DeleteEntry(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteEntryResponse{}, nil
}

func (s *Server) PostEntry(ctx context.Context, r *pb.PostEntryRequest) (*pb.PostEntryResponse, error) {
	v, err := s.Svc.PostEntry(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.PostEntryResponse{Entry: entryTo(v)}, nil
}

func (s *Server) ReverseEntry(ctx context.Context, r *pb.ReverseEntryRequest) (*pb.ReverseEntryResponse, error) {
	d, err := optDate(r.AccountingDate, "/accountingDate")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.ReverseEntry(ctx, actor(ctx), r.GetId(), r.GetReason(), d)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.ReverseEntryResponse{Entry: entryTo(v)}, nil
}

// ── états ──

func (s *Server) GetGeneralLedger(ctx context.Context, r *pb.GetGeneralLedgerRequest) (*pb.GetGeneralLedgerResponse, error) {
	from, err := optDate(r.From, "/from")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	to, err := optDate(r.To, "/to")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	l, err := s.Svc.GeneralLedger(ctx, r.GetAccountId(), from, to, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.GetGeneralLedgerResponse{Account: accountTo(l.Account), OpeningBalance: l.Opening, ClosingBalance: l.Closing, Currency: l.Currency, Items: []*pb.LedgerLine{},
		Page: &commonv1.PageResponse{Page: int32(l.Page), PageSize: int32(l.PageSize), Total: int64(l.Total), TotalPages: int32(l.Pages)}}
	for _, x := range l.Lines {
		out.Items = append(out.Items, &pb.LedgerLine{EntryId: x.EntryID, EntryNumber: x.EntryNumber, AccountingDate: date(x.AccountingDate), JournalCode: x.JournalCode, Label: x.Label,
			Reference: x.Reference, Memo: x.Memo, Debit: x.Debit, Credit: x.Credit, Balance: x.Balance})
	}
	return out, nil
}

func (s *Server) GetTrialBalance(ctx context.Context, r *pb.GetTrialBalanceRequest) (*pb.GetTrialBalanceResponse, error) {
	from, err := reqDate(r.GetFrom(), "/from")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	to, err := reqDate(r.GetTo(), "/to")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	tb, err := s.Svc.GetTrialBalance(ctx, from, to)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.GetTrialBalanceResponse{From: date(tb.From), To: date(tb.To), Currency: tb.Currency, TotalDebit: tb.TotalDebit, TotalCredit: tb.TotalCredit, Balanced: tb.Balanced, Rows: []*pb.TrialBalanceRow{}}
	for _, x := range tb.Rows {
		out.Rows = append(out.Rows, &pb.TrialBalanceRow{AccountId: x.AccountID, Code: x.Code, Label: x.Label, Kind: x.Kind, OpeningBalance: x.Opening, Debit: x.Debit, Credit: x.Credit, ClosingBalance: x.Closing})
	}
	return out, nil
}

func (s *Server) ExportGeneralLedger(ctx context.Context, r *pb.ExportGeneralLedgerRequest) (*pb.ExportGeneralLedgerResponse, error) {
	from, err := optDate(r.From, "/from")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	to, err := optDate(r.To, "/to")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	b, err := s.Svc.ExportGeneralLedger(ctx, r.GetAccountId(), from, to)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.ExportGeneralLedgerResponse{Csv: b}, nil
}

func (s *Server) ExportTrialBalance(ctx context.Context, r *pb.ExportTrialBalanceRequest) (*pb.ExportTrialBalanceResponse, error) {
	from, err := reqDate(r.GetFrom(), "/from")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	to, err := reqDate(r.GetTo(), "/to")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	b, err := s.Svc.ExportTrialBalance(ctx, from, to)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.ExportTrialBalanceResponse{Csv: b}, nil
}
