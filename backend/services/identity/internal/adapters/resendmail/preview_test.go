package resendmail

import (
	"os"
	"strings"
	"testing"

	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"google.golang.org/protobuf/proto"
)

// TestPreview écrit un aperçu HTML des gabarits quand MAIL_PREVIEW_OUT est défini (outil de relecture, pas une vérification).
func TestPreview(t *testing.T) {
	out := os.Getenv("MAIL_PREVIEW_OUT")
	if out == "" {
		t.Skip("définir MAIL_PREVIEW_OUT pour produire l'aperçu")
	}
	s := New(Sink{APIKey: "x", From: "x", SiteURL: "https://filterfinder.be", BackofficeURL: "https://admin.filterfinder.be"})
	cases := []struct {
		typ string
		msg proto.Message
	}{
		{"identity.email_verification_requested.v1", &eventsv1.EmailVerificationRequested{Email: "jean@example.be", Token: "TOKEN"}},
		{"identity.password_reset_requested.v1", &eventsv1.PasswordResetRequested{Email: "jean@example.be", Code: "482913"}},
		{"identity.staff_invitation_requested.v1", &eventsv1.StaffInvitationRequested{Email: "staff@example.be", Token: "TOKEN"}},
		{"identity.staff_two_factor_reset.v1", &eventsv1.StaffTwoFactorReset{Email: "staff@example.be"}},
		{"identity.registration_attempt_on_existing_account.v1", &eventsv1.RegistrationAttemptOnExistingAccount{Email: "jean@example.be"}},
		{"identity.suspicious_login_alert_requested.v1", &eventsv1.SuspiciousLoginAlertRequested{Email: "jean@example.be", Failures: 5, LastIp: "81.240.12.7"}},
	}
	var b strings.Builder
	b.WriteString(`<!doctype html><meta charset="utf-8"><title>Aperçu des e-mails</title><body style="margin:0;background:#e9eaec;font-family:system-ui,sans-serif">`)
	for _, c := range cases {
		b.WriteString(`<h2 style="margin:28px 0 0;padding:0 12px;font-size:14px;color:#333">` + c.typ + `</h2>`)
		b.WriteString(`<div style="display:flex;gap:12px;overflow-x:auto;padding:8px 12px">`)
		for _, lang := range []string{"fr", "en", "es"} {
			// La langue voyage dans l'enveloppe : le gabarit, lui, est le même pour les trois.
			raw, _ := proto.Marshal(c.msg)
			m, ok := s.compose(&eventsv1.Envelope{Type: c.typ, Payload: raw, Locale: lang})
			if !ok {
				t.Fatalf("%s : non composé", c.typ)
			}
			html, _, err := s.render(m)
			if err != nil {
				t.Fatalf("%s : %v", c.typ, err)
			}
			b.WriteString(`<div style="flex:0 0 auto"><code style="font-size:12px;color:#555">` + lang + ` — ` + m.Subject + `</code>`)
			b.WriteString(`<iframe style="width:620px;height:700px;border:0;display:block" srcdoc="` + strings.ReplaceAll(html, `"`, "&quot;") + `"></iframe></div>`)
		}
		b.WriteString(`</div>`)
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("aperçu écrit : %s", out)
}
