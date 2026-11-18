package resendmail

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/layout.html
var templatesFS embed.FS

var layout = template.Must(template.ParseFS(templatesFS, "templates/layout.html"))

// row est une paire libellé / valeur affichée dans le tableau de détails.
type row struct{ Label, Value string }

// message est le CONTENU d'un e-mail, indépendant du gabarit : le rendu HTML et la version texte en découlent tous deux.
// Les champs facultatifs (Code, CTAURL, Rows, Note) masquent la section correspondante du gabarit quand ils sont vides.
type message struct {
	To        string // destinataire
	Locale    string // langue retenue pour ce message : "fr", "en" ou "es"
	Subject   string
	Preheader string // première ligne affichée par la boîte de réception, sous l'objet
	Title     string
	// Paragraphs : corps du message. `html/template` échappe chaque paragraphe (aucune balise injectable par les données).
	Paragraphs []string
	Code       string // code à recopier (réinitialisation du mot de passe)
	CTALabel   string
	CTAURL     string
	Rows       []row
	Note       string // encadré d'avertissement en bas de message

	// Remplis par render : constants pour tous les messages.
	BrandName     string
	FooterLine    string
	FallbackLabel string
	Lang          string
	Year          int
}

// render produit le corps HTML et le repli texte brut (exigé par les filtres anti-spam et les lecteurs en mode texte).
// Les libellés communs (lien de repli, pied de page) suivent la langue déjà retenue par compose.
func (s *Sink) render(m message) (html, text string, err error) {
	p := packFor(m.Locale)
	m.BrandName = s.brandName()
	m.Lang = p.Code
	m.Year = time.Now().Year()
	m.FallbackLabel = p.FallbackLabel
	m.FooterLine = p.FooterLine

	var buf bytes.Buffer
	if err := layout.Execute(&buf, m); err != nil {
		return "", "", fmt.Errorf("gabarit : %w", err)
	}
	return buf.String(), plainText(m, p), nil
}

// plainText reconstruit la version texte à partir du même contenu (jamais une conversion du HTML).
func plainText(m message, p langPack) string {
	var b strings.Builder
	b.WriteString(m.Title)
	b.WriteString("\n\n")
	for _, p := range m.Paragraphs {
		b.WriteString(p)
		b.WriteString("\n\n")
	}
	if m.Code != "" {
		b.WriteString(p.CodeLabel)
		b.WriteString(m.Code)
		b.WriteString("\n\n")
	}
	if m.CTAURL != "" {
		b.WriteString(m.CTALabel)
		b.WriteString(" : ")
		b.WriteString(m.CTAURL)
		b.WriteString("\n\n")
	}
	for _, r := range m.Rows {
		b.WriteString(r.Label)
		b.WriteString(" : ")
		b.WriteString(r.Value)
		b.WriteString("\n")
	}
	if len(m.Rows) > 0 {
		b.WriteString("\n")
	}
	if m.Note != "" {
		b.WriteString(m.Note)
		b.WriteString("\n\n")
	}
	b.WriteString("— ")
	b.WriteString(m.BrandName)
	b.WriteString("\n")
	b.WriteString(m.FooterLine)
	b.WriteString("\n")
	return b.String()
}
