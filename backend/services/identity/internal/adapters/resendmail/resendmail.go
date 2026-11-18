// Package resendmail envoie les e-mails transactionnels d'identity (vérification d'adresse, réinitialisation du mot de
// passe, invitation au backoffice, alertes de sécurité) via l'API HTTP de Resend.
//
// Il s'abonne aux mêmes événements que devmail : identity écrit dans l'OUTBOX (même transaction que les données), le
// relais les publie sur NATS JetStream, et ce puits les consomme. Conséquences voulues :
//   - un e-mail n'est jamais envoyé pour une transaction annulée ;
//   - une panne de Resend ne fait pas échouer l'inscription : le message est simplement rejoué (NAK) ;
//   - l'identifiant d'événement sert de clé d'idempotence, donc un rejeu ne provoque PAS de second envoi.
//
// SECRET : la clé d'API est lue dans l'environnement (RESEND_API_KEY). Elle n'est jamais journalisée, ni écrite dans un
// fichier, ni passée en argument de ligne de commande. Les jetons et codes contenus dans les e-mails ne sont JAMAIS
// journalisés non plus (seuls le type d'événement et une adresse masquée le sont).
package resendmail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"google.golang.org/protobuf/proto"
)

// DefaultBaseURL est l'API publique de Resend.
const DefaultBaseURL = "https://api.resend.com"

// defaultMaxAge : âge au-delà duquel un événement n'est plus transformé en e-mail. Deux situations réelles l'imposent :
//   - PREMIER démarrage de ce puits sur un flux déjà alimenté : le consommateur durable part du début du flux et
//     rejouerait TOUT l'historique (anciennes vérifications d'adresse, anciens codes de réinitialisation) ;
//   - reprise après une longue panne : un code valable 15 minutes n'a plus aucun intérêt des heures plus tard, et un
//     e-mail qui arrive hors contexte inquiète le destinataire.
const defaultMaxAge = 2 * time.Hour

// Sink consomme les événements d'identity et envoie l'e-mail correspondant.
type Sink struct {
	APIKey        string        // RESEND_API_KEY (obligatoire)
	From          string        // expéditeur vérifié chez Resend, ex. « FilterFinder <bonjour@filterfinder.be> »
	ReplyTo       string        // adresse de réponse réelle (facultatif)
	BrandName     string        // nom affiché dans le bandeau et le pied (défaut : « FilterFinder »)
	SiteURL       string        // ex. https://filterfinder.be
	BackofficeURL string        // ex. https://admin.filterfinder.be
	BaseURL       string        // surchargeable en test (défaut : DefaultBaseURL)
	Timeout       time.Duration // défaut 10 s
	MaxAge        time.Duration // au-delà, l'événement n'est plus envoyé (défaut 2 h)
	Client        *http.Client
	Log           *slog.Logger
}

// New construit un puits prêt à l'emploi avec un client HTTP dédié.
func New(s Sink) *Sink {
	if s.BaseURL == "" {
		s.BaseURL = DefaultBaseURL
	}
	if s.Timeout <= 0 {
		s.Timeout = 10 * time.Second
	}
	if s.MaxAge <= 0 {
		s.MaxAge = defaultMaxAge
	}
	if s.Client == nil {
		s.Client = &http.Client{Timeout: s.Timeout}
	}
	return &s
}

func (s *Sink) brandName() string {
	if s.BrandName == "" {
		return "FilterFinder"
	}
	return s.BrandName
}

// Run s'abonne au flux d'identity et envoie un e-mail par événement concerné, jusqu'à l'annulation du contexte.
// Une erreur renvoyée au bus entraîne un NAK : l'événement est rejoué (Resend indisponible, quota atteint…).
func (s *Sink) Run(ctx context.Context, bus *natsbus.Bus, log *slog.Logger) (stop func(), err error) {
	if s.APIKey == "" {
		return nil, fmt.Errorf("resendmail : RESEND_API_KEY est obligatoire")
	}
	if s.From == "" {
		return nil, fmt.Errorf("resendmail : MAIL_FROM est obligatoire (expéditeur vérifié chez Resend)")
	}
	s.Log = log
	return bus.Subscribe(ctx, "resendmail", "identity.>", func(ctx context.Context, _ string, b []byte) error {
		var env eventsv1.Envelope
		if err := proto.Unmarshal(b, &env); err != nil {
			return nil // message illisible : inutile de le rejouer
		}
		if s.stale(&env, time.Now()) {
			s.log().Info("e-mail ignoré : événement trop ancien", "event", env.Type, "occurredAt", env.OccurredAt.AsTime().UTC().Format(time.RFC3339))
			return nil
		}
		m, ok := s.compose(&env)
		if !ok {
			return nil // événement sans e-mail associé
		}
		return s.send(ctx, env.EventId, env.Type, m)
	})
}

// stale : l'événement est-il trop vieux pour mériter encore un e-mail ? Une enveloppe sans date est envoyée (la clé
// d'idempotence empêche de toute façon le doublon).
func (s *Sink) stale(env *eventsv1.Envelope, now time.Time) bool {
	if env.GetOccurredAt() == nil {
		return false
	}
	return now.Sub(env.GetOccurredAt().AsTime()) > s.MaxAge
}

// Types d'événements donnant lieu à un e-mail. Ils servent aussi de clés aux jeux de libellés (voir locale.go) :
// ajouter une langue ne touche pas à ce fichier, ajouter un e-mail impose de le traduire dans les trois langues.
const (
	evVerification    = "identity.email_verification_requested.v1"
	evPasswordReset   = "identity.password_reset_requested.v1"
	evStaffInvitation = "identity.staff_invitation_requested.v1"
	evTwoFactorReset  = "identity.staff_two_factor_reset.v1"
	evExistingAccount = "identity.registration_attempt_on_existing_account.v1"
	evSuspiciousLogin = "identity.suspicious_login_alert_requested.v1"
	evNewDeviceLogin  = "identity.new_device_login_alert_requested.v1"
	evEmailChangeReq  = "identity.email_change_requested.v1"
	evEmailChangeDone = "identity.email_change_confirmed.v1"
)

// defaultResetMinutes : durée affichée quand l'événement ne porte pas d'échéance (voir resetCodeTTL côté application).
const defaultResetMinutes = 15

// draft assemble la partie RÉDACTIONNELLE du message dans la langue du destinataire : les marqueurs {brand} et
// {validity} sont remplacés ici, une fois pour toutes. L'appelant complète ensuite le destinataire, le lien et le code.
func (s *Sink) draft(p langPack, eventType, validity string) (message, bool) {
	t, ok := p.Mails[eventType]
	if !ok {
		return message{}, false
	}
	r := strings.NewReplacer("{brand}", s.brandName(), "{validity}", validity)
	m := message{
		Locale:    p.Code,
		Subject:   r.Replace(t.Subject),
		Preheader: r.Replace(t.Preheader),
		Title:     r.Replace(t.Title),
		CTALabel:  r.Replace(t.CTALabel),
		Note:      r.Replace(t.Note),
	}
	for _, para := range t.Paragraphs {
		m.Paragraphs = append(m.Paragraphs, r.Replace(para))
	}
	return m, true
}

// compose traduit un événement en message. ok vaut faux quand l'événement ne donne lieu à aucun e-mail.
//
// La LANGUE vient du champ `locale` de l'événement (celle choisie par le client à l'inscription ou déduite de son
// compte Google) ; l'enveloppe sert de repli, et toute valeur inconnue retombe sur le français.
func (s *Sink) compose(env *eventsv1.Envelope) (message, bool) {
	switch env.Type {

	case evVerification:
		var e eventsv1.EmailVerificationRequested
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		m, ok := s.draft(packFor(e.Locale, env.Locale), env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.Email
		m.CTAURL = s.SiteURL + "/verify-email?token=" + e.Token
		return m, true

	case evPasswordReset:
		var e eventsv1.PasswordResetRequested
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		p := packFor(e.Locale, env.Locale)
		minutes := defaultResetMinutes
		if e.ExpiresAt != nil {
			if d := time.Until(e.ExpiresAt.AsTime()).Round(time.Minute); d > 0 {
				minutes = int(d.Minutes())
			}
		}
		m, ok := s.draft(p, env.Type, p.validity(minutes))
		if !ok {
			return message{}, false
		}
		m.To, m.Code = e.Email, e.Code
		m.CTAURL = s.SiteURL + "/reset"
		return m, true

	case evStaffInvitation:
		var e eventsv1.StaffInvitationRequested
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		m, ok := s.draft(packFor(e.Locale, env.Locale), env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.Email
		m.CTAURL = s.BackofficeURL + "/invitation?token=" + e.Token
		return m, true

	case evTwoFactorReset:
		var e eventsv1.StaffTwoFactorReset
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		m, ok := s.draft(packFor(e.Locale, env.Locale), env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.Email
		m.CTAURL = s.BackofficeURL + "/login"
		return m, true

	case evExistingAccount:
		var e eventsv1.RegistrationAttemptOnExistingAccount
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		m, ok := s.draft(packFor(e.Locale, env.Locale), env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.Email
		m.CTAURL = s.SiteURL + "/sign"
		return m, true

	case evEmailChangeReq: // envoyé à la NOUVELLE adresse : lien de confirmation (le jeton n'est jamais journalisé)
		var e eventsv1.EmailChangeRequested
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		m, ok := s.draft(packFor(e.Locale, env.Locale), env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.NewEmail
		m.CTAURL = s.SiteURL + "/confirm-email-change?token=" + e.Token
		return m, true

	case evEmailChangeDone: // envoyé à l'ANCIENNE adresse : information et alerte
		var e eventsv1.EmailChangeConfirmed
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		p := packFor(e.Locale, env.Locale)
		m, ok := s.draft(p, env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.OldEmail
		m.CTAURL = s.SiteURL + "/reset"
		m.Rows = []row{{Label: p.RowNewEmail, Value: e.NewEmail}}
		if env.OccurredAt != nil {
			m.Rows = append(m.Rows, row{Label: p.RowDate, Value: env.OccurredAt.AsTime().UTC().Format(p.DateFormat)})
		}
		return m, true

	case evNewDeviceLogin:
		var e eventsv1.NewDeviceLoginAlertRequested
		if proto.Unmarshal(env.Payload, &e) != nil || e.Token == "" {
			return message{}, false
		}
		p := packFor(e.Locale, env.Locale)
		m, ok := s.draft(p, env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.Email
		m.CTAURL = s.SiteURL + "/secure-account?token=" + e.Token
		if e.At != nil {
			m.Rows = append(m.Rows, row{Label: p.RowDate, Value: e.At.AsTime().UTC().Format(p.DateFormat)})
		}
		m.Rows = append(m.Rows, row{Label: p.RowDevice, Value: strings.TrimSpace(e.Browser + " · " + e.Os)})
		if place := strings.Trim(strings.Join([]string{e.City, e.Country}, ", "), ", "); place != "" {
			m.Rows = append(m.Rows, row{Label: p.RowPlace, Value: place})
		}
		if e.Ip != "" {
			m.Rows = append(m.Rows, row{Label: p.RowLastIP, Value: e.Ip})
		}
		return m, true

	case evSuspiciousLogin:
		var e eventsv1.SuspiciousLoginAlertRequested
		if proto.Unmarshal(env.Payload, &e) != nil {
			return message{}, false
		}
		p := packFor(e.Locale, env.Locale)
		m, ok := s.draft(p, env.Type, "")
		if !ok {
			return message{}, false
		}
		m.To = e.Email
		m.CTAURL = s.SiteURL + "/reset"
		m.Rows = []row{{Label: p.RowFailures, Value: strconv.Itoa(int(e.Failures))}}
		if e.LastIp != "" {
			m.Rows = append(m.Rows, row{Label: p.RowLastIP, Value: e.LastIp})
		}
		if env.OccurredAt != nil {
			m.Rows = append(m.Rows, row{Label: p.RowDate, Value: env.OccurredAt.AsTime().UTC().Format(p.DateFormat)})
		}
		return m, true
	}
	return message{}, false
}

// resendRequest est le corps attendu par POST /emails.
type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

// send poste le message à Resend.
//
// Codes de retour : 2xx = envoyé (ACK). 429 et 5xx = incident passager, l'erreur remonte au bus pour un REJEU.
// Les autres 4xx (adresse invalide, expéditeur non vérifié…) ne seront jamais acceptés : on journalise et on ACK,
// pour ne pas bloquer indéfiniment la file sur un message impossible à délivrer.
func (s *Sink) send(ctx context.Context, eventID, eventType string, m message) error {
	html, text, err := s.render(m)
	if err != nil {
		s.log().Error("resendmail : rendu du gabarit", "event", eventType, "error", err.Error())
		return nil // un gabarit cassé ne se répare pas par un rejeu
	}

	body, err := json.Marshal(resendRequest{
		From: s.From, To: []string{m.To}, Subject: m.Subject, HTML: html, Text: text, ReplyTo: s.ReplyTo,
	})
	if err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(s.BaseURL, "/")+"/emails", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	// Clé d'idempotence : un rejeu du même événement n'envoie pas un second e-mail.
	if eventID != "" {
		req.Header.Set("Idempotency-Key", eventID)
	}

	res, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("resendmail : appel à Resend : %w", err) // réseau : rejeu
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		s.log().Info("e-mail envoyé", "event", eventType, "to", maskEmail(m.To))
		return nil
	case res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500:
		return fmt.Errorf("resendmail : Resend a répondu %d (rejeu)", res.StatusCode)
	default:
		// Le corps d'erreur de Resend décrit le problème (expéditeur non vérifié, adresse invalide…) sans jamais
		// reprendre le contenu du message : on peut le journaliser.
		s.log().Error("resendmail : e-mail refusé", "event", eventType, "to", maskEmail(m.To), "status", res.StatusCode, "detail", readDetail(res.Body))
		return nil
	}
}

func (s *Sink) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}

// readDetail lit le champ `message` de l'erreur Resend, sans jamais dépasser quelques kilo-octets.
func readDetail(r io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil {
		return ""
	}
	var e struct {
		Message string `json:"message"`
		Name    string `json:"name"`
	}
	if json.Unmarshal(raw, &e) == nil && e.Message != "" {
		return e.Message
	}
	return strings.TrimSpace(string(raw))
}

// maskEmail ne laisse que la première lettre du nom et le domaine : les journaux restent exploitables sans exposer
// l'adresse complète.
func maskEmail(e string) string {
	at := strings.LastIndex(e, "@")
	if at <= 0 {
		return "***"
	}
	return e[:1] + "***" + e[at:]
}
