package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/services/catalog/internal/application"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
)

// mapProductErr traduit les violations propres aux produits.
func mapProductErr(err error) error {
	pe := pgErr(err)
	if pe == nil {
		return err
	}
	switch {
	case pe.Code == "23505" && pe.ConstraintName == "products_sku_key":
		return application.ErrSKUTaken
	case pe.Code == "23505" && pe.ConstraintName == "products_slug_key":
		return application.ErrSlugTaken
	case pe.Code == "23505" && pe.ConstraintName == "products_brand_ref_key":
		return application.ErrRefTaken
	case pe.Code == "23514" && pe.Message == "product_identifiers_frozen":
		return application.ErrFrozen
	case pe.Code == "23503" && strings.Contains(pe.ConstraintName, "brand"):
		return application.ErrBrandMissing
	case pe.Code == "23503" && strings.Contains(pe.ConstraintName, "category"):
		return application.ErrCategoryMissing
	case pe.Code == "23503" && strings.Contains(pe.ConstraintName, "sector"):
		return application.ErrSectorMissing
	}
	return err
}

func (s *Store) NextSKU(ctx context.Context) (string, error) {
	var n int64
	if err := s.q(ctx).QueryRow(ctx, `SELECT nextval('product_sku_seq')`).Scan(&n); err != nil {
		return "", err
	}
	return fmt.Sprintf("FF-%06d", n), nil
}

func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var ok bool
	err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE slug = $1)`, slug).Scan(&ok)
	return ok, err
}

func (s *Store) ProductCurrency(ctx context.Context) (string, error) {
	var c string
	err := s.q(ctx).QueryRow(ctx, `SELECT base_currency FROM catalog_settings`).Scan(&c)
	return c, err
}

func attrsJSON(as []domain.AttrValue) string {
	m := map[string]any{}
	for _, a := range as {
		switch {
		case a.Text != nil:
			m[a.Key] = *a.Text
		case a.Number != nil:
			m[a.Key] = *a.Number
		case a.Bool != nil:
			m[a.Key] = *a.Bool
		}
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func originsJSON(o map[string]string) string {
	if o == nil {
		o = map[string]string{}
	}
	b, _ := json.Marshal(o)
	return string(b)
}

func refNorm(p *string) *string {
	if p == nil {
		return nil
	}
	n := domain.NormalizeRef(*p)
	if n == "" {
		return nil
	}
	return &n
}

func (s *Store) writeChildren(ctx context.Context, p *domain.Product) error {
	q := s.q(ctx)
	for _, t := range []string{"product_sectors", "product_media", "product_cross_refs"} {
		if _, err := q.Exec(ctx, `DELETE FROM `+t+` WHERE product_id = $1`, p.ID); err != nil {
			return err
		}
	}
	for _, id := range p.SectorIDs {
		if _, err := q.Exec(ctx, `INSERT INTO product_sectors (product_id, sector_id) VALUES ($1,$2)`, p.ID, id); err != nil {
			return err
		}
	}
	for i, m := range p.Media {
		if _, err := q.Exec(ctx, `INSERT INTO product_media (id, product_id, url, alt, sort_order) VALUES ($1,$2,$3,$4,$5)`, m.ID, p.ID, m.URL, m.Alt, i); err != nil {
			return err
		}
	}
	for _, x := range p.CrossRefs {
		if _, err := q.Exec(ctx, `INSERT INTO product_cross_refs (id, product_id, brand, number, number_norm, kind) VALUES (gen_random_uuid(),$1,$2,$3,$4,$5)`, p.ID, x.Brand, x.Number, domain.NormalizeRef(x.Number), x.Kind); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateProduct(ctx context.Context, p *domain.Product) error {
	ref := p.ManufacturerRef
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO products (id, sku, slug, name, description, brand_id, category_id, manufacturer_ref, manufacturer_ref_norm, status, fulfillment_mode, lead_time_days,
		price_mode, price_amount, gtin, sale_unit, min_order_quantity, weight_grams, length_mm, width_mm, height_mm, attributes, origins, version, published_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22::jsonb,$23::jsonb,$24,$25,$26,$27)`,
		p.ID, p.SKU, p.Slug, toJSON(p.Name), toJSON(p.Description), p.BrandID, p.CategoryID, ref, refNorm(ref), p.Status, p.FulfillmentMode, p.LeadTimeDays,
		p.PriceMode, p.PriceAmount, p.GTIN, p.SaleUnit, p.MinOrderQuantity, p.WeightGrams, p.LengthMM, p.WidthMM, p.HeightMM, attrsJSON(p.Attributes), originsJSON(p.Origins), p.Version, p.PublishedAt, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return mapProductErr(err)
	}
	return mapProductErr(s.writeChildren(ctx, p))
}

func (s *Store) UpdateProduct(ctx context.Context, p *domain.Product) error {
	ref := p.ManufacturerRef
	tag, err := s.q(ctx).Exec(ctx, `UPDATE products SET sku=$2, slug=$3, name=$4::jsonb, description=$5::jsonb, brand_id=$6, category_id=$7, manufacturer_ref=$8, manufacturer_ref_norm=$9, status=$10,
		fulfillment_mode=$11, lead_time_days=$12, price_mode=$13, price_amount=$14, gtin=$15, sale_unit=$16, min_order_quantity=$17, weight_grams=$18, length_mm=$19, width_mm=$20, height_mm=$21,
		attributes=$22::jsonb, origins=$23::jsonb, published_at=$24, version=version+1, updated_at=$25 WHERE id=$1 AND version=$26`,
		p.ID, p.SKU, p.Slug, toJSON(p.Name), toJSON(p.Description), p.BrandID, p.CategoryID, ref, refNorm(ref), p.Status, p.FulfillmentMode, p.LeadTimeDays,
		p.PriceMode, p.PriceAmount, p.GTIN, p.SaleUnit, p.MinOrderQuantity, p.WeightGrams, p.LengthMM, p.WidthMM, p.HeightMM, attrsJSON(p.Attributes), originsJSON(p.Origins), p.PublishedAt, p.UpdatedAt, p.Version)
	if err != nil {
		return mapProductErr(err)
	}
	if tag.RowsAffected() == 0 { // la ligne a disparu, ou sa version a changé depuis la lecture
		var exists bool
		if err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, p.ID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return application.ErrStale
		}
		return application.ErrNotFound
	}
	p.Version++
	return mapProductErr(s.writeChildren(ctx, p))
}

const prodCols = `id::text, sku, slug, name, description, brand_id::text, category_id::text, manufacturer_ref, status, fulfillment_mode, lead_time_days, price_mode, price_amount,
	gtin, sale_unit, min_order_quantity, weight_grams, length_mm, width_mm, height_mm, attributes, origins, version, published_at, created_at, updated_at`

func scanProduct(r pgx.Row, cur string) (*domain.Product, error) {
	var p domain.Product
	var name, desc, attrs, origins []byte
	if err := r.Scan(&p.ID, &p.SKU, &p.Slug, &name, &desc, &p.BrandID, &p.CategoryID, &p.ManufacturerRef, &p.Status, &p.FulfillmentMode, &p.LeadTimeDays, &p.PriceMode, &p.PriceAmount,
		&p.GTIN, &p.SaleUnit, &p.MinOrderQuantity, &p.WeightGrams, &p.LengthMM, &p.WidthMM, &p.HeightMM, &attrs, &origins, &p.Version, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	var err error
	if p.Name, err = fromJSON(name); err != nil {
		return nil, err
	}
	if p.Description, err = fromJSON(desc); err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(attrs, &m); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		a := domain.AttrValue{Key: k}
		switch v := m[k].(type) {
		case string:
			a.Text = &v
		case float64:
			a.Number = &v
		case bool:
			a.Bool = &v
		}
		p.Attributes = append(p.Attributes, a)
	}
	p.Origins = map[string]string{}
	if err := json.Unmarshal(origins, &p.Origins); err != nil {
		return nil, err
	}
	p.Currency = cur
	return &p, nil
}

// loadChildren charge secteurs, médias et références croisées de plusieurs produits en 3 requêtes.
func (s *Store) loadChildren(ctx context.Context, ps []*domain.Product) error {
	if len(ps) == 0 {
		return nil
	}
	ids := make([]string, len(ps))
	by := map[string]*domain.Product{}
	for i, p := range ps {
		ids[i], by[p.ID] = p.ID, p
		p.SectorIDs, p.Media, p.CrossRefs = []string{}, []domain.Media{}, []domain.CrossRef{}
	}
	q := s.q(ctx)
	rows, err := q.Query(ctx, `SELECT product_id::text, sector_id::text FROM product_sectors WHERE product_id = ANY($1::uuid[]) ORDER BY sector_id`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pid, sid string
		if err := rows.Scan(&pid, &sid); err != nil {
			rows.Close()
			return err
		}
		by[pid].SectorIDs = append(by[pid].SectorIDs, sid)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT product_id::text, id::text, url, alt, sort_order FROM product_media WHERE product_id = ANY($1::uuid[]) ORDER BY product_id, sort_order`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pid string
		var m domain.Media
		if err := rows.Scan(&pid, &m.ID, &m.URL, &m.Alt, &m.SortOrder); err != nil {
			rows.Close()
			return err
		}
		by[pid].Media = append(by[pid].Media, m)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT product_id::text, brand, number, kind FROM product_cross_refs WHERE product_id = ANY($1::uuid[]) ORDER BY product_id, kind, lower(brand), number_norm`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var pid string
		var x domain.CrossRef
		if err := rows.Scan(&pid, &x.Brand, &x.Number, &x.Kind); err != nil {
			return err
		}
		by[pid].CrossRefs = append(by[pid].CrossRefs, x)
	}
	return rows.Err()
}

func (s *Store) ProductByID(ctx context.Context, id string, lock bool) (*domain.Product, error) {
	cur, err := s.ProductCurrency(ctx)
	if err != nil {
		return nil, err
	}
	sql := `SELECT ` + prodCols + ` FROM products WHERE id = $1`
	if lock {
		sql += ` FOR UPDATE`
	}
	p, err := scanProduct(s.q(ctx).QueryRow(ctx, sql, id), cur)
	if err != nil {
		return nil, err
	}
	return p, s.loadChildren(ctx, []*domain.Product{p})
}

func (s *Store) ListProducts(ctx context.Context, f application.ProductFilter, limit, offset int) ([]*domain.Product, int, error) {
	where, args := ` WHERE true`, []any{}
	if f.Query != "" {
		args = append(args, like(f.Query))
		where += fmt.Sprintf(` AND (sku ILIKE $%[1]d OR name::text ILIKE $%[1]d OR manufacturer_ref ILIKE $%[1]d OR slug ILIKE $%[1]d)`, len(args))
	}
	if f.CategoryID != nil {
		args = append(args, *f.CategoryID)
		where += fmt.Sprintf(` AND category_id IN (WITH RECURSIVE d AS (SELECT id FROM categories WHERE id = $%d UNION ALL SELECT c.id FROM categories c JOIN d ON c.parent_id = d.id) SELECT id FROM d)`, len(args))
	}
	if f.BrandID != nil {
		args = append(args, *f.BrandID)
		where += fmt.Sprintf(` AND brand_id = $%d`, len(args))
	}
	if f.Status != nil {
		args = append(args, *f.Status)
		where += fmt.Sprintf(` AND status = $%d`, len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM products`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	cur, err := s.ProductCurrency(ctx)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+prodCols+` FROM products`+where+fmt.Sprintf(` ORDER BY updated_at DESC, id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Product, error) { return scanProduct(r, cur) })
	if err != nil {
		return nil, 0, err
	}
	return items, total, s.loadChildren(ctx, items)
}

func (s *Store) DeleteProduct(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return mapProductErr(err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (s *Store) AddProductVersion(ctx context.Context, productID string, v domain.ProductVersion) error {
	b, err := json.Marshal(v.Changes)
	if err != nil {
		return err
	}
	_, err = s.q(ctx).Exec(ctx, `INSERT INTO product_versions (product_id, version, at, actor_id, action, changes) VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, productID, v.Version, v.At, v.ActorID, v.Action, string(b))
	return err
}

func (s *Store) ListProductVersions(ctx context.Context, productID string, limit int) ([]domain.ProductVersion, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT version, at, actor_id::text, action, changes FROM product_versions WHERE product_id = $1 ORDER BY version DESC LIMIT $2`, productID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.ProductVersion, error) {
		var v domain.ProductVersion
		var ch []byte
		if err := r.Scan(&v.Version, &v.At, &v.ActorID, &v.Action, &ch); err != nil {
			return v, err
		}
		return v, json.Unmarshal(ch, &v.Changes)
	})
}

func (s *Store) AttributeDefsForCategory(ctx context.Context, categoryID string) ([]*domain.AttributeDef, error) {
	rows, err := s.q(ctx).Query(ctx, `WITH RECURSIVE up AS (
		SELECT id, parent_id FROM categories WHERE id = $1
		UNION ALL SELECT c.id, c.parent_id FROM categories c JOIN up ON c.id = up.parent_id)
		SELECT `+attrCols+` FROM attribute_defs WHERE category_id IN (SELECT id FROM up) ORDER BY sort_order, key`, categoryID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.AttributeDef, error) { return scanAttr(r) })
}
