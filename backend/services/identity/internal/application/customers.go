package application

import (
	"context"
	"errors"
	"strings"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
)

func (s *Service) ListCustomers(ctx context.Context, query, kind, status string, page, size int) (*Page[*domain.User], error) {
	if kind != "" && kind != domain.KindIndividual && kind != domain.KindCompany {
		return nil, invalid("/kind", "type de compte inconnu")
	}
	if status != "" && status != domain.StatusActive && status != domain.StatusSuspended {
		return nil, invalid("/status", "statut inconnu")
	}
	limit, offset, p, sz := paginate(page, size)
	items, total, err := s.Customers.ListCustomers(ctx, strings.TrimSpace(query), kind, status, limit, offset)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &Page[*domain.User]{Items: items, Page: p, PageSize: sz, Total: total, Pages: pages(total, sz)}, nil
}

func (s *Service) GetCustomer(ctx context.Context, id string) (*domain.User, error) {
	u, err := s.Users.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperr.NotFound("client introuvable")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return u, nil
}

func customerSnapshot(u *domain.User) map[string]any {
	return map[string]any{"email": u.Email, "kind": u.Kind, "status": u.Status}
}

// SetCustomerStatus suspend ou réactive un client. Une suspension coupe toutes ses sessions.
func (s *Service) SetCustomerStatus(ctx context.Context, a Actor, id, status string) (*domain.User, error) {
	if status != domain.StatusActive && status != domain.StatusSuspended {
		return nil, invalid("/status", "le statut doit être « active » ou « suspended »")
	}
	var out *domain.User
	err := s.tx(ctx, func(ctx context.Context) error {
		u, err := s.GetCustomer(ctx, id)
		if err != nil {
			return err
		}
		out = u
		if u.Status == status {
			return nil
		}
		before := customerSnapshot(u)
		u.Status = status
		if err := s.Users.Update(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		if status == domain.StatusSuspended {
			if err := s.Sessions.RevokeAllExcept(ctx, u.ID, "", s.Now()); err != nil {
				return apperr.Internal(err)
			}
		}
		action := "customer.suspend"
		if status == domain.StatusActive {
			action = "customer.reactivate"
		}
		return s.audit(ctx, a, action, "customer", u.ID, before, customerSnapshot(u))
	})
	return out, err
}

// ── entreprises ─────────────────────────────────────────────────────────────

func (s *Service) ListCompanies(ctx context.Context, query, status string, page, size int) (*Page[*domain.Company], error) {
	switch status {
	case "", domain.CompanyPending, domain.CompanyApproved, domain.CompanyRejected:
	default:
		return nil, invalid("/status", "statut inconnu")
	}
	limit, offset, p, sz := paginate(page, size)
	items, total, err := s.Customers.ListCompanies(ctx, strings.TrimSpace(query), status, limit, offset)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &Page[*domain.Company]{Items: items, Page: p, PageSize: sz, Total: total, Pages: pages(total, sz)}, nil
}

func (s *Service) ownerOfCompany(ctx context.Context, id string) (*domain.User, error) {
	u, err := s.Customers.UserByCompanyID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperr.NotFound("entreprise introuvable")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return u, nil
}

func (s *Service) GetCompany(ctx context.Context, id string) (*domain.Company, error) {
	u, err := s.ownerOfCompany(ctx, id)
	if err != nil {
		return nil, err
	}
	return u.Company, nil
}

func (s *Service) CountPendingCompanies(ctx context.Context) (int, error) {
	n, err := s.Customers.CountCompanies(ctx, domain.CompanyPending)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}

func companySnapshot(c *domain.Company) map[string]any {
	m := map[string]any{"legalName": c.LegalName, "taxId": c.TaxID, "country": c.Country, "status": c.State()}
	if c.RejectionReason != "" {
		m["rejectionReason"] = c.RejectionReason
	}
	return m
}

// decideCompany applique une décision sur une entreprise EN ATTENTE. Action sensible : la permission est revérifiée en base.
func (s *Service) decideCompany(ctx context.Context, a Actor, id, reason string, approve bool) (*domain.Company, error) {
	var out *domain.Company
	err := s.tx(ctx, func(ctx context.Context) error {
		perms, err := s.actorPerms(ctx, a)
		if err != nil {
			return err
		}
		if !domain.HasPermission(perms, "companies:approve") {
			return apperr.PermissionDenied(apperr.CodeForbidden, "action non autorisée")
		}
		u, err := s.ownerOfCompany(ctx, id)
		if err != nil {
			return err
		}
		c := u.Company
		if c.State() != domain.CompanyPending {
			return conflict("cette entreprise a déjà fait l'objet d'une décision")
		}
		if approve && !u.EmailVerified {
			return apperr.New(codes.FailedPrecondition, apperr.CodeEmailNotVerified, "l'adresse e-mail du titulaire n'est pas vérifiée")
		}
		before := companySnapshot(c)
		now := s.Now()
		c.DecidedAt = &now
		if approve {
			c.Status, c.Approved = domain.CompanyApproved, true
		} else {
			c.Status, c.Approved, c.RejectionReason = domain.CompanyRejected, false, reason
		}
		if err := s.Users.Update(ctx, u); err != nil {
			return apperr.Internal(err)
		}
		action := "company.reject"
		if approve {
			action = "company.approve"
		}
		out = c
		return s.audit(ctx, a, action, "company", c.ID, before, companySnapshot(c))
	})
	return out, err
}

func (s *Service) ApproveCompany(ctx context.Context, a Actor, id string) (*domain.Company, error) {
	return s.decideCompany(ctx, a, id, "", true)
}

func (s *Service) RejectCompany(ctx context.Context, a Actor, id, reason string) (*domain.Company, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || len([]rune(reason)) > 500 {
		return nil, invalid("/reason", "le motif est obligatoire (500 caractères au maximum)")
	}
	return s.decideCompany(ctx, a, id, reason, false)
}
