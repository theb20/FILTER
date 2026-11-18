package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var _ application.PreferencesRepo = (*Store)(nil)

// owner renvoie la colonne propriétaire selon le monde du compte.
func owner(kind domain.PrincipalKind) (string, error) {
	switch kind {
	case domain.PrincipalStaff:
		return "staff_id", nil
	case domain.PrincipalCustomer:
		return "customer_id", nil
	}
	return "", application.ErrNotFound
}

func (s *Store) LoadPreferences(ctx context.Context, kind domain.PrincipalKind, id string) (*domain.Preferences, error) {
	col, err := owner(kind)
	if err != nil {
		return nil, err
	}
	var p domain.Preferences
	err = s.q(ctx).QueryRow(ctx, `SELECT guide_enabled, guide_animations, guide_seen FROM user_preferences WHERE `+col+` = $1`, id).
		Scan(&p.Guide.Enabled, &p.Guide.Animations, &p.Guide.Seen)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if p.Guide.Seen == nil {
		p.Guide.Seen = []string{}
	}
	return &p, nil
}

func (s *Store) SavePreferences(ctx context.Context, kind domain.PrincipalKind, id string, p domain.Preferences, at time.Time) error {
	col, err := owner(kind)
	if err != nil {
		return err
	}
	seen := p.Guide.Seen
	if seen == nil {
		seen = []string{}
	}
	_, err = s.q(ctx).Exec(ctx, `INSERT INTO user_preferences (`+col+`, guide_enabled, guide_animations, guide_seen, updated_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (`+col+`) WHERE `+col+` IS NOT NULL DO UPDATE SET guide_enabled = EXCLUDED.guide_enabled, guide_animations = EXCLUDED.guide_animations,
		guide_seen = EXCLUDED.guide_seen, updated_at = EXCLUDED.updated_at`, id, p.Guide.Enabled, p.Guide.Animations, seen, at)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23503" { // clé étrangère : le compte n'existe pas
		return application.ErrNotFound
	}
	return err
}
