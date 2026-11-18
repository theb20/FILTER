package server

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/catalogadmin/v1"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Vitrine : lecture PUBLIQUE du catalogue. Le gateway relaie vers le service catalog et ne montre que ce qui est publié
// (statut `active`). Les textes sont servis dans la langue demandée, avec repli sur la langue par défaut.

const defaultLocale = "fr"

// wantLocale : première langue de l'en-tête Accept-Language (ou celle déduite par le gateway), sinon « fr ».
func wantLocale(ctx context.Context, accept api.OptString) string {
	l := accept.Or("")
	if l == "" {
		l = infoFrom(ctx).Locale
	}
	l = strings.ToLower(strings.TrimSpace(strings.SplitN(strings.SplitN(l, ",", 2)[0], ";", 2)[0]))
	if i := strings.IndexAny(l, "-_"); i > 0 {
		l = l[:i]
	}
	if l == "" {
		return defaultLocale
	}
	return l
}

// pickText choisit le texte dans la langue demandée ; `fallback` vaut true si on a dû servir une autre langue.
func pickText(l *pb.Localized, locale string) (text string, fallback bool) {
	t := l.GetTexts()
	if v, ok := t[locale]; ok && v != "" {
		return v, false
	}
	if v, ok := t[defaultLocale]; ok && v != "" {
		return v, locale != defaultLocale
	}
	for _, v := range t {
		if v != "" {
			return v, true
		}
	}
	return "", false
}

// allCategories lit toutes les catégories (l'arbre est petit) ; indexées par identifiant.
func (h *handler) allCategories(ctx context.Context, c pb.CatalogAdminServiceClient) ([]*pb.Category, map[string]*pb.Category, error) {
	var all []*pb.Category
	for p := int32(1); p < 50; p++ {
		res, err := c.ListCategories(ctx, &pb.ListCategoriesRequest{Page: &commonv1.PageRequest{Page: p, PageSize: 100}})
		if err != nil {
			return nil, nil, err
		}
		all = append(all, res.GetItems()...)
		if p >= res.GetPage().GetTotalPages() {
			break
		}
	}
	by := make(map[string]*pb.Category, len(all))
	for _, x := range all {
		by[x.GetId()] = x
	}
	return all, by, nil
}

func categoryRef(by map[string]*pb.Category, id, locale string) api.CategoryRef {
	var chain []*pb.Category
	for cur, n := by[id], 0; cur != nil && n < 50; n++ {
		chain = append([]*pb.Category{cur}, chain...)
		if cur.ParentId == nil {
			break
		}
		cur = by[cur.GetParentId()]
	}
	out := api.CategoryRef{ID: uuidOf(id), Path: []api.CategoryRefPathItem{}}
	for _, x := range chain {
		name, _ := pickText(x.GetName(), locale)
		out.Path = append(out.Path, api.CategoryRefPathItem{ID: uuidOf(x.GetId()), Slug: x.GetSlug(), Name: name})
		out.Slug, out.Name = x.GetSlug(), name
	}
	return out
}

func publicBrand(b *pb.Brand) api.Brand {
	out := api.Brand{ID: uuidOf(b.GetId()), Slug: b.GetSlug(), Name: b.GetName()}
	if b.LogoUrl != nil {
		if u, err := url.Parse(b.GetLogoUrl()); err == nil {
			out.LogoUrl = api.NewOptNilURI(*u)
		}
	}
	return out
}

func publicSector(s *pb.Sector, locale string) api.Sector {
	name, _ := pickText(s.GetName(), locale)
	out := api.Sector{ID: uuidOf(s.GetId()), Slug: s.GetSlug(), Name: name}
	if s.ImageUrl != nil {
		if u, err := url.Parse(s.GetImageUrl()); err == nil {
			out.ImageUrl = api.NewOptNilURI(*u)
		}
	}
	return out
}

func moneyXOF(amount int64, cur string) api.Money {
	return api.Money{Amount: itoa(amount), Currency: cur}
}

func itoa(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	b := [20]byte{}
	i := len(b)
	for {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// publicPrice : prix hors taxes dans la devise de base. Sur devis : montant 0 (le site affiche « sur devis » d'après le mode).
func publicPrice(p *pb.Product) api.PriceInfo {
	amt := int64(0)
	if p.PriceAmount != nil {
		amt = p.GetPriceAmount()
	}
	return api.PriceInfo{Mode: api.NewOptPriceInfoMode(api.PriceInfoMode(p.GetPriceMode())), UnitPrice: moneyXOF(amt, p.GetCurrency()), TaxIncluded: false, Source: api.PriceInfoSourceFixed, FxRateId: api.NilUUID{Null: true}, VolumeTiers: []api.VolumeTier{}}
}

// publicAvailability : tant que le stock n'existe pas, « disponibilité sur demande » (jamais « rupture »).
func publicAvailability(p *pb.Product) api.Availability {
	a := api.Availability{ShipsWithinDays: api.DayRange{}}
	a.Quantity.SetToNull()
	a.LeadTimeDays.SetToNull()
	if p.GetFulfillmentMode() == "made_to_order" {
		a.Status = api.AvailabilityStatusMadeToOrder
		if p.LeadTimeDays != nil {
			a.LeadTimeDays = api.NewOptNilInt32(p.GetLeadTimeDays())
			a.ShipsWithinDays = api.DayRange{Min: p.GetLeadTimeDays(), Max: p.GetLeadTimeDays()}
		}
		return a
	}
	a.Status = api.AvailabilityStatusOnRequest
	return a
}

func (h *handler) brandIndex(ctx context.Context, c pb.CatalogAdminServiceClient) (map[string]*pb.Brand, error) {
	by := map[string]*pb.Brand{}
	for p := int32(1); p < 50; p++ {
		res, err := c.ListBrands(ctx, &pb.ListBrandsRequest{Page: &commonv1.PageRequest{Page: p, PageSize: 100}})
		if err != nil {
			return nil, err
		}
		for _, b := range res.GetItems() {
			by[b.GetId()] = b
		}
		if p >= res.GetPage().GetTotalPages() {
			break
		}
	}
	return by, nil
}

func (h *handler) summary(p *pb.Product, cats map[string]*pb.Category, brands map[string]*pb.Brand, locale string) api.ProductSummary {
	name, _ := pickText(p.GetName(), locale)
	out := api.ProductSummary{ID: uuidOf(p.GetId()), Sku: p.GetSku(), Name: name, Category: categoryRef(cats, p.GetCategoryId(), locale), Price: publicPrice(p), Availability: publicAvailability(p)}
	if b := brands[p.GetBrandId()]; b != nil {
		out.Brand = api.NewOptNilBrand(publicBrand(b))
	} else {
		out.Brand.SetToNull()
	}
	out.Image.SetToNull()
	if len(p.GetMedia()) > 0 {
		m := p.GetMedia()[0]
		if u, err := url.Parse(m.GetUrl()); err == nil {
			img := api.ImageRef{URL: *u}
			if m.Alt != nil {
				img.Alt = api.NewOptNilString(m.GetAlt())
			} else {
				img.Alt.SetToNull()
			}
			out.Image = api.NewOptNilImageRef(img)
		}
	}
	return out
}

// ── catégories ──

func categoryTree(all []*pb.Category, locale string, root *string, depth int) []api.CategoryNode {
	kids := map[string][]*pb.Category{}
	var roots []*pb.Category
	for _, c := range all {
		if c.ParentId == nil {
			roots = append(roots, c)
		} else {
			kids[c.GetParentId()] = append(kids[c.GetParentId()], c)
		}
	}
	var build func(c *pb.Category, d int) api.CategoryNode
	build = func(c *pb.Category, d int) api.CategoryNode {
		name, _ := pickText(c.GetName(), locale)
		n := api.CategoryNode{ID: uuidOf(c.GetId()), Slug: c.GetSlug(), Name: name, Depth: c.GetDepth(), ProductCount: api.NewOptInt64(c.GetProductCount()), Children: []api.CategoryNode{}, ImageUrl: nilStr(c.ImageUrl)}
		if c.ParentId != nil {
			n.ParentId = api.NewNilUUID(uuidOf(c.GetParentId()))
		} else {
			n.ParentId.SetToNull()
		}
		if depth == 0 || d < depth {
			for _, k := range kids[c.GetId()] {
				n.Children = append(n.Children, build(k, d+1))
			}
		}
		return n
	}
	var start []*pb.Category
	if root != nil {
		start = kids[*root]
	} else {
		start = roots
	}
	out := make([]api.CategoryNode, 0, len(start))
	for _, c := range start {
		out = append(out, build(c, 1))
	}
	return out
}

func (h *handler) ListCategories(ctx context.Context, p api.ListCategoriesParams) (api.ListCategoriesRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	all, _, err := h.allCategories(ctx, c)
	if err != nil {
		return nil, err
	}
	out := api.ListCategoriesOKApplicationJSON(categoryTree(all, locale, optUUIDPtr(p.ParentId), p.Depth.Or(0)))
	return &out, nil
}

func (h *handler) findCategory(all []*pb.Category, idOrSlug string) *pb.Category {
	for _, c := range all {
		if c.GetId() == idOrSlug || c.GetSlug() == idOrSlug {
			return c
		}
	}
	return nil
}

func (h *handler) GetCategory(ctx context.Context, p api.GetCategoryParams) (api.GetCategoryRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	all, _, err := h.allCategories(ctx, c)
	if err != nil {
		return nil, err
	}
	cat := h.findCategory(all, p.IdOrSlug)
	if cat == nil {
		return nil, apperr.NotFound("catégorie introuvable")
	}
	id := cat.GetId()
	n := categoryTree(all, locale, &id, 0)
	name, _ := pickText(cat.GetName(), locale)
	node := api.CategoryNode{ID: uuidOf(id), Slug: cat.GetSlug(), Name: name, Depth: cat.GetDepth(), ProductCount: api.NewOptInt64(cat.GetProductCount()), Children: n}
	if cat.ParentId != nil {
		node.ParentId = api.NewNilUUID(uuidOf(cat.GetParentId()))
	} else {
		node.ParentId.SetToNull()
	}
	return &node, nil
}

func (h *handler) ListCategoryAttributes(ctx context.Context, p api.ListCategoryAttributesParams) (api.ListCategoryAttributesRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	all, by, err := h.allCategories(ctx, c)
	if err != nil {
		return nil, err
	}
	cat := h.findCategory(all, p.IdOrSlug)
	if cat == nil {
		return nil, apperr.NotFound("catégorie introuvable")
	}
	var chain []*pb.Category // racine → catégorie : les attributs des ancêtres sont hérités
	for cur, n := cat, 0; cur != nil && n < 50; n++ {
		chain = append([]*pb.Category{cur}, chain...)
		if cur.ParentId == nil {
			break
		}
		cur = by[cur.GetParentId()]
	}
	out := api.ListCategoryAttributesOKApplicationJSON{}
	for _, anc := range chain {
		for pg := int32(1); pg < 50; pg++ {
			id := anc.GetId()
			res, err := c.ListAttributeDefs(ctx, &pb.ListAttributeDefsRequest{Page: &commonv1.PageRequest{Page: pg, PageSize: 100}, CategoryId: &id})
			if err != nil {
				return nil, err
			}
			for _, d := range res.GetItems() {
				if p.FilterableOnly.Or(false) && !d.GetFilterable() {
					continue
				}
				label, _ := pickText(d.GetLabel(), locale)
				a := api.AttributeDef{ID: uuidOf(d.GetId()), CategoryId: uuidOf(d.GetCategoryId()), Key: d.GetKey(), Label: label, Type: api.AttributeDefType(strings.ToLower(strings.TrimPrefix(d.GetType().String(), "ATTRIBUTE_TYPE_"))),
					Filterable: d.GetFilterable(), Required: d.GetRequired(), SortOrder: d.GetSortOrder(), Options: []api.AttributeOption{}}
				if d.Unit != nil {
					a.Unit = api.NewNilString(d.GetUnit())
				} else {
					a.Unit.SetToNull()
				}
				if d.GetCategoryId() != cat.GetId() {
					a.InheritedFrom = api.NewNilUUID(uuidOf(d.GetCategoryId()))
				} else {
					a.InheritedFrom.SetToNull()
				}
				for _, o := range d.GetOptions() {
					l, _ := pickText(o.GetLabel(), locale)
					a.Options = append(a.Options, api.AttributeOption{Value: o.GetValue(), Label: l})
				}
				out = append(out, a)
			}
			if pg >= res.GetPage().GetTotalPages() {
				break
			}
		}
	}
	return &out, nil
}

// ── marques et secteurs ──

func (h *handler) ListBrands(ctx context.Context, p api.ListBrandsParams) (api.ListBrandsRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := c.ListBrands(ctx, &pb.ListBrandsRequest{Page: page(p.Page, p.PageSize)})
	if err != nil {
		return nil, err
	}
	out := &api.BrandPage{Items: make([]api.Brand, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, b := range res.GetItems() {
		out.Items = append(out.Items, publicBrand(b))
	}
	return out, nil
}

func (h *handler) GetBrand(ctx context.Context, p api.GetBrandParams) (api.GetBrandRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	by, err := h.brandIndex(ctx, c)
	if err != nil {
		return nil, err
	}
	for _, b := range by {
		if b.GetSlug() == p.Slug {
			out := publicBrand(b)
			return &out, nil
		}
	}
	return nil, apperr.NotFound("marque introuvable")
}

func (h *handler) ListSectors(ctx context.Context, p api.ListSectorsParams) (api.ListSectorsRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	out := api.ListSectorsOKApplicationJSON{}
	for pg := int32(1); pg < 50; pg++ {
		res, err := c.ListSectors(ctx, &pb.ListSectorsRequest{Page: &commonv1.PageRequest{Page: pg, PageSize: 100}})
		if err != nil {
			return nil, err
		}
		for _, s := range res.GetItems() {
			out = append(out, publicSector(s, locale))
		}
		if pg >= res.GetPage().GetTotalPages() {
			break
		}
	}
	return &out, nil
}

// ── produits ──

func (h *handler) ListProducts(ctx context.Context, p api.ListProductsParams) (api.ListProductsRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	active := "active"
	res, err := c.ListProducts(ctx, &pb.ListProductsRequest{Page: page(p.Page, p.PageSize), CategoryId: optUUIDPtr(p.CategoryId), BrandId: optUUIDPtr(p.BrandId), Status: &active})
	if err != nil {
		return nil, err
	}
	_, cats, err := h.allCategories(ctx, c)
	if err != nil {
		return nil, err
	}
	brands, err := h.brandIndex(ctx, c)
	if err != nil {
		return nil, err
	}
	out := &api.ProductPage{Items: make([]api.ProductSummary, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, x := range res.GetItems() {
		out.Items = append(out.Items, h.summary(x, cats, brands, locale))
	}
	return out, nil
}

// publishedProduct : un produit n'est visible que s'il est publié ; sinon 404 (jamais de fuite d'un brouillon).
func (h *handler) publishedProduct(ctx context.Context, c pb.CatalogAdminServiceClient, idOrSku string) (*pb.Product, error) {
	if _, err := uuid.Parse(idOrSku); err == nil {
		res, err := c.GetProduct(ctx, &pb.GetProductRequest{Id: idOrSku})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return nil, apperr.NotFound("produit introuvable")
			}
			return nil, err
		}
		if res.GetProduct().GetStatus() != "active" {
			return nil, apperr.NotFound("produit introuvable")
		}
		return res.GetProduct(), nil
	}
	active := "active"
	res, err := c.ListProducts(ctx, &pb.ListProductsRequest{Page: &commonv1.PageRequest{Page: 1, PageSize: 50}, Query: idOrSku, Status: &active})
	if err != nil {
		return nil, err
	}
	for _, x := range res.GetItems() {
		if strings.EqualFold(x.GetSku(), idOrSku) || strings.EqualFold(x.GetSlug(), idOrSku) {
			return x, nil
		}
	}
	return nil, apperr.NotFound("produit introuvable")
}

func (h *handler) GetProduct(ctx context.Context, p api.GetProductParams) (api.GetProductRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	x, err := h.publishedProduct(ctx, c, p.IdOrSku)
	if err != nil {
		return nil, err
	}
	all, cats, err := h.allCategories(ctx, c)
	if err != nil {
		return nil, err
	}
	_ = all
	brands, err := h.brandIndex(ctx, c)
	if err != nil {
		return nil, err
	}
	sum := h.summary(x, cats, brands, locale)
	name, fb := pickText(x.GetName(), locale)
	out := &api.Product{ID: sum.ID, Sku: sum.Sku, Name: name, Brand: sum.Brand, Category: sum.Category, Status: api.ProductStatusActive, Price: sum.Price, Availability: sum.Availability,
		Attributes: []api.AttributeValue{}, Media: []api.ProductMedia{}, Sectors: []api.Sector{}, CrossReferences: []api.CrossReference{}, Compatibilities: []api.Compatibility{},
		Fulfillment: api.FulfillmentInfo{Mode: api.FulfillmentInfoMode(x.GetFulfillmentMode())}, Locale: api.NewOptLocale(api.Locale(locale))}
	out.Fulfillment.LeadTimeDays.SetToNull()
	if x.LeadTimeDays != nil {
		out.Fulfillment.LeadTimeDays = api.NewNilInt32(x.GetLeadTimeDays())
	}
	if x.Slug != "" {
		out.Slug = api.NewOptNilString(x.GetSlug())
	} else {
		out.Slug.SetToNull()
	}
	out.Description.SetToNull()
	if x.Description != nil {
		d, f := pickText(x.GetDescription(), locale)
		fb = fb || f
		out.Description = api.NewOptNilString(d)
	}
	out.TranslationFallback = api.NewOptBool(fb)
	// attributs : libellés et unités viennent des définitions de la catégorie et de ses ancêtres
	defs := map[string]*pb.AttributeDef{}
	for cur, n := cats[x.GetCategoryId()], 0; cur != nil && n < 50; n++ {
		id := cur.GetId()
		res, err := c.ListAttributeDefs(ctx, &pb.ListAttributeDefsRequest{Page: &commonv1.PageRequest{Page: 1, PageSize: 100}, CategoryId: &id})
		if err != nil {
			return nil, err
		}
		for _, d := range res.GetItems() {
			if _, ok := defs[d.GetKey()]; !ok {
				defs[d.GetKey()] = d
			}
		}
		if cur.ParentId == nil {
			break
		}
		cur = cats[cur.GetParentId()]
	}
	for _, a := range x.GetAttributes() {
		d := defs[a.GetKey()]
		if d == nil {
			continue
		}
		label, _ := pickText(d.GetLabel(), locale)
		av := api.AttributeValue{Key: a.GetKey(), Label: label, Type: api.AttributeValueType(strings.ToLower(strings.TrimPrefix(d.GetType().String(), "ATTRIBUTE_TYPE_")))}
		if d.Unit != nil {
			av.Unit = api.NewNilString(d.GetUnit())
		} else {
			av.Unit.SetToNull()
		}
		switch v := a.GetValue().(type) {
		case *pb.AttributeValue_Text:
			s := v.Text
			for _, o := range d.GetOptions() { // liste de choix : on affiche le libellé de l'option
				if o.GetValue() == s {
					s, _ = pickText(o.GetLabel(), locale)
				}
			}
			av.Value = api.NewStringAttributeValueValue(s)
		case *pb.AttributeValue_Number:
			av.Value = api.NewFloat64AttributeValueValue(v.Number)
		case *pb.AttributeValue_Boolean:
			av.Value = api.NewBoolAttributeValueValue(v.Boolean)
		default:
			continue
		}
		out.Attributes = append(out.Attributes, av)
	}
	for _, m := range x.GetMedia() {
		u, err := url.Parse(m.GetUrl())
		if err != nil {
			continue
		}
		out.Media = append(out.Media, api.ProductMedia{ID: uuidOf(m.GetId()), URL: *u, Alt: nilStr(m.Alt), SortOrder: api.NewOptInt32(m.GetSortOrder())})
	}
	for _, cr := range x.GetCrossReferences() {
		out.CrossReferences = append(out.CrossReferences, api.CrossReference{Brand: cr.GetBrand(), Number: cr.GetNumber(), Kind: api.CrossReferenceKind(cr.GetKind())})
	}
	if len(x.GetSectorIds()) > 0 {
		want := map[string]bool{}
		for _, s := range x.GetSectorIds() {
			want[s] = true
		}
		for pg := int32(1); pg < 50; pg++ {
			res, err := c.ListSectors(ctx, &pb.ListSectorsRequest{Page: &commonv1.PageRequest{Page: pg, PageSize: 100}})
			if err != nil {
				return nil, err
			}
			for _, s := range res.GetItems() {
				if want[s.GetId()] {
					out.Sectors = append(out.Sectors, publicSector(s, locale))
				}
			}
			if pg >= res.GetPage().GetTotalPages() {
				break
			}
		}
	}
	return out, nil
}

func (h *handler) ListRelatedProducts(ctx context.Context, p api.ListRelatedProductsParams) (api.ListRelatedProductsRes, error) {
	c, err := h.catalog()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	x, err := h.publishedProduct(ctx, c, p.ID.String())
	if err != nil {
		return nil, err
	}
	limit := p.Limit.Or(6)
	active := "active"
	cat := x.GetCategoryId()
	res, err := c.ListProducts(ctx, &pb.ListProductsRequest{Page: &commonv1.PageRequest{Page: 1, PageSize: int32(limit + 1)}, CategoryId: &cat, Status: &active})
	if err != nil {
		return nil, err
	}
	_, cats, err := h.allCategories(ctx, c)
	if err != nil {
		return nil, err
	}
	brands, err := h.brandIndex(ctx, c)
	if err != nil {
		return nil, err
	}
	out := api.ListRelatedProductsOKApplicationJSON{}
	for _, r := range res.GetItems() {
		if r.GetId() == x.GetId() || len(out) >= limit {
			continue
		}
		out = append(out, h.summary(r, cats, brands, locale))
	}
	return &out, nil
}
