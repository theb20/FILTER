package server

import (
	"context"

	"github.com/google/uuid"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

func toCompany(c *identityv1.Company) api.Company {
	id, _ := uuid.Parse(c.GetId())
	out := api.Company{ID: id, LegalName: c.GetLegalName(), TaxId: c.GetTaxId(), Country: api.CountryCode(c.GetCountry()), TaxIdVerified: c.GetTaxIdVerified()}
	switch c.GetStatus() {
	case identityv1.CompanyStatus_COMPANY_STATUS_APPROVED:
		out.Status = api.CompanyStatusApproved
	case identityv1.CompanyStatus_COMPANY_STATUS_REJECTED:
		out.Status = api.CompanyStatusRejected
	default:
		out.Status = api.CompanyStatusPending
	}
	if c.RejectionReason != nil {
		out.RejectionReason = api.NewNilString(c.GetRejectionReason())
	} else {
		out.RejectionReason.SetToNull()
	}
	if c.DecidedAt != nil {
		out.DecidedAt = api.NewNilDateTime(c.GetDecidedAt().AsTime())
	} else {
		out.DecidedAt.SetToNull()
	}
	return out
}

// toCustomerAdmin : le nombre de commandes et la valeur vie client viendront du service des commandes (absent pour l'instant : 0 et null).
func toCustomerAdmin(c *identityv1.Customer) api.CustomerAdmin {
	id, _ := uuid.Parse(c.GetId())
	out := api.CustomerAdmin{ID: id, Email: c.GetEmail(), CreatedAt: c.GetCreatedAt().AsTime(), Kind: api.CustomerAdminKindIndividual, Status: api.CustomerAdminStatus(c.GetStatus())}
	if c.GetKind() == identityv1.UserKind_USER_KIND_COMPANY {
		out.Kind = api.CustomerAdminKindCompany
	}
	for _, f := range []struct {
		dst *api.NilString
		src *string
	}{{&out.FirstName, c.FirstName}, {&out.LastName, c.LastName}} {
		if f.src != nil {
			*f.dst = api.NewNilString(*f.src)
		} else {
			f.dst.SetToNull()
		}
	}
	out.Phone.SetToNull()
	out.LifetimeValue.SetToNull()
	if c.Company != nil {
		out.Company = api.NewNilCompany(toCompany(c.Company))
	} else {
		out.Company.SetToNull()
	}
	return out
}

func (h *handler) AdminCustomersList(ctx context.Context, p api.AdminCustomersListParams) (api.AdminCustomersListRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &identityv1.ListCustomersRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")}
	if k, ok := p.Kind.Get(); ok {
		kind := identityv1.UserKind_USER_KIND_INDIVIDUAL
		if k == api.AdminCustomersListKindCompany {
			kind = identityv1.UserKind_USER_KIND_COMPANY
		}
		req.Kind = &kind
	}
	if st, ok := p.Status.Get(); ok {
		v := string(st)
		req.Status = &v
	}
	res, err := h.id.ListCustomers(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.CustomerAdminPage{Items: make([]api.CustomerAdmin, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, c := range res.GetItems() {
		out.Items = append(out.Items, toCustomerAdmin(c))
	}
	return out, nil
}

func (h *handler) AdminCustomersGet(ctx context.Context, p api.AdminCustomersGetParams) (api.AdminCustomersGetRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.GetCustomer(ctx, &identityv1.GetCustomerRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	c := toCustomerAdmin(res.GetCustomer())
	return &c, nil
}

func (h *handler) AdminCustomersUpdate(ctx context.Context, req *api.CustomerStatusUpdate, p api.AdminCustomersUpdateParams) (api.AdminCustomersUpdateRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.SetCustomerStatus(ctx, &identityv1.SetCustomerStatusRequest{Id: p.ID.String(), Status: string(req.Status)})
	if err != nil {
		return nil, err
	}
	c := toCustomerAdmin(res.GetCustomer())
	return &c, nil
}

func (h *handler) AdminCompaniesList(ctx context.Context, p api.AdminCompaniesListParams) (api.AdminCompaniesListRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &identityv1.ListCompaniesRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")}
	if st, ok := p.Status.Get(); ok {
		v := identityv1.CompanyStatus_COMPANY_STATUS_PENDING
		switch st {
		case api.AdminCompaniesListStatusApproved:
			v = identityv1.CompanyStatus_COMPANY_STATUS_APPROVED
		case api.AdminCompaniesListStatusRejected:
			v = identityv1.CompanyStatus_COMPANY_STATUS_REJECTED
		}
		req.Status = &v
	}
	res, err := h.id.ListCompanies(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.CompanyPage{Items: make([]api.Company, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, c := range res.GetItems() {
		out.Items = append(out.Items, toCompany(c))
	}
	return out, nil
}

func (h *handler) AdminCompaniesGet(ctx context.Context, p api.AdminCompaniesGetParams) (api.AdminCompaniesGetRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.GetCompanyById(ctx, &identityv1.GetCompanyByIdRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	c := toCompany(res.GetCompany())
	return &c, nil
}

func (h *handler) AdminCompaniesApprove(ctx context.Context, p api.AdminCompaniesApproveParams) (api.AdminCompaniesApproveRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.ApproveCompany(ctx, &identityv1.ApproveCompanyRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	c := toCompany(res.GetCompany())
	return &c, nil
}

func (h *handler) AdminCompaniesReject(ctx context.Context, req *api.RejectCompanyRequest, p api.AdminCompaniesRejectParams) (api.AdminCompaniesRejectRes, error) {
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := h.id.RejectCompany(ctx, &identityv1.RejectCompanyRequest{Id: p.ID.String(), Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	c := toCompany(res.GetCompany())
	return &c, nil
}
