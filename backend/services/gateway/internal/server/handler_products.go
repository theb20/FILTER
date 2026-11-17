package server

import (
	"context"
	"net/url"

	"github.com/google/uuid"
	pb "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Produits : le gateway relaie vers le service catalog. `catalog:publish` n'est exigée que par la route de publication.

func nilI32(p *int32) api.OptNilInt32 {
	var o api.OptNilInt32
	if p == nil {
		o.SetToNull()
		return o
	}
	return api.NewOptNilInt32(*p)
}

func i32p(o api.OptNilInt32) *int32 {
	if v, ok := o.Get(); ok {
		return &v
	}
	return nil
}

func nilStrIn(o api.OptNilString) *string {
	if v, ok := o.Get(); ok {
		return &v
	}
	return nil
}

func toProduct(p *pb.Product) api.ProductAdmin {
	out := api.ProductAdmin{ID: uuidOf(p.GetId()), Sku: p.GetSku(), Slug: p.GetSlug(), Name: locAPI(p.GetName()), CategoryId: uuidOf(p.GetCategoryId()),
		Status: api.ProductAdminStatus(p.GetStatus()), FulfillmentMode: api.ProductAdminFulfillmentMode(p.GetFulfillmentMode()), PriceMode: api.ProductAdminPriceMode(p.GetPriceMode()),
		Currency: p.GetCurrency(), Version: p.GetVersion(), ManufacturerRef: nilStr(p.ManufacturerRef), LeadTimeDays: nilI32(p.LeadTimeDays),
		Gtin: nilStr(p.Gtin), SaleUnit: nilStr(p.SaleUnit), MinOrderQuantity: nilI32(p.MinOrderQuantity), WeightGrams: nilI32(p.WeightGrams),
		LengthMm: nilI32(p.LengthMm), WidthMm: nilI32(p.WidthMm), HeightMm: nilI32(p.HeightMm),
		Attributes: []api.ProductAttributeInput{}, SectorIds: []uuid.UUID{}, Media: []api.ProductMedia{}, CrossReferences: []api.CrossReference{},
		CreatedAt: api.NewOptDateTime(p.GetCreatedAt().AsTime()), UpdatedAt: api.NewOptDateTime(p.GetUpdatedAt().AsTime())}
	if p.Description != nil {
		out.Description = api.NewOptNilLocalizedText(locAPI(p.GetDescription()))
	} else {
		out.Description.SetToNull()
	}
	if p.BrandId != nil {
		out.BrandId = api.NewOptNilUUID(uuidOf(p.GetBrandId()))
	} else {
		out.BrandId.SetToNull()
	}
	if p.PriceAmount != nil {
		out.PriceAmount = api.NewOptNilInt64(p.GetPriceAmount())
	} else {
		out.PriceAmount.SetToNull()
	}
	if p.PublishedAt != nil {
		out.PublishedAt = api.NewOptNilDateTime(p.GetPublishedAt().AsTime())
	} else {
		out.PublishedAt.SetToNull()
	}
	if len(p.GetOrigins()) > 0 {
		out.Origins = api.NewOptProductAdminOrigins(api.ProductAdminOrigins(p.GetOrigins()))
	}
	for _, a := range p.GetAttributes() {
		var v api.ProductAttributeInputValue
		switch x := a.GetValue().(type) {
		case *pb.AttributeValue_Text:
			v = api.NewStringProductAttributeInputValue(x.Text)
		case *pb.AttributeValue_Number:
			v = api.NewFloat64ProductAttributeInputValue(x.Number)
		case *pb.AttributeValue_Boolean:
			v = api.NewBoolProductAttributeInputValue(x.Boolean)
		}
		out.Attributes = append(out.Attributes, api.ProductAttributeInput{Key: a.GetKey(), Value: v})
	}
	for _, s := range p.GetSectorIds() {
		out.SectorIds = append(out.SectorIds, uuidOf(s))
	}
	for _, m := range p.GetMedia() {
		u, _ := url.Parse(m.GetUrl())
		if u == nil {
			u = &url.URL{}
		}
		out.Media = append(out.Media, api.ProductMedia{ID: uuidOf(m.GetId()), URL: *u, Alt: nilStr(m.Alt), SortOrder: api.NewOptInt32(m.GetSortOrder())})
	}
	for _, x := range p.GetCrossReferences() {
		out.CrossReferences = append(out.CrossReferences, api.CrossReference{Brand: x.GetBrand(), Number: x.GetNumber(), Kind: api.CrossReferenceKind(x.GetKind())})
	}
	return out
}

func productProto(id string, in *api.ProductInput) (*pb.Product, *int32) {
	p := &pb.Product{Id: id, Sku: in.Sku.Or(""), Slug: in.Slug.Or(""), Name: locProto(in.Name), CategoryId: in.CategoryId.String(), BrandId: strPtr(in.BrandId), ManufacturerRef: nilStrIn(in.ManufacturerRef),
		FulfillmentMode: string(in.FulfillmentMode), LeadTimeDays: i32p(in.LeadTimeDays), PriceMode: string(in.PriceMode), Gtin: nilStrIn(in.Gtin), SaleUnit: nilStrIn(in.SaleUnit),
		MinOrderQuantity: i32p(in.MinOrderQuantity), WeightGrams: i32p(in.WeightGrams), LengthMm: i32p(in.LengthMm), WidthMm: i32p(in.WidthMm), HeightMm: i32p(in.HeightMm)}
	if d, ok := in.Description.Get(); ok {
		p.Description = locProto(d)
	}
	if v, ok := in.PriceAmount.Get(); ok {
		p.PriceAmount = &v
	}
	for _, a := range in.Attributes {
		v := &pb.AttributeValue{Key: a.Key}
		switch {
		case a.Value.IsString():
			s, _ := a.Value.GetString()
			v.Value = &pb.AttributeValue_Text{Text: s}
		case a.Value.IsFloat64():
			n, _ := a.Value.GetFloat64()
			v.Value = &pb.AttributeValue_Number{Number: n}
		case a.Value.IsBool():
			b, _ := a.Value.GetBool()
			v.Value = &pb.AttributeValue_Boolean{Boolean: b}
		}
		p.Attributes = append(p.Attributes, v)
	}
	for _, s := range in.SectorIds {
		p.SectorIds = append(p.SectorIds, s.String())
	}
	for _, m := range in.Media {
		p.Media = append(p.Media, &pb.ProductMedia{Url: m.URL.String(), Alt: nilStrIn(m.Alt)})
	}
	for _, x := range in.CrossReferences {
		p.CrossReferences = append(p.CrossReferences, &pb.CrossReference{Brand: x.Brand, Number: x.Number, Kind: string(x.Kind)})
	}
	return p, i32p(in.Version)
}

func (h *handler) AdminProductsList(ctx context.Context, p api.AdminProductsListParams) (api.AdminProductsListRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	req := &pb.ListProductsRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or(""), CategoryId: optUUIDPtr(p.CategoryId), BrandId: optUUIDPtr(p.BrandId)}
	if v, ok := p.Status.Get(); ok {
		s := string(v)
		req.Status = &s
	}
	res, err := c.ListProducts(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &api.ProductsPage{Items: make([]api.ProductAdmin, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toProduct(x))
	}
	return out, nil
}

func (h *handler) AdminProductsCreate(ctx context.Context, req *api.ProductInput) (api.AdminProductsCreateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	p, _ := productProto("", req)
	res, err := c.CreateProduct(ctx, &pb.CreateProductRequest{Product: p})
	if err != nil {
		return nil, err
	}
	x := toProduct(res.GetProduct())
	return &x, nil
}

func (h *handler) AdminProductsGet(ctx context.Context, p api.AdminProductsGetParams) (api.AdminProductsGetRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetProduct(ctx, &pb.GetProductRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toProduct(res.GetProduct())
	return &x, nil
}

func (h *handler) AdminProductsUpdate(ctx context.Context, req *api.ProductInput, p api.AdminProductsUpdateParams) (api.AdminProductsUpdateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	pr, ver := productProto(p.ID.String(), req)
	res, err := c.UpdateProduct(ctx, &pb.UpdateProductRequest{Product: pr, ExpectedVersion: ver})
	if err != nil {
		return nil, err
	}
	x := toProduct(res.GetProduct())
	return &x, nil
}

func (h *handler) AdminProductsDelete(ctx context.Context, p api.AdminProductsDeleteParams) (api.AdminProductsDeleteRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteProduct(ctx, &pb.DeleteProductRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminProductsDeleteNoContent{}, nil
}

func (h *handler) AdminProductSubmitReview(ctx context.Context, p api.AdminProductSubmitReviewParams) (api.AdminProductSubmitReviewRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.SubmitProductReview(ctx, &pb.SubmitProductReviewRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toProduct(res.GetProduct())
	return &x, nil
}

func (h *handler) AdminProductPublish(ctx context.Context, p api.AdminProductPublishParams) (api.AdminProductPublishRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.PublishProduct(ctx, &pb.PublishProductRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toProduct(res.GetProduct())
	return &x, nil
}

func (h *handler) AdminProductArchive(ctx context.Context, p api.AdminProductArchiveParams) (api.AdminProductArchiveRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ArchiveProduct(ctx, &pb.ArchiveProductRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toProduct(res.GetProduct())
	return &x, nil
}

func (h *handler) AdminProductRestore(ctx context.Context, p api.AdminProductRestoreParams) (api.AdminProductRestoreRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.RestoreProduct(ctx, &pb.RestoreProductRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toProduct(res.GetProduct())
	return &x, nil
}

func (h *handler) AdminProductVersions(ctx context.Context, p api.AdminProductVersionsParams) (api.AdminProductVersionsRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListProductVersions(ctx, &pb.ListProductVersionsRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	out := &api.ProductVersionsPage{Items: make([]api.ProductVersion, 0, len(res.GetItems()))}
	for _, v := range res.GetItems() {
		item := api.ProductVersion{Version: v.GetVersion(), At: v.GetAt().AsTime(), Action: v.GetAction(), Changes: api.ProductVersionChanges(rawMap(v.GetChanges()))}
		if v.ActorId != nil {
			item.ActorId = api.NewOptNilUUID(uuidOf(v.GetActorId()))
		} else {
			item.ActorId.SetToNull()
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}
