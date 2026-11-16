package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	"github.com/theb20/FILTER/backend/services/catalog/internal/application"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func i32(p *int) *int32 {
	if p == nil {
		return nil
	}
	v := int32(*p)
	return &v
}

func intp(p *int32) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

func productTo(p *domain.Product) *pb.Product {
	out := &pb.Product{Id: p.ID, Sku: p.SKU, Slug: p.Slug, Name: locTo(p.Name), Description: locTo(p.Description), BrandId: p.BrandID, CategoryId: p.CategoryID, ManufacturerRef: p.ManufacturerRef,
		Status: p.Status, FulfillmentMode: p.FulfillmentMode, LeadTimeDays: i32(p.LeadTimeDays), PriceMode: p.PriceMode, PriceAmount: p.PriceAmount, Currency: p.Currency, Gtin: p.GTIN, SaleUnit: p.SaleUnit,
		MinOrderQuantity: i32(p.MinOrderQuantity), WeightGrams: i32(p.WeightGrams), LengthMm: i32(p.LengthMM), WidthMm: i32(p.WidthMM), HeightMm: i32(p.HeightMM),
		SectorIds: p.SectorIDs, Origins: p.Origins, Version: int32(p.Version), CreatedAt: timestamppb.New(p.CreatedAt), UpdatedAt: timestamppb.New(p.UpdatedAt)}
	if p.PublishedAt != nil {
		out.PublishedAt = timestamppb.New(*p.PublishedAt)
	}
	for _, a := range p.Attributes {
		v := &pb.AttributeValue{Key: a.Key}
		switch {
		case a.Text != nil:
			v.Value = &pb.AttributeValue_Text{Text: *a.Text}
		case a.Number != nil:
			v.Value = &pb.AttributeValue_Number{Number: *a.Number}
		case a.Bool != nil:
			v.Value = &pb.AttributeValue_Boolean{Boolean: *a.Bool}
		}
		out.Attributes = append(out.Attributes, v)
	}
	for _, m := range p.Media {
		out.Media = append(out.Media, &pb.ProductMedia{Id: m.ID, Url: m.URL, Alt: m.Alt, SortOrder: int32(m.SortOrder)})
	}
	for _, x := range p.CrossRefs {
		out.CrossReferences = append(out.CrossReferences, &pb.CrossReference{Brand: x.Brand, Number: x.Number, Kind: x.Kind})
	}
	return out
}

func productFrom(p *pb.Product) domain.Product {
	out := domain.Product{SKU: p.GetSku(), Slug: p.GetSlug(), Name: locFrom(p.GetName()), Description: locFrom(p.GetDescription()), BrandID: p.BrandId, CategoryID: p.GetCategoryId(), ManufacturerRef: p.ManufacturerRef,
		FulfillmentMode: p.GetFulfillmentMode(), LeadTimeDays: intp(p.LeadTimeDays), PriceMode: p.GetPriceMode(), PriceAmount: p.PriceAmount, GTIN: p.Gtin, SaleUnit: p.SaleUnit,
		MinOrderQuantity: intp(p.MinOrderQuantity), WeightGrams: intp(p.WeightGrams), LengthMM: intp(p.LengthMm), WidthMM: intp(p.WidthMm), HeightMM: intp(p.HeightMm), SectorIDs: p.GetSectorIds()}
	for _, a := range p.GetAttributes() {
		v := domain.AttrValue{Key: a.GetKey()}
		switch x := a.GetValue().(type) {
		case *pb.AttributeValue_Text:
			v.Text = &x.Text
		case *pb.AttributeValue_Number:
			v.Number = &x.Number
		case *pb.AttributeValue_Boolean:
			v.Bool = &x.Boolean
		}
		out.Attributes = append(out.Attributes, v)
	}
	for _, m := range p.GetMedia() {
		out.Media = append(out.Media, domain.Media{URL: m.GetUrl(), Alt: m.Alt})
	}
	for _, x := range p.GetCrossReferences() {
		out.CrossRefs = append(out.CrossRefs, domain.CrossRef{Brand: x.GetBrand(), Number: x.GetNumber(), Kind: x.GetKind()})
	}
	return out
}

func (s *Server) ListProducts(ctx context.Context, r *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	p, err := s.Svc.ListProducts(ctx, application.ProductFilter{Query: r.GetQuery(), CategoryID: r.CategoryId, BrandID: r.BrandId, Status: r.Status}, int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListProductsResponse{Items: []*pb.Product{}, Page: pageResp(p)}
	for _, x := range p.Items {
		out.Items = append(out.Items, productTo(x))
	}
	return out, nil
}

func (s *Server) GetProduct(ctx context.Context, r *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	x, err := s.Svc.GetProduct(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetProductResponse{Product: productTo(x)}, nil
}

func (s *Server) CreateProduct(ctx context.Context, r *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	x, err := s.Svc.CreateProduct(ctx, actor(ctx), productFrom(r.GetProduct()), domain.OriginManual)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateProductResponse{Product: productTo(x)}, nil
}

func (s *Server) UpdateProduct(ctx context.Context, r *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	x, err := s.Svc.UpdateProduct(ctx, actor(ctx), r.GetProduct().GetId(), productFrom(r.GetProduct()), intp(r.ExpectedVersion), domain.OriginManual)
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateProductResponse{Product: productTo(x)}, nil
}

func (s *Server) DeleteProduct(ctx context.Context, r *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	if err := s.Svc.DeleteProduct(ctx, actor(ctx), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteProductResponse{}, nil
}

func (s *Server) SubmitProductReview(ctx context.Context, r *pb.SubmitProductReviewRequest) (*pb.SubmitProductReviewResponse, error) {
	x, err := s.Svc.SubmitProductReview(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.SubmitProductReviewResponse{Product: productTo(x)}, nil
}

func (s *Server) PublishProduct(ctx context.Context, r *pb.PublishProductRequest) (*pb.PublishProductResponse, error) {
	x, err := s.Svc.PublishProduct(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.PublishProductResponse{Product: productTo(x)}, nil
}

func (s *Server) ArchiveProduct(ctx context.Context, r *pb.ArchiveProductRequest) (*pb.ArchiveProductResponse, error) {
	x, err := s.Svc.ArchiveProduct(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.ArchiveProductResponse{Product: productTo(x)}, nil
}

func (s *Server) RestoreProduct(ctx context.Context, r *pb.RestoreProductRequest) (*pb.RestoreProductResponse, error) {
	x, err := s.Svc.RestoreProduct(ctx, actor(ctx), r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.RestoreProductResponse{Product: productTo(x)}, nil
}

func (s *Server) ListProductVersions(ctx context.Context, r *pb.ListProductVersionsRequest) (*pb.ListProductVersionsResponse, error) {
	vs, err := s.Svc.ProductVersions(ctx, r.GetId())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListProductVersionsResponse{Items: []*pb.ProductVersion{}}
	for _, v := range vs {
		ch, err := structpb.NewStruct(v.Changes)
		if err != nil {
			return nil, apperr.ToGRPC(apperr.Internal(err))
		}
		out.Items = append(out.Items, &pb.ProductVersion{Version: int32(v.Version), At: timestamppb.New(v.At), ActorId: v.ActorID, Action: v.Action, Changes: ch})
	}
	return out, nil
}
