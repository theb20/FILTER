package server

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	pb "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

// Contenu du site : le gateway relaie vers le service content (collections et réglages). La permission des routes du
// personnel est contrôlée avant ; les lectures publiques ne servent que ce qui est publié, dans la langue demandée.

func (h *handler) content() (pb.ContentAdminServiceClient, error) {
	if h.cms == nil {
		return nil, apperr.New(codes.Unimplemented, apperr.CodeNotImplemented, "le service content n'est pas configuré")
	}
	return h.cms, nil
}

// docMap : le document tel que l'API le décrit (données + identifiant et dates).
func docMap(d *pb.Document) map[string]any {
	m := d.GetData().AsMap()
	if m == nil {
		m = map[string]any{}
	}
	m["id"] = d.GetId()
	m["createdAt"] = d.GetCreatedAt().AsTime().Format("2006-01-02T15:04:05.999Z07:00")
	m["updatedAt"] = d.GetUpdatedAt().AsTime().Format("2006-01-02T15:04:05.999Z07:00")
	return m
}

func decodeDoc[T any](d *pb.Document) (*T, error) {
	b, err := json.Marshal(docMap(d))
	if err != nil {
		return nil, apperr.Internal(err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, apperr.Internal(err)
	}
	return &out, nil
}

// inputStruct : le corps de la requête (type généré) sous forme d'objet générique pour le service.
func inputStruct(in any) (*structpb.Struct, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, apperr.Internal(err)
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s, nil
}

func (h *handler) contentList(ctx context.Context, kind, q string, where map[string]string, p, size api.OptInt) ([]*pb.Document, *commonv1.PageResponse, error) {
	c, err := h.content()
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.ListDocuments(ctx, &pb.ListDocumentsRequest{Kind: kind, Page: page(p, size), Query: q, Where: where})
	if err != nil {
		return nil, nil, err
	}
	return res.GetItems(), res.GetPage(), nil
}

func (h *handler) contentGet(ctx context.Context, kind, id string) (*pb.Document, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.GetDocument(ctx, &pb.GetDocumentRequest{Kind: kind, IdOrKey: id})
	if err != nil {
		return nil, err
	}
	return res.GetDocument(), nil
}

// contentCreate : `defaults` fixe les champs que seule une route dédiée peut changer (ex. publication : brouillon à la création).
func (h *handler) contentCreate(ctx context.Context, kind string, in any, defaults map[string]any) (*pb.Document, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	data, err := inputStruct(in)
	if err != nil {
		return nil, err
	}
	for k, v := range defaults {
		data.Fields[k], _ = structpb.NewValue(v)
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.CreateDocument(ctx, &pb.CreateDocumentRequest{Kind: kind, Data: data})
	if err != nil {
		return nil, err
	}
	return res.GetDocument(), nil
}

// contentUpdate : `keep` liste les champs conservés tels quels lors d'un remplacement complet (ex. état de publication).
func (h *handler) contentUpdate(ctx context.Context, kind, id string, in any, replace bool, keep []string) (*pb.Document, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	data, err := inputStruct(in)
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	if len(keep) > 0 {
		cur, err := c.GetDocument(ctx, &pb.GetDocumentRequest{Kind: kind, IdOrKey: id})
		if err != nil {
			return nil, err
		}
		for _, f := range keep {
			if v, ok := cur.GetDocument().GetData().GetFields()[f]; ok {
				data.Fields[f] = v
			}
		}
	}
	res, err := c.UpdateDocument(ctx, &pb.UpdateDocumentRequest{Kind: kind, Id: id, Data: data, Replace: replace})
	if err != nil {
		return nil, err
	}
	return res.GetDocument(), nil
}

func (h *handler) contentDelete(ctx context.Context, kind, id string) error {
	c, err := h.content()
	if err != nil {
		return err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	_, err = c.DeleteDocument(ctx, &pb.DeleteDocumentRequest{Kind: kind, Id: id})
	return err
}

func settingsGetAs[T any](h *handler, ctx context.Context, key string) (*T, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := c.GetSettings(ctx, &pb.GetSettingsRequest{Key: key})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(res.GetData().AsMap())
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, apperr.Internal(err)
	}
	return &out, nil
}

func settingsPutAs[T any](h *handler, ctx context.Context, key string, in *T) (*T, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	data, err := inputStruct(in)
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.staffCtx(ctx)
	defer cancel()
	res, err := c.PutSettings(ctx, &pb.PutSettingsRequest{Key: key, Data: data})
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(res.GetData().AsMap())
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, apperr.Internal(err)
	}
	return &out, nil
}

// ── lectures publiques ──

// pickLoc : texte d'un champ « par langue » dans la langue demandée, avec repli sur la langue par défaut.
func pickLoc(v any, locale string) string {
	m, _ := v.(map[string]any)
	if s, ok := m[locale].(string); ok && s != "" {
		return s
	}
	if s, ok := m[defaultLocale].(string); ok {
		return s
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }
func num(m map[string]any, k string) int64  { n, _ := m[k].(float64); return int64(n) }
func flag(m map[string]any, k string) bool  { b, _ := m[k].(bool); return b }

// publicAll lit toutes les pages d'une collection filtrée (collections de taille modeste).
func (h *handler) publicAll(ctx context.Context, kind string, where map[string]string) ([]*pb.Document, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	var out []*pb.Document
	for p := int32(1); p < 50; p++ {
		res, err := c.ListDocuments(ctx, &pb.ListDocumentsRequest{Kind: kind, Page: &commonv1.PageRequest{Page: p, PageSize: 100}, Where: where})
		if err != nil {
			return nil, err
		}
		out = append(out, res.GetItems()...)
		if p >= res.GetPage().GetTotalPages() {
			break
		}
	}
	return out, nil
}

func (h *handler) ListFaq(ctx context.Context, p api.ListFaqParams) (api.ListFaqRes, error) {
	locale := wantLocale(ctx, p.AcceptLanguage)
	docs, err := h.publicAll(ctx, "faq", map[string]string{"published": "true"})
	if err != nil {
		return nil, err
	}
	out := api.ListFaqOKApplicationJSON{}
	for _, d := range docs {
		m := d.GetData().AsMap()
		out = append(out, api.PublicFaqItem{ID: uuidOf(d.GetId()), Question: pickLoc(m["question"], locale), Answer: pickLoc(m["answer"], locale)})
	}
	return &out, nil
}

func (h *handler) ListHomePartners(ctx context.Context) (api.ListHomePartnersRes, error) {
	docs, err := h.publicAll(ctx, "home_partner", map[string]string{"published": "true"})
	if err != nil {
		return nil, err
	}
	str := func(v any) string { s, _ := v.(string); return s }
	out := api.ListHomePartnersOKApplicationJSON{}
	for _, d := range docs {
		m := d.GetData().AsMap()
		out = append(out, api.PublicHomePartner{ID: uuidOf(d.GetId()), Name: str(m["name"]), LogoUrl: str(m["logoUrl"]), Link: api.NewOptString(str(m["link"]))})
	}
	return &out, nil
}

func (h *handler) ListHeroCards(ctx context.Context, p api.ListHeroCardsParams) (api.ListHeroCardsRes, error) {
	locale := wantLocale(ctx, p.AcceptLanguage)
	docs, err := h.publicAll(ctx, "hero_card", map[string]string{"published": "true"})
	if err != nil {
		return nil, err
	}
	str := func(v any) string { s, _ := v.(string); return s }
	out := api.ListHeroCardsOKApplicationJSON{}
	for _, d := range docs {
		m := d.GetData().AsMap()
		out = append(out, api.PublicHeroCard{ID: uuidOf(d.GetId()), Key: api.NewOptString(str(m["cardKey"])), Title: pickLoc(m["title"], locale),
			Text: api.NewOptString(pickLoc(m["text"], locale)), Cta: api.NewOptString(pickLoc(m["cta"], locale)), ImageUrl: str(m["imageUrl"]),
			Link: api.NewOptString(str(m["link"])), BgColor: api.NewOptString(str(m["bgColor"])), TextColor: api.NewOptString(str(m["textColor"]))})
	}
	return &out, nil
}

func (h *handler) GetPage(ctx context.Context, p api.GetPageParams) (api.GetPageRes, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := c.GetDocument(ctx, &pb.GetDocumentRequest{Kind: "page", IdOrKey: p.Slug})
	if err != nil {
		return nil, err
	}
	m := res.GetDocument().GetData().AsMap()
	if !flag(m, "published") {
		return nil, apperr.NotFound("page introuvable")
	}
	out := &api.PublicPage{Slug: str(m, "slug"), Kind: api.PublicPageKind(str(m, "kind")), Title: pickLoc(m["title"], locale), Body: pickLoc(m["body"], locale), UpdatedAt: res.GetDocument().GetUpdatedAt().AsTime()}
	if u := str(m, "imageUrl"); u != "" {
		out.ImageUrl = api.NewOptString(u)
	}
	return out, nil
}

func (h *handler) ListShippingZones(ctx context.Context, p api.ListShippingZonesParams) (api.ListShippingZonesRes, error) {
	locale := wantLocale(ctx, p.AcceptLanguage)
	docs, err := h.publicAll(ctx, "shipping_zone", map[string]string{"active": "true"})
	if err != nil {
		return nil, err
	}
	out := api.ListShippingZonesOKApplicationJSON{}
	for _, d := range docs {
		m := d.GetData().AsMap()
		out = append(out, api.PublicShippingZone{Code: str(m, "code"), Name: pickLoc(m["name"], locale), Fee: num(m, "fee"), DaysMin: num(m, "daysMin"), DaysMax: num(m, "daysMax"), FreeEligible: flag(m, "freeEligible")})
	}
	return &out, nil
}

func (h *handler) ListBlogCategories(ctx context.Context, p api.ListBlogCategoriesParams) (api.ListBlogCategoriesRes, error) {
	locale := wantLocale(ctx, p.AcceptLanguage)
	docs, err := h.publicAll(ctx, "blog_category", nil)
	if err != nil {
		return nil, err
	}
	out := api.ListBlogCategoriesOKApplicationJSON{}
	for _, d := range docs {
		m := d.GetData().AsMap()
		out = append(out, api.PublicBlogCategory{Slug: str(m, "slug"), Name: pickLoc(m["name"], locale)})
	}
	return &out, nil
}

func (h *handler) ListBlogTags(ctx context.Context, p api.ListBlogTagsParams) (api.ListBlogTagsRes, error) {
	locale := wantLocale(ctx, p.AcceptLanguage)
	docs, err := h.publicAll(ctx, "blog_tag", nil)
	if err != nil {
		return nil, err
	}
	out := api.ListBlogTagsOKApplicationJSON{}
	for _, d := range docs {
		m := d.GetData().AsMap()
		out = append(out, api.PublicBlogCategory{Slug: str(m, "slug"), Name: pickLoc(m["name"], locale)})
	}
	return &out, nil
}

// publicArticle compose un article publié avec son auteur (lu une fois par appel).
func (h *handler) publicArticle(d *pb.Document, authors map[string]map[string]any, locale string) api.PublicArticle {
	m := d.GetData().AsMap()
	a := api.PublicArticle{Slug: str(m, "slug"), Title: pickLoc(m["title"], locale), Body: pickLoc(m["body"], locale), Kind: api.PublicArticleKind(str(m, "kind")), Tags: []string{}, ProductIds: []uuid.UUID{}}
	if s := pickLoc(m["excerpt"], locale); s != "" {
		a.Excerpt = api.NewOptString(s)
	}
	if s := str(m, "categorySlug"); s != "" {
		a.CategorySlug = api.NewOptString(s)
	}
	if s := str(m, "imageUrl"); s != "" {
		if u, err := url.Parse(s); err == nil {
			a.ImageUrl = api.NewOptURI(*u)
		}
	}
	if n := num(m, "readMin"); n > 0 {
		a.ReadMin = api.NewOptInt64(n)
	}
	if g := str(m, "guide"); g != "" {
		a.Guide = api.NewOptPublicArticleGuide(api.PublicArticleGuide(g))
	}
	a.Dossier, a.Featured = api.NewOptBool(flag(m, "dossier")), api.NewOptBool(flag(m, "featured"))
	for _, t := range asStrings(m["tags"]) {
		a.Tags = append(a.Tags, t)
	}
	for _, id := range asStrings(m["productIds"]) {
		if u, err := uuid.Parse(id); err == nil {
			a.ProductIds = append(a.ProductIds, u)
		}
	}
	a.PublishedAt = d.GetCreatedAt().AsTime()
	if s := str(m, "publishedAt"); s != "" {
		if t, err := parseTime(s); err == nil {
			a.PublishedAt = t
		}
	}
	if au := authors[str(m, "authorId")]; au != nil {
		pa := api.PublicAuthor{Name: str(au, "name")}
		if s := pickLoc(au["role"], locale); s != "" {
			pa.Role = api.NewOptString(s)
		}
		if s := pickLoc(au["bio"], locale); s != "" {
			pa.Bio = api.NewOptString(s)
		}
		if s := str(au, "avatarUrl"); s != "" {
			if u, err := url.Parse(s); err == nil {
				pa.AvatarUrl = api.NewOptURI(*u)
			}
		}
		a.Author = api.NewOptPublicAuthor(pa)
	}
	return a
}

func asStrings(v any) []string {
	l, _ := v.([]any)
	out := make([]string, 0, len(l))
	for _, x := range l {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (h *handler) authorIndex(ctx context.Context) (map[string]map[string]any, error) {
	docs, err := h.publicAll(ctx, "author", nil)
	if err != nil {
		return nil, err
	}
	by := map[string]map[string]any{}
	for _, d := range docs {
		by[d.GetId()] = d.GetData().AsMap()
	}
	return by, nil
}

func (h *handler) ListBlogArticles(ctx context.Context, p api.ListBlogArticlesParams) (api.ListBlogArticlesRes, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	authors, err := h.authorIndex(ctx)
	if err != nil {
		return nil, err
	}
	where := map[string]string{"status": "published"}
	if v, ok := p.Category.Get(); ok && v != "" {
		where["categorySlug"] = v
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := c.ListDocuments(ctx, &pb.ListDocumentsRequest{Kind: "article", Page: page(p.Page, p.PageSize), Where: where})
	if err != nil {
		return nil, err
	}
	out := &api.PublicArticlePage{Items: make([]api.PublicArticle, 0, len(res.GetItems())), Page: toPageInfo(res.GetPage())}
	for _, d := range res.GetItems() {
		out.Items = append(out.Items, h.publicArticle(d, authors, locale))
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].PublishedAt.After(out.Items[j].PublishedAt) })
	return out, nil
}

func (h *handler) GetBlogArticle(ctx context.Context, p api.GetBlogArticleParams) (api.GetBlogArticleRes, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	locale := wantLocale(ctx, p.AcceptLanguage)
	authors, err := h.authorIndex(ctx)
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := c.GetDocument(ctx, &pb.GetDocumentRequest{Kind: "article", IdOrKey: p.Slug})
	if err != nil {
		return nil, err
	}
	if str(res.GetDocument().GetData().AsMap(), "status") != "published" {
		return nil, apperr.NotFound("article introuvable")
	}
	a := h.publicArticle(res.GetDocument(), authors, locale)
	return &a, nil
}

func (h *handler) ListProductReviews(ctx context.Context, p api.ListProductReviewsParams) (api.ListProductReviewsRes, error) {
	return h.reviews(ctx, map[string]string{"status": "published", "productId": p.ID.String()})
}

// ListSiteReviews : témoignages publiés qui ne concernent pas un produit précis (page d'accueil).
func (h *handler) ListSiteReviews(ctx context.Context, _ api.ListSiteReviewsParams) (api.ListSiteReviewsRes, error) {
	return h.reviews(ctx, map[string]string{"status": "published", "productId": ""})
}

func (h *handler) reviews(ctx context.Context, where map[string]string) (*api.PublicReviewList, error) {
	docs, err := h.publicAll(ctx, "review", where)
	if err != nil {
		return nil, err
	}
	out := &api.PublicReviewList{Items: []api.PublicReview{}}
	var sum int64
	for _, d := range docs {
		m := d.GetData().AsMap()
		r := api.PublicReview{ID: uuidOf(d.GetId()), Rating: num(m, "rating"), Title: str(m, "title"), Text: str(m, "text"), Author: str(m, "author"), Verified: flag(m, "verified"), Date: d.GetCreatedAt().AsTime()}
		if s := str(m, "company"); s != "" {
			r.Company = api.NewOptString(s)
		}
		sum += r.Rating
		out.Items = append(out.Items, r)
	}
	out.Count = int64(len(out.Items))
	if out.Count > 0 {
		out.Average = float64(sum*10/out.Count) / 10
	}
	return out, nil
}

// ── formulaires publics : le contenu est enregistré, jamais renvoyé ──

func (h *handler) submit(ctx context.Context, kind string, in any, defaults map[string]any) error {
	c, err := h.content()
	if err != nil {
		return err
	}
	data, err := inputStruct(in)
	if err != nil {
		return err
	}
	for k, v := range defaults {
		data.Fields[k], _ = structpb.NewValue(v)
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	_, err = c.CreateDocument(ctx, &pb.CreateDocumentRequest{Kind: kind, Data: data})
	return err
}

func (h *handler) SubmitContact(ctx context.Context, req *api.ContactSubmission) (api.SubmitContactRes, error) {
	if err := h.submit(ctx, "contact_message", req, map[string]any{"status": "new"}); err != nil {
		return nil, err
	}
	return &api.SubmitContactAccepted{}, nil
}

func (h *handler) SubmitQuoteRequest(ctx context.Context, req *api.QuoteSubmission) (api.SubmitQuoteRequestRes, error) {
	if err := h.submit(ctx, "quote_request", req, map[string]any{"status": "new"}); err != nil {
		return nil, err
	}
	return &api.SubmitQuoteRequestAccepted{}, nil
}

// SubscribeNewsletter : réponse identique que l'adresse soit déjà inscrite ou non (aucune fuite d'existence).
func (h *handler) SubscribeNewsletter(ctx context.Context, req *api.NewsletterSubmission) (api.SubscribeNewsletterRes, error) {
	err := h.submit(ctx, "newsletter_subscriber", req, map[string]any{"status": "subscribed"})
	if err != nil && !strings.Contains(err.Error(), "ALREADY_EXISTS") && !isConflict(err) {
		return nil, err
	}
	return &api.SubscribeNewsletterAccepted{}, nil
}

func (h *handler) ValidatePromoCode(ctx context.Context, req *api.PromoCheckRequest) (api.ValidatePromoCodeRes, error) {
	c, err := h.content()
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	res, err := c.GetDocument(ctx, &pb.GetDocumentRequest{Kind: "promo_code", IdOrKey: strings.ToUpper(strings.TrimSpace(req.Code))})
	if err != nil {
		if isNotFound(err) {
			return &api.PromoCheck{Status: api.PromoCheckStatusInvalid}, nil
		}
		return nil, err
	}
	m := res.GetDocument().GetData().AsMap()
	now := h.srv.now()
	if !flag(m, "active") {
		return &api.PromoCheck{Status: api.PromoCheckStatusInvalid}, nil
	}
	if s := str(m, "startsAt"); s != "" {
		if t, err := parseTime(s); err == nil && now.Before(t) {
			return &api.PromoCheck{Status: api.PromoCheckStatusInvalid}, nil
		}
	}
	if s := str(m, "expiresAt"); s != "" {
		if t, err := parseTime(s); err == nil && now.After(t) {
			return &api.PromoCheck{Status: api.PromoCheckStatusExpired}, nil
		}
	}
	out := &api.PromoCheck{Status: api.PromoCheckStatusValid, Type: api.NewOptPromoCheckType(api.PromoCheckType(str(m, "type"))), Value: api.NewOptInt64(num(m, "value"))}
	if min := num(m, "minSubtotal"); min > 0 {
		out.MinSubtotal = api.NewOptInt64(min)
		if req.Subtotal < min {
			out.Status = api.PromoCheckStatusMinimum
		}
	}
	return out, nil
}
