package resendmail

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	eventsv1 "github.com/theb20/FILTER/backend/pkg/gen/events/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// capture monte un faux Resend et renvoie le puits, la dernière requête reçue et ses en-têtes.
func capture(t *testing.T, status int, body string) (*Sink, *resendRequest, *http.Header) {
	t.Helper()
	var got resendRequest
	var hdr http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	s := New(Sink{
		APIKey: "re_test_key", From: "FilterFinder <bonjour@example.test>", BaseURL: srv.URL,
		SiteURL: "https://site.test", BackofficeURL: "https://admin.test",
	})
	return s, &got, &hdr
}

func envelope(t *testing.T, id, typ string, payload proto.Message) *eventsv1.Envelope {
	t.Helper()
	b, err := proto.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal : %v", err)
	}
	return &eventsv1.Envelope{EventId: id, Type: typ, Payload: b}
}

func TestVerificationEmail(t *testing.T) {
	s, got, hdr := capture(t, 200, `{"id":"abc"}`)
	env := envelope(t, "evt-1", "identity.email_verification_requested.v1",
		&eventsv1.EmailVerificationRequested{Email: "jean@example.test", Token: "tok-123"})

	m, ok := s.compose(env)
	if !ok {
		t.Fatal("événement non pris en charge")
	}
	if err := s.send(context.Background(), env.EventId, env.Type, m); err != nil {
		t.Fatalf("send : %v", err)
	}

	if got.From != "FilterFinder <bonjour@example.test>" {
		t.Errorf("expéditeur = %q", got.From)
	}
	if len(got.To) != 1 || got.To[0] != "jean@example.test" {
		t.Errorf("destinataire = %v", got.To)
	}
	if hdr.Get("Authorization") != "Bearer re_test_key" {
		t.Errorf("autorisation = %q", hdr.Get("Authorization"))
	}
	// Un rejeu du même événement ne doit pas provoquer un second envoi : Resend s'appuie sur cette clé.
	if hdr.Get("Idempotency-Key") != "evt-1" {
		t.Errorf("clé d'idempotence = %q", hdr.Get("Idempotency-Key"))
	}
	link := "https://site.test/verify-email?token=tok-123"
	if !strings.Contains(got.HTML, link) {
		t.Error("le lien de vérification manque dans le HTML")
	}
	if !strings.Contains(got.Text, link) {
		t.Error("le lien de vérification manque dans la version texte")
	}
	if got.Text == "" || got.HTML == "" {
		t.Error("les deux versions (HTML et texte) sont obligatoires")
	}
}

func TestPasswordResetCarriesCode(t *testing.T) {
	s, got, _ := capture(t, 200, `{"id":"abc"}`)
	env := envelope(t, "evt-2", "identity.password_reset_requested.v1",
		&eventsv1.PasswordResetRequested{Email: "jean@example.test", Code: "482913"})

	m, _ := s.compose(env)
	if err := s.send(context.Background(), env.EventId, env.Type, m); err != nil {
		t.Fatalf("send : %v", err)
	}
	if !strings.Contains(got.HTML, "482913") || !strings.Contains(got.Text, "482913") {
		t.Error("le code de réinitialisation manque")
	}
}

func TestStaffInvitationUsesBackofficeURL(t *testing.T) {
	s, got, _ := capture(t, 200, `{"id":"abc"}`)
	env := envelope(t, "evt-3", "identity.staff_invitation_requested.v1",
		&eventsv1.StaffInvitationRequested{Email: "staff@example.test", Token: "inv-9"})

	m, _ := s.compose(env)
	if err := s.send(context.Background(), env.EventId, env.Type, m); err != nil {
		t.Fatalf("send : %v", err)
	}
	if !strings.Contains(got.HTML, "https://admin.test/invitation?token=inv-9") {
		t.Error("le lien d'invitation ne pointe pas vers le backoffice")
	}
}

// Les données du message sont échappées : une adresse contenant du HTML ne peut pas injecter de balise.
func TestDataIsEscaped(t *testing.T) {
	s, got, _ := capture(t, 200, `{"id":"abc"}`)
	env := envelope(t, "evt-4", "identity.suspicious_login_alert_requested.v1",
		&eventsv1.SuspiciousLoginAlertRequested{Email: "jean@example.test", Failures: 5, LastIp: `1.2.3.4"><script>alert(1)</script>`})

	m, _ := s.compose(env)
	if err := s.send(context.Background(), env.EventId, env.Type, m); err != nil {
		t.Fatalf("send : %v", err)
	}
	if strings.Contains(got.HTML, "<script>") {
		t.Error("balise script non échappée dans le corps HTML")
	}
}

// 5xx et 429 : incident passager côté Resend, l'erreur doit remonter pour que NATS rejoue l'événement.
func TestTransientErrorsAreRetried(t *testing.T) {
	for _, status := range []int{429, 500, 502} {
		s, _, _ := capture(t, status, `{"message":"oups"}`)
		env := envelope(t, "evt-5", "identity.email_verification_requested.v1",
			&eventsv1.EmailVerificationRequested{Email: "jean@example.test", Token: "t"})
		m, _ := s.compose(env)
		if err := s.send(context.Background(), env.EventId, env.Type, m); err == nil {
			t.Errorf("statut %d : une erreur était attendue (rejeu)", status)
		}
	}
}

// 4xx définitif (adresse invalide, expéditeur non vérifié) : on journalise et on acquitte, sinon la file se bloque.
func TestPermanentErrorIsAcknowledged(t *testing.T) {
	s, _, _ := capture(t, 422, `{"message":"Invalid to field","name":"validation_error"}`)
	env := envelope(t, "evt-6", "identity.email_verification_requested.v1",
		&eventsv1.EmailVerificationRequested{Email: "cassé", Token: "t"})
	m, _ := s.compose(env)
	if err := s.send(context.Background(), env.EventId, env.Type, m); err != nil {
		t.Errorf("422 ne doit pas être rejoué : %v", err)
	}
}

func TestUnknownEventSendsNothing(t *testing.T) {
	s, _, _ := capture(t, 200, `{}`)
	if _, ok := s.compose(&eventsv1.Envelope{Type: "identity.autre.v1"}); ok {
		t.Error("un événement sans e-mail associé ne doit rien composer")
	}
}

func TestMaskEmail(t *testing.T) {
	for in, want := range map[string]string{
		"jean.dupont@example.test": "j***@example.test",
		"sans-arobase":             "***",
		"":                         "***",
	} {
		if got := maskEmail(in); got != want {
			t.Errorf("maskEmail(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestRunRequiresKeyAndSender(t *testing.T) {
	if _, err := (&Sink{From: "a@b.test"}).Run(context.Background(), nil, nil); err == nil {
		t.Error("sans clé d'API, Run doit refuser de démarrer")
	}
	if _, err := (&Sink{APIKey: "k"}).Run(context.Background(), nil, nil); err == nil {
		t.Error("sans expéditeur, Run doit refuser de démarrer")
	}
}

// ── Langues ───────────────────────────────────────────────────────────────────

// Tout e-mail doit exister dans TOUTES les langues : un ajout non traduit échoue ici plutôt qu'en production.
func TestEveryMailIsTranslated(t *testing.T) {
	all := []string{evVerification, evPasswordReset, evStaffInvitation, evTwoFactorReset, evExistingAccount, evSuspiciousLogin, evEmailChangeReq, evEmailChangeDone}
	for code, p := range packs {
		if p.Code != code {
			t.Errorf("paquet %q : champ Code = %q", code, p.Code)
		}
		for name, value := range map[string]string{"FallbackLabel": p.FallbackLabel, "FooterLine": p.FooterLine, "CodeLabel": p.CodeLabel, "ValidityFmt": p.ValidityFmt, "DateFormat": p.DateFormat, "RowFailures": p.RowFailures, "RowLastIP": p.RowLastIP, "RowDate": p.RowDate, "RowNewEmail": p.RowNewEmail} {
			if value == "" {
				t.Errorf("paquet %q : libellé commun %s vide", code, name)
			}
		}
		for _, typ := range all {
			m, ok := p.Mails[typ]
			if !ok {
				t.Errorf("paquet %q : %s non traduit", code, typ)
				continue
			}
			if m.Subject == "" || m.Title == "" || m.CTALabel == "" || len(m.Paragraphs) == 0 {
				t.Errorf("paquet %q / %s : libellé obligatoire manquant", code, typ)
			}
		}
	}
	for _, code := range []string{"fr", "en", "es"} {
		if _, ok := packs[code]; !ok {
			t.Errorf("langue %q absente", code)
		}
	}
}

// La langue vient du champ `locale` de l'événement ; l'enveloppe sert de repli ; l'inconnu retombe sur le français.
func TestLocaleSelection(t *testing.T) {
	cases := []struct{ payload, envelope, wantLang, wantSubject string }{
		{"fr", "", "fr", "Confirmez votre adresse e-mail"},
		{"en", "", "en", "Confirm your email address"},
		{"es", "", "es", "Confirme su dirección de correo"},
		{"en-GB", "", "en", "Confirm your email address"},   // étiquette complète : seuls les deux premiers caractères comptent
		{"ES", "", "es", "Confirme su dirección de correo"}, // casse indifférente
		{"", "en", "en", "Confirm your email address"},      // repli sur l'enveloppe
		{"de", "", "fr", "Confirmez votre adresse e-mail"},  // langue non servie
		{"", "", "fr", "Confirmez votre adresse e-mail"},    // rien du tout
	}
	for _, c := range cases {
		s, got, _ := capture(t, 200, `{"id":"x"}`)
		env := envelope(t, "evt", evVerification, &eventsv1.EmailVerificationRequested{Email: "jean@example.test", Token: "tok", Locale: c.payload})
		env.Locale = c.envelope

		m, ok := s.compose(env)
		if !ok {
			t.Fatalf("locale %q : non composé", c.payload)
		}
		if m.Subject != c.wantSubject {
			t.Errorf("locale %q/%q : objet = %q, attendu %q", c.payload, c.envelope, m.Subject, c.wantSubject)
		}
		if err := s.send(context.Background(), "evt", env.Type, m); err != nil {
			t.Fatalf("send : %v", err)
		}
		if !strings.Contains(got.HTML, `<html lang="`+c.wantLang+`">`) {
			t.Errorf("locale %q/%q : attribut lang attendu %q", c.payload, c.envelope, c.wantLang)
		}
	}
}

// Les marqueurs {brand} et {validity} doivent tous être remplacés, dans chaque langue.
func TestPlaceholdersAreResolved(t *testing.T) {
	for _, code := range []string{"fr", "en", "es"} {
		s, _, _ := capture(t, 200, `{"id":"x"}`)
		s.BrandName = "FilterFinder"

		inv := envelope(t, "e", evStaffInvitation, &eventsv1.StaffInvitationRequested{Email: "a@b.test", Token: "t", Locale: code})
		m, _ := s.compose(inv)
		if !strings.Contains(m.Subject, "FilterFinder") {
			t.Errorf("%s : {brand} non remplacé dans l'objet (%q)", code, m.Subject)
		}

		res := envelope(t, "e", evPasswordReset, &eventsv1.PasswordResetRequested{
			Email: "a@b.test", Code: "123456", Locale: code,
			ExpiresAt: timestamppb.New(time.Now().Add(15 * time.Minute)),
		})
		m2, _ := s.compose(res)
		for _, txt := range append([]string{m2.Subject, m2.Preheader, m2.Title, m2.Note}, m2.Paragraphs...) {
			if strings.Contains(txt, "{brand}") || strings.Contains(txt, "{validity}") {
				t.Errorf("%s : marqueur non remplacé dans %q", code, txt)
			}
		}
		if !strings.Contains(m2.Preheader, "15") {
			t.Errorf("%s : durée de validité absente (%q)", code, m2.Preheader)
		}
	}
}

// Les détails de l'alerte de connexion (libellés ET format de date) suivent la langue.
func TestAlertRowsAreLocalised(t *testing.T) {
	want := map[string]string{"fr": "Tentatives échouées", "en": "Failed attempts", "es": "Intentos fallidos"}
	for code, label := range want {
		s, _, _ := capture(t, 200, `{"id":"x"}`)
		env := envelope(t, "e", evSuspiciousLogin, &eventsv1.SuspiciousLoginAlertRequested{Email: "a@b.test", Failures: 5, LastIp: "81.240.12.7", Locale: code})
		env.OccurredAt = timestamppb.New(time.Date(2026, 3, 9, 14, 5, 0, 0, time.UTC))

		m, _ := s.compose(env)
		if len(m.Rows) != 3 {
			t.Fatalf("%s : %d lignes de détail, 3 attendues", code, len(m.Rows))
		}
		if m.Rows[0].Label != label || m.Rows[0].Value != "5" {
			t.Errorf("%s : première ligne = %q / %q", code, m.Rows[0].Label, m.Rows[0].Value)
		}
		if !strings.Contains(m.Rows[2].Value, "2026") {
			t.Errorf("%s : date mal formatée (%q)", code, m.Rows[2].Value)
		}
	}
}

// La version texte reprend la langue du message (préfixe du code compris).
func TestPlainTextFollowsLocale(t *testing.T) {
	want := map[string]string{"fr": "Code : 482913", "en": "Code: 482913", "es": "Código: 482913"}
	for code, line := range want {
		s, got, _ := capture(t, 200, `{"id":"x"}`)
		env := envelope(t, "e", evPasswordReset, &eventsv1.PasswordResetRequested{Email: "a@b.test", Code: "482913", Locale: code})
		m, _ := s.compose(env)
		if err := s.send(context.Background(), "e", env.Type, m); err != nil {
			t.Fatalf("send : %v", err)
		}
		if !strings.Contains(got.Text, line) {
			t.Errorf("%s : version texte sans %q", code, line)
		}
	}
}

// Au premier démarrage, le consommateur durable repart du DÉBUT du flux : sans ce garde-fou, tout l'historique
// (anciennes inscriptions, anciens codes) repartirait en e-mails.
func TestEvenementTropAncienIgnore(t *testing.T) {
	s, _, _ := capture(t, 200, `{"id":"abc"}`)
	now := time.Now()
	env := envelope(t, "evt-vieux", "identity.email_verification_requested.v1",
		&eventsv1.EmailVerificationRequested{Email: "jean@example.test", Token: "tok-123"})

	if s.MaxAge != 2*time.Hour {
		t.Fatalf("âge maximal par défaut = %s", s.MaxAge)
	}
	if s.stale(env, now) {
		t.Error("une enveloppe sans date ne doit pas être écartée")
	}
	env.OccurredAt = timestamppb.New(now.Add(-3 * time.Hour))
	if !s.stale(env, now) {
		t.Error("un événement de 3 h doit être écarté")
	}
	env.OccurredAt = timestamppb.New(now.Add(-5 * time.Minute))
	if s.stale(env, now) {
		t.Error("un événement de 5 min doit être envoyé")
	}
}

// Changement d'adresse : le lien part à la NOUVELLE adresse, l'alerte à l'ANCIENNE ; jamais l'inverse.
func TestEmailChangeMailsGoToTheRightAddress(t *testing.T) {
	for _, code := range []string{"fr", "en", "es"} {
		s, _, _ := capture(t, 200, `{"id":"x"}`)
		s.SiteURL, s.BrandName = "https://site.example", "FilterFinder"

		req := envelope(t, "e1", evEmailChangeReq, &eventsv1.EmailChangeRequested{UserId: "u", OldEmail: "ancienne@example.test", NewEmail: "nouvelle@example.test", Token: "SECRET-TOKEN", Locale: code})
		m, ok := s.compose(req)
		if !ok || m.To != "nouvelle@example.test" || m.CTAURL != "https://site.example/confirm-email-change?token=SECRET-TOKEN" {
			t.Fatalf("%s demande : %+v ok=%v", code, m, ok)
		}
		if strings.Contains(m.Subject+m.Title+strings.Join(m.Paragraphs, " "), "{") {
			t.Errorf("%s : marqueur non remplacé", code)
		}

		done := envelope(t, "e2", evEmailChangeDone, &eventsv1.EmailChangeConfirmed{UserId: "u", OldEmail: "ancienne@example.test", NewEmail: "nouvelle@example.test", Locale: code})
		m, ok = s.compose(done)
		if !ok || m.To != "ancienne@example.test" || strings.Contains(m.CTAURL, "token") {
			t.Fatalf("%s confirmation : %+v ok=%v", code, m, ok)
		}
		if len(m.Rows) == 0 || m.Rows[0].Value != "nouvelle@example.test" || m.Rows[0].Label == "" {
			t.Errorf("%s : la nouvelle adresse doit figurer dans l'alerte : %+v", code, m.Rows)
		}
		if strings.Contains(m.Subject+m.Title+strings.Join(m.Paragraphs, " "), "{") {
			t.Errorf("%s : marqueur non remplacé", code)
		}
	}
}
