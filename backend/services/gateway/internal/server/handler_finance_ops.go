package server

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/financeadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/grpc/codes"
)

// Finance F2 : tiers, catégories, dépenses et revenus, règlements, récurrences, pièces justificatives.

func nilDate(s *string) api.OptNilDate {
	var o api.OptNilDate
	if s == nil {
		o.SetToNull()
		return o
	}
	return api.NewOptNilDate(parseDate(*s))
}

func datePtrNil(o api.OptNilDate) *string {
	if v, ok := o.Get(); ok {
		s := dateStr(v)
		return &s
	}
	return nil
}

func natPtr[T ~string](v T, ok bool) *string {
	if !ok {
		return nil
	}
	s := string(v)
	return &s
}

// ── tiers ──

func toThirdParty(t *pb.ThirdParty) api.FinanceThirdParty {
	return api.FinanceThirdParty{ID: uuidOf(t.GetId()), Kind: api.FinanceThirdPartyKind(t.GetKind()), ExternalRef: nilStr(t.ExternalRef), DisplayName: t.GetDisplayName(), Active: t.GetActive()}
}

func (h *handler) AdminFinanceThirdPartiesList(ctx context.Context, p api.AdminFinanceThirdPartiesListParams) (api.AdminFinanceThirdPartiesListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListThirdPartiesRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")}
	if v, ok := p.Kind.Get(); ok {
		k := string(v)
		req.Kind = &k
	}
	res, err := c.ListThirdParties(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.ThirdPartiesPage{Items: make([]api.FinanceThirdParty, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toThirdParty(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceThirdPartiesCreate(ctx context.Context, req *api.ThirdPartyInput) (api.AdminFinanceThirdPartiesCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateThirdParty(ctx, &pb.CreateThirdPartyRequest{ThirdParty: &pb.ThirdParty{Kind: string(req.Kind), ExternalRef: nilStrIn(req.ExternalRef), DisplayName: req.DisplayName}})
	if err != nil {
		return nil, err
	}
	out := toThirdParty(res.GetThirdParty())
	return &out, nil
}

func (h *handler) AdminFinanceThirdPartiesUpdate(ctx context.Context, req *api.ThirdPartyInput, p api.AdminFinanceThirdPartiesUpdateParams) (api.AdminFinanceThirdPartiesUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateThirdParty(ctx, &pb.UpdateThirdPartyRequest{ThirdParty: &pb.ThirdParty{Id: p.ID.String(), ExternalRef: nilStrIn(req.ExternalRef), DisplayName: req.DisplayName, Active: req.Active.Or(true)}})
	if err != nil {
		return nil, err
	}
	out := toThirdParty(res.GetThirdParty())
	return &out, nil
}

// ── catégories ──

func toOpCategory(c *pb.OperationCategory) api.FinanceOperationCategory {
	out := api.FinanceOperationCategory{ID: uuidOf(c.GetId()), Kind: api.FinanceOperationCategoryKind(c.GetKind()), Label: c.GetLabel(), AccountId: uuidOf(c.GetAccountId()), AccountCode: c.GetAccountCode(), AccountLabel: c.GetAccountLabel(), Active: c.GetActive()}
	if c.Nature != nil {
		out.Nature = api.NewOptNilFinanceOperationCategoryNature(api.FinanceOperationCategoryNature(c.GetNature()))
	} else {
		out.Nature.SetToNull()
	}
	return out
}

func (h *handler) AdminFinanceCategoriesList(ctx context.Context, p api.AdminFinanceCategoriesListParams) (api.AdminFinanceCategoriesListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListOperationCategoriesRequest{}
	if v, ok := p.Kind.Get(); ok {
		k := string(v)
		req.Kind = &k
	}
	if v, ok := p.Active.Get(); ok {
		req.Active = &v
	}
	res, err := c.ListOperationCategories(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.OperationCategoriesList{Items: make([]api.FinanceOperationCategory, 0, len(res.GetItems()))}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toOpCategory(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceCategoriesCreate(ctx context.Context, req *api.OperationCategoryInput) (api.AdminFinanceCategoriesCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	v, ok := req.Nature.Get()
	res, err := c.CreateOperationCategory(ctx, &pb.CreateOperationCategoryRequest{Category: &pb.OperationCategory{Kind: string(req.Kind), Label: req.Label, AccountId: req.AccountId.String(), Nature: natPtr(v, ok)}})
	if err != nil {
		return nil, err
	}
	out := toOpCategory(res.GetCategory())
	return &out, nil
}

func (h *handler) AdminFinanceCategoriesUpdate(ctx context.Context, req *api.OperationCategoryInput, p api.AdminFinanceCategoriesUpdateParams) (api.AdminFinanceCategoriesUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	v, ok := req.Nature.Get()
	res, err := c.UpdateOperationCategory(ctx, &pb.UpdateOperationCategoryRequest{Category: &pb.OperationCategory{Id: p.ID.String(), Label: req.Label, AccountId: req.AccountId.String(), Nature: natPtr(v, ok), Active: req.Active.Or(true)}})
	if err != nil {
		return nil, err
	}
	out := toOpCategory(res.GetCategory())
	return &out, nil
}

// ── opérations ──

func toOperation(o *pb.Operation) api.FinanceOperation {
	out := api.FinanceOperation{ID: uuidOf(o.GetId()), Kind: api.FinanceOperationKind(o.GetKind()), OperationDate: parseDate(o.GetOperationDate()), DueDate: nilDate(o.DueDate), ThirdPartyId: nilUUIDStr(o.ThirdPartyId),
		ThirdPartyName: nilStr(o.ThirdPartyName), CategoryId: uuidOf(o.GetCategoryId()), CategoryLabel: o.GetCategoryLabel(), Label: o.GetLabel(), Reference: nilStr(o.Reference), AmountNet: o.GetAmountNet(),
		TaxAmount: o.GetTaxAmount(), TaxAccountId: nilUUIDStr(o.TaxAccountId), AmountGross: o.GetAmountGross(), CounterpartyAccountId: uuidOf(o.GetCounterpartyAccountId()),
		CounterpartyAccountCode: o.GetCounterpartyAccountCode(), SettlementMode: api.FinanceOperationSettlementMode(o.GetSettlementMode()), Currency: o.GetCurrency(), Status: api.FinanceOperationStatus(o.GetStatus()),
		EntryId: nilUUIDStr(o.EntryId), EntryNumber: nilStr(o.EntryNumber), SettledAmount: o.GetSettledAmount(), Remaining: o.GetRemaining(), DuplicateConfirmed: o.GetDuplicateConfirmed(),
		RecurringTemplateId: nilUUIDStr(o.RecurringTemplateId), AttachmentCount: o.GetAttachmentCount(), CreatedBy: uuidOf(o.GetCreatedBy()), CreatedAt: o.GetCreatedAt().AsTime(), Payments: []api.OperationPayment{}}
	if o.Nature != nil {
		out.Nature = api.NewOptNilFinanceOperationNature(api.FinanceOperationNature(o.GetNature()))
	} else {
		out.Nature.SetToNull()
	}
	for _, p := range o.GetPayments() {
		out.Payments = append(out.Payments, api.OperationPayment{ID: uuidOf(p.GetId()), PaymentDate: parseDate(p.GetPaymentDate()), Amount: p.GetAmount(), AccountId: uuidOf(p.GetAccountId()), AccountCode: p.GetAccountCode(),
			EntryId: uuidOf(p.GetEntryId()), EntryStatus: api.OperationPaymentEntryStatus(p.GetEntryStatus()), EntryNumber: nilStr(p.EntryNumber)})
	}
	return out
}

func operationProto(id string, in *api.OperationInput) *pb.Operation {
	v, ok := in.Nature.Get()
	return &pb.Operation{Id: id, Kind: string(in.Kind), OperationDate: dateStr(in.OperationDate), DueDate: datePtrNil(in.DueDate), ThirdPartyId: strPtr(in.ThirdPartyId), CategoryId: in.CategoryId.String(), Label: in.Label,
		Reference: nilStrIn(in.Reference), AmountNet: in.AmountNet, TaxAmount: in.TaxAmount, TaxAccountId: strPtr(in.TaxAccountId), CounterpartyAccountId: in.CounterpartyAccountId.String(),
		SettlementMode: string(in.SettlementMode), Nature: natPtr(v, ok)}
}

func (h *handler) AdminFinanceOperationsList(ctx context.Context, p api.AdminFinanceOperationsListParams) (api.AdminFinanceOperationsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListOperationsRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or(""), CategoryId: optUUIDPtr(p.CategoryId), ThirdPartyId: optUUIDPtr(p.ThirdPartyId), From: optDatePtr(p.From), To: optDatePtr(p.To)}
	if v, ok := p.Kind.Get(); ok {
		k := string(v)
		req.Kind = &k
	}
	if v, ok := p.Status.Get(); ok {
		k := string(v)
		req.Status = &k
	}
	res, err := c.ListOperations(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.OperationsPage{Items: make([]api.FinanceOperation, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage()), TotalNet: res.GetTotalNet(), TotalTax: res.GetTotalTax(), TotalGross: res.GetTotalGross(), Currency: res.GetCurrency()}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toOperation(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceOperationsCreate(ctx context.Context, req *api.OperationInput) (api.AdminFinanceOperationsCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateOperation(ctx, &pb.CreateOperationRequest{Operation: operationProto("", req), DuplicateConfirmed: req.DuplicateConfirmed.Or(false)})
	if err != nil {
		return nil, err
	}
	out := toOperation(res.GetOperation())
	return &out, nil
}

func (h *handler) AdminFinanceOperationsGet(ctx context.Context, p api.AdminFinanceOperationsGetParams) (api.AdminFinanceOperationsGetRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetOperation(ctx, &pb.GetOperationRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toOperation(res.GetOperation())
	return &out, nil
}

func (h *handler) AdminFinanceOperationsUpdate(ctx context.Context, req *api.OperationInput, p api.AdminFinanceOperationsUpdateParams) (api.AdminFinanceOperationsUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateOperation(ctx, &pb.UpdateOperationRequest{Operation: operationProto(p.ID.String(), req), DuplicateConfirmed: req.DuplicateConfirmed.Or(false)})
	if err != nil {
		return nil, err
	}
	out := toOperation(res.GetOperation())
	return &out, nil
}

func (h *handler) AdminFinanceOperationsDelete(ctx context.Context, p api.AdminFinanceOperationsDeleteParams) (api.AdminFinanceOperationsDeleteRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteOperation(ctx, &pb.DeleteOperationRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminFinanceOperationsDeleteNoContent{}, nil
}

func (h *handler) AdminFinanceOperationsGenerateEntry(ctx context.Context, req *api.OperationEntryInput, p api.AdminFinanceOperationsGenerateEntryParams) (api.AdminFinanceOperationsGenerateEntryRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GenerateOperationEntry(ctx, &pb.GenerateOperationEntryRequest{Id: p.ID.String(), JournalId: req.JournalId.String()})
	if err != nil {
		return nil, err
	}
	out := toOperation(res.GetOperation())
	return &out, nil
}

func (h *handler) AdminFinanceOperationsCancelEntry(ctx context.Context, p api.AdminFinanceOperationsCancelEntryParams) (api.AdminFinanceOperationsCancelEntryRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CancelOperationEntry(ctx, &pb.CancelOperationEntryRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := toOperation(res.GetOperation())
	return &out, nil
}

func (h *handler) AdminFinanceOperationsPay(ctx context.Context, req *api.OperationPaymentInput, p api.AdminFinanceOperationsPayParams) (api.AdminFinanceOperationsPayRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.RecordOperationPayment(ctx, &pb.RecordOperationPaymentRequest{Id: p.ID.String(), PaymentDate: dateStr(req.PaymentDate), Amount: req.Amount, AccountId: req.AccountId.String(), JournalId: req.JournalId.String()})
	if err != nil {
		return nil, err
	}
	out := toOperation(res.GetOperation())
	return &out, nil
}

// ── récurrences ──

func toRecurring(r *pb.RecurringTemplate) api.FinanceRecurring {
	out := api.FinanceRecurring{ID: uuidOf(r.GetId()), Kind: api.FinanceRecurringKind(r.GetKind()), CategoryId: uuidOf(r.GetCategoryId()), CategoryLabel: r.GetCategoryLabel(), ThirdPartyId: nilUUIDStr(r.ThirdPartyId),
		Label: r.GetLabel(), AmountNet: r.GetAmountNet(), TaxAmount: r.GetTaxAmount(), TaxAccountId: nilUUIDStr(r.TaxAccountId), CounterpartyAccountId: uuidOf(r.GetCounterpartyAccountId()), JournalId: uuidOf(r.GetJournalId()),
		SettlementMode: api.FinanceRecurringSettlementMode(r.GetSettlementMode()), DayOfMonth: r.GetDayOfMonth(), StartDate: parseDate(r.GetStartDate()), EndDate: nilDate(r.EndDate), Active: r.GetActive(), LastGeneratedMonth: nilDate(r.LastGeneratedMonth)}
	if r.Nature != nil {
		out.Nature = api.NewOptNilFinanceRecurringNature(api.FinanceRecurringNature(r.GetNature()))
	} else {
		out.Nature.SetToNull()
	}
	return out
}

func recurringProto(id string, in *api.RecurringInput) *pb.RecurringTemplate {
	v, ok := in.Nature.Get()
	return &pb.RecurringTemplate{Id: id, Kind: string(in.Kind), CategoryId: in.CategoryId.String(), ThirdPartyId: strPtr(in.ThirdPartyId), Label: in.Label, AmountNet: in.AmountNet, TaxAmount: in.TaxAmount,
		TaxAccountId: strPtr(in.TaxAccountId), CounterpartyAccountId: in.CounterpartyAccountId.String(), JournalId: in.JournalId.String(), SettlementMode: string(in.SettlementMode), Nature: natPtr(v, ok),
		DayOfMonth: in.DayOfMonth, StartDate: dateStr(in.StartDate), EndDate: datePtrNil(in.EndDate), Active: in.Active.Or(true)}
}

func (h *handler) AdminFinanceRecurringList(ctx context.Context) (api.AdminFinanceRecurringListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListRecurringTemplates(ctx, &pb.ListRecurringTemplatesRequest{})
	if err != nil {
		return nil, err
	}
	out := &api.RecurringList{Items: make([]api.FinanceRecurring, 0, len(res.GetItems()))}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toRecurring(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceRecurringCreate(ctx context.Context, req *api.RecurringInput) (api.AdminFinanceRecurringCreateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateRecurringTemplate(ctx, &pb.CreateRecurringTemplateRequest{Template: recurringProto("", req)})
	if err != nil {
		return nil, err
	}
	out := toRecurring(res.GetTemplate())
	return &out, nil
}

func (h *handler) AdminFinanceRecurringUpdate(ctx context.Context, req *api.RecurringInput, p api.AdminFinanceRecurringUpdateParams) (api.AdminFinanceRecurringUpdateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateRecurringTemplate(ctx, &pb.UpdateRecurringTemplateRequest{Template: recurringProto(p.ID.String(), req)})
	if err != nil {
		return nil, err
	}
	out := toRecurring(res.GetTemplate())
	return &out, nil
}

func (h *handler) AdminFinanceRecurringGenerate(ctx context.Context, req *api.RecurringGenerateInput) (api.AdminFinanceRecurringGenerateRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GenerateRecurring(ctx, &pb.GenerateRecurringRequest{UpTo: dateStr(req.UpTo)})
	if err != nil {
		return nil, err
	}
	return &api.RecurringGenerateResult{Created: res.GetCreated(), Skipped: res.GetSkipped()}, nil
}

// ── pièces justificatives ──

func toAttachment(a *pb.Attachment) api.FinanceAttachment {
	return api.FinanceAttachment{ID: uuidOf(a.GetId()), OperationId: uuidOf(a.GetOperationId()), Filename: a.GetFilename(), ContentType: api.FinanceAttachmentContentType(a.GetContentType()), Size: a.GetSize(), SHA256: a.GetSha256(), CreatedAt: a.GetCreatedAt().AsTime()}
}

func (h *handler) AdminFinanceAttachmentsList(ctx context.Context, p api.AdminFinanceAttachmentsListParams) (api.AdminFinanceAttachmentsListRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListAttachments(ctx, &pb.ListAttachmentsRequest{OperationId: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := &api.AttachmentsList{Items: make([]api.FinanceAttachment, 0, len(res.GetItems()))}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toAttachment(x))
	}
	return out, nil
}

func (h *handler) AdminFinanceAttachmentsUpload(ctx context.Context, req *api.AttachmentUpload, p api.AdminFinanceAttachmentsUploadParams) (api.AdminFinanceAttachmentsUploadRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(req.DataBase64)
	if err != nil {
		return nil, apperr.Validation("fichier illisible", apperr.Field{Pointer: "/dataBase64", Code: apperr.CodeValidation, Message: "contenu base64 invalide"})
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UploadAttachment(ctx, &pb.UploadAttachmentRequest{OperationId: p.ID.String(), Filename: req.Filename, Data: data})
	if err != nil {
		return nil, err
	}
	out := toAttachment(res.GetAttachment())
	return &out, nil
}

func (h *handler) AdminFinanceAttachmentsGet(ctx context.Context, p api.AdminFinanceAttachmentsGetParams) (api.AdminFinanceAttachmentsGetRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetAttachment(ctx, &pb.GetAttachmentRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	return &api.AttachmentContent{Attachment: toAttachment(res.GetAttachment()), DataBase64: base64.StdEncoding.EncodeToString(res.GetData())}, nil
}

func (h *handler) AdminFinanceAttachmentsDelete(ctx context.Context, p api.AdminFinanceAttachmentsDeleteParams) (api.AdminFinanceAttachmentsDeleteRes, error) {
	c, err := h.finance()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteAttachment(ctx, &pb.DeleteAttachmentRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminFinanceAttachmentsDeleteNoContent{}, nil
}

var (
	_ = codes.OK
	_ = time.Second
)
