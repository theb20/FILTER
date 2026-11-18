package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

// Sécurité du compte : sessions actives, appareils connus, historique des connexions, images (voir 00009).

func (s *Store) ActiveSessions(ctx context.Context, userID string, now time.Time) ([]domain.ActiveSession, error) {
	rows, err := s.q(ctx).Query(ctx, `
		SELECT f.id::text, f.created_at, t.created_at, t.ip, t.user_agent
		FROM token_families f
		JOIN LATERAL (SELECT created_at, expires_at, ip, user_agent FROM refresh_tokens
		              WHERE family_id = f.id ORDER BY created_at DESC LIMIT 1) t ON true
		WHERE f.user_id = $1 AND f.revoked_at IS NULL AND t.expires_at > $2
		ORDER BY t.created_at DESC`, userID, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.ActiveSession, error) {
		var a domain.ActiveSession
		err := r.Scan(&a.FamilyID, &a.CreatedAt, &a.LastSeenAt, &a.IP, &a.UserAgent)
		return a, err
	})
}

func (s *Store) RevokeUserSession(ctx context.Context, userID, familyID string, at time.Time) (bool, error) {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE token_families SET revoked_at = $3 WHERE id::text = $2 AND user_id = $1 AND revoked_at IS NULL`, userID, familyID, at)
	return tag.RowsAffected() == 1, err
}

func (s *Store) RevokeUserSessionsExcept(ctx context.Context, userID, keepFamilyID string, at time.Time) (int, error) {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE token_families SET revoked_at = $3 WHERE user_id = $1 AND revoked_at IS NULL AND id::text <> $2`, userID, keepFamilyID, at)
	return int(tag.RowsAffected()), err
}

func (s *Store) TouchDevice(ctx context.Context, userID, deviceKey string, at, cutoff time.Time) (existed, hadOthers bool, err error) {
	err = s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `DELETE FROM known_devices WHERE user_id = $1 AND last_seen_at < $2`, userID, cutoff); err != nil {
			return err
		}
		// Verrou par compte : deux connexions simultanées depuis un appareil inconnu ne déclenchent qu'une alerte.
		if _, err := db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('known_devices:' || $1, 0))`, userID); err != nil {
			return err
		}
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM known_devices WHERE user_id = $1 AND device_key = $2),
		                                   EXISTS (SELECT 1 FROM known_devices WHERE user_id = $1 AND device_key <> $2)`, userID, deviceKey).Scan(&existed, &hadOthers); err != nil {
			return err
		}
		_, err := db.Exec(ctx, `INSERT INTO known_devices (user_id, device_key, first_seen_at, last_seen_at) VALUES ($1, $2, $3, $3)
		                        ON CONFLICT (user_id, device_key) DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at`, userID, deviceKey, at)
		return err
	})
	return existed, hadOthers, err
}

func (s *Store) InsertLoginEvent(ctx context.Context, ev *domain.LoginEvent, cutoff time.Time) error {
	return s.Do(ctx, func(ctx context.Context) error {
		db := s.q(ctx)
		if _, err := db.Exec(ctx, `DELETE FROM login_events WHERE at < $1`, cutoff); err != nil {
			return err
		}
		var city, region, cc, country string
		var lat, lon *float64
		if l := ev.Location; l != nil {
			city, region, cc, country, lat, lon = l.City, l.Region, l.CountryCode, l.Country, &l.Latitude, &l.Longitude
		}
		_, err := db.Exec(ctx, `INSERT INTO login_events (id, user_id, at, ip, user_agent, method, new_device, city, region, country_code, country, latitude, longitude)
		                        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			ev.ID, ev.UserID, ev.At, ev.IP, ev.UserAgent, ev.Method, ev.NewDevice, city, region, cc, country, lat, lon)
		return err
	})
}

func (s *Store) LoginEvents(ctx context.Context, userID string, since time.Time, offset, limit int) ([]domain.LoginEvent, int, error) {
	db := s.q(ctx)
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM login_events WHERE user_id = $1 AND at >= $2`, userID, since).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(ctx, `SELECT id::text, user_id::text, at, ip, user_agent, method, new_device, city, region, country_code, country, latitude, longitude
	                            FROM login_events WHERE user_id = $1 AND at >= $2 ORDER BY at DESC, id DESC OFFSET $3 LIMIT $4`, userID, since, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (domain.LoginEvent, error) {
		var e domain.LoginEvent
		var l domain.GeoLocation
		var lat, lon *float64
		if err := r.Scan(&e.ID, &e.UserID, &e.At, &e.IP, &e.UserAgent, &e.Method, &e.NewDevice, &l.City, &l.Region, &l.CountryCode, &l.Country, &lat, &lon); err != nil {
			return e, err
		}
		if lat != nil && lon != nil {
			l.Latitude, l.Longitude = *lat, *lon
			e.Location = &l
		}
		return e, nil
	})
	return list, total, err
}

func (s *Store) PutImage(ctx context.Context, userID string, img domain.UserImage) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO user_images (user_id, kind, content_type, data, updated_at) VALUES ($1,$2,$3,$4,$5)
	                              ON CONFLICT (user_id, kind) DO UPDATE SET content_type = EXCLUDED.content_type, data = EXCLUDED.data, updated_at = EXCLUDED.updated_at`,
		userID, img.Kind, img.ContentType, img.Data, img.UpdatedAt)
	return err
}

func (s *Store) Image(ctx context.Context, userID, kind string) (*domain.UserImage, error) {
	img := domain.UserImage{Kind: kind}
	err := s.q(ctx).QueryRow(ctx, `SELECT content_type, data, updated_at FROM user_images WHERE user_id = $1 AND kind = $2`, userID, kind).
		Scan(&img.ContentType, &img.Data, &img.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	return &img, err
}

func (s *Store) DeleteImage(ctx context.Context, userID, kind string) error {
	_, err := s.q(ctx).Exec(ctx, `DELETE FROM user_images WHERE user_id = $1 AND kind = $2`, userID, kind)
	return err
}

func (s *Store) ImageTimes(ctx context.Context, userID string) (avatar, cover *time.Time, err error) {
	err = s.q(ctx).QueryRow(ctx, `SELECT (SELECT updated_at FROM user_images WHERE user_id = $1 AND kind = 'avatar'),
	                                     (SELECT updated_at FROM user_images WHERE user_id = $1 AND kind = 'cover')`, userID).Scan(&avatar, &cover)
	return avatar, cover, err
}
