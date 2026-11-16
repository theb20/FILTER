package server

import (
	"context"
	"net/url"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/grpc/codes"
)

// Taxonomie du catalogue : le gateway relaie vers le service catalog ; la permission de chaque route est contrôlée avant.

func (h *handler) catalog() (pb.CatalogAdminServiceClient, error) {
	if h.cat == nil {
		return nil, apperr.New(codes.Unimplemented, apperr.CodeNotImplemented, "le service catalog n'est pas configuré")
	}
	return h.cat, nil
}

func uuidOf(s string) uuid.UUID { id, _ := uuid.Parse(s); return id }

func strPtr(id api.OptNilUUID) *string {
	if v, ok := id.Get(); ok {
		s := v.String()
		return &s
	}
	return nil
}

func optUUIDPtr(id api.OptUUID) *string {
	if v, ok := id.Get(); ok {
		s := v.String()
		return &s
	}
	return nil
}

func uriPtr(u api.OptNilURI) *string {
	if v, ok := u.Get(); ok {
		s := v.String()
		return &s
	}
	return nil
}

func toURI(s *string) api.OptNilURI {
	var out api.OptNilURI
	if s == nil {
		out.SetToNull()
		return out
	}
	u, err := url.Parse(*s)
	if err != nil {
		out.SetToNull()
		return out
	}
	return api.NewOptNilURI(*u)
}

func locProto(l api.LocalizedText) *pb.Localized { return &pb.Localized{Texts: map[string]string(l)} }

func locAPI(l *pb.Localized) api.LocalizedText { return api.LocalizedText(l.GetTexts()) }

func nilStr(s *string) api.OptNilString {
	var o api.OptNilString
	if s == nil {
		o.SetToNull()
		return o
	}
	return api.NewOptNilString(*s)
}

// ── catégories ──

func toCategory(c *pb.Category) api.CategoryAdmin {
	out := api.CategoryAdmin{ID: uuidOf(c.GetId()), Slug: c.GetSlug(), SortOrder: api.NewOptInt32(c.GetSortOrder()), Name: locAPI(c.GetName()), Depth: api.NewOptInt32(c.GetDepth()),
		ProductCount: api.NewOptInt64(c.GetProductCount()), CreatedAt: api.NewOptDateTime(c.GetCreatedAt().AsTime()), UpdatedAt: api.NewOptDateTime(c.GetUpdatedAt().AsTime())}
	if c.ParentId != nil {
		out.ParentId = api.NewOptNilUUID(uuidOf(c.GetParentId()))
	} else {
		out.ParentId.SetToNull()
	}
	if c.Description != nil {
		out.Description = api.NewOptNilLocalizedText(locAPI(c.GetDescription()))
	} else {
		out.Description.SetToNull()
	}
	return out
}

func categoryProto(id string, in *api.CategoryInput) *pb.Category {
	c := &pb.Category{Id: id, Slug: in.Slug, ParentId: strPtr(in.ParentId), SortOrder: in.SortOrder.Or(0), Name: locProto(in.Name)}
	if d, ok := in.Description.Get(); ok {
		c.Description = locProto(d)
	}
	return c
}

func (h *handler) AdminCategoriesList(ctx context.Context, p api.AdminCategoriesListParams) (api.AdminCategoriesListRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListCategories(ctx, &pb.ListCategoriesRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or(""), ParentId: optUUIDPtr(p.ParentId)})
	if err != nil {
		return nil, err
	}
	out := &api.CategoriesPage{Items: make([]api.CategoryAdmin, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toCategory(x))
	}
	return out, nil
}

func (h *handler) AdminCategoriesCreate(ctx context.Context, req *api.CategoryInput) (api.AdminCategoriesCreateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateCategory(ctx, &pb.CreateCategoryRequest{Category: categoryProto("", req)})
	if err != nil {
		return nil, err
	}
	x := toCategory(res.GetCategory())
	return &x, nil
}

func (h *handler) AdminCategoriesGet(ctx context.Context, p api.AdminCategoriesGetParams) (api.AdminCategoriesGetRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetCategory(ctx, &pb.GetCategoryRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toCategory(res.GetCategory())
	return &x, nil
}

func (h *handler) AdminCategoriesUpdate(ctx context.Context, req *api.CategoryInput, p api.AdminCategoriesUpdateParams) (api.AdminCategoriesUpdateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateCategory(ctx, &pb.UpdateCategoryRequest{Category: categoryProto(p.ID.String(), req)})
	if err != nil {
		return nil, err
	}
	x := toCategory(res.GetCategory())
	return &x, nil
}

func (h *handler) AdminCategoriesDelete(ctx context.Context, p api.AdminCategoriesDeleteParams) (api.AdminCategoriesDeleteRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteCategory(ctx, &pb.DeleteCategoryRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminCategoriesDeleteNoContent{}, nil
}

func (h *handler) AdminMoveCategory(ctx context.Context, req *api.AdminMoveCategoryRequest, p api.AdminMoveCategoryParams) (api.AdminMoveCategoryRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	var parent *string
	if v, ok := req.ParentId.Get(); ok {
		s := v.String()
		parent = &s
	}
	res, err := c.MoveCategory(ctx, &pb.MoveCategoryRequest{Id: p.ID.String(), ParentId: parent, Position: req.Position})
	if err != nil {
		return nil, err
	}
	x := toCategory(res.GetCategory())
	return &x, nil
}

// ── attributs ──

func attrTypeProto(t api.AttributeDefInputType) pb.AttributeType {
	switch t {
	case api.AttributeDefInputTypeText:
		return pb.AttributeType_ATTRIBUTE_TYPE_TEXT
	case api.AttributeDefInputTypeNumber:
		return pb.AttributeType_ATTRIBUTE_TYPE_NUMBER
	case api.AttributeDefInputTypeBoolean:
		return pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN
	case api.AttributeDefInputTypeEnum:
		return pb.AttributeType_ATTRIBUTE_TYPE_ENUM
	}
	return pb.AttributeType_ATTRIBUTE_TYPE_UNSPECIFIED
}

func toAttr(a *pb.AttributeDef) api.AttributeDefAdmin {
	t := api.AttributeDefAdminTypeText
	switch a.GetType() {
	case pb.AttributeType_ATTRIBUTE_TYPE_NUMBER:
		t = api.AttributeDefAdminTypeNumber
	case pb.AttributeType_ATTRIBUTE_TYPE_BOOLEAN:
		t = api.AttributeDefAdminTypeBoolean
	case pb.AttributeType_ATTRIBUTE_TYPE_ENUM:
		t = api.AttributeDefAdminTypeEnum
	}
	out := api.AttributeDefAdmin{ID: uuidOf(a.GetId()), CategoryId: uuidOf(a.GetCategoryId()), Key: a.GetKey(), Label: locAPI(a.GetLabel()), Type: t, Unit: nilStr(a.Unit),
		Filterable: api.NewOptBool(a.GetFilterable()), Required: api.NewOptBool(a.GetRequired()), SortOrder: api.NewOptInt32(a.GetSortOrder()), Options: []api.AttributeDefAdminOptionsItem{}}
	for _, o := range a.GetOptions() {
		out.Options = append(out.Options, api.AttributeDefAdminOptionsItem{Value: o.GetValue(), Label: locAPI(o.GetLabel())})
	}
	return out
}

func attrProto(id string, in *api.AttributeDefInput) *pb.AttributeDef {
	a := &pb.AttributeDef{Id: id, CategoryId: in.CategoryId.String(), Key: in.Key, Label: locProto(in.Label), Type: attrTypeProto(in.Type), Filterable: in.Filterable.Or(false), Required: in.Required.Or(false), SortOrder: in.SortOrder.Or(0)}
	if u, ok := in.Unit.Get(); ok {
		a.Unit = &u
	}
	for _, o := range in.Options {
		a.Options = append(a.Options, &pb.AttributeOption{Value: o.Value, Label: locProto(o.Label)})
	}
	return a
}

func (h *handler) AdminAttributeDefsList(ctx context.Context, p api.AdminAttributeDefsListParams) (api.AdminAttributeDefsListRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListAttributeDefs(ctx, &pb.ListAttributeDefsRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or(""), CategoryId: optUUIDPtr(p.CategoryId)})
	if err != nil {
		return nil, err
	}
	out := &api.AttributeDefsPage{Items: make([]api.AttributeDefAdmin, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toAttr(x))
	}
	return out, nil
}

func (h *handler) AdminAttributeDefsCreate(ctx context.Context, req *api.AttributeDefInput) (api.AdminAttributeDefsCreateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateAttributeDef(ctx, &pb.CreateAttributeDefRequest{Attribute: attrProto("", req)})
	if err != nil {
		return nil, err
	}
	x := toAttr(res.GetAttribute())
	return &x, nil
}

func (h *handler) AdminAttributeDefsGet(ctx context.Context, p api.AdminAttributeDefsGetParams) (api.AdminAttributeDefsGetRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetAttributeDef(ctx, &pb.GetAttributeDefRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toAttr(res.GetAttribute())
	return &x, nil
}

func (h *handler) AdminAttributeDefsUpdate(ctx context.Context, req *api.AttributeDefInput, p api.AdminAttributeDefsUpdateParams) (api.AdminAttributeDefsUpdateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateAttributeDef(ctx, &pb.UpdateAttributeDefRequest{Attribute: attrProto(p.ID.String(), req)})
	if err != nil {
		return nil, err
	}
	x := toAttr(res.GetAttribute())
	return &x, nil
}

func (h *handler) AdminAttributeDefsDelete(ctx context.Context, p api.AdminAttributeDefsDeleteParams) (api.AdminAttributeDefsDeleteRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteAttributeDef(ctx, &pb.DeleteAttributeDefRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminAttributeDefsDeleteNoContent{}, nil
}

// ── marques ──

func toBrand(b *pb.Brand) api.BrandAdmin {
	return api.BrandAdmin{ID: uuidOf(b.GetId()), Slug: b.GetSlug(), Name: b.GetName(), LogoUrl: toURI(b.LogoUrl), CreatedAt: api.NewOptDateTime(b.GetCreatedAt().AsTime()), UpdatedAt: api.NewOptDateTime(b.GetUpdatedAt().AsTime())}
}

func (h *handler) AdminBrandsList(ctx context.Context, p api.AdminBrandsListParams) (api.AdminBrandsListRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListBrands(ctx, &pb.ListBrandsRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")})
	if err != nil {
		return nil, err
	}
	out := &api.BrandsPage{Items: make([]api.BrandAdmin, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toBrand(x))
	}
	return out, nil
}

func (h *handler) AdminBrandsCreate(ctx context.Context, req *api.BrandAdminInput) (api.AdminBrandsCreateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateBrand(ctx, &pb.CreateBrandRequest{Brand: &pb.Brand{Slug: req.Slug, Name: req.Name, LogoUrl: uriPtr(req.LogoUrl)}})
	if err != nil {
		return nil, err
	}
	x := toBrand(res.GetBrand())
	return &x, nil
}

func (h *handler) AdminBrandsGet(ctx context.Context, p api.AdminBrandsGetParams) (api.AdminBrandsGetRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetBrand(ctx, &pb.GetBrandRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toBrand(res.GetBrand())
	return &x, nil
}

func (h *handler) AdminBrandsUpdate(ctx context.Context, req *api.BrandAdminInput, p api.AdminBrandsUpdateParams) (api.AdminBrandsUpdateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateBrand(ctx, &pb.UpdateBrandRequest{Brand: &pb.Brand{Id: p.ID.String(), Slug: req.Slug, Name: req.Name, LogoUrl: uriPtr(req.LogoUrl)}})
	if err != nil {
		return nil, err
	}
	x := toBrand(res.GetBrand())
	return &x, nil
}

func (h *handler) AdminBrandsDelete(ctx context.Context, p api.AdminBrandsDeleteParams) (api.AdminBrandsDeleteRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteBrand(ctx, &pb.DeleteBrandRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminBrandsDeleteNoContent{}, nil
}

// ── secteurs ──

func toSector(s *pb.Sector) api.SectorAdmin {
	return api.SectorAdmin{ID: uuidOf(s.GetId()), Slug: s.GetSlug(), Name: locAPI(s.GetName()), ImageUrl: toURI(s.ImageUrl), CreatedAt: api.NewOptDateTime(s.GetCreatedAt().AsTime()), UpdatedAt: api.NewOptDateTime(s.GetUpdatedAt().AsTime())}
}

func (h *handler) AdminSectorsList(ctx context.Context, p api.AdminSectorsListParams) (api.AdminSectorsListRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListSectors(ctx, &pb.ListSectorsRequest{Page: page(p.Page, p.PageSize), Query: p.Q.Or("")})
	if err != nil {
		return nil, err
	}
	out := &api.SectorsPage{Items: make([]api.SectorAdmin, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, toSector(x))
	}
	return out, nil
}

func (h *handler) AdminSectorsCreate(ctx context.Context, req *api.SectorAdminInput) (api.AdminSectorsCreateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateSector(ctx, &pb.CreateSectorRequest{Sector: &pb.Sector{Slug: req.Slug, Name: locProto(req.Name), ImageUrl: uriPtr(req.ImageUrl)}})
	if err != nil {
		return nil, err
	}
	x := toSector(res.GetSector())
	return &x, nil
}

func (h *handler) AdminSectorsGet(ctx context.Context, p api.AdminSectorsGetParams) (api.AdminSectorsGetRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetSector(ctx, &pb.GetSectorRequest{Id: p.ID.String()})
	if err != nil {
		return nil, err
	}
	x := toSector(res.GetSector())
	return &x, nil
}

func (h *handler) AdminSectorsUpdate(ctx context.Context, req *api.SectorAdminInput, p api.AdminSectorsUpdateParams) (api.AdminSectorsUpdateRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.UpdateSector(ctx, &pb.UpdateSectorRequest{Sector: &pb.Sector{Id: p.ID.String(), Slug: req.Slug, Name: locProto(req.Name), ImageUrl: uriPtr(req.ImageUrl)}})
	if err != nil {
		return nil, err
	}
	x := toSector(res.GetSector())
	return &x, nil
}

func (h *handler) AdminSectorsDelete(ctx context.Context, p api.AdminSectorsDeleteParams) (api.AdminSectorsDeleteRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if _, err := c.DeleteSector(ctx, &pb.DeleteSectorRequest{Id: p.ID.String()}); err != nil {
		return nil, err
	}
	return &api.AdminSectorsDeleteNoContent{}, nil
}
