// Package postgres : adaptateur PostgreSQL du service content. L'unicité (slug, code, e-mail) est garantie par la base.
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
	"github.com/theb20/FILTER/backend/services/content/internal/application"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const MigrationsTable = "goose_content"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// Migrate applique les migrations de l'outbox puis celles de content.
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

func mapErr(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return application.ErrTaken
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

const cols = `id, kind, data, version, created_at, updated_at`

func scan(row pgx.Row) (*application.Document, error) {
	var d application.Document
	var raw []byte
	if err := row.Scan(&d.ID, &d.Kind, &raw, &d.Version, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	if err := json.Unmarshal(raw, &d.Data); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) Insert(ctx context.Context, d *application.Document, uniq *string, sort int) error {
	raw, _ := json.Marshal(d.Data)
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO documents (id, kind, uniq, sort_order, data) VALUES ($1, $2, $3, $4, $5)`, d.ID, d.Kind, uniq, sort, raw)
	return mapErr(err)
}

func (s *Store) Update(ctx context.Context, d *application.Document, uniq *string, sort int) error {
	raw, _ := json.Marshal(d.Data)
	tag, err := s.q(ctx).Exec(ctx, `UPDATE documents SET uniq = $3, sort_order = $4, data = $5, version = version + 1, updated_at = now() WHERE kind = $1 AND id = $2`, d.Kind, d.ID, uniq, sort, raw)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (s *Store) ByID(ctx context.Context, kind, id string) (*application.Document, error) {
	return scan(s.q(ctx).QueryRow(ctx, `SELECT `+cols+` FROM documents WHERE kind = $1 AND id = $2`, kind, id))
}

func (s *Store) ByKey(ctx context.Context, kind, key string) (*application.Document, error) {
	return scan(s.q(ctx).QueryRow(ctx, `SELECT `+cols+` FROM documents WHERE kind = $1 AND uniq = $2`, kind, key))
}

func (s *Store) List(ctx context.Context, kind, query string, where map[string]string, limit, offset int) ([]*application.Document, int, error) {
	args := []any{kind}
	cond := ` WHERE kind = $1`
	if query != "" {
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)+"%")
		cond += fmt.Sprintf(` AND data::text ILIKE $%d`, len(args))
	}
	for field, v := range where { // le nom du champ est LIÉ comme paramètre, jamais concaténé
		if v == "" { // valeur vide : le champ est absent du document
			args = append(args, field)
			cond += fmt.Sprintf(` AND data ->> $%d IS NULL`, len(args))
			continue
		}
		args = append(args, field, v)
		cond += fmt.Sprintf(` AND data ->> $%d = $%d`, len(args)-1, len(args))
	}
	var total int
	if err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM documents`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := s.q(ctx).Query(ctx, `SELECT `+cols+` FROM documents`+cond+fmt.Sprintf(` ORDER BY sort_order, created_at DESC, id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*application.Document
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

func (s *Store) Delete(ctx context.Context, kind, id string) error {
	tag, err := s.q(ctx).Exec(ctx, `DELETE FROM documents WHERE kind = $1 AND id = $2`, kind, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (s *Store) GetSettings(ctx context.Context, key string) (*application.Settings, error) {
	var st application.Settings
	var raw []byte
	if err := s.q(ctx).QueryRow(ctx, `SELECT data, version, updated_at FROM settings WHERE key = $1`, key).Scan(&raw, &st.Version, &st.UpdatedAt); err != nil {
		return nil, mapErr(err)
	}
	if err := json.Unmarshal(raw, &st.Data); err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) PutSettings(ctx context.Context, key string, data map[string]any) (*application.Settings, error) {
	raw, _ := json.Marshal(data)
	if _, err := s.q(ctx).Exec(ctx, `INSERT INTO settings (key, data) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET data = EXCLUDED.data, version = settings.version + 1, updated_at = now()`, key, raw); err != nil {
		return nil, mapErr(err)
	}
	return s.GetSettings(ctx, key)
}
