// Package domain : collections de contenu du site et leurs règles de validation. Aucune dépendance technique.
package domain

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
)

// DefaultLocale : langue obligatoire des textes par langue (le site retombe dessus quand une traduction manque).
const DefaultLocale = "fr"

// Field décrit un champ d'une collection. Types : str, text, slug, code, loc, int, num, bool, url, email, ts, uuid, enum, strs, uuids, items.
type Field struct {
	Name           string
	Type           string
	Required       bool
	Enum           []string
	Min, Max       int
	HasMin, HasMax bool
}

// Kind décrit une collection (ou un réglage unique).
type Kind struct {
	Key         string
	Unique      string // champ à valeur unique dans la collection (slug, code, e-mail) ; vide = aucun
	Upper       bool   // la valeur unique est mise en majuscules (codes promotionnels)
	Sort        string // champ numérique d'ordre d'affichage
	NoAuditData bool   // données personnelles : le journal d'audit ne conserve pas le contenu
	Fields      []Field
}

var (
	slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	codeRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{2,63}$`)
	langRe = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
)

func bad(field, msg string) error {
	return apperr.Validation(msg, apperr.Field{Pointer: "/" + field, Code: apperr.CodeValidation, Message: msg})
}

func (k *Kind) field(name string) *Field {
	for i := range k.Fields {
		if k.Fields[i].Name == name {
			return &k.Fields[i]
		}
	}
	return nil
}

// Normalize valide `data` et renvoie sa forme canonique (champs inconnus refusés, valeurs vides retirées, majuscules des codes).
func (k *Kind) Normalize(data map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(data))
	for name := range data {
		if k.field(name) == nil {
			return nil, bad(name, "champ inconnu")
		}
	}
	for _, f := range k.Fields {
		v, present := data[f.Name]
		if !present || v == nil {
			if f.Required {
				return nil, bad(f.Name, "ce champ est obligatoire")
			}
			continue
		}
		nv, err := f.check(v)
		if err != nil {
			return nil, err
		}
		if nv == nil { // valeur vide (texte vide, liste vide) : le champ est retiré
			if f.Required {
				return nil, bad(f.Name, "ce champ est obligatoire")
			}
			continue
		}
		if k.Upper && f.Name == k.Unique {
			nv = strings.ToUpper(nv.(string))
		}
		out[f.Name] = nv
	}
	return out, nil
}

func (f Field) check(v any) (any, error) {
	switch f.Type {
	case "str", "text", "slug", "code", "url", "email", "ts", "uuid", "enum":
		s, ok := v.(string)
		if !ok {
			return nil, bad(f.Name, "doit être un texte")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		limit := 255
		if f.Type == "text" {
			limit = 20000
		}
		if len([]rune(s)) > limit {
			return nil, bad(f.Name, fmt.Sprintf("%d caractères au maximum", limit))
		}
		switch f.Type {
		case "slug":
			if len(s) > 80 || !slugRe.MatchString(s) {
				return nil, bad(f.Name, "minuscules, chiffres et tirets uniquement (ex. filtres-hydrauliques)")
			}
		case "code":
			s = strings.ToUpper(s)
			if !codeRe.MatchString(s) {
				return nil, bad(f.Name, "majuscules, chiffres, tirets ou soulignés (3 à 64 caractères)")
			}
		case "url":
			u, err := url.Parse(s)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(s) > 2000 {
				return nil, bad(f.Name, "adresse web invalide (http:// ou https://)")
			}
		case "email":
			a, err := mail.ParseAddress(s)
			if err != nil || a.Address != s || len(s) > 254 {
				return nil, bad(f.Name, "adresse e-mail invalide")
			}
			s = strings.ToLower(s)
		case "ts":
			if _, err := time.Parse(time.RFC3339, s); err != nil {
				return nil, bad(f.Name, "date invalide (format RFC 3339)")
			}
		case "uuid":
			if _, err := uuid.Parse(s); err != nil {
				return nil, bad(f.Name, "identifiant invalide")
			}
		case "enum":
			for _, e := range f.Enum {
				if e == s {
					return s, nil
				}
			}
			return nil, bad(f.Name, "valeur non autorisée")
		}
		return s, nil
	case "loc":
		m, ok := v.(map[string]any)
		if !ok {
			return nil, bad(f.Name, "doit être un texte par langue")
		}
		out := map[string]any{}
		for lang, t := range m {
			if !langRe.MatchString(lang) {
				return nil, bad(f.Name, "code de langue invalide : "+lang)
			}
			s, ok := t.(string)
			if !ok {
				return nil, bad(f.Name, "chaque traduction doit être un texte")
			}
			if s = strings.TrimSpace(s); s != "" {
				if len([]rune(s)) > 20000 {
					return nil, bad(f.Name, "20000 caractères au maximum par langue")
				}
				out[lang] = s
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		if _, ok := out[DefaultLocale]; !ok {
			return nil, bad(f.Name, "la langue par défaut (fr) est obligatoire")
		}
		return out, nil
	case "int", "num":
		n, ok := v.(float64)
		if !ok {
			return nil, bad(f.Name, "doit être un nombre")
		}
		if f.Type == "int" && n != float64(int64(n)) {
			return nil, bad(f.Name, "doit être un nombre entier")
		}
		if (f.HasMin && n < float64(f.Min)) || (f.HasMax && n > float64(f.Max)) {
			return nil, bad(f.Name, "valeur hors limites")
		}
		return n, nil
	case "bool":
		b, ok := v.(bool)
		if !ok {
			return nil, bad(f.Name, "doit être vrai ou faux")
		}
		return b, nil
	case "strs", "uuids":
		l, ok := v.([]any)
		if !ok {
			return nil, bad(f.Name, "doit être une liste")
		}
		if len(l) > 50 {
			return nil, bad(f.Name, "50 éléments au maximum")
		}
		out := make([]any, 0, len(l))
		for _, x := range l {
			s, ok := x.(string)
			if !ok {
				return nil, bad(f.Name, "la liste ne contient que des textes")
			}
			if s = strings.TrimSpace(s); s == "" {
				continue
			}
			if f.Type == "uuids" {
				if _, err := uuid.Parse(s); err != nil {
					return nil, bad(f.Name, "identifiant invalide : "+s)
				}
			} else if len(s) > 80 {
				return nil, bad(f.Name, "80 caractères au maximum par élément")
			}
			out = append(out, s)
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	case "items": // lignes d'une demande de devis
		l, ok := v.([]any)
		if !ok {
			return nil, bad(f.Name, "doit être une liste")
		}
		if len(l) > 100 {
			return nil, bad(f.Name, "100 lignes au maximum")
		}
		out := make([]any, 0, len(l))
		for _, x := range l {
			m, ok := x.(map[string]any)
			if !ok {
				return nil, bad(f.Name, "ligne invalide")
			}
			ref, _ := m["reference"].(string)
			if ref = strings.TrimSpace(ref); ref == "" || len(ref) > 255 {
				return nil, bad(f.Name, "référence obligatoire (255 caractères au maximum)")
			}
			q, _ := m["quantity"].(float64)
			if q < 1 || q > 100000 || q != float64(int64(q)) {
				return nil, bad(f.Name, "quantité invalide")
			}
			line := map[string]any{"reference": ref, "quantity": q}
			if id, ok := m["productId"].(string); ok && id != "" {
				if _, err := uuid.Parse(id); err != nil {
					return nil, bad(f.Name, "identifiant de produit invalide")
				}
				line["productId"] = id
			}
			if s, ok := m["name"].(string); ok && strings.TrimSpace(s) != "" && len(s) <= 255 {
				line["name"] = strings.TrimSpace(s)
			}
			out = append(out, line)
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	}
	return nil, bad(f.Name, "type de champ non géré : "+f.Type)
}
