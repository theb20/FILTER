package grpcserver

import (
	"context"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/services/finance/internal/application"
	"github.com/theb20/FILTER/backend/services/finance/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func optT(p *time.Time) *string {
	if p == nil {
		return nil
	}
	s := date(*p)
	return &s
}

// ── tiers ──

func tpTo(t *domain.ThirdParty) *pb.ThirdParty {
	return &pb.ThirdParty{Id: t.ID, Kind: t.Kind, ExternalRef: t.ExternalRef, DisplayName: t.DisplayName, Active: t.Active}
}

func (s *Server) ListThirdParties(ctx context.Context, r *pb.ListThirdPartiesRequest) (*pb.ListThirdPartiesResponse, error) {
	p, err := s.Svc.ListThirdParties(ctx, r.GetQuery(), r.Kind, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListThirdPartiesResponse{Items: []*pb.ThirdParty{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, tpTo(x))
	}
	return out, nil
}

func (s *Server) CreateThirdParty(ctx context.Context, r *pb.CreateThirdPartyRequest) (*pb.CreateThirdPartyResponse, error) {
	in := r.GetThirdParty()
	v, err := s.Svc.CreateThirdParty(ctx, actor(ctx), domain.ThirdParty{Kind: in.GetKind(), ExternalRef: in.ExternalRef, DisplayName: in.GetDisplayName()})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateThirdPartyResponse{ThirdParty: tpTo(v)}, nil
}

func (s *Server) UpdateThirdParty(ctx context.Context, r *pb.UpdateThirdPartyRequest) (*pb.UpdateThirdPartyResponse, error) {
	in := r.GetThirdParty()
	v, err := s.Svc.UpdateThirdParty(ctx, actor(ctx), in.GetId(), domain.ThirdParty{ExternalRef: in.ExternalRef, DisplayName: in.GetDisplayName()}, in.GetActive())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateThirdPartyResponse{ThirdParty: tpTo(v)}, nil
}

// ── catégories ──

func catTo(c *domain.OpCategory) *pb.OperationCategory {
	return &pb.OperationCategory{Id: c.ID, Kind: c.Kind, Label: c.Label, AccountId: c.AccountID, AccountCode: c.AccountCode, AccountLabel: c.AccountLabel, Nature: c.Nature, Active: c.Active}
}

func (s *Server) ListOperationCategories(ctx context.Context, r *pb.ListOperationCategoriesRequest) (*pb.ListOperationCategoriesResponse, error) {
	v, err := s.Svc.ListOpCategories(ctx, r.Kind, r.Active)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListOperationCategoriesResponse{Items: []*pb.OperationCategory{}}
	for _, c := range v {
		out.Items = append(out.Items, catTo(c))
	}
	return out, nil
}

func (s *Server) CreateOperationCategory(ctx context.Context, r *pb.CreateOperationCategoryRequest) (*pb.CreateOperationCategoryResponse, error) {
	in := r.GetCategory()
	v, err := s.Svc.CreateOpCategory(ctx, actor(ctx), domain.OpCategory{Kind: in.GetKind(), Label: in.GetLabel(), AccountID: in.GetAccountId(), Nature: in.Nature})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateOperationCategoryResponse{Category: catTo(v)}, nil
}

func (s *Server) UpdateOperationCategory(ctx context.Context, r *pb.UpdateOperationCategoryRequest) (*pb.UpdateOperationCategoryResponse, error) {
	in := r.GetCategory()
	v, err := s.Svc.UpdateOpCategory(ctx, actor(ctx), in.GetId(), domain.OpCategory{Label: in.GetLabel(), AccountID: in.GetAccountId(), Nature: in.Nature}, in.GetActive())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateOperationCategoryResponse{Category: catTo(v)}, nil
}

// ── opérations ──

func opTo(o *domain.Operation) *pb.Operation {
	out := &pb.Operation{Id: o.ID, Kind: o.Kind, OperationDate: date(o.OperationDate), DueDate: optT(o.DueDate), ThirdPartyId: o.ThirdPartyID, ThirdPartyName: o.ThirdPartyName,
		CategoryId: o.CategoryID, CategoryLabel: o.CategoryLabel, Label: o.Label, Reference: o.Reference, AmountNet: o.AmountNet, TaxAmount: o.TaxAmount, TaxAccountId: o.TaxAccountID,
		AmountGross: o.Gross(), CounterpartyAccountId: o.CounterpartyAccountID, CounterpartyAccountCode: o.CounterpartyAccountCode, SettlementMode: o.SettlementMode, Nature: o.Nature,
		Currency: o.Currency, Status: o.Status, EntryId: o.EntryID, EntryNumber: o.EntryNumber, SettledAmount: o.SettledAmount, Remaining: o.Remaining(), DuplicateConfirmed: o.DuplicateConfirmed,
		RecurringTemplateId: o.RecurringTemplateID, AttachmentCount: int32(o.AttachmentCount), CreatedBy: o.CreatedBy, CreatedAt: timestamppb.New(o.CreatedAt), Payments: []*pb.OperationPayment{}}
	for _, p := range o.Payments {
		out.Payments = append(out.Payments, &pb.OperationPayment{Id: p.ID, PaymentDate: date(p.PaymentDate), Amount: p.Amount, AccountId: p.AccountID, AccountCode: p.AccountCode, EntryId: p.EntryID, EntryStatus: p.EntryStatus, EntryNumber: p.EntryNumber})
	}
	return out
}

func opFrom(o *pb.Operation) (domain.Operation, error) {
	d, err := reqDate(o.GetOperationDate(), "/operationDate")
	if err != nil {
		return domain.Operation{}, err
	}
	due, err := optDate(o.DueDate, "/dueDate")
	if err != nil {
		return domain.Operation{}, err
	}
	return domain.Operation{Kind: o.GetKind(), OperationDate: d, DueDate: due, ThirdPartyID: o.ThirdPartyId, CategoryID: o.GetCategoryId(), Label: o.GetLabel(), Reference: o.Reference,
		AmountNet: o.GetAmountNet(), TaxAmount: o.GetTaxAmount(), TaxAccountID: o.TaxAccountId, CounterpartyAccountID: o.GetCounterpartyAccountId(), SettlementMode: o.GetSettlementMode(), Nature: o.Nature}, nil
}

func (s *Server) ListOperations(ctx context.Context, r *pb.ListOperationsRequest) (*pb.ListOperationsResponse, error) {
	from, err := optDate(r.From, "/from")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	to, err := optDate(r.To, "/to")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	p, tot, cur, err := s.Svc.ListOperations(ctx, application.OperationFilter{Query: r.GetQuery(), Kind: r.Kind, Status: r.Status, CategoryID: r.CategoryId, ThirdPartyID: r.ThirdPartyId, From: from, To: to},
		int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListOperationsResponse{Items: []*pb.Operation{}, Page: pageResp(p), TotalNet: tot.Net, TotalTax: tot.Tax, TotalGross: tot.Gross, Currency: cur}
	for _, x := range p.Items {
		out.Items = append(out.Items, opTo(x))
	}
	return out, nil
}

func (s *Server) GetOperation(ctx context.Context, r *pb.GetOperationRequest) (*pb.GetOperationResponse, error) {
	v, err := s.Svc.GetOperation(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetOperationResponse{Operation: opTo(v)}, nil
}

func (s *Server) CreateOperation(ctx context.Context, r *pb.CreateOperationRequest) (*pb.CreateOperationResponse, error) {
	in, err := opFrom(r.GetOperation())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.CreateOperation(ctx, actor(ctx), in, r.GetDuplicateConfirmed())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateOperationResponse{Operation: opTo(v)}, nil
}

func (s *Server) UpdateOperation(ctx context.Context, r *pb.UpdateOperationRequest) (*pb.UpdateOperationResponse, error) {
	in, err := opFrom(r.GetOperation())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.UpdateOperation(ctx, actor(ctx), r.GetOperation().GetId(), in, r.GetDuplicateConfirmed())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateOperationResponse{Operation: opTo(v)}, nil
}

func (s *Server) DeleteOperation(ctx context.Context, r *pb.DeleteOperationRequest) (*pb.DeleteOperationResponse, error) {
	if err := s.Svc.DeleteOperation(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteOperationResponse{}, nil
}

func (s *Server) GenerateOperationEntry(ctx context.Context, r *pb.GenerateOperationEntryRequest) (*pb.GenerateOperationEntryResponse, error) {
	v, err := s.Svc.GenerateOperationEntry(ctx, actor(ctx), r.GetId(), r.GetJournalId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GenerateOperationEntryResponse{Operation: opTo(v)}, nil
}

func (s *Server) CancelOperationEntry(ctx context.Context, r *pb.CancelOperationEntryRequest) (*pb.CancelOperationEntryResponse, error) {
	v, err := s.Svc.CancelOperationEntry(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CancelOperationEntryResponse{Operation: opTo(v)}, nil
}

func (s *Server) RecordOperationPayment(ctx context.Context, r *pb.RecordOperationPaymentRequest) (*pb.RecordOperationPaymentResponse, error) {
	d, err := reqDate(r.GetPaymentDate(), "/paymentDate")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.RecordOperationPayment(ctx, actor(ctx), r.GetId(), d, r.GetAmount(), r.GetAccountId(), r.GetJournalId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.RecordOperationPaymentResponse{Operation: opTo(v)}, nil
}

// ── récurrences ──

func recTo(r *domain.Recurring) *pb.RecurringTemplate {
	return &pb.RecurringTemplate{Id: r.ID, Kind: r.Kind, CategoryId: r.CategoryID, CategoryLabel: r.CategoryLabel, ThirdPartyId: r.ThirdPartyID, Label: r.Label, AmountNet: r.AmountNet, TaxAmount: r.TaxAmount,
		TaxAccountId: r.TaxAccountID, CounterpartyAccountId: r.CounterpartyAccountID, JournalId: r.JournalID, SettlementMode: r.SettlementMode, Nature: r.Nature, DayOfMonth: int32(r.DayOfMonth),
		StartDate: date(r.Start), EndDate: optT(r.End), Active: r.Active, LastGeneratedMonth: optT(r.LastGeneratedMonth)}
}

func recFrom(t *pb.RecurringTemplate) (domain.Recurring, error) {
	st, err := reqDate(t.GetStartDate(), "/startDate")
	if err != nil {
		return domain.Recurring{}, err
	}
	en, err := optDate(t.EndDate, "/endDate")
	if err != nil {
		return domain.Recurring{}, err
	}
	return domain.Recurring{Kind: t.GetKind(), CategoryID: t.GetCategoryId(), ThirdPartyID: t.ThirdPartyId, Label: t.GetLabel(), AmountNet: t.GetAmountNet(), TaxAmount: t.GetTaxAmount(), TaxAccountID: t.TaxAccountId,
		CounterpartyAccountID: t.GetCounterpartyAccountId(), JournalID: t.GetJournalId(), SettlementMode: t.GetSettlementMode(), Nature: t.Nature, DayOfMonth: int(t.GetDayOfMonth()), Start: st, End: en}, nil
}

func (s *Server) ListRecurringTemplates(ctx context.Context, _ *pb.ListRecurringTemplatesRequest) (*pb.ListRecurringTemplatesResponse, error) {
	v, err := s.Svc.ListRecurring(ctx)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListRecurringTemplatesResponse{Items: []*pb.RecurringTemplate{}}
	for _, x := range v {
		out.Items = append(out.Items, recTo(x))
	}
	return out, nil
}

func (s *Server) CreateRecurringTemplate(ctx context.Context, r *pb.CreateRecurringTemplateRequest) (*pb.CreateRecurringTemplateResponse, error) {
	in, err := recFrom(r.GetTemplate())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.CreateRecurring(ctx, actor(ctx), in)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateRecurringTemplateResponse{Template: recTo(v)}, nil
}

func (s *Server) UpdateRecurringTemplate(ctx context.Context, r *pb.UpdateRecurringTemplateRequest) (*pb.UpdateRecurringTemplateResponse, error) {
	in, err := recFrom(r.GetTemplate())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	v, err := s.Svc.UpdateRecurring(ctx, actor(ctx), r.GetTemplate().GetId(), in, r.GetTemplate().GetActive())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateRecurringTemplateResponse{Template: recTo(v)}, nil
}

func (s *Server) GenerateRecurring(ctx context.Context, r *pb.GenerateRecurringRequest) (*pb.GenerateRecurringResponse, error) {
	up, err := reqDate(r.GetUpTo(), "/upTo")
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	c, k, err := s.Svc.GenerateRecurring(ctx, actor(ctx), up)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GenerateRecurringResponse{Created: int32(c), Skipped: int32(k)}, nil
}

// ── pièces ──

func attTo(a *domain.Attachment) *pb.Attachment {
	return &pb.Attachment{Id: a.ID, OperationId: a.OperationID, Filename: a.Filename, ContentType: a.ContentType, Size: a.Size, Sha256: a.SHA256, CreatedAt: timestamppb.New(a.CreatedAt)}
}

func (s *Server) ListAttachments(ctx context.Context, r *pb.ListAttachmentsRequest) (*pb.ListAttachmentsResponse, error) {
	v, err := s.Svc.ListAttachments(ctx, r.GetOperationId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListAttachmentsResponse{Items: []*pb.Attachment{}}
	for _, a := range v {
		out.Items = append(out.Items, attTo(a))
	}
	return out, nil
}

func (s *Server) UploadAttachment(ctx context.Context, r *pb.UploadAttachmentRequest) (*pb.UploadAttachmentResponse, error) {
	v, err := s.Svc.UploadAttachment(ctx, actor(ctx), r.GetOperationId(), r.GetFilename(), r.GetData())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UploadAttachmentResponse{Attachment: attTo(v)}, nil
}

func (s *Server) GetAttachment(ctx context.Context, r *pb.GetAttachmentRequest) (*pb.GetAttachmentResponse, error) {
	a, d, err := s.Svc.GetAttachment(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetAttachmentResponse{Attachment: attTo(a), Data: d}, nil
}

func (s *Server) DeleteAttachment(ctx context.Context, r *pb.DeleteAttachmentRequest) (*pb.DeleteAttachmentResponse, error) {
	if err := s.Svc.DeleteAttachment(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteAttachmentResponse{}, nil
}
