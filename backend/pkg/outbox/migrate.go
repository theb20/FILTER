package outbox

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// Migrate applique les migrations goose de fsys avec une table de version dédiée (chaque module a la sienne :
// l'outbox et le service ne partagent jamais la même numérotation).
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS, table string) error {
	p, err := goose.NewProvider(goose.DialectPostgres, db, fsys, goose.WithTableName(table))
	if err != nil {
		return err
	}
	_, err = p.Up(ctx)
	return err
}

// MigrateOutbox applique les migrations de l'outbox.
func MigrateOutbox(ctx context.Context, db *sql.DB) error {
	return Migrate(ctx, db, Migrations(), MigrationsTable)
}
