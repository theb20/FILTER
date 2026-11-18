package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/theb20/FILTER/backend/pkg/apperr"
)

var (
	phoneRe    = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	localeRe   = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)
	countryRe  = regexp.MustCompile(`^[A-Z]{2}$`)
	policyRe   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,32}$`)
	anonIDRe   = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
)

func bad(ptr, msg string) error {
	return apperr.Validation("données invalides", apperr.Field{Pointer: ptr, Code: apperr.CodeValidation, Message: msg})
}

// ProfilePatch : modification PARTIELLE du profil (nil = inchangé ; chaîne vide = effacer, sauf locale et devise).
type ProfilePatch struct {
	FirstName, LastName, Phone, Locale, Currency *string
}

// Apply valide puis applique le patch. Rien n'est modifié si une valeur est refusée.
func (p ProfilePatch) Apply(u *User) error {
	name := func(v *string, ptr string) (string, bool, error) {
		if v == nil {
			return "", false, nil
		}
		t := strings.TrimSpace(*v)
		if utf8.RuneCountInString(t) > 100 || strings.ContainsAny(t, "\x00\r\n\t") {
			return "", false, bad(ptr, "100 caractères au maximum, sans caractère de contrôle")
		}
		return t, true, nil
	}
	fn, fnSet, err := name(p.FirstName, "/firstName")
	if err != nil {
		return err
	}
	ln, lnSet, err := name(p.LastName, "/lastName")
	if err != nil {
		return err
	}
	if p.Phone != nil && *p.Phone != "" && !phoneRe.MatchString(*p.Phone) {
		return bad("/phone", "format E.164 attendu (ex. +2250747225087)")
	}
	if p.Locale != nil && (len(*p.Locale) > 16 || !localeRe.MatchString(*p.Locale)) {
		return bad("/locale", "code de langue invalide")
	}
	if p.Currency != nil && !currencyRe.MatchString(*p.Currency) {
		return bad("/preferredCurrency", "code ISO 4217 attendu (ex. XOF)")
	}
	if fnSet {
		u.FirstName = fn
	}
	if lnSet {
		u.LastName = ln
	}
	if p.Phone != nil {
		u.Phone = *p.Phone
	}
	if p.Locale != nil {
		u.Locale = *p.Locale
	}
	if p.Currency != nil {
		u.PreferredCurrency = *p.Currency
	}
	return nil
}

// CompanyData : données saisies par le titulaire.
type CompanyData struct{ LegalName, TaxID, Country, JobTitle string }

// Normalize nettoie et valide la saisie.
func (d CompanyData) Normalize() (CompanyData, error) {
	d.LegalName, d.TaxID, d.JobTitle = strings.TrimSpace(d.LegalName), strings.TrimSpace(d.TaxID), strings.TrimSpace(d.JobTitle)
	switch {
	case d.LegalName == "" || utf8.RuneCountInString(d.LegalName) > 200:
		return d, bad("/legalName", "obligatoire, 200 caractères au maximum")
	case d.TaxID == "" || utf8.RuneCountInString(d.TaxID) > 64:
		return d, bad("/taxId", "obligatoire, 64 caractères au maximum")
	case !countryRe.MatchString(d.Country):
		return d, bad("/country", "code pays ISO 3166-1 alpha-2 attendu (ex. CI)")
	case utf8.RuneCountInString(d.JobTitle) > 100:
		return d, bad("/jobTitle", "100 caractères au maximum")
	}
	return d, nil
}

// LegalChanged : une modification des données LÉGALES (pas de la fonction) exige une revalidation par le personnel.
func (c *Company) LegalChanged(d CompanyData) bool {
	return c.LegalName != d.LegalName || c.TaxID != d.TaxID || c.Country != d.Country
}

// Consentement (cookies / traitements). `necessary` ne se refuse pas : sans lui le site ne fonctionne pas.
type ConsentPurposes struct{ Necessary, Preferences, Functional, Analytics, Marketing bool }

// ConsentRecord est une preuve de consentement : JOURNAL APPEND-ONLY, jamais modifié.
type ConsentRecord struct {
	ID            string
	UserID        string // vide : visiteur
	AnonymousID   string
	Purposes      ConsentPurposes
	PolicyVersion string
	Locale        string
	IPNet         string // adresse tronquée (IPv4 /24, IPv6 /48) : preuve sans identifier précisément la connexion
	RecordedAt    time.Time
}

// Validate vérifie la forme d'un consentement entrant.
func (c *ConsentRecord) Validate() error {
	if !c.Purposes.Necessary {
		return bad("/purposes/necessary", "les cookies nécessaires ne peuvent pas être refusés")
	}
	if !policyRe.MatchString(c.PolicyVersion) {
		return bad("/policyVersion", "version de politique invalide (1 à 32 caractères : lettres, chiffres, . _ -)")
	}
	if c.AnonymousID != "" && !anonIDRe.MatchString(c.AnonymousID) {
		return bad("/anonymousId", "identifiant invalide (8 à 64 caractères : lettres, chiffres, _ -)")
	}
	if c.Locale != "" && (len(c.Locale) > 16 || !localeRe.MatchString(c.Locale)) {
		return bad("/locale", "code de langue invalide")
	}
	return nil
}
