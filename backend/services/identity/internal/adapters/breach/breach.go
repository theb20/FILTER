// Package breach refuse les mots de passe connus comme compromis.
//
// Source principale : service « Pwned Passwords » par PRÉFIXE de condensat (k-anonymat) : seuls les 5 premiers
// caractères du SHA-1 quittent le serveur ; ni le mot de passe, ni son condensat complet ne sont envoyés, journalisés
// ou conservés. Si le service ne répond pas dans le délai imparti, on bascule sur une liste LOCALE des mots de passe
// les plus courants, SANS jamais bloquer l'inscription. Le tout derrière l'interface application.BreachChecker.
package breach

import (
	"bufio"
	"context"
	"crypto/sha1" //nolint:gosec // SHA-1 imposé par le protocole k-anonymat de Pwned Passwords (pas un usage de sécurité)
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL est l'API publique par plage de condensat.
const DefaultBaseURL = "https://api.pwnedpasswords.com/range/"

// Checker interroge le service par préfixe, avec repli local.
type Checker struct {
	BaseURL  string
	Client   *http.Client
	Timeout  time.Duration // délai court (défaut 1,5 s)
	Fallback *Local
	Log      *slog.Logger
}

// NewChecker construit un contrôleur avec un client HTTP dédié (pas de redirection, pas de cookie).
func NewChecker(baseURL string, timeout time.Duration, fallback *Local, log *slog.Logger) *Checker {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	return &Checker{
		BaseURL: baseURL, Timeout: timeout, Fallback: fallback, Log: log,
		Client: &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

// Compromised ne renvoie JAMAIS d'erreur : une panne du service externe ne doit pas bloquer l'inscription.
func (c *Checker) Compromised(ctx context.Context, password string) (bool, error) {
	sum := sha1.Sum([]byte(password)) //nolint:gosec
	full := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := full[:5], full[5:]

	hit, err := c.queryRange(ctx, prefix, suffix)
	if err == nil {
		return hit, nil
	}
	// Journal SANS secret : ni mot de passe, ni préfixe, ni condensat. Seulement le fait du repli.
	if c.Log != nil {
		c.Log.Warn("contrôle des mots de passe compromis : service indisponible, repli sur la liste locale")
	}
	if c.Fallback != nil {
		return c.Fallback.Contains(password), nil
	}
	return false, nil
}

func (c *Checker) queryRange(ctx context.Context, prefix, suffix string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+prefix, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Add-Padding", "true") // masque le nombre réel de résultats
	req.Header.Set("User-Agent", "filterfinder-identity")
	resp, err := c.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, &statusError{resp.StatusCode}
	}
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 4<<20))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		sfx, count, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(sfx, suffix) {
			continue
		}
		// Les lignes de REMPLISSAGE (padding) ont un compteur à 0 : elles ne signalent aucune fuite.
		if n, err := strconv.Atoi(strings.TrimSpace(count)); err == nil && n > 0 {
			return true, nil
		}
	}
	return false, sc.Err()
}

type statusError struct{ code int }

func (e *statusError) Error() string { return "statut HTTP " + strconv.Itoa(e.code) }
