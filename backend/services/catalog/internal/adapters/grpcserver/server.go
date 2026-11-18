// Package grpcserver expose le service catalog en gRPC (contrat proto/catalogadmin/v1).
package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	"github.com/theb20/FILTER/backend/services/catalog/internal/application"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedCatalogAdminServiceServer
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

func locTo(l domain.Localized) *pb.Localized {
	if l == nil {
		return nil
	}
	return &pb.Localized{Texts: map[string]string(l)}
}

func locFrom(l *pb.Localized) domain.Localized {
	if l == nil {
		return nil
	}
	return domain.Localized(l.GetTexts())
}

// ── catégories ──

func catTo(c *domain.Category) *pb.Category {
	return &pb.Category{Id: c.ID, Slug: c.Slug, ParentId: c.ParentID, SortOrder: int32(c.SortOrder), Name: locTo(c.Name), Description: locTo(c.Description), ImageUrl: c.ImageURL, Depth: int32(c.Depth), ProductCount: c.ProductCount, CreatedAt: timestamppb.New(c.CreatedAt), UpdatedAt: timestamppb.New(c.UpdatedAt)}
}

func catFrom(c *pb.Category) domain.Category {
	return domain.Category{Slug: c.GetSlug(), ParentID: c.ParentId, SortOrder: int(c.GetSortOrder()), Name: locFrom(c.GetName()), Description: locFrom(c.GetDescription()), ImageURL: c.ImageUrl}
}

func (s *Server) ListCategories(ctx context.Context, r *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	p, err := s.Svc.ListCategories(ctx, r.GetQuery(), r.ParentId, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListCategoriesResponse{Items: []*pb.Category{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, catTo(x))
	}
	return out, nil
}

func (s *Server) GetCategory(ctx context.Context, r *pb.GetCategoryRequest) (*pb.GetCategoryResponse, error) {
	x, err := s.Svc.GetCategory(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetCategoryResponse{Category: catTo(x)}, nil
}

func (s *Server) CreateCategory(ctx context.Context, r *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	x, err := s.Svc.CreateCategory(ctx, actor(ctx), catFrom(r.GetCategory()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateCategoryResponse{Category: catTo(x)}, nil
}

func (s *Server) UpdateCategory(ctx context.Context, r *pb.UpdateCategoryRequest) (*pb.UpdateCategoryResponse, error) {
	x, err := s.Svc.UpdateCategory(ctx, actor(ctx), r.GetCategory().GetId(), catFrom(r.GetCategory()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateCategoryResponse{Category: catTo(x)}, nil
}

func (s *Server) DeleteCategory(ctx context.Context, r *pb.DeleteCategoryRequest) (*pb.DeleteCategoryResponse, error) {
	if err := s.Svc.DeleteCategory(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteCategoryResponse{}, nil
}

func (s *Server) MoveCategory(ctx context.Context, r *pb.MoveCategoryRequest) (*pb.MoveCategoryResponse, error) {
	x, err := s.Svc.MoveCategory(ctx, actor(ctx), r.GetId(), r.ParentId, int(r.GetPosition()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.MoveCategoryResponse{Category: catTo(x)}, nil
}

// ── attributs ──

func typeTo(t string) pb.AttributeType {
	switch t {
	case domain.AttrText:
		return pb.AttributeType_ATTRIBUTE_TYPE_TEXT
	case domain.AttrNumber:
		return pb.AttributeType_ATTRIBUTE_TYPE_NUMBER
	case domain.AttrBoolean:
		return pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN
	case domain.AttrEnum:
		return pb.AttributeType_ATTRIBUTE_TYPE_ENUM
	}
	return pb.AttributeType_ATTRIBUTE_TYPE_UNSPECIFIED
}

func typeFrom(t pb.AttributeType) string {
	switch t {
	case pb.AttributeType_ATTRIBUTE_TYPE_TEXT:
		return domain.AttrText
	case pb.AttributeType_ATTRIBUTE_TYPE_NUMBER:
		return domain.AttrNumber
	case pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN:
		return domain.AttrBoolean
	case pb.AttributeType_ATTRIBUTE_TYPE_ENUM:
		return domain.AttrEnum
	}
	return ""
}

func attrTo(a *domain.AttributeDef) *pb.AttributeDef {
	out := &pb.AttributeDef{Id: a.ID, CategoryId: a.CategoryID, Key: a.Key, Label: locTo(a.Label), Type: typeTo(a.Type), Unit: a.Unit, Filterable: a.Filterable, Required: a.Required, SortOrder: int32(a.SortOrder)}
	for _, o := range a.Options {
		out.Options = append(out.Options, &pb.AttributeOption{Value: o.Value, Label: locTo(o.Label)})
	}
	return out
}

func attrFrom(a *pb.AttributeDef) domain.AttributeDef {
	out := domain.AttributeDef{CategoryID: a.GetCategoryId(), Key: a.GetKey(), Label: locFrom(a.GetLabel()), Type: typeFrom(a.GetType()), Unit: a.Unit, Filterable: a.GetFilterable(), Required: a.GetRequired(), SortOrder: int(a.GetSortOrder())}
	for _, o := range a.GetOptions() {
		out.Options = append(out.Options, domain.AttributeOption{Value: o.GetValue(), Label: locFrom(o.GetLabel())})
	}
	return out
}

func (s *Server) ListAttributeDefs(ctx context.Context, r *pb.ListAttributeDefsRequest) (*pb.ListAttributeDefsResponse, error) {
	p, err := s.Svc.ListAttributeDefs(ctx, r.GetQuery(), r.CategoryId, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListAttributeDefsResponse{Items: []*pb.AttributeDef{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, attrTo(x))
	}
	return out, nil
}

func (s *Server) GetAttributeDef(ctx context.Context, r *pb.GetAttributeDefRequest) (*pb.GetAttributeDefResponse, error) {
	x, err := s.Svc.GetAttributeDef(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetAttributeDefResponse{Attribute: attrTo(x)}, nil
}

func (s *Server) CreateAttributeDef(ctx context.Context, r *pb.CreateAttributeDefRequest) (*pb.CreateAttributeDefResponse, error) {
	x, err := s.Svc.CreateAttributeDef(ctx, actor(ctx), attrFrom(r.GetAttribute()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateAttributeDefResponse{Attribute: attrTo(x)}, nil
}

func (s *Server) UpdateAttributeDef(ctx context.Context, r *pb.UpdateAttributeDefRequest) (*pb.UpdateAttributeDefResponse, error) {
	x, err := s.Svc.UpdateAttributeDef(ctx, actor(ctx), r.GetAttribute().GetId(), attrFrom(r.GetAttribute()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateAttributeDefResponse{Attribute: attrTo(x)}, nil
}

func (s *Server) DeleteAttributeDef(ctx context.Context, r *pb.DeleteAttributeDefRequest) (*pb.DeleteAttributeDefResponse, error) {
	if err := s.Svc.DeleteAttributeDef(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteAttributeDefResponse{}, nil
}

// ── marques ──

func brandTo(b *domain.Brand) *pb.Brand {
	return &pb.Brand{Id: b.ID, Slug: b.Slug, Name: b.Name, LogoUrl: b.LogoURL, CreatedAt: timestamppb.New(b.CreatedAt), UpdatedAt: timestamppb.New(b.UpdatedAt)}
}

func brandFrom(b *pb.Brand) domain.Brand {
	return domain.Brand{Slug: b.GetSlug(), Name: b.GetName(), LogoURL: b.LogoUrl}
}

func (s *Server) ListBrands(ctx context.Context, r *pb.ListBrandsRequest) (*pb.ListBrandsResponse, error) {
	p, err := s.Svc.ListBrands(ctx, r.GetQuery(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListBrandsResponse{Items: []*pb.Brand{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, brandTo(x))
	}
	return out, nil
}

func (s *Server) GetBrand(ctx context.Context, r *pb.GetBrandRequest) (*pb.GetBrandResponse, error) {
	x, err := s.Svc.GetBrand(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetBrandResponse{Brand: brandTo(x)}, nil
}

func (s *Server) CreateBrand(ctx context.Context, r *pb.CreateBrandRequest) (*pb.CreateBrandResponse, error) {
	x, err := s.Svc.CreateBrand(ctx, actor(ctx), brandFrom(r.GetBrand()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateBrandResponse{Brand: brandTo(x)}, nil
}

func (s *Server) UpdateBrand(ctx context.Context, r *pb.UpdateBrandRequest) (*pb.UpdateBrandResponse, error) {
	x, err := s.Svc.UpdateBrand(ctx, actor(ctx), r.GetBrand().GetId(), brandFrom(r.GetBrand()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateBrandResponse{Brand: brandTo(x)}, nil
}

func (s *Server) DeleteBrand(ctx context.Context, r *pb.DeleteBrandRequest) (*pb.DeleteBrandResponse, error) {
	if err := s.Svc.DeleteBrand(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteBrandResponse{}, nil
}

// ── secteurs ──

func sectorTo(x *domain.Sector) *pb.Sector {
	return &pb.Sector{Id: x.ID, Slug: x.Slug, Name: locTo(x.Name), ImageUrl: x.ImageURL, CreatedAt: timestamppb.New(x.CreatedAt), UpdatedAt: timestamppb.New(x.UpdatedAt)}
}

func sectorFrom(x *pb.Sector) domain.Sector {
	return domain.Sector{Slug: x.GetSlug(), Name: locFrom(x.GetName()), ImageURL: x.ImageUrl}
}

func (s *Server) ListSectors(ctx context.Context, r *pb.ListSectorsRequest) (*pb.ListSectorsResponse, error) {
	p, err := s.Svc.ListSectors(ctx, r.GetQuery(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListSectorsResponse{Items: []*pb.Sector{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, sectorTo(x))
	}
	return out, nil
}

func (s *Server) GetSector(ctx context.Context, r *pb.GetSectorRequest) (*pb.GetSectorResponse, error) {
	x, err := s.Svc.GetSector(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetSectorResponse{Sector: sectorTo(x)}, nil
}

func (s *Server) CreateSector(ctx context.Context, r *pb.CreateSectorRequest) (*pb.CreateSectorResponse, error) {
	x, err := s.Svc.CreateSector(ctx, actor(ctx), sectorFrom(r.GetSector()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateSectorResponse{Sector: sectorTo(x)}, nil
}

func (s *Server) UpdateSector(ctx context.Context, r *pb.UpdateSectorRequest) (*pb.UpdateSectorResponse, error) {
	x, err := s.Svc.UpdateSector(ctx, actor(ctx), r.GetSector().GetId(), sectorFrom(r.GetSector()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateSectorResponse{Sector: sectorTo(x)}, nil
}

func (s *Server) DeleteSector(ctx context.Context, r *pb.DeleteSectorRequest) (*pb.DeleteSectorResponse, error) {
	if err := s.Svc.DeleteSector(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteSectorResponse{}, nil
}
