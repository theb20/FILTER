package application_test

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"testing"
	"time"

	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/services/identity/internal/application"
	"github.com/theb20/FILTER/backend/services/identity/internal/domain"
)

const (
	uaChromeWin  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36"
	uaFirefoxAnd = "Mozilla/5.0 (Android 14; Mobile; rv:132.0) Gecko/132.0 Firefox/132.0"
	uaSafariIOS  = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Mobile/15E148 Safari/604.1"
)

// fakeGeo : position fixe pour une adresse connue du test, nil sinon (comme une adresse absente de la base).
type fakeGeo map[string]*domain.GeoLocation

func (f fakeGeo) Locate(ip string) *domain.GeoLocation { return f[ip] }

var abidjan = &domain.GeoLocation{City: "Abidjan", Region: "Abidjan", CountryCode: "CI", Country: "Côte d'Ivoire", Latitude: 5.36, Longitude: -4.01}

func secEnv(t *testing.T) (*env, string) {
	t.Helper()
	e := newEnv(t, nil)
	e.svc.Geo = fakeGeo{"41.202.10.1": abidjan}
	register(t, e, "sec@example.com", pw)
	u, err := e.store.ByEmail(context.Background(), "sec@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return e, u.ID
}

func loginUA(t *testing.T, e *env, ip, ua string) *application.Session {
	t.Helper()
	s, err := e.svc.Login(context.Background(), application.LoginInput{Email: "sec@example.com", Password: pw, IP: ip, UserAgent: ua})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	return s
}

func testSecuriteAppareilsConnectesEtDeconnexion(t *testing.T) {
	ctx := context.Background()
	e, uid := secEnv(t)
	pc := loginUA(t, e, "41.202.10.1", uaChromeWin)
	e.clock.Advance(time.Minute)
	phone := loginUA(t, e, "10.0.0.5", uaFirefoxAnd) // adresse privée : aucune position

	list, err := e.svc.ListMySessions(ctx, uid, pc.FamilyID)
	if err != nil || len(list) != 2 {
		t.Fatalf("sessions = %d, %v", len(list), err)
	}
	if list[0].FamilyID != phone.FamilyID || list[0].Current || list[0].Device.Browser != "Firefox" || list[0].Device.OS != "Android" || list[0].Device.Kind != "mobile" || list[0].Location != nil {
		t.Fatalf("session la plus récente inattendue : %+v", list[0])
	}
	if !list[1].Current || list[1].Device.Browser != "Chrome" || list[1].Device.OS != "Windows" || list[1].Location == nil || list[1].Location.City != "Abidjan" {
		t.Fatalf("session courante inattendue : %+v", list[1])
	}

	// Un autre compte ne peut pas déconnecter cette session.
	register(t, e, "autre@example.com", pw)
	other, _ := e.store.ByEmail(ctx, "autre@example.com")
	if err := e.svc.RevokeMySession(ctx, other.ID, phone.FamilyID); code(err) != "NOT_FOUND" {
		t.Fatalf("session d'un autre compte : %v", err)
	}
	if err := e.svc.RevokeMySession(ctx, uid, phone.FamilyID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.RevokeMySession(ctx, uid, phone.FamilyID); code(err) != "NOT_FOUND" {
		t.Fatalf("déjà révoquée : %v", err)
	}
	if _, err := e.svc.Refresh(ctx, phone.RefreshToken, "10.0.0.5", uaFirefoxAnd); err == nil {
		t.Fatal("l'appareil déconnecté ne doit plus pouvoir renouveler sa session")
	}

	tablet := loginUA(t, e, "10.0.0.6", uaSafariIOS)
	n, err := e.svc.RevokeMyOtherSessions(ctx, uid, pc.FamilyID)
	if err != nil || n != 1 {
		t.Fatalf("autres appareils déconnectés = %d, %v", n, err)
	}
	if _, err := e.svc.Refresh(ctx, tablet.RefreshToken, "10.0.0.6", uaSafariIOS); err == nil {
		t.Fatal("les autres appareils doivent être déconnectés")
	}
	if _, err := e.svc.Refresh(ctx, pc.RefreshToken, "41.202.10.1", uaChromeWin); err != nil {
		t.Fatalf("l'appareil courant doit rester connecté : %v", err)
	}
	if _, err := e.svc.RevokeMyOtherSessions(ctx, uid, ""); code(err) != "VALIDATION_FAILED" {
		t.Fatalf("sans session courante, rien ne doit être révoqué : %v", err)
	}
}

func testSecuriteAlerteNouvelAppareilEtLienDeSecours(t *testing.T) {
	ctx := context.Background()
	e, uid := secEnv(t)
	alerts := func() []*eventsv1.NewDeviceLoginAlertRequested {
		var out []*eventsv1.NewDeviceLoginAlertRequested
		for _, ev := range e.events.OfType(application.EvNewDeviceLogin) {
			out = append(out, ev.Payload.(*eventsv1.NewDeviceLoginAlertRequested))
		}
		return out
	}
	loginUA(t, e, "41.202.10.1", uaChromeWin) // premier appareil du compte : aucune alerte
	loginUA(t, e, "41.202.10.1", uaChromeWin) // même appareil : aucune alerte
	if n := len(alerts()); n != 0 {
		t.Fatalf("alertes après le premier appareil = %d", n)
	}
	phone := loginUA(t, e, "41.202.10.1", uaFirefoxAnd)
	a := alerts()
	if len(a) != 1 || a[0].Email != "sec@example.com" || a[0].Browser != "Firefox" || a[0].Os != "Android" || a[0].City != "Abidjan" || a[0].Token == "" {
		t.Fatalf("alerte inattendue : %+v", a)
	}
	evs, _, _ := e.svc.ListMyLoginEvents(ctx, uid, 1, 10)
	if len(evs) != 3 || !evs[0].NewDevice || evs[1].NewDevice || evs[0].Location == nil || evs[0].Method != domain.LoginMethodPassword {
		t.Fatalf("historique inattendu : %+v", evs)
	}

	if err := e.svc.SecureAccountFromAlert(ctx, a[0].Token); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Refresh(ctx, phone.RefreshToken, "41.202.10.1", uaFirefoxAnd); err == nil {
		t.Fatal("« ce n'était pas moi » doit déconnecter tous les appareils")
	}
	if list, _ := e.svc.ListMySessions(ctx, uid, ""); len(list) != 0 {
		t.Fatalf("sessions restantes = %d", len(list))
	}
	if err := e.svc.SecureAccountFromAlert(ctx, a[0].Token); code(err) != "INVALID_CODE" {
		t.Fatalf("le lien ne sert qu'une fois : %v", err)
	}
	if err := e.svc.SecureAccountFromAlert(ctx, "jeton-inconnu-0123456789"); code(err) != "INVALID_CODE" {
		t.Fatalf("jeton inconnu : %v", err)
	}
}

func testSecuriteHistoriqueConserve90Jours(t *testing.T) {
	ctx := context.Background()
	e, uid := secEnv(t)
	loginUA(t, e, "41.202.10.1", uaChromeWin)
	e.clock.Advance(91 * 24 * time.Hour)
	if evs, total, _ := e.svc.ListMyLoginEvents(ctx, uid, 1, 10); total != 0 || len(evs) != 0 {
		t.Fatalf("au-delà de 90 jours, l'historique doit être vide : %d", total)
	}
	// L'appareil a été oublié avec l'historique : la connexion suivante n'est pas une alerte (aucun autre appareil connu).
	loginUA(t, e, "41.202.10.1", uaChromeWin)
	for i := 0; i < 3; i++ {
		e.clock.Advance(time.Hour)
		loginUA(t, e, "41.202.10.1", uaChromeWin)
	}
	evs, total, err := e.svc.ListMyLoginEvents(ctx, uid, 2, 3)
	if err != nil || total != 4 || len(evs) != 1 {
		t.Fatalf("pagination : total=%d page=%d %v", total, len(evs), err)
	}
	if n := len(e.events.OfType(application.EvNewDeviceLogin)); n != 0 {
		t.Fatalf("alertes = %d", n)
	}
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func testSecuritePhotoEtCouverture(t *testing.T) {
	ctx := context.Background()
	e, uid := secEnv(t)
	if u, _ := e.svc.GetUser(ctx, uid); u.AvatarUpdatedAt != nil || u.CoverUpdatedAt != nil {
		t.Fatal("aucune image au départ")
	}
	img, err := e.svc.PutMyImage(ctx, uid, domain.ImageAvatar, pngBytes(t, 4, 4))
	if err != nil || img.ContentType != "image/png" {
		t.Fatalf("photo : %v %+v", err, img)
	}
	if u, _ := e.svc.GetUser(ctx, uid); u.AvatarUpdatedAt == nil || u.CoverUpdatedAt != nil {
		t.Fatalf("dates des images : %+v %+v", u.AvatarUpdatedAt, u.CoverUpdatedAt)
	}
	got, err := e.svc.GetMyImage(ctx, uid, domain.ImageAvatar)
	if err != nil || !bytes.Equal(got.Data, pngBytes(t, 4, 4)) {
		t.Fatalf("relecture : %v", err)
	}
	for name, data := range map[string][]byte{
		"texte":          []byte("<svg onload=alert(1)>"),
		"PNG tronqué":    pngBytes(t, 4, 4)[:20],
		"trop grande":    append(pngBytes(t, 4, 4), make([]byte, domain.MaxImageBytes)...),
		"côté trop long": pngBytes(t, 5000, 1),
		"vide":           nil,
	} {
		if _, err := e.svc.PutMyImage(ctx, uid, domain.ImageCover, data); code(err) != "VALIDATION_FAILED" {
			t.Fatalf("%s : %v", name, err)
		}
	}
	if _, err := e.svc.PutMyImage(ctx, uid, "banner", pngBytes(t, 4, 4)); code(err) != "VALIDATION_FAILED" {
		t.Fatalf("type inconnu : %v", err)
	}
	if _, err := e.svc.GetMyImage(ctx, uid, domain.ImageCover); code(err) != "NOT_FOUND" {
		t.Fatalf("couverture absente : %v", err)
	}
	// Images d'un compte invisibles depuis un autre compte.
	register(t, e, "autre@example.com", pw)
	other, _ := e.store.ByEmail(ctx, "autre@example.com")
	if _, err := e.svc.GetMyImage(ctx, other.ID, domain.ImageAvatar); code(err) != "NOT_FOUND" {
		t.Fatalf("isolation : %v", err)
	}
	if err := e.svc.DeleteMyImage(ctx, uid, domain.ImageAvatar); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.DeleteMyImage(ctx, uid, domain.ImageAvatar); err != nil {
		t.Fatalf("suppression idempotente : %v", err)
	}
	if u, _ := e.svc.GetUser(ctx, uid); u.AvatarUpdatedAt != nil {
		t.Fatal("la photo doit avoir disparu")
	}
}
