// Package server assemble le gateway : middlewares (CORS, CSRF, limitation de débit, journalisation), sécurité des
// jetons et handlers des routes validées. Le serveur HTTP lui-même (`internal/api`) est GÉNÉRÉ depuis openapi.yaml.
package server

import "time"

// Config regroupe les réglages du gateway. Les valeurs numériques sont les défauts techniques validés (G5).
type Config struct {
	Env            string
	AllowedOrigins []string // liste blanche CORS / CSRF (site web) ; jamais "*"
	TrustProxy     bool     // lire X-Forwarded-For (derrière un proxy de confiance seulement)

	CookieName   string // jeton de rafraîchissement
	CookiePath   string
	CookieDomain string // vide = cookie « host-only » (l'hôte de l'API seul) ; sinon domaine parent, ex. exemple.ci
	CookieSecure bool
	HSTS         bool

	UploadDir     string // images téléversées (devis, logo, cachet) ; à remplacer par un stockage objet en production
	DefaultLocale string
	MaxBodyBytes  int64

	AnonPerMinute     int // 120 / min / IP
	UserPerMinute     int // 600 / min / utilisateur
	LoginPerMinute    int // 10 / min / IP
	RegisterPerHour   int // 5 / h / IP
	JWKSMinRefresh    time.Duration
	ClockLeeway       time.Duration
	RequestTimeoutRPC time.Duration
}

// DefaultConfig renvoie les défauts (production : cookies Secure, HSTS).
func DefaultConfig() Config {
	return Config{
		Env: "prod", CookieName: "ff_refresh", CookiePath: "/v1/auth", CookieSecure: true, HSTS: true,
		DefaultLocale: "fr", MaxBodyBytes: 1 << 20, UploadDir: "uploads",
		AnonPerMinute: 120, UserPerMinute: 600, LoginPerMinute: 10, RegisterPerHour: 5,
		JWKSMinRefresh: 30 * time.Second, ClockLeeway: 30 * time.Second, RequestTimeoutRPC: 10 * time.Second,
	}
}
