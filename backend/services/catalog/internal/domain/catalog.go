// Package domain : règles du catalogue (taxonomie), sans dépendance technique.
package domain

import (
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/theb20/FILTER/backend/pkg/apperr"
)

// DefaultLanguage : langue par défaut, obligatoire dans tout texte localisé (les autres langues se replient dessus).
const DefaultLanguage = "fr"

// Localized : texte par langue (BCP 47), ex. {"fr": "Pompes", "en": "Pumps"}.
type Localized map[string]string

var (
	langRE = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	slugRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	keyRE  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	skuRE  = regexp.MustCompile(`^[A-Z0-9]+(-[A-Z0-9]+)*$`)
)

type fieldErrs []apperr.Field

func (f *fieldErrs) add(ptr, msg string) {
	*f = append(*f, apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
}
func (f fieldErrs) err(msg string) error {
	if len(f) == 0 {
		return nil
	}
	return apperr.Validation(msg, f...)
}

// Check valide un texte localisé : langue par défaut obligatoire et non vide, codes de langue valides, longueur bornée.
func (l Localized) check(ptr string, max int, required bool, f *fieldErrs) {
	if len(l) == 0 {
		if required {
			f.add(ptr, "le texte dans la langue par défaut ("+DefaultLanguage+") est obligatoire")
		}
		return
	}
	if strings.TrimSpace(l[DefaultLanguage]) == "" {
		f.add(ptr, "le texte dans la langue par défaut ("+DefaultLanguage+") est obligatoire")
	}
	for lang, v := range l {
		if !langRE.MatchString(lang) {
			f.add(ptr+"/"+lang, "code de langue invalide")
		}
		if strings.TrimSpace(v) == "" && lang != DefaultLanguage {
			f.add(ptr+"/"+lang, "texte vide : retirez la langue plutôt que de la laisser vide")
		}
		if utf8.RuneCountInString(v) > max {
			f.add(ptr+"/"+lang, "trop long")
		}
	}
}

// Normalize retire les espaces autour des textes.
func (l Localized) Normalize() Localized {
	if l == nil {
		return nil
	}
	out := make(Localized, len(l))
	for k, v := range l {
		out[k] = strings.TrimSpace(v)
	}
	return out
}

// ── catégories ──────────────────────────────────────────────────────────────

type Category struct {
	ID, Slug     string
	ParentID     *string
	SortOrder    int
	Name         Localized
	Description  Localized // nil = aucune
	ImageURL     *string   // facultative : https://… ou /chemin du site
	Depth        int
	ProductCount int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (c *Category) Validate() error {
	var f fieldErrs
	c.Slug = strings.TrimSpace(c.Slug)
	if !slugRE.MatchString(c.Slug) || len(c.Slug) > 80 {
		f.add("/slug", "minuscules, chiffres et tirets uniquement (80 caractères maximum)")
	}
	if c.SortOrder < 0 {
		f.add("/sortOrder", "doit être supérieur ou égal à 0")
	}
	c.Name, c.Description = c.Name.Normalize(), c.Description.Normalize()
	c.Name.check("/name", 200, true, &f)
	c.Description.check("/description", 5000, false, &f)
	if c.ImageURL != nil {
		v := strings.TrimSpace(*c.ImageURL)
		switch {
		case v == "":
			c.ImageURL = nil
		case len(v) > 2000 || !(strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://") || (strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//"))):
			f.add("/imageUrl", "adresse https://… ou chemin du site commençant par /")
		default:
			c.ImageURL = &v
		}
	}
	return f.err("catégorie invalide")
}

// ── attributs ───────────────────────────────────────────────────────────────

const (
	AttrText    = "text"
	AttrNumber  = "number"
	AttrBoolean = "boolean"
	AttrEnum    = "enum"
)

type AttributeOption struct {
	Value string
	Label Localized
}

type AttributeDef struct {
	ID, CategoryID, Key string
	Label               Localized
	Type                string
	Unit                *string
	Filterable          bool
	Required            bool
	Options             []AttributeOption
	SortOrder           int
}

func (a *AttributeDef) Validate() error {
	var f fieldErrs
	if !keyRE.MatchString(a.Key) {
		f.add("/key", "lettres minuscules, chiffres et _ ; commence par une lettre (2 à 64 caractères)")
	}
	a.Label = a.Label.Normalize()
	a.Label.check("/label", 200, true, &f)
	switch a.Type {
	case AttrText, AttrNumber, AttrBoolean, AttrEnum:
	default:
		f.add("/type", "text, number, boolean ou enum")
	}
	if a.Unit != nil {
		u := strings.TrimSpace(*a.Unit)
		if u == "" || utf8.RuneCountInString(u) > 20 {
			f.add("/unit", "unité vide ou trop longue : laissez le champ absent")
		}
		if a.Type != AttrNumber {
			f.add("/unit", "une unité n'a de sens que pour un attribut numérique")
		}
		a.Unit = &u
	}
	if a.SortOrder < 0 {
		f.add("/sortOrder", "doit être supérieur ou égal à 0")
	}
	if a.Type == AttrEnum {
		if len(a.Options) == 0 {
			f.add("/options", "au moins une option est obligatoire pour un attribut de type enum")
		}
		seen := map[string]bool{}
		for i, o := range a.Options {
			ptr := "/options/" + itoa(i)
			o.Label = o.Label.Normalize()
			a.Options[i] = o
			o.Label.check(ptr+"/label", 200, true, &f)
			v := strings.TrimSpace(o.Value)
			a.Options[i].Value = v
			if v == "" || len(v) > 100 {
				f.add(ptr+"/value", "valeur obligatoire (100 caractères maximum)")
			}
			if seen[v] {
				f.add(ptr+"/value", "valeur en double")
			}
			seen[v] = true
		}
	} else if len(a.Options) > 0 {
		f.add("/options", "des options ne sont permises que pour le type enum")
	}
	if a.Filterable && a.Type == AttrText {
		f.add("/filterable", "un attribut texte libre ne peut pas servir de filtre (utilisez un type enum)")
	}
	return f.err("attribut invalide")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// ── marques et secteurs ─────────────────────────────────────────────────────

type Brand struct {
	ID, Slug, Name       string
	LogoURL              *string
	CreatedAt, UpdatedAt time.Time
}

func checkURL(ptr string, u **string, f *fieldErrs) {
	if *u == nil {
		return
	}
	v := strings.TrimSpace(**u)
	if v == "" {
		*u = nil
		return
	}
	if !(strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://")) || len(v) > 2000 || strings.ContainsAny(v, " \t\n\"<>") {
		f.add(ptr, "adresse http(s) valide attendue")
	}
	*u = &v
}

func (b *Brand) Validate() error {
	var f fieldErrs
	b.Slug, b.Name = strings.TrimSpace(b.Slug), strings.TrimSpace(b.Name)
	if !slugRE.MatchString(b.Slug) || len(b.Slug) > 80 {
		f.add("/slug", "minuscules, chiffres et tirets uniquement (80 caractères maximum)")
	}
	if b.Name == "" || utf8.RuneCountInString(b.Name) > 120 {
		f.add("/name", "le nom est obligatoire (120 caractères maximum)")
	}
	checkURL("/logoUrl", &b.LogoURL, &f)
	return f.err("marque invalide")
}

type Sector struct {
	ID, Slug             string
	Name                 Localized
	ImageURL             *string
	CreatedAt, UpdatedAt time.Time
}

func (s *Sector) Validate() error {
	var f fieldErrs
	s.Slug = strings.TrimSpace(s.Slug)
	if !slugRE.MatchString(s.Slug) || len(s.Slug) > 80 {
		f.add("/slug", "minuscules, chiffres et tirets uniquement (80 caractères maximum)")
	}
	s.Name = s.Name.Normalize()
	s.Name.check("/name", 200, true, &f)
	checkURL("/imageUrl", &s.ImageURL, &f)
	return f.err("secteur invalide")
}

// SortedKeys sert aux snapshots d'audit déterministes.
func SortedKeys(l Localized) []string {
	k := make([]string, 0, len(l))
	for x := range l {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}
