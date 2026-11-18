package postgres

import (
	"context"

	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

var _ application.ConsentRepo = (*Store)(nil)

func (s *Store) AppendConsent(ctx context.Context, c *domain.ConsentRecord) error {
	var uid *string
	if c.UserID != "" {
		uid = &c.UserID
	}
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO consents (id, user_id, anonymous_id, necessary, preferences, functional, analytics, marketing, policy_version, locale, ip_net, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		c.ID, uid, c.AnonymousID, c.Purposes.Necessary, c.Purposes.Preferences, c.Purposes.Functional, c.Purposes.Analytics, c.Purposes.Marketing, c.PolicyVersion, c.Locale, c.IPNet, c.RecordedAt)
	if isFK(err) {
		return application.ErrNotFound
	}
	return err
}

func (s *Store) LatestUserConsent(ctx context.Context, userID string) (*domain.ConsentRecord, error) {
	c := domain.ConsentRecord{UserID: userID}
	err := s.q(ctx).QueryRow(ctx, `SELECT id::text, anonymous_id, necessary, preferences, functional, analytics, marketing, policy_version, locale, ip_net, recorded_at
		FROM consents WHERE user_id = $1 ORDER BY recorded_at DESC, id DESC LIMIT 1`, userID).
		Scan(&c.ID, &c.AnonymousID, &c.Purposes.Necessary, &c.Purposes.Preferences, &c.Purposes.Functional, &c.Purposes.Analytics, &c.Purposes.Marketing, &c.PolicyVersion, &c.Locale, &c.IPNet, &c.RecordedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return &c, nil
}
