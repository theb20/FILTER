package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func companyStatusToProto(s string) identityv1.CompanyStatus {
	switch s {
	case domain.CompanyApproved:
		return identityv1.CompanyStatus_COMPANY_STATUS_APPROVED
	case domain.CompanyRejected:
		return identityv1.CompanyStatus_COMPANY_STATUS_REJECTED
	}
	return identityv1.CompanyStatus_COMPANY_STATUS_PENDING
}

func companyStatusFromProto(s *identityv1.CompanyStatus) string {
	if s == nil {
		return ""
	}
	switch *s {
	case identityv1.CompanyStatus_COMPANY_STATUS_PENDING:
		return domain.CompanyPending
	case identityv1.CompanyStatus_COMPANY_STATUS_APPROVED:
		return domain.CompanyApproved
	case identityv1.CompanyStatus_COMPANY_STATUS_REJECTED:
		return domain.CompanyRejected
	}
	return ""
}

func companyToProto(c *domain.Company) *identityv1.Company {
	out := &identityv1.Company{Id: c.ID, LegalName: c.LegalName, TaxId: c.TaxID, Country: c.Country, Status: companyStatusToProto(c.State())}
	if c.RejectionReason != "" {
		out.RejectionReason = &c.RejectionReason
	}
	if c.DecidedAt != nil {
		out.DecidedAt = timestamppb.New(*c.DecidedAt)
	}
	return out
}

func customerToProto(u *domain.User) *identityv1.Customer {
	out := &identityv1.Customer{Id: u.ID, Email: u.Email, Kind: kindToProto(u.Kind), Status: u.Status, CreatedAt: timestamppb.New(u.CreatedAt)}
	if u.FirstName != "" {
		out.FirstName = &u.FirstName
	}
	if u.LastName != "" {
		out.LastName = &u.LastName
	}
	if u.Company != nil {
		out.Company = companyToProto(u.Company)
	}
	return out
}

func (s *Server) ListCustomers(ctx context.Context, r *identityv1.ListCustomersRequest) (*identityv1.ListCustomersResponse, error) {
	kind := ""
	if r.Kind != nil {
		kind = domain.KindIndividual
		if r.GetKind() == identityv1.UserKind_USER_KIND_COMPANY {
			kind = domain.KindCompany
		}
	}
	p, err := s.Svc.ListCustomers(ctx, r.GetQuery(), kind, r.GetStatus(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &identityv1.ListCustomersResponse{Page: pageResp(p), Items: []*identityv1.Customer{}}
	for _, u := range p.Items {
		out.Items = append(out.Items, customerToProto(u))
	}
	return out, nil
}

func (s *Server) GetCustomer(ctx context.Context, r *identityv1.GetCustomerRequest) (*identityv1.GetCustomerResponse, error) {
	u, err := s.Svc.GetCustomer(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetCustomerResponse{Customer: customerToProto(u)}, nil
}

func (s *Server) SetCustomerStatus(ctx context.Context, r *identityv1.SetCustomerStatusRequest) (*identityv1.SetCustomerStatusResponse, error) {
	u, err := s.Svc.SetCustomerStatus(ctx, actor(ctx), r.GetId(), r.GetStatus())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.SetCustomerStatusResponse{Customer: customerToProto(u)}, nil
}

func (s *Server) ListCompanies(ctx context.Context, r *identityv1.ListCompaniesRequest) (*identityv1.ListCompaniesResponse, error) {
	p, err := s.Svc.ListCompanies(ctx, r.GetQuery(), companyStatusFromProto(r.Status), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &identityv1.ListCompaniesResponse{Page: pageResp(p), Items: []*identityv1.Company{}}
	for _, c := range p.Items {
		out.Items = append(out.Items, companyToProto(c))
	}
	return out, nil
}

func (s *Server) GetCompanyById(ctx context.Context, r *identityv1.GetCompanyByIdRequest) (*identityv1.GetCompanyByIdResponse, error) {
	c, err := s.Svc.GetCompany(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetCompanyByIdResponse{Company: companyToProto(c)}, nil
}

func (s *Server) ApproveCompany(ctx context.Context, r *identityv1.ApproveCompanyRequest) (*identityv1.ApproveCompanyResponse, error) {
	c, err := s.Svc.ApproveCompany(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.ApproveCompanyResponse{Company: companyToProto(c)}, nil
}

func (s *Server) RejectCompany(ctx context.Context, r *identityv1.RejectCompanyRequest) (*identityv1.RejectCompanyResponse, error) {
	c, err := s.Svc.RejectCompany(ctx, actor(ctx), r.GetId(), r.GetReason())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.RejectCompanyResponse{Company: companyToProto(c)}, nil
}

func (s *Server) CountPendingCompanies(ctx context.Context, _ *identityv1.CountPendingCompaniesRequest) (*identityv1.CountPendingCompaniesResponse, error) {
	n, err := s.Svc.CountPendingCompanies(ctx)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.CountPendingCompaniesResponse{Count: int64(n)}, nil
}
