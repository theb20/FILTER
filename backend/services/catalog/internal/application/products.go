package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
)

const lifecycleConflictCode = apperr.CodeConflict

func productFail(err error) error {
	switch {
	case errors.Is(err, ErrSKUTaken):
		return conflict("ce SKU est déjà utilisé")
	case errors.Is(err, ErrRefTaken):
		return conflict("cette référence constructeur existe déjà pour cette marque")
	case errors.Is(err, ErrBrandMissing):
		return invalid("/brandId", "marque introuvable")
	case errors.Is(err, ErrSectorMissing):
		return invalid("/sectorIds", "secteur introuvable")
	case errors.Is(err, ErrStale):
		return conflict("ce produit a été modifié par quelqu'un d'autre : rechargez la fiche avant d'enregistrer")
	case errors.Is(err, ErrFrozen):
		return invalid("/sku", "le SKU et l'adresse sont figés après la première publication")
	}
	return fail(err)
}

// ptrEq compare deux valeurs pointées.
func snapshot(p *domain.Product) map[string]any {
	m := map[string]any{
		"sku": p.SKU, "slug": p.Slug, "name": loc(p.Name), "description": loc(p.Description), "brandId": strOrNil(p.BrandID), "categoryId": p.CategoryID,
		"manufacturerRef": strOrNil(p.ManufacturerRef), "status": p.Status, "fulfillmentMode": p.FulfillmentMode, "leadTimeDays": intOrNil(p.LeadTimeDays),
		"priceMode": p.PriceMode, "priceAmount": int64OrNil(p.PriceAmount), "gtin": strOrNil(p.GTIN), "saleUnit": strOrNil(p.SaleUnit),
		"minOrderQuantity": intOrNil(p.MinOrderQuantity), "weightGrams": intOrNil(p.WeightGrams), "lengthMm": intOrNil(p.LengthMM), "widthMm": intOrNil(p.WidthMM), "heightMm": intOrNil(p.HeightMM),
	}
	attrs := map[string]any{}
	for _, a := range p.Attributes {
		switch {
		case a.Text != nil:
			attrs[a.Key] = *a.Text
		case a.Number != nil:
			attrs[a.Key] = *a.Number
		case a.Bool != nil:
			attrs[a.Key] = *a.Bool
		}
	}
	m["attributes"] = attrs
	sec := append([]string{}, p.SectorIDs...)
	sort.Strings(sec)
	m["sectorIds"] = anySlice(sec)
	media := make([]any, 0, len(p.Media))
	for _, x := range p.Media {
		media = append(media, map[string]any{"url": x.URL, "alt": strOrNil(x.Alt)})
	}
	m["media"] = media
	xr := make([]any, 0, len(p.CrossRefs))
	for _, x := range p.CrossRefs {
		xr = append(xr, map[string]any{"brand": x.Brand, "number": x.Number, "kind": x.Kind})
	}
	m["crossReferences"] = xr
	return m
}

func anySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return int64(*p)
}

func int64OrNil(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// normalizeJSON ramène toute valeur à sa forme JSON (nombres en float64, slices en []any) pour des comparaisons fiables.
func normalizeJSON(v any) any {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = normalizeJSON(e)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalizeJSON(e)
		}
		return out
	}
	return v
}

// diff : par champ modifié, {before, after, origin}. `before == nil` : création (tout champ non vide est « nouveau »).
func diff(before, after map[string]any, origin string) map[string]any {
	out := map[string]any{}
	for k, a := range after {
		var b any
		if before != nil {
			b = before[k]
		}
		na, nb := normalizeJSON(a), normalizeJSON(b)
		if reflect.DeepEqual(na, nb) {
			continue
		}
		if before == nil && isEmpty(na) {
			continue
		}
		out[k] = map[string]any{"before": nb, "after": na, "origin": origin}
	}
	return out
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case map[string]any:
		return len(x) == 0
	case []any:
		return len(x) == 0
	}
	return false
}

func mergeOrigins(cur map[string]string, changes map[string]any, origin string) map[string]string {
	out := map[string]string{}
	for k, v := range cur {
		out[k] = v
	}
	for k := range changes {
		out[k] = origin
	}
	return out
}

func (s *Service) validateProduct(ctx context.Context, p *domain.Product, requirePublishable bool) error {
	if err := p.ValidateShape(); err != nil {
		return err
	}
	defs, err := s.Store.AttributeDefsForCategory(ctx, p.CategoryID)
	if err != nil {
		return fail(err)
	}
	if err := p.CheckAttributes(defs); err != nil {
		return err
	}
	if requirePublishable {
		if probs := p.PublishProblems(defs); len(probs) > 0 {
			return apperr.Validation("publication impossible : la fiche est incomplète", probs...)
		}
	}
	return nil
}

func (s *Service) uniqueSlug(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "produit"
	}
	slug := base
	for i := 2; i < 1000; i++ {
		ok, err := s.Store.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !ok {
			return slug, nil
		}
		suffix := fmt.Sprintf("-%d", i)
		b := base
		if len(b)+len(suffix) > 80 {
			b = strings.Trim(b[:80-len(suffix)], "-")
		}
		slug = b + suffix
	}
	return "", errors.New("aucun identifiant d'URL disponible")
}

// CreateProduct : toujours en brouillon. SKU et slug sont générés s'ils sont vides.
func (s *Service) CreateProduct(ctx context.Context, a Actor, in domain.Product, origin string) (*domain.Product, error) {
	in.ID, in.Status, in.Version, in.PublishedAt = s.NewID(), domain.StatusDraft, 1, nil
	in.Currency = domain.BaseCurrency
	if origin == "" {
		origin = domain.OriginManual
	}
	if err := s.validateProduct(ctx, &in, false); err != nil {
		return nil, err
	}
	now := s.Now()
	in.CreatedAt, in.UpdatedAt = now, now
	for i := range in.Media {
		in.Media[i].ID = s.NewID()
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		if in.SKU == "" {
			sku, err := s.Store.NextSKU(ctx)
			if err != nil {
				return err
			}
			in.SKU = sku
		}
		if in.Slug == "" {
			slug, err := s.uniqueSlug(ctx, domain.Slugify(in.Name[domain.DefaultLanguage]))
			if err != nil {
				return err
			}
			in.Slug = slug
		}
		ch := diff(nil, snapshot(&in), origin)
		in.Origins = mergeOrigins(nil, ch, origin)
		if err := s.Store.CreateProduct(ctx, &in); err != nil {
			return productFail(err)
		}
		if err := s.Store.AddProductVersion(ctx, in.ID, domain.ProductVersion{Version: 1, At: now, ActorID: actorPtr(a), Action: "create", Changes: ch}); err != nil {
			return err
		}
		return s.audit(ctx, a, "product.create", "product", in.ID, nil, snapshot(&in))
	})
	if err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, in.ID)
}

func actorPtr(a Actor) *string {
	if a.ID == "" {
		return nil
	}
	return &a.ID
}

// UpdateProduct : remplace la fiche. SKU et slug vides = inchangés ; figés après publication. `expected` protège des écritures concurrentes.
func (s *Service) UpdateProduct(ctx context.Context, a Actor, id string, in domain.Product, expected *int, origin string) (*domain.Product, error) {
	if origin == "" {
		origin = domain.OriginManual
	}
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.Store.ProductByID(ctx, id, true)
		if err != nil {
			return err
		}
		if expected != nil && *expected != cur.Version {
			return productFail(ErrStale)
		}
		if cur.Status == domain.StatusArchived {
			return conflict("ce produit est archivé : restaurez-le avant de le modifier")
		}
		in.ID, in.Status, in.Version, in.PublishedAt, in.CreatedAt, in.Currency = cur.ID, cur.Status, cur.Version, cur.PublishedAt, cur.CreatedAt, cur.Currency
		if in.SKU == "" {
			in.SKU = cur.SKU
		}
		if in.Slug == "" {
			in.Slug = cur.Slug
		}
		if err := s.validateProduct(ctx, &in, cur.Status == domain.StatusActive); err != nil {
			return err
		}
		if cur.PublishedAt != nil && (in.SKU != cur.SKU || in.Slug != cur.Slug) {
			return productFail(ErrFrozen)
		}
		for i := range in.Media {
			in.Media[i].ID = s.NewID()
		}
		before, after := snapshot(cur), snapshot(&in)
		ch := diff(before, after, origin)
		if len(ch) == 0 {
			return nil // rien n'a changé : pas de nouvelle version
		}
		in.Origins = mergeOrigins(cur.Origins, ch, origin)
		in.UpdatedAt = s.Now()
		if err := s.Store.UpdateProduct(ctx, &in); err != nil {
			return productFail(err)
		}
		if err := s.Store.AddProductVersion(ctx, id, domain.ProductVersion{Version: in.Version, At: in.UpdatedAt, ActorID: actorPtr(a), Action: "update", Changes: ch}); err != nil {
			return err
		}
		return s.audit(ctx, a, "product.update", "product", id, before, after)
	})
	if err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, id)
}

func (s *Service) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	p, err := s.Store.ProductByID(ctx, id, false)
	if err != nil {
		return nil, fail(err)
	}
	return p, nil
}

func (s *Service) ListProducts(ctx context.Context, f ProductFilter, page, size int) (*Page[*domain.Product], error) {
	page, size = paging(page, size)
	f.Query = strings.TrimSpace(f.Query)
	items, total, err := s.Store.ListProducts(ctx, f, size, (page-1)*size)
	if err != nil {
		return nil, fail(err)
	}
	return pageOf(items, total, page, size), nil
}

func (s *Service) ProductVersions(ctx context.Context, id string) ([]domain.ProductVersion, error) {
	if _, err := s.GetProduct(ctx, id); err != nil {
		return nil, err
	}
	v, err := s.Store.ListProductVersions(ctx, id, 100)
	if err != nil {
		return nil, fail(err)
	}
	return v, nil
}

// transition applique un changement d'état sous verrou ; `check` peut refuser.
func (s *Service) transition(ctx context.Context, a Actor, id, action string, from []string, to string, publish bool) (*domain.Product, error) {
	err := s.tx(ctx, func(ctx context.Context) error {
		cur, err := s.Store.ProductByID(ctx, id, true)
		if err != nil {
			return err
		}
		ok := false
		for _, f := range from {
			ok = ok || cur.Status == f
		}
		if !ok {
			return apperr.New(apperrConflictKind, lifecycleConflictCode, fmt.Sprintf("action impossible : le produit est « %s »", statusLabel(cur.Status)))
		}
		next := *cur
		next.Status = to
		if publish {
			defs, err := s.Store.AttributeDefsForCategory(ctx, cur.CategoryID)
			if err != nil {
				return err
			}
			if probs := next.PublishProblems(defs); len(probs) > 0 {
				return apperr.Validation("publication impossible : la fiche est incomplète", probs...)
			}
			if next.PublishedAt == nil {
				now := s.Now()
				next.PublishedAt = &now
			}
		}
		next.UpdatedAt = s.Now()
		ch := map[string]any{"status": map[string]any{"before": cur.Status, "after": to, "origin": domain.OriginManual}}
		next.Origins = mergeOrigins(cur.Origins, ch, domain.OriginManual)
		if err := s.Store.UpdateProduct(ctx, &next); err != nil {
			return productFail(err)
		}
		if err := s.Store.AddProductVersion(ctx, id, domain.ProductVersion{Version: next.Version, At: next.UpdatedAt, ActorID: actorPtr(a), Action: action, Changes: ch}); err != nil {
			return err
		}
		return s.audit(ctx, a, "product."+action, "product", id, map[string]any{"status": cur.Status}, map[string]any{"status": to})
	})
	if err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, id)
}

func statusLabel(s string) string {
	switch s {
	case domain.StatusDraft:
		return "brouillon"
	case domain.StatusReview:
		return "à valider"
	case domain.StatusActive:
		return "publié"
	case domain.StatusArchived:
		return "archivé"
	}
	return s
}

func (s *Service) SubmitProductReview(ctx context.Context, a Actor, id string) (*domain.Product, error) {
	return s.transition(ctx, a, id, "submit_review", []string{domain.StatusDraft}, domain.StatusReview, false)
}

// PublishProduct : la permission catalog:publish est contrôlée par le gateway.
func (s *Service) PublishProduct(ctx context.Context, a Actor, id string) (*domain.Product, error) {
	return s.transition(ctx, a, id, "publish", []string{domain.StatusDraft, domain.StatusReview}, domain.StatusActive, true)
}

func (s *Service) ArchiveProduct(ctx context.Context, a Actor, id string) (*domain.Product, error) {
	return s.transition(ctx, a, id, "archive", []string{domain.StatusDraft, domain.StatusReview, domain.StatusActive}, domain.StatusArchived, false)
}

func (s *Service) RestoreProduct(ctx context.Context, a Actor, id string) (*domain.Product, error) {
	return s.transition(ctx, a, id, "restore", []string{domain.StatusArchived}, domain.StatusDraft, false)
}

// DeleteProduct : un produit jamais publié est supprimé ; sinon il est archivé (les commandes et devis y renvoient).
func (s *Service) DeleteProduct(ctx context.Context, a Actor, id string) error {
	cur, err := s.GetProduct(ctx, id)
	if err != nil {
		return err
	}
	if cur.PublishedAt != nil {
		if cur.Status == domain.StatusArchived {
			return nil
		}
		_, err := s.ArchiveProduct(ctx, a, id)
		return err
	}
	return s.tx(ctx, func(ctx context.Context) error {
		if err := s.Store.DeleteProduct(ctx, id); err != nil {
			return productFail(err)
		}
		return s.audit(ctx, a, "product.delete", "product", id, snapshot(cur), nil)
	})
}
