package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg" // DecodeConfig des JPEG
	_ "image/png"  // DecodeConfig des PNG
	"strings"
	"time"
)

// ── Sécurité du compte : appareils connectés, historique des connexions, photo et couverture ─────────────────

// LoginHistoryRetention : durée de conservation de l'historique des connexions et des appareils connus (décision du
// 18 nov. 2026). Au-delà, les lignes sont purgées ; un appareil purgé redevient « nouveau » (nouvelle alerte).
const LoginHistoryRetention = 90 * 24 * time.Hour

// LoginAlertTTL : validité du lien « ce n'était pas moi » de l'e-mail d'alerte.
const LoginAlertTTL = 7 * 24 * time.Hour

// PurposeLoginAlert : jeton du lien « ce n'était pas moi » (révoque toutes les sessions du compte).
const PurposeLoginAlert = "login_alert"

// Méthodes de connexion enregistrées dans l'historique.
const (
	LoginMethodPassword = "password"
	LoginMethodGoogle   = "google"
)

// GeoLocation : position APPROXIMATIVE déduite d'une adresse IP (souvent la ville de l'opérateur).
type GeoLocation struct {
	City, Region, CountryCode, Country string
	Latitude, Longitude                float64
}

// Device : appareil déduit de l'en-tête User-Agent. Approximatif par nature (l'en-tête est déclaratif).
type Device struct {
	Browser, OS string
	Kind        string // desktop | mobile | tablet | unknown
}

// ActiveSession : une session active (famille de jetons non révoquée, dernier jeton non expiré).
type ActiveSession struct {
	FamilyID   string
	CreatedAt  time.Time
	LastSeenAt time.Time // dernier renouvellement du jeton
	IP         string
	UserAgent  string
}

// LoginEvent : une connexion RÉUSSIE (l'historique des échecs reste dans le ralentissement progressif).
type LoginEvent struct {
	ID        string
	UserID    string
	At        time.Time
	IP        string
	UserAgent string
	Method    string
	NewDevice bool
	Location  *GeoLocation
}

// ParseUserAgent reconnaît les navigateurs et systèmes courants. L'ordre des tests compte : Edge et Opera se
// déclarent aussi « Chrome », Chrome se déclare aussi « Safari ».
func ParseUserAgent(ua string) Device {
	l := strings.ToLower(ua)
	d := Device{Browser: "Navigateur inconnu", OS: "Système inconnu", Kind: "unknown"}
	switch {
	case strings.Contains(l, "edg/") || strings.Contains(l, "edga/") || strings.Contains(l, "edgios/"):
		d.Browser = "Edge"
	case strings.Contains(l, "opr/") || strings.Contains(l, "opera"):
		d.Browser = "Opera"
	case strings.Contains(l, "samsungbrowser/"):
		d.Browser = "Samsung Internet"
	case strings.Contains(l, "firefox/") || strings.Contains(l, "fxios/"):
		d.Browser = "Firefox"
	case strings.Contains(l, "chrome/") || strings.Contains(l, "crios/"):
		d.Browser = "Chrome"
	case strings.Contains(l, "safari/") && strings.Contains(l, "version/"):
		d.Browser = "Safari"
	}
	switch {
	case strings.Contains(l, "ipad"):
		d.OS, d.Kind = "iPadOS", "tablet"
	case strings.Contains(l, "iphone") || strings.Contains(l, "ipod"):
		d.OS, d.Kind = "iOS", "mobile"
	case strings.Contains(l, "android"):
		d.OS, d.Kind = "Android", "mobile"
		if !strings.Contains(l, "mobile") {
			d.Kind = "tablet"
		}
	case strings.Contains(l, "windows"):
		d.OS, d.Kind = "Windows", "desktop"
	case strings.Contains(l, "mac os x") || strings.Contains(l, "macintosh"):
		d.OS, d.Kind = "macOS", "desktop"
	case strings.Contains(l, "cros"):
		d.OS, d.Kind = "ChromeOS", "desktop"
	case strings.Contains(l, "linux"):
		d.OS, d.Kind = "Linux", "desktop"
	}
	return d
}

// DeviceKey identifie un appareil « connu » d'un compte : navigateur, système et type d'appareil (sans la version,
// qui change à chaque mise à jour). Condensat : l'en-tête brut n'est pas conservé dans cette table.
func DeviceKey(userID string, d Device) string {
	h := sha256.Sum256([]byte(userID + "|" + d.Browser + "|" + d.OS + "|" + d.Kind))
	return hex.EncodeToString(h[:])
}

// ── Photo de profil et couverture ───────────────────────────────────────────────────────────────────────────

// Types d'image du compte.
const (
	ImageAvatar = "avatar"
	ImageCover  = "cover"
)

// MaxImageBytes : taille maximale d'une image (le site redimensionne et réencode avant l'envoi).
const MaxImageBytes = 600 * 1024

// maxImageSide : côté maximal accepté (garde-fou contre les images « bombes de décompression »).
const maxImageSide = 4096

// ErrImageInvalid : contenu qui n'est pas une image PNG, JPEG ou WebP valide, ou trop grande.
var ErrImageInvalid = errors.New("image invalide : PNG, JPEG ou WebP de 600 Kio et 4096 pixels de côté au plus")

// ValidImageKind indique si `k` est un type d'image du compte.
func ValidImageKind(k string) bool { return k == ImageAvatar || k == ImageCover }

// SniffImage reconnaît l'image par son CONTENU (jamais par une extension ou un type déclaré) et renvoie son type MIME.
func SniffImage(b []byte) (string, error) {
	if len(b) == 0 || len(b) > MaxImageBytes {
		return "", ErrImageInvalid
	}
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return checkDims(b, "image/png")
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return checkDims(b, "image/jpeg")
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "image/webp", nil // en-tête vérifié ; la bibliothèque standard ne décode pas le WebP
	}
	return "", ErrImageInvalid
}

func checkDims(b []byte, ct string) (string, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxImageSide || cfg.Height > maxImageSide {
		return "", ErrImageInvalid
	}
	return ct, nil
}

// UserImage : photo de profil ou couverture stockée.
type UserImage struct {
	Kind        string
	ContentType string
	Data        []byte
	UpdatedAt   time.Time
}
