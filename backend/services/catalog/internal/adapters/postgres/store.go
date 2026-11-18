// Package postgres : adaptateur PostgreSQL du service catalog. Les invariants (cycles, héritage des clés, unicités) sont dans la base.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/theb20/FILTER/backend/pkg/outbox"
	"github.com/theb20/FILTER/backend/pkg/pgxtx"
	"github.com/theb20/FILTER/backend/services/catalog/internal/application"
	"github.com/theb20/FILTER/backend/services/catalog/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const MigrationsTable = "goose_catalog"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// Migrate applique les migrations de l'outbox puis celles de catalog.
func Migrate(ctx context.Context, url string) error {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := outbox.MigrateOutbox(ctx, db); err != nil {
		return err
	}
	return outbox.Migrate(ctx, db, Migrations(), MigrationsTable)
}

type Store struct{ Pool *pgxpool.Pool }

var _ application.Store = (*Store)(nil)

func New(ctx context.Context, url string) (*Store, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Store{Pool: p}, nil
}

func (s *Store) Close()                           { s.Pool.Close() }
func (s *Store) q(ctx context.Context) pgxtx.DBTX { return pgxtx.Querier(ctx, s.Pool) }
func (s *Store) Do(ctx context.Context, f func(context.Context) error) error {
	return pgxtx.Run(ctx, s.Pool, f)
}

func pgErr(err error) *pgconn.PgError {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe
	}
	return nil
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

// mapErr traduit les violations de contraintes en erreurs métier ; `fk` est l'erreur renvoyée pour une clé étrangère (23503).
func mapErr(err error, fk error) error {
	pe := pgErr(err)
	if pe == nil {
		return err
	}
	switch {
	case pe.Code == "23505" && strings.Contains(pe.ConstraintName, "slug"):
		return application.ErrSlugTaken
	case pe.Code == "23505" && strings.Contains(pe.ConstraintName, "name"):
		return application.ErrNameTaken
	case pe.Code == "23505" && strings.Contains(pe.ConstraintName, "key_per_category"):
		return application.ErrKeyTaken
	case pe.Code == "23514" && pe.Message == "category_cycle":
		return application.ErrCycle
	case pe.Code == "23514" && pe.Message == "attribute_key_inherited":
		return application.ErrKeyInherited
	case pe.Code == "23503":
		return fk
	}
	return err
}

func toJSON(l domain.Localized) any {
	if l == nil {
		return nil
	}
	b, _ := json.Marshal(l)
	return string(b)
}

func fromJSON(b []byte) (domain.Localized, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var l domain.Localized
	return l, json.Unmarshal(b, &l)
}

func like(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// ── catégories ──────────────────────────────────────────────────────────────

// La profondeur est calculée à la lecture (racine = 0).
const catSelect = `WITH RECURSIVE tree AS (
    SELECT id, 0 AS depth FROM categories WHERE parent_id IS NULL
    UNION ALL
    SELECT c.id, t.depth + 1 FROM categories c JOIN tree t ON c.parent_id = t.id
) SELECT c.id::text, c.slug, c.parent_id::text, c.sort_order, c.name, c.description, c.image_url, t.depth, (SELECT count(*) FROM products p WHERE p.category_id = c.id), c.created_at, c.updated_at
  FROM categories c JOIN tree t ON t.id = c.id`

func scanCategory(r pgx.Row) (*domain.Category, error) {
	var c domain.Category
	var name, desc []byte
	if err := r.Scan(&c.ID, &c.Slug, &c.ParentID, &c.SortOrder, &name, &desc, &c.ImageURL, &c.Depth, &c.ProductCount, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	var err error
	if c.Name, err = fromJSON(name); err != nil {
		return nil, err
	}
	c.Description, err = fromJSON(desc)
	return &c, err
}

func (s *Store) CreateCategory(ctx context.Context, c *domain.Category) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO categories (id, slug, parent_id, sort_order, name, description, image_url) VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7)`,
		c.ID, c.Slug, c.ParentID, c.SortOrder, toJSON(c.Name), toJSON(c.Description), c.ImageURL)
	return mapErr(err, application.ErrParentNotFound)
}

func (s *Store) UpdateCategory(ctx context.Context, c *domain.Category) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE categories SET slug=$2, name=$3::jsonb, description=$4::jsonb, image_url=$5, updated_at=now() WHERE id=$1`, c.ID, c.Slug, toJSON(c.Name), toJSON(c.Description), c.ImageURL)
	if err = mapErr(err, err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) CategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	return scanCategory(s.q(ctx).QueryRow(ctx, catSelect+` WHERE c.id = $1`, id))
}

func (s *Store) ListCategories(ctx context.Context, query string, parentID *string, limit, offset int) ([]*domain.Category, int, error) {
	where, args := ` WHERE true`, []any{}
	if query != "" {
		args = append(args, like(query))
		where += fmt.Sprintf(` AND (c.slug ILIKE $%[1]d OR c.name::text ILIKE $%[1]d)`, len(args))
	}
	if parentID != nil {
		args = append(args, *parentID)
		where += fmt.Sprintf(` AND c.parent_id = $%d`, len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM categories c`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, catSelect+where+fmt.Sprintf(` ORDER BY t.depth, c.sort_order, c.slug LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Category, error) { return scanCategory(r) })
	return items, total, err
}

func (s *Store) DeleteCategory(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err = mapErr(err, application.ErrInUse); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) MoveCategory(ctx context.Context, id string, parentID *string, position int) error {
	// Un seul écrivain sur l'arbre à la fois (déplacements rares) : la renumérotation des frères reste cohérente.
	if _, err := s.q(ctx).Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('catalog_tree'))`); err != nil {
		return err
	}
	var exists bool
	if err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM categories WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return application.ErrNotFound
	}
	if parentID != nil {
		var ok bool
		if err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM categories WHERE id = $1)`, *parentID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return application.ErrParentNotFound
		}
	}
	if _, err := s.q(ctx).Exec(ctx, `UPDATE categories SET parent_id = $2, updated_at = now() WHERE id = $1`, id, parentID); err != nil {
		return mapErr(err, application.ErrParentNotFound)
	}
	rows, err := s.q(ctx).Query(ctx, `SELECT id::text FROM categories WHERE parent_id IS NOT DISTINCT FROM $1::uuid AND id <> $2 ORDER BY sort_order, slug`, parentID, id)
	if err != nil {
		return err
	}
	sibs, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if position > len(sibs) {
		position = len(sibs)
	}
	order := append(sibs[:position:position], append([]string{id}, sibs[position:]...)...)
	for i, sid := range order {
		if _, err := s.q(ctx).Exec(ctx, `UPDATE categories SET sort_order = $2 WHERE id = $1 AND sort_order <> $2`, sid, i); err != nil {
			return err
		}
	}
	return nil
}

// ── attributs ───────────────────────────────────────────────────────────────

const attrCols = `id::text, category_id::text, key, label, type, unit, filterable, required, options, sort_order`

func scanAttr(r pgx.Row) (*domain.AttributeDef, error) {
	var a domain.AttributeDef
	var label, opts []byte
	if err := r.Scan(&a.ID, &a.CategoryID, &a.Key, &label, &a.Type, &a.Unit, &a.Filterable, &a.Required, &opts, &a.SortOrder); err != nil {
		return nil, notFound(err)
	}
	var err error
	if a.Label, err = fromJSON(label); err != nil {
		return nil, err
	}
	var raw []struct {
		Value string           `json:"value"`
		Label domain.Localized `json:"label"`
	}
	if err := json.Unmarshal(opts, &raw); err != nil {
		return nil, err
	}
	for _, o := range raw {
		a.Options = append(a.Options, domain.AttributeOption{Value: o.Value, Label: o.Label})
	}
	return &a, nil
}

func optsJSON(o []domain.AttributeOption) string {
	type jo struct {
		Value string           `json:"value"`
		Label domain.Localized `json:"label"`
	}
	out := make([]jo, 0, len(o))
	for _, x := range o {
		out = append(out, jo{x.Value, x.Label})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func (s *Store) CreateAttributeDef(ctx context.Context, a *domain.AttributeDef) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO attribute_defs (id, category_id, key, label, type, unit, filterable, required, options, sort_order) VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9::jsonb,$10)`,
		a.ID, a.CategoryID, a.Key, toJSON(a.Label), a.Type, a.Unit, a.Filterable, a.Required, optsJSON(a.Options), a.SortOrder)
	return mapErr(err, application.ErrCategoryMissing)
}

func (s *Store) UpdateAttributeDef(ctx context.Context, a *domain.AttributeDef) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE attribute_defs SET label=$2::jsonb, unit=$3, filterable=$4, required=$5, options=$6::jsonb, sort_order=$7, updated_at=now() WHERE id=$1`,
		a.ID, toJSON(a.Label), a.Unit, a.Filterable, a.Required, optsJSON(a.Options), a.SortOrder)
	if err = mapErr(err, err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) AttributeDefByID(ctx context.Context, id string) (*domain.AttributeDef, error) {
	return scanAttr(s.q(ctx).QueryRow(ctx, `SELECT `+attrCols+` FROM attribute_defs WHERE id = $1`, id))
}

func (s *Store) ListAttributeDefs(ctx context.Context, query string, categoryID *string, limit, offset int) ([]*domain.AttributeDef, int, error) {
	where, args := ` WHERE true`, []any{}
	if query != "" {
		args = append(args, like(query))
		where += fmt.Sprintf(` AND (key ILIKE $%[1]d OR label::text ILIKE $%[1]d)`, len(args))
	}
	if categoryID != nil {
		args = append(args, *categoryID)
		where += fmt.Sprintf(` AND category_id = $%d`, len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM attribute_defs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+attrCols+` FROM attribute_defs`+where+fmt.Sprintf(` ORDER BY category_id, sort_order, key LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.AttributeDef, error) { return scanAttr(r) })
	return items, total, err
}

func (s *Store) DeleteAttributeDef(ctx context.Context, id string) error {
	var used bool
	if err := s.q(ctx).QueryRow(ctx, `WITH RECURSIVE d AS (
		SELECT a.category_id AS id, a.key FROM attribute_defs a WHERE a.id = $1
		UNION ALL SELECT c.id, d.key FROM categories c JOIN d ON c.parent_id = d.id)
		SELECT EXISTS (SELECT 1 FROM products p JOIN d ON p.category_id = d.id WHERE p.attributes ? d.key)`, id).Scan(&used); err != nil {
		return err
	}
	if used {
		return application.ErrInUse
	}
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM attribute_defs WHERE id = $1`, id)
	if err = mapErr(err, application.ErrInUse); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

// ── marques ─────────────────────────────────────────────────────────────────

const brandCols = `id::text, slug, name, logo_url, created_at, updated_at`

func scanBrand(r pgx.Row) (*domain.Brand, error) {
	var b domain.Brand
	if err := r.Scan(&b.ID, &b.Slug, &b.Name, &b.LogoURL, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	return &b, nil
}

func (s *Store) CreateBrand(ctx context.Context, b *domain.Brand) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO brands (id, slug, name, logo_url) VALUES ($1,$2,$3,$4)`, b.ID, b.Slug, b.Name, b.LogoURL)
	return mapErr(err, err)
}

func (s *Store) UpdateBrand(ctx context.Context, b *domain.Brand) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE brands SET slug=$2, name=$3, logo_url=$4, updated_at=now() WHERE id=$1`, b.ID, b.Slug, b.Name, b.LogoURL)
	if err = mapErr(err, err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) BrandByID(ctx context.Context, id string) (*domain.Brand, error) {
	return scanBrand(s.q(ctx).QueryRow(ctx, `SELECT `+brandCols+` FROM brands WHERE id = $1`, id))
}

func (s *Store) ListBrands(ctx context.Context, query string, limit, offset int) ([]*domain.Brand, int, error) {
	where, args := ``, []any{}
	if query != "" {
		args = append(args, like(query))
		where = ` WHERE name ILIKE $1 OR slug ILIKE $1`
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM brands`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+brandCols+` FROM brands`+where+fmt.Sprintf(` ORDER BY lower(name) LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Brand, error) { return scanBrand(r) })
	return items, total, err
}

func (s *Store) DeleteBrand(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM brands WHERE id = $1`, id)
	if err = mapErr(err, application.ErrInUse); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

// ── secteurs ────────────────────────────────────────────────────────────────

const sectorCols = `id::text, slug, name, image_url, created_at, updated_at`

func scanSector(r pgx.Row) (*domain.Sector, error) {
	var x domain.Sector
	var name []byte
	if err := r.Scan(&x.ID, &x.Slug, &name, &x.ImageURL, &x.CreatedAt, &x.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	var err error
	x.Name, err = fromJSON(name)
	return &x, err
}

func (s *Store) CreateSector(ctx context.Context, x *domain.Sector) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO sectors (id, slug, name, image_url) VALUES ($1,$2,$3::jsonb,$4)`, x.ID, x.Slug, toJSON(x.Name), x.ImageURL)
	return mapErr(err, err)
}

func (s *Store) UpdateSector(ctx context.Context, x *domain.Sector) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE sectors SET slug=$2, name=$3::jsonb, image_url=$4, updated_at=now() WHERE id=$1`, x.ID, x.Slug, toJSON(x.Name), x.ImageURL)
	if err = mapErr(err, err); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) SectorByID(ctx context.Context, id string) (*domain.Sector, error) {
	return scanSector(s.q(ctx).QueryRow(ctx, `SELECT `+sectorCols+` FROM sectors WHERE id = $1`, id))
}

func (s *Store) ListSectors(ctx context.Context, query string, limit, offset int) ([]*domain.Sector, int, error) {
	where, args := ``, []any{}
	if query != "" {
		args = append(args, like(query))
		where = ` WHERE slug ILIKE $1 OR name::text ILIKE $1`
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM sectors`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+sectorCols+` FROM sectors`+where+fmt.Sprintf(` ORDER BY slug LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*domain.Sector, error) { return scanSector(r) })
	return items, total, err
}

func (s *Store) DeleteSector(ctx context.Context, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM sectors WHERE id = $1`, id)
	if err = mapErr(err, application.ErrInUse); err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return err
}
