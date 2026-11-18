package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) UpdateUser(ctx context.Context, r *identityv1.UpdateUserRequest) (*identityv1.UpdateUserResponse, error) {
	u, err := s.Svc.UpdateProfile(ctx, r.GetId(), domain.ProfilePatch{
		FirstName: r.FirstName, LastName: r.LastName, Phone: r.PhoneE164, Locale: r.Locale, Currency: r.PreferredCurrency,
	})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.UpdateUserResponse{User: userToProto(u)}, nil
}

func (s *Server) RequestEmailChange(ctx context.Context, r *identityv1.RequestEmailChangeRequest) (*identityv1.RequestEmailChangeResponse, error) {
	if err := s.Svc.RequestEmailChange(ctx, r.GetUserId(), r.GetNewEmail(), r.GetPassword()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.RequestEmailChangeResponse{Status: "accepted"}, nil
}

func (s *Server) ConfirmEmailChange(ctx context.Context, r *identityv1.ConfirmEmailChangeRequest) (*identityv1.ConfirmEmailChangeResponse, error) {
	return &identityv1.ConfirmEmailChangeResponse{}, apperr.ToGRPC(s.Svc.ConfirmEmailChange(ctx, r.GetToken()))
}

func (s *Server) GetCompany(ctx context.Context, r *identityv1.GetCompanyRequest) (*identityv1.GetCompanyResponse, error) {
	c, err := s.Svc.MyCompany(ctx, r.GetUserId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetCompanyResponse{Company: companyToProto(c)}, nil
}

func (s *Server) PutCompany(ctx context.Context, r *identityv1.PutCompanyRequest) (*identityv1.PutCompanyResponse, error) {
	in := r.GetCompany()
	c, err := s.Svc.PutMyCompany(ctx, r.GetUserId(), domain.CompanyData{LegalName: in.GetLegalName(), TaxID: in.GetTaxId(), Country: in.GetCountry(), JobTitle: in.GetJobTitle()})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.PutCompanyResponse{Company: companyToProto(c)}, nil
}

func consentToProto(c *domain.ConsentRecord) *identityv1.ConsentRecord {
	return &identityv1.ConsentRecord{
		Id: c.ID, PolicyVersion: c.PolicyVersion, RecordedAt: timestamppb.New(c.RecordedAt),
		Purposes: &identityv1.ConsentPurposes{Necessary: c.Purposes.Necessary, Preferences: c.Purposes.Preferences, Functional: c.Purposes.Functional, Analytics: c.Purposes.Analytics, Marketing: c.Purposes.Marketing},
	}
}

func (s *Server) RecordConsent(ctx context.Context, r *identityv1.RecordConsentRequest) (*identityv1.RecordConsentResponse, error) {
	p := r.GetPurposes()
	c, err := s.Svc.RecordConsent(ctx, application.ConsentInput{
		UserID: r.GetUserId(), AnonymousID: r.GetAnonymousId(), PolicyVersion: r.GetPolicyVersion(), Locale: r.GetLocale(), ClientIP: r.GetClientIp(),
		Purposes: domain.ConsentPurposes{Necessary: p.GetNecessary(), Preferences: p.GetPreferences(), Functional: p.GetFunctional(), Analytics: p.GetAnalytics(), Marketing: p.GetMarketing()},
	})
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.RecordConsentResponse{Record: consentToProto(c)}, nil
}

func (s *Server) GetConsents(ctx context.Context, r *identityv1.GetConsentsRequest) (*identityv1.GetConsentsResponse, error) {
	c, err := s.Svc.MyConsent(ctx, r.GetUserId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.GetConsentsResponse{Record: consentToProto(c)}, nil
}

func (s *Server) CheckPermission(ctx context.Context, r *identityv1.CheckPermissionRequest) (*identityv1.CheckPermissionResponse, error) {
	ok, err := s.Svc.CheckPermission(ctx, r.GetStaffId(), r.GetPermission())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &identityv1.CheckPermissionResponse{Allowed: ok}, nil
}
