package server_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	identityv1 "github.com/theb20/FILTER/backend/pkg/gen/identity/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (f *fakeID) ListCustomers(_ context.Context, in *identityv1.ListCustomersRequest, _ ...grpc.CallOption) (*identityv1.ListCustomersResponse, error) {
	adminCalls["ListCustomers"]++
	first := "Awa"
	return &identityv1.ListCustomersResponse{
		Items: []*identityv1.Customer{{Id: uuid.NewString(), Email: "awa@exemple.ci", Kind: identityv1.UserKind_USER_KIND_COMPANY, FirstName: &first, Status: "active", CreatedAt: timestamppb.Now(),
			Company: &identityv1.Company{Id: uuid.NewString(), LegalName: "Pièces Plus", TaxId: "CI-1", Country: "CI", Status: identityv1.CompanyStatus_COMPANY_STATUS_PENDING}}},
		Page: &commonv1.PageResponse{Page: 1, PageSize: 20, Total: 1, TotalPages: 1},
	}, nil
}

func (f *fakeID) ApproveCompany(_ context.Context, in *identityv1.ApproveCompanyRequest, _ ...grpc.CallOption) (*identityv1.ApproveCompanyResponse, error) {
	adminCalls["ApproveCompany"]++
	return &identityv1.ApproveCompanyResponse{Company: &identityv1.Company{Id: in.GetId(), LegalName: "X", TaxId: "1", Country: "CI", Status: identityv1.CompanyStatus_COMPANY_STATUS_APPROVED, DecidedAt: timestamppb.Now()}}, nil
}

func TestClientsEtEntreprisesDuPersonnel(t *testing.T) {
	r := newRig(t, nil)
	sub := uuid.NewString()

	if rec := r.do(req{"GET", "/v1/admin/customers", "", bearer(staffToken(t, r, sub, "orders:read"))}); rec.Code != 403 {
		t.Fatalf("sans customers:read : %d", rec.Code)
	}
	rec := r.do(req{"GET", "/v1/admin/customers?kind=company", "", bearer(staffToken(t, r, sub, "customers:read"))})
	if rec.Code != 200 {
		t.Fatalf("liste : %d %s", rec.Code, rec.Body.String())
	}
	var page struct {
		Items []struct {
			Kind    string          `json:"kind"`
			Company json.RawMessage `json:"company"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || len(page.Items) != 1 || page.Items[0].Kind != "company" || string(page.Items[0].Company) == "null" {
		t.Fatalf("corps : %v %s", err, rec.Body.String())
	}

	// l'approbation exige companies:approve, pas customers:write
	id := uuid.NewString()
	before := adminCalls["ApproveCompany"]
	if rec := r.do(req{"POST", "/v1/admin/companies/" + id + "/approve", "", bearer(staffToken(t, r, sub, "customers:write"))}); rec.Code != 403 {
		t.Fatalf("approbation sans permission : %d", rec.Code)
	}
	if adminCalls["ApproveCompany"] != before {
		t.Fatal("identity appelé malgré le refus")
	}
	if rec := r.do(req{"POST", "/v1/admin/companies/" + id + "/approve", "", bearer(staffToken(t, r, sub, "companies:approve"))}); rec.Code != 200 {
		t.Fatalf("approbation : %d %s", rec.Code, rec.Body.String())
	}
}
