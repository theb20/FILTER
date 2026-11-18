package application

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── Sécurité du compte client : appareils connectés, historique des connexions, alerte « nouvel appareil »,
// photo de profil et couverture. Décisions du 18 nov. 2026 : géolocalisation par une base LOCALE (DB-IP, aucune
// adresse IP envoyée à un tiers), historique conservé 90 jours, e-mail d'alerte pour un appareil jamais vu.

// EvNewDeviceLogin : connexion réussie depuis un appareil inconnu de ce compte.
const EvNewDeviceLogin = "identity.new_device_login_alert_requested.v1"

// SecurityRepo persiste l'historique des connexions, les appareils connus et les images du compte, et lit les
// sessions actives (familles de jetons).
type SecurityRepo interface {
	// ActiveSessions : familles non révoquées dont le dernier jeton n'a pas expiré à `now`, la plus récente d'abord.
	ActiveSessions(ctx context.Context, userID string, now time.Time) ([]domain.ActiveSession, error)
	// RevokeUserSession révoque UNE famille de CE compte ; false si elle n'existe pas, est à un autre compte ou est déjà révoquée.
	RevokeUserSession(ctx context.Context, userID, familyID string, at time.Time) (bool, error)
	// RevokeUserSessionsExcept révoque les familles actives du compte sauf `keepFamilyID` et renvoie leur nombre.
	RevokeUserSessionsExcept(ctx context.Context, userID, keepFamilyID string, at time.Time) (int, error)
	// TouchDevice enregistre l'appareil (purge d'abord ceux vus avant `cutoff`) : `existed` si l'appareil était connu,
	// `hadOthers` si le compte avait déjà au moins un AUTRE appareil connu.
	TouchDevice(ctx context.Context, userID, deviceKey string, at, cutoff time.Time) (existed, hadOthers bool, err error)
	// InsertLoginEvent ajoute la connexion et purge l'historique antérieur à `cutoff` (tous comptes).
	InsertLoginEvent(ctx context.Context, ev *domain.LoginEvent, cutoff time.Time) error
	// LoginEvents : connexions du compte postérieures à `since`, la plus récente d'abord, et leur nombre total.
	LoginEvents(ctx context.Context, userID string, since time.Time, offset, limit int) ([]domain.LoginEvent, int, error)
	PutImage(ctx context.Context, userID string, img domain.UserImage) error
	Image(ctx context.Context, userID, kind string) (*domain.UserImage, error) // ErrNotFound si absente
	DeleteImage(ctx context.Context, userID, kind string) error
	ImageTimes(ctx context.Context, userID string) (avatar, cover *time.Time, err error)
}

// GeoLocator déduit une position APPROXIMATIVE d'une adresse IP ; nil si inconnue (adresse privée, base absente…).
type GeoLocator interface {
	Locate(ip string) *domain.GeoLocation
}

// SessionView : une session active telle que présentée au titulaire du compte.
type SessionView struct {
	domain.ActiveSession
	Current  bool
	Device   domain.Device
	Location *domain.GeoLocation
}

const maxUserAgent = 512

func (s *Service) locate(ip string) *domain.GeoLocation {
	if s.Geo == nil || ip == "" {
		return nil
	}
	if a, err := netip.ParseAddr(ip); err != nil || a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() || a.IsUnspecified() {
		return nil
	}
	return s.Geo.Locate(ip)
}

// recordLogin : appelé DANS la transaction d'une connexion réussie (mot de passe ou Google). Le premier appareil
// d'un compte est enregistré sans alerte ; un appareil inconnu ensuite déclenche l'e-mail d'alerte.
func (s *Service) recordLogin(ctx context.Context, u *domain.User, ip, ua, method string, now time.Time) error {
	if s.Security == nil {
		return nil
	}
	if len(ua) > maxUserAgent {
		ua = ua[:maxUserAgent]
	}
	cutoff := now.Add(-domain.LoginHistoryRetention)
	dev := domain.ParseUserAgent(ua)
	existed, hadOthers, err := s.Security.TouchDevice(ctx, u.ID, domain.DeviceKey(u.ID, dev), now, cutoff)
	if err != nil {
		return apperr.Internal(err)
	}
	ev := &domain.LoginEvent{ID: s.NewID(), UserID: u.ID, At: now, IP: ip, UserAgent: ua, Method: method, NewDevice: !existed && hadOthers, Location: s.locate(ip)}
	if err := s.Security.InsertLoginEvent(ctx, ev, cutoff); err != nil {
		return apperr.Internal(err)
	}
	if !ev.NewDevice {
		return nil
	}
	plain, hash, err := domain.NewOpaqueToken()
	if err != nil {
		return apperr.Internal(err)
	}
	if err := s.Tokens.Put(ctx, &domain.OneTimeToken{Hash: hash, Purpose: domain.PurposeLoginAlert, UserID: u.ID, ExpiresAt: now.Add(domain.LoginAlertTTL)}); err != nil {
		return apperr.Internal(err)
	}
	msg := &eventsv1.NewDeviceLoginAlertRequested{UserId: u.ID, Email: u.Email, Locale: u.Locale, At: timestamppb.New(now), Ip: ip, Browser: dev.Browser, Os: dev.OS, Token: plain}
	if ev.Location != nil {
		msg.City, msg.Country = ev.Location.City, ev.Location.Country
	}
	return s.wrap(s.publish(ctx, EvNewDeviceLogin, u.ID, msg, u.Locale))
}

// ListMySessions : appareils connectés du compte ; `currentSID` (claim `sid` du jeton) marque l'appareil courant.
func (s *Service) ListMySessions(ctx context.Context, userID, currentSID string) ([]SessionView, error) {
	list, err := s.Security.ActiveSessions(ctx, userID, s.Now())
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]SessionView, 0, len(list))
	for _, a := range list {
		out = append(out, SessionView{ActiveSession: a, Current: a.FamilyID == currentSID, Device: domain.ParseUserAgent(a.UserAgent), Location: s.locate(a.IP)})
	}
	return out, nil
}

// RevokeMySession déconnecte un appareil du compte.
func (s *Service) RevokeMySession(ctx context.Context, userID, sessionID string) error {
	ok, err := s.Security.RevokeUserSession(ctx, userID, sessionID, s.Now())
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		return apperr.NotFound("session introuvable")
	}
	return nil
}

// RevokeMyOtherSessions déconnecte tous les appareils sauf le courant.
func (s *Service) RevokeMyOtherSessions(ctx context.Context, userID, currentSID string) (int, error) {
	if currentSID == "" {
		return 0, apperr.New(codes.InvalidArgument, apperr.CodeValidation, "session courante inconnue")
	}
	n, err := s.Security.RevokeUserSessionsExcept(ctx, userID, currentSID, s.Now())
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}

// ListMyLoginEvents : historique des connexions (fenêtre de conservation seulement).
func (s *Service) ListMyLoginEvents(ctx context.Context, userID string, page, size int) ([]domain.LoginEvent, int, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	since := s.Now().Add(-domain.LoginHistoryRetention)
	list, total, err := s.Security.LoginEvents(ctx, userID, since, (page-1)*size, size)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return list, total, nil
}

// SecureAccountFromAlert : lien « ce n'était pas moi ». Jeton à usage unique ; toutes les sessions sont révoquées.
func (s *Service) SecureAccountFromAlert(ctx context.Context, token string) error {
	now := s.Now()
	return s.tx(ctx, func(ctx context.Context) error {
		userID, ok, err := s.Tokens.Consume(ctx, domain.PurposeLoginAlert, domain.HashToken(token), now)
		if err != nil {
			return apperr.Internal(err)
		}
		if !ok {
			return apperr.New(codes.InvalidArgument, apperr.CodeInvalidCode, "lien invalide ou expiré")
		}
		return s.wrap(s.Sessions.RevokeAllExcept(ctx, userID, "", now))
	})
}

// PutMyImage remplace la photo de profil ou la couverture (contenu vérifié).
func (s *Service) PutMyImage(ctx context.Context, userID, kind string, data []byte) (*domain.UserImage, error) {
	if !domain.ValidImageKind(kind) {
		return nil, apperr.New(codes.InvalidArgument, apperr.CodeValidation, "type d'image inconnu")
	}
	ct, err := domain.SniffImage(data)
	if err != nil {
		return nil, apperr.New(codes.InvalidArgument, apperr.CodeValidation, err.Error())
	}
	img := domain.UserImage{Kind: kind, ContentType: ct, Data: data, UpdatedAt: s.Now().UTC().Truncate(time.Microsecond)}
	if err := s.Security.PutImage(ctx, userID, img); err != nil {
		return nil, apperr.Internal(err)
	}
	return &img, nil
}

// GetMyImage lit la photo de profil ou la couverture.
func (s *Service) GetMyImage(ctx context.Context, userID, kind string) (*domain.UserImage, error) {
	if !domain.ValidImageKind(kind) {
		return nil, apperr.NotFound("image introuvable")
	}
	img, err := s.Security.Image(ctx, userID, kind)
	if errors.Is(err, ErrNotFound) {
		return nil, apperr.NotFound("image introuvable")
	} else if err != nil {
		return nil, apperr.Internal(err)
	}
	return img, nil
}

// DeleteMyImage retire la photo de profil ou la couverture (idempotent).
func (s *Service) DeleteMyImage(ctx context.Context, userID, kind string) error {
	if !domain.ValidImageKind(kind) {
		return apperr.New(codes.InvalidArgument, apperr.CodeValidation, "type d'image inconnu")
	}
	return s.wrap(s.Security.DeleteImage(ctx, userID, kind))
}

// withImages complète le compte avec les dates de sa photo et de sa couverture (clé de cache du site).
func (s *Service) withImages(ctx context.Context, u *domain.User) error {
	if s.Security == nil || u == nil {
		return nil
	}
	a, c, err := s.Security.ImageTimes(ctx, u.ID)
	if err != nil {
		return apperr.Internal(err)
	}
	u.AvatarUpdatedAt, u.CoverUpdatedAt = a, c
	return nil
}
