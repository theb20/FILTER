// Package devmail est une BOÎTE D'ENVOI DE DÉVELOPPEMENT : tant que le service notification n'existe pas, elle écrit dans
// un fichier les e-mails que le système aurait envoyés (liens d'invitation, de vérification…), pour pouvoir tester les
// parcours en local. Les jetons y figurent EN CLAIR : à n'activer qu'avec ENV=dev (refusé ailleurs par cmd/identity).
package devmail

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"github.com/theb20/FILTER/backend/pkg/natsbus"
	"google.golang.org/protobuf/proto"
)

// Sink écrit les « e-mails » dans un fichier.
type Sink struct {
	Path          string
	BackofficeURL string // ex. http://localhost:3100
	SiteURL       string // ex. http://localhost:5173
	mu            sync.Mutex
}

// Run s'abonne aux événements d'identity et écrit un message par événement concerné, jusqu'à l'annulation du contexte.
func (s *Sink) Run(ctx context.Context, bus *natsbus.Bus, log *slog.Logger) (stop func(), err error) {
	return bus.Subscribe(ctx, "devmail", "identity.>", func(_ context.Context, _ string, b []byte) error {
		var env eventsv1.Envelope
		if err := proto.Unmarshal(b, &env); err != nil {
			return nil // message illisible : on ne le rejoue pas
		}
		if line := s.render(&env); line != "" {
			if err := s.write(line); err != nil {
				log.Error("devmail : écriture", "error", err.Error())
			}
		}
		return nil
	})
}

func (s *Sink) render(env *eventsv1.Envelope) string {
	switch env.Type {
	case "identity.staff_invitation_requested.v1":
		var m eventsv1.StaffInvitationRequested
		if proto.Unmarshal(env.Payload, &m) != nil {
			return ""
		}
		return fmt.Sprintf("Invitation au backoffice pour %s\n  Lien : %s/invitation?token=%s", m.Email, s.BackofficeURL, m.Token)
	case "identity.staff_two_factor_reset.v1":
		var m eventsv1.StaffTwoFactorReset
		if proto.Unmarshal(env.Payload, &m) != nil {
			return ""
		}
		return fmt.Sprintf("Double authentification réinitialisée pour %s : un nouvel enregistrement sera demandé à la prochaine connexion.", m.Email)
	case "identity.email_verification_requested.v1":
		var m eventsv1.EmailVerificationRequested
		if proto.Unmarshal(env.Payload, &m) != nil {
			return ""
		}
		return fmt.Sprintf("Vérification d'adresse pour %s\n  Lien : %s/verify-email?token=%s", m.Email, s.SiteURL, m.Token)
	case "identity.registration_attempt_on_existing_account.v1":
		var m eventsv1.RegistrationAttemptOnExistingAccount
		if proto.Unmarshal(env.Payload, &m) != nil {
			return ""
		}
		return fmt.Sprintf("Tentative d'inscription sur un compte existant : %s", m.Email)
	case "identity.suspicious_login_alert_requested.v1":
		var m eventsv1.SuspiciousLoginAlertRequested
		if proto.Unmarshal(env.Payload, &m) != nil {
			return ""
		}
		return fmt.Sprintf("Alerte : %d échecs de connexion sur le compte %s (dernière adresse IP %s)", m.Failures, m.Email, m.LastIp)
	}
	return ""
}

func (s *Sink) write(msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "[%s] %s\n\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	return err
}
