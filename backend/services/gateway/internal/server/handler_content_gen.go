// Code généré depuis la spécification des collections : relais HTTP → service content pour les routes d'administration.
// La permission de chaque route est contrôlée avant (security.go) ; ici on ne fait que traduire.

package server

import (
	"context"

	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

func (h *handler) AdminFaqList(ctx context.Context, p api.AdminFaqListParams) (api.AdminFaqListRes, error) {
	docs, pg, err := h.contentList(ctx, "faq", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsFaqPage{Items: make([]api.CmsFaq, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsFaq](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminFaqCreate(ctx context.Context, req *api.CmsFaqInput) (api.AdminFaqCreateRes, error) {
	d, err := h.contentCreate(ctx, "faq", req, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsFaq](d)
}

func (h *handler) AdminFaqGet(ctx context.Context, p api.AdminFaqGetParams) (api.AdminFaqGetRes, error) {
	d, err := h.contentGet(ctx, "faq", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsFaq](d)
}

func (h *handler) AdminFaqUpdate(ctx context.Context, req *api.CmsFaqInput, p api.AdminFaqUpdateParams) (api.AdminFaqUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "faq", p.ID.String(), req, true, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsFaq](d)
}

func (h *handler) AdminFaqDelete(ctx context.Context, p api.AdminFaqDeleteParams) (api.AdminFaqDeleteRes, error) {
	if err := h.contentDelete(ctx, "faq", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminFaqDeleteNoContent{}, nil
}

func (h *handler) AdminPagesList(ctx context.Context, p api.AdminPagesListParams) (api.AdminPagesListRes, error) {
	docs, pg, err := h.contentList(ctx, "page", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsPagePage{Items: make([]api.CmsPage, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsPage](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminPagesCreate(ctx context.Context, req *api.CmsPageInput) (api.AdminPagesCreateRes, error) {
	d, err := h.contentCreate(ctx, "page", req, map[string]any{"published": false})
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPage](d)
}

func (h *handler) AdminPagesGet(ctx context.Context, p api.AdminPagesGetParams) (api.AdminPagesGetRes, error) {
	d, err := h.contentGet(ctx, "page", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPage](d)
}

func (h *handler) AdminPagesUpdate(ctx context.Context, req *api.CmsPageInput, p api.AdminPagesUpdateParams) (api.AdminPagesUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "page", p.ID.String(), req, true, []string{"published"})
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPage](d)
}

func (h *handler) AdminPagesPublish(ctx context.Context, p api.AdminPagesPublishParams) (api.AdminPagesPublishRes, error) {
	d, err := h.contentUpdate(ctx, "page", p.ID.String(), map[string]any{"published": true}, false, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPage](d)
}

func (h *handler) AdminPagesUnpublish(ctx context.Context, p api.AdminPagesUnpublishParams) (api.AdminPagesUnpublishRes, error) {
	d, err := h.contentUpdate(ctx, "page", p.ID.String(), map[string]any{"published": false}, false, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPage](d)
}

func (h *handler) AdminPagesDelete(ctx context.Context, p api.AdminPagesDeleteParams) (api.AdminPagesDeleteRes, error) {
	if err := h.contentDelete(ctx, "page", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminPagesDeleteNoContent{}, nil
}

func (h *handler) AdminAuthorsList(ctx context.Context, p api.AdminAuthorsListParams) (api.AdminAuthorsListRes, error) {
	docs, pg, err := h.contentList(ctx, "author", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsAuthorPage{Items: make([]api.CmsAuthor, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsAuthor](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminAuthorsCreate(ctx context.Context, req *api.CmsAuthorInput) (api.AdminAuthorsCreateRes, error) {
	d, err := h.contentCreate(ctx, "author", req, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsAuthor](d)
}

func (h *handler) AdminAuthorsGet(ctx context.Context, p api.AdminAuthorsGetParams) (api.AdminAuthorsGetRes, error) {
	d, err := h.contentGet(ctx, "author", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsAuthor](d)
}

func (h *handler) AdminAuthorsUpdate(ctx context.Context, req *api.CmsAuthorInput, p api.AdminAuthorsUpdateParams) (api.AdminAuthorsUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "author", p.ID.String(), req, true, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsAuthor](d)
}

func (h *handler) AdminAuthorsDelete(ctx context.Context, p api.AdminAuthorsDeleteParams) (api.AdminAuthorsDeleteRes, error) {
	if err := h.contentDelete(ctx, "author", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminAuthorsDeleteNoContent{}, nil
}

func (h *handler) AdminBlogCategoriesList(ctx context.Context, p api.AdminBlogCategoriesListParams) (api.AdminBlogCategoriesListRes, error) {
	docs, pg, err := h.contentList(ctx, "blog_category", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsBlogCategoryPage{Items: make([]api.CmsBlogCategory, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsBlogCategory](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminBlogCategoriesCreate(ctx context.Context, req *api.CmsBlogCategoryInput) (api.AdminBlogCategoriesCreateRes, error) {
	d, err := h.contentCreate(ctx, "blog_category", req, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsBlogCategory](d)
}

func (h *handler) AdminBlogCategoriesGet(ctx context.Context, p api.AdminBlogCategoriesGetParams) (api.AdminBlogCategoriesGetRes, error) {
	d, err := h.contentGet(ctx, "blog_category", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsBlogCategory](d)
}

func (h *handler) AdminBlogCategoriesUpdate(ctx context.Context, req *api.CmsBlogCategoryInput, p api.AdminBlogCategoriesUpdateParams) (api.AdminBlogCategoriesUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "blog_category", p.ID.String(), req, true, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsBlogCategory](d)
}

func (h *handler) AdminBlogCategoriesDelete(ctx context.Context, p api.AdminBlogCategoriesDeleteParams) (api.AdminBlogCategoriesDeleteRes, error) {
	if err := h.contentDelete(ctx, "blog_category", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminBlogCategoriesDeleteNoContent{}, nil
}

func (h *handler) AdminBlogTagsList(ctx context.Context, p api.AdminBlogTagsListParams) (api.AdminBlogTagsListRes, error) {
	docs, pg, err := h.contentList(ctx, "blog_tag", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsBlogTagPage{Items: make([]api.CmsBlogTag, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsBlogTag](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminBlogTagsCreate(ctx context.Context, req *api.CmsBlogTagInput) (api.AdminBlogTagsCreateRes, error) {
	d, err := h.contentCreate(ctx, "blog_tag", req, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsBlogTag](d)
}

func (h *handler) AdminBlogTagsGet(ctx context.Context, p api.AdminBlogTagsGetParams) (api.AdminBlogTagsGetRes, error) {
	d, err := h.contentGet(ctx, "blog_tag", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsBlogTag](d)
}

func (h *handler) AdminBlogTagsUpdate(ctx context.Context, req *api.CmsBlogTagInput, p api.AdminBlogTagsUpdateParams) (api.AdminBlogTagsUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "blog_tag", p.ID.String(), req, true, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsBlogTag](d)
}

func (h *handler) AdminBlogTagsDelete(ctx context.Context, p api.AdminBlogTagsDeleteParams) (api.AdminBlogTagsDeleteRes, error) {
	if err := h.contentDelete(ctx, "blog_tag", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminBlogTagsDeleteNoContent{}, nil
}

func (h *handler) AdminArticlesList(ctx context.Context, p api.AdminArticlesListParams) (api.AdminArticlesListRes, error) {
	docs, pg, err := h.contentList(ctx, "article", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsArticlePage{Items: make([]api.CmsArticle, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsArticle](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminArticlesCreate(ctx context.Context, req *api.CmsArticleInput) (api.AdminArticlesCreateRes, error) {
	d, err := h.contentCreate(ctx, "article", req, map[string]any{"status": "draft"})
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsArticle](d)
}

func (h *handler) AdminArticlesGet(ctx context.Context, p api.AdminArticlesGetParams) (api.AdminArticlesGetRes, error) {
	d, err := h.contentGet(ctx, "article", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsArticle](d)
}

func (h *handler) AdminArticlesUpdate(ctx context.Context, req *api.CmsArticleInput, p api.AdminArticlesUpdateParams) (api.AdminArticlesUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "article", p.ID.String(), req, true, []string{"status"})
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsArticle](d)
}

func (h *handler) AdminArticlesPublish(ctx context.Context, p api.AdminArticlesPublishParams) (api.AdminArticlesPublishRes, error) {
	d, err := h.contentUpdate(ctx, "article", p.ID.String(), map[string]any{"status": "published"}, false, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsArticle](d)
}

func (h *handler) AdminArticlesUnpublish(ctx context.Context, p api.AdminArticlesUnpublishParams) (api.AdminArticlesUnpublishRes, error) {
	d, err := h.contentUpdate(ctx, "article", p.ID.String(), map[string]any{"status": "draft"}, false, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsArticle](d)
}

func (h *handler) AdminArticlesDelete(ctx context.Context, p api.AdminArticlesDeleteParams) (api.AdminArticlesDeleteRes, error) {
	if err := h.contentDelete(ctx, "article", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminArticlesDeleteNoContent{}, nil
}

func (h *handler) AdminDeliveryZonesList(ctx context.Context, p api.AdminDeliveryZonesListParams) (api.AdminDeliveryZonesListRes, error) {
	docs, pg, err := h.contentList(ctx, "shipping_zone", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsShippingZonePage{Items: make([]api.CmsShippingZone, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsShippingZone](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminDeliveryZonesCreate(ctx context.Context, req *api.CmsShippingZoneInput) (api.AdminDeliveryZonesCreateRes, error) {
	d, err := h.contentCreate(ctx, "shipping_zone", req, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsShippingZone](d)
}

func (h *handler) AdminDeliveryZonesGet(ctx context.Context, p api.AdminDeliveryZonesGetParams) (api.AdminDeliveryZonesGetRes, error) {
	d, err := h.contentGet(ctx, "shipping_zone", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsShippingZone](d)
}

func (h *handler) AdminDeliveryZonesUpdate(ctx context.Context, req *api.CmsShippingZoneInput, p api.AdminDeliveryZonesUpdateParams) (api.AdminDeliveryZonesUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "shipping_zone", p.ID.String(), req, true, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsShippingZone](d)
}

func (h *handler) AdminDeliveryZonesDelete(ctx context.Context, p api.AdminDeliveryZonesDeleteParams) (api.AdminDeliveryZonesDeleteRes, error) {
	if err := h.contentDelete(ctx, "shipping_zone", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminDeliveryZonesDeleteNoContent{}, nil
}

func (h *handler) AdminPromoCodesList(ctx context.Context, p api.AdminPromoCodesListParams) (api.AdminPromoCodesListRes, error) {
	docs, pg, err := h.contentList(ctx, "promo_code", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsPromoCodePage{Items: make([]api.CmsPromoCode, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsPromoCode](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminPromoCodesCreate(ctx context.Context, req *api.CmsPromoCodeInput) (api.AdminPromoCodesCreateRes, error) {
	d, err := h.contentCreate(ctx, "promo_code", req, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPromoCode](d)
}

func (h *handler) AdminPromoCodesGet(ctx context.Context, p api.AdminPromoCodesGetParams) (api.AdminPromoCodesGetRes, error) {
	d, err := h.contentGet(ctx, "promo_code", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPromoCode](d)
}

func (h *handler) AdminPromoCodesUpdate(ctx context.Context, req *api.CmsPromoCodeInput, p api.AdminPromoCodesUpdateParams) (api.AdminPromoCodesUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "promo_code", p.ID.String(), req, true, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsPromoCode](d)
}

func (h *handler) AdminPromoCodesDelete(ctx context.Context, p api.AdminPromoCodesDeleteParams) (api.AdminPromoCodesDeleteRes, error) {
	if err := h.contentDelete(ctx, "promo_code", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminPromoCodesDeleteNoContent{}, nil
}

func (h *handler) AdminReviewsList(ctx context.Context, p api.AdminReviewsListParams) (api.AdminReviewsListRes, error) {
	docs, pg, err := h.contentList(ctx, "review", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsReviewPage{Items: make([]api.CmsReview, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsReview](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminReviewsCreate(ctx context.Context, req *api.CmsReviewInput) (api.AdminReviewsCreateRes, error) {
	d, err := h.contentCreate(ctx, "review", req, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsReview](d)
}

func (h *handler) AdminReviewsGet(ctx context.Context, p api.AdminReviewsGetParams) (api.AdminReviewsGetRes, error) {
	d, err := h.contentGet(ctx, "review", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsReview](d)
}

func (h *handler) AdminReviewsUpdate(ctx context.Context, req *api.CmsReviewInput, p api.AdminReviewsUpdateParams) (api.AdminReviewsUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "review", p.ID.String(), req, true, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsReview](d)
}

func (h *handler) AdminReviewsDelete(ctx context.Context, p api.AdminReviewsDeleteParams) (api.AdminReviewsDeleteRes, error) {
	if err := h.contentDelete(ctx, "review", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminReviewsDeleteNoContent{}, nil
}

func (h *handler) AdminContactMessagesList(ctx context.Context, p api.AdminContactMessagesListParams) (api.AdminContactMessagesListRes, error) {
	docs, pg, err := h.contentList(ctx, "contact_message", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsContactMessagePage{Items: make([]api.CmsContactMessage, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsContactMessage](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminContactMessagesGet(ctx context.Context, p api.AdminContactMessagesGetParams) (api.AdminContactMessagesGetRes, error) {
	d, err := h.contentGet(ctx, "contact_message", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsContactMessage](d)
}

func (h *handler) AdminContactMessagesUpdate(ctx context.Context, req *api.CmsContactMessageUpdate, p api.AdminContactMessagesUpdateParams) (api.AdminContactMessagesUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "contact_message", p.ID.String(), req, false, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsContactMessage](d)
}

func (h *handler) AdminContactMessagesDelete(ctx context.Context, p api.AdminContactMessagesDeleteParams) (api.AdminContactMessagesDeleteRes, error) {
	if err := h.contentDelete(ctx, "contact_message", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminContactMessagesDeleteNoContent{}, nil
}

func (h *handler) AdminNewsletterSubscribersList(ctx context.Context, p api.AdminNewsletterSubscribersListParams) (api.AdminNewsletterSubscribersListRes, error) {
	docs, pg, err := h.contentList(ctx, "newsletter_subscriber", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsNewsletterSubscriberPage{Items: make([]api.CmsNewsletterSubscriber, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsNewsletterSubscriber](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminNewsletterSubscribersGet(ctx context.Context, p api.AdminNewsletterSubscribersGetParams) (api.AdminNewsletterSubscribersGetRes, error) {
	d, err := h.contentGet(ctx, "newsletter_subscriber", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsNewsletterSubscriber](d)
}

func (h *handler) AdminNewsletterSubscribersUpdate(ctx context.Context, req *api.CmsNewsletterSubscriberUpdate, p api.AdminNewsletterSubscribersUpdateParams) (api.AdminNewsletterSubscribersUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "newsletter_subscriber", p.ID.String(), req, false, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsNewsletterSubscriber](d)
}

func (h *handler) AdminNewsletterSubscribersDelete(ctx context.Context, p api.AdminNewsletterSubscribersDeleteParams) (api.AdminNewsletterSubscribersDeleteRes, error) {
	if err := h.contentDelete(ctx, "newsletter_subscriber", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminNewsletterSubscribersDeleteNoContent{}, nil
}

func (h *handler) AdminQuoteRequestsList(ctx context.Context, p api.AdminQuoteRequestsListParams) (api.AdminQuoteRequestsListRes, error) {
	docs, pg, err := h.contentList(ctx, "quote_request", p.Q.Or(""), nil, p.Page, p.PageSize)
	if err != nil {
		return nil, err
	}
	out := &api.CmsQuoteRequestPage{Items: make([]api.CmsQuoteRequest, 0, len(docs)), Page: toPageInfo(pg)}
	for _, d := range docs {
		v, err := decodeDoc[api.CmsQuoteRequest](d)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *v)
	}
	return out, nil
}

func (h *handler) AdminQuoteRequestsGet(ctx context.Context, p api.AdminQuoteRequestsGetParams) (api.AdminQuoteRequestsGetRes, error) {
	d, err := h.contentGet(ctx, "quote_request", p.ID.String())
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsQuoteRequest](d)
}

func (h *handler) AdminQuoteRequestsUpdate(ctx context.Context, req *api.CmsQuoteRequestUpdate, p api.AdminQuoteRequestsUpdateParams) (api.AdminQuoteRequestsUpdateRes, error) {
	d, err := h.contentUpdate(ctx, "quote_request", p.ID.String(), req, false, nil)
	if err != nil {
		return nil, err
	}
	return decodeDoc[api.CmsQuoteRequest](d)
}

func (h *handler) AdminQuoteRequestsDelete(ctx context.Context, p api.AdminQuoteRequestsDeleteParams) (api.AdminQuoteRequestsDeleteRes, error) {
	if err := h.contentDelete(ctx, "quote_request", p.ID.String()); err != nil {
		return nil, err
	}
	return &api.AdminQuoteRequestsDeleteNoContent{}, nil
}

func (h *handler) AdminSiteSettingsGet(ctx context.Context) (api.AdminSiteSettingsGetRes, error) {
	return settingsGetAs[api.CmsSiteSettings](h, ctx, "site")
}

func (h *handler) AdminSiteSettingsPut(ctx context.Context, req *api.CmsSiteSettings) (api.AdminSiteSettingsPutRes, error) {
	return settingsPutAs[api.CmsSiteSettings](h, ctx, "site", req)
}

func (h *handler) GetSiteSettings(ctx context.Context) (api.GetSiteSettingsRes, error) {
	return settingsGetAs[api.CmsSiteSettings](h, ctx, "site")
}

func (h *handler) AdminCommerceSettingsGet(ctx context.Context) (api.AdminCommerceSettingsGetRes, error) {
	return settingsGetAs[api.CmsCommerceSettings](h, ctx, "commerce")
}

func (h *handler) AdminCommerceSettingsPut(ctx context.Context, req *api.CmsCommerceSettings) (api.AdminCommerceSettingsPutRes, error) {
	return settingsPutAs[api.CmsCommerceSettings](h, ctx, "commerce", req)
}

func (h *handler) GetCommerceSettings(ctx context.Context) (api.GetCommerceSettingsRes, error) {
	return settingsGetAs[api.CmsCommerceSettings](h, ctx, "commerce")
}
