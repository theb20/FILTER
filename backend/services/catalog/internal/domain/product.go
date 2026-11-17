package domain

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/theb20/FILTER/backend/pkg/apperr"
)

const (
	StatusDraft    = "draft"
	StatusReview   = "review"
	StatusActive   = "active"
	StatusArchived = "archived"

	FulfillStock       = "stock"
	FulfillMadeToOrder = "made_to_order"

	PriceFixed   = "fixed"
	PriceOnQuote = "on_quote"

	OriginManual = "manual"
	OriginImport = "import"

	// BaseCurrency : devise de base du catalogue (décision du 17 nov. 2026).
	BaseCurrency = "XOF"
)

const (
	maxMedia     = 50
	maxCrossRefs = 200
)

type AttrValue struct {
	Key    string
	Text   *string
	Number *float64
	Bool   *bool
}

type Media struct {
	ID        string
	URL       string
	Alt       *string
	SortOrder int
}

type CrossRef struct{ Brand, Number, Kind string }

type Product struct {
	ID, SKU, Slug                                              string
	Name, Description                                          Localized
	BrandID                                                    *string
	CategoryID                                                 string
	ManufacturerRef                                            *string
	Status, FulfillmentMode                                    string
	LeadTimeDays                                               *int
	PriceMode                                                  string
	PriceAmount                                                *int64
	Currency                                                   string
	GTIN, SaleUnit                                             *string
	MinOrderQuantity, WeightGrams, LengthMM, WidthMM, HeightMM *int
	Attributes                                                 []AttrValue
	SectorIDs                                                  []string
	Media                                                      []Media
	CrossRefs                                                  []CrossRef
	Origins                                                    map[string]string
	Version                                                    int
	PublishedAt                                                *time.Time
	CreatedAt, UpdatedAt                                       time.Time
}

type ProductVersion struct {
	Version int
	At      time.Time
	ActorID *string
	Action  string
	Changes map[string]any
}

var accents = strings.NewReplacer("à", "a", "â", "a", "ä", "a", "á", "a", "ã", "a", "å", "a", "é", "e", "è", "e", "ê", "e", "ë", "e", "î", "i", "ï", "i", "í", "i", "ô", "o", "ö", "o", "ó", "o", "õ", "o", "ù", "u", "û", "u", "ü", "u", "ú", "u", "ç", "c", "ñ", "n", "œ", "oe", "æ", "ae", "ÿ", "y")

// Slugify : minuscules, accents retirés, tout caractère hors a-z0-9 devient un tiret ; 80 caractères au plus.
func Slugify(s string) string {
	s = accents.Replace(strings.ToLower(strings.TrimSpace(s)))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		out = strings.Trim(out[:80], "-")
	}
	return out
}

// NormalizeRef : clé de comparaison d'une référence (majuscules, sans espaces ni ponctuation). « 6204-2RS / C3 » = « 6204 2rs c3 ».
func NormalizeRef(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// GTINValid : longueur 8, 12, 13 ou 14 et chiffre de contrôle GS1 correct.
func GTINValid(g string) bool {
	switch len(g) {
	case 8, 12, 13, 14:
	default:
		return false
	}
	sum := 0
	for i := 0; i < len(g)-1; i++ {
		if g[i] < '0' || g[i] > '9' {
			return false
		}
		d := int(g[i] - '0')
		if (len(g)-1-i)%2 == 1 {
			d *= 3
		}
		sum += d
	}
	last := g[len(g)-1]
	return last >= '0' && last <= '9' && (10-sum%10)%10 == int(last-'0')
}

func optText(p **string, ptr string, max int, f *fieldErrs) {
	if *p == nil {
		return
	}
	v := strings.TrimSpace(**p)
	if v == "" {
		*p = nil
		return
	}
	if utf8.RuneCountInString(v) > max {
		f.add(ptr, "trop long")
	}
	*p = &v
}

func optNonNeg(v *int, ptr string, min int, f *fieldErrs) {
	if v != nil && *v < min {
		f.add(ptr, "doit être supérieur ou égal à "+itoa(min))
	}
}

// ValidateShape vérifie la forme de la fiche (indépendante de la catégorie et du cycle de vie).
// Un SKU et un slug vides sont acceptés : le service les génère.
func (p *Product) ValidateShape() error {
	var f fieldErrs
	p.SKU = strings.ToUpper(strings.TrimSpace(p.SKU))
	if p.SKU != "" && !(len(p.SKU) <= 64 && skuRE.MatchString(p.SKU)) {
		f.add("/sku", "majuscules, chiffres et tirets uniquement (64 caractères maximum)")
	}
	p.Slug = strings.TrimSpace(p.Slug)
	if p.Slug != "" && !(len(p.Slug) <= 80 && slugRE.MatchString(p.Slug)) {
		f.add("/slug", "minuscules, chiffres et tirets uniquement (80 caractères maximum)")
	}
	p.Name, p.Description = p.Name.Normalize(), p.Description.Normalize()
	p.Name.check("/name", 300, true, &f)
	p.Description.check("/description", 20000, false, &f)
	if p.CategoryID == "" {
		f.add("/categoryId", "la catégorie est obligatoire")
	}
	optText(&p.ManufacturerRef, "/manufacturerRef", 100, &f)
	switch p.FulfillmentMode {
	case FulfillStock:
	case FulfillMadeToOrder:
		if p.LeadTimeDays == nil {
			f.add("/leadTimeDays", "le délai est obligatoire pour un produit fabriqué à la commande")
		}
	default:
		f.add("/fulfillmentMode", "stock ou made_to_order")
	}
	optNonNeg(p.LeadTimeDays, "/leadTimeDays", 0, &f)
	switch p.PriceMode {
	case PriceFixed:
		if p.PriceAmount != nil && *p.PriceAmount < 0 {
			f.add("/priceAmount", "le prix ne peut pas être négatif")
		}
	case PriceOnQuote:
		if p.PriceAmount != nil {
			f.add("/priceAmount", "un produit sur devis n'a pas de prix")
		}
	default:
		f.add("/priceMode", "fixed ou on_quote")
	}
	optText(&p.GTIN, "/gtin", 14, &f)
	if p.GTIN != nil && !GTINValid(*p.GTIN) {
		f.add("/gtin", "code-barres GTIN/EAN invalide (longueur ou chiffre de contrôle)")
	}
	optText(&p.SaleUnit, "/saleUnit", 50, &f)
	optNonNeg(p.MinOrderQuantity, "/minOrderQuantity", 1, &f)
	optNonNeg(p.WeightGrams, "/weightGrams", 0, &f)
	optNonNeg(p.LengthMM, "/lengthMm", 0, &f)
	optNonNeg(p.WidthMM, "/widthMm", 0, &f)
	optNonNeg(p.HeightMM, "/heightMm", 0, &f)

	seen := map[string]bool{}
	for i, a := range p.Attributes {
		ptr := "/attributes/" + itoa(i)
		n := 0
		if a.Text != nil {
			n++
		}
		if a.Number != nil {
			n++
			if math.IsNaN(*a.Number) || math.IsInf(*a.Number, 0) {
				f.add(ptr+"/value", "nombre invalide")
			}
		}
		if a.Bool != nil {
			n++
		}
		if !keyRE.MatchString(a.Key) {
			f.add(ptr+"/key", "clé d'attribut invalide")
		}
		if n != 1 {
			f.add(ptr+"/value", "une seule valeur est attendue")
		}
		if seen[a.Key] {
			f.add(ptr+"/key", "attribut en double")
		}
		seen[a.Key] = true
	}
	ids := map[string]bool{}
	for i, id := range p.SectorIDs {
		if ids[id] {
			f.add("/sectorIds/"+itoa(i), "secteur en double")
		}
		ids[id] = true
	}
	if len(p.Media) > maxMedia {
		f.add("/media", "trop de médias")
	}
	for i := range p.Media {
		m := &p.Media[i]
		ptr := "/media/" + itoa(i)
		m.URL = strings.TrimSpace(m.URL)
		if !(strings.HasPrefix(m.URL, "https://") || strings.HasPrefix(m.URL, "http://")) || len(m.URL) > 2000 || strings.ContainsAny(m.URL, " \t\n\"<>") {
			f.add(ptr+"/url", "adresse http(s) valide attendue")
		}
		optText(&m.Alt, ptr+"/alt", 300, &f)
		m.SortOrder = i
	}
	if len(p.CrossRefs) > maxCrossRefs {
		f.add("/crossReferences", "trop de références croisées")
	}
	xs := map[string]bool{}
	for i := range p.CrossRefs {
		x := &p.CrossRefs[i]
		ptr := "/crossReferences/" + itoa(i)
		x.Brand, x.Number = strings.TrimSpace(x.Brand), strings.TrimSpace(x.Number)
		if x.Brand == "" || utf8.RuneCountInString(x.Brand) > 100 {
			f.add(ptr+"/brand", "marque obligatoire (100 caractères maximum)")
		}
		if NormalizeRef(x.Number) == "" || utf8.RuneCountInString(x.Number) > 100 {
			f.add(ptr+"/number", "numéro obligatoire (100 caractères maximum)")
		}
		switch x.Kind {
		case "oem", "equivalent", "supersedes":
		default:
			f.add(ptr+"/kind", "oem, equivalent ou supersedes")
		}
		k := strings.ToLower(x.Brand) + "|" + NormalizeRef(x.Number) + "|" + x.Kind
		if xs[k] {
			f.add(ptr, "référence croisée en double")
		}
		xs[k] = true
	}
	return f.err("fiche produit invalide")
}

// CheckAttributes vérifie les valeurs contre les définitions de la catégorie (héritées comprises).
func (p *Product) CheckAttributes(defs []*AttributeDef) error {
	var f fieldErrs
	by := map[string]*AttributeDef{}
	for _, d := range defs {
		by[d.Key] = d
	}
	for i, a := range p.Attributes {
		ptr := "/attributes/" + itoa(i) + "/value"
		d, ok := by[a.Key]
		if !ok {
			f.add("/attributes/"+itoa(i)+"/key", "cet attribut n'existe pas pour la catégorie du produit")
			continue
		}
		switch d.Type {
		case AttrText:
			if a.Text == nil || strings.TrimSpace(*a.Text) == "" || utf8.RuneCountInString(*a.Text) > 2000 {
				f.add(ptr, "texte attendu (2000 caractères maximum)")
			}
		case AttrNumber:
			if a.Number == nil {
				f.add(ptr, "nombre attendu")
			}
		case AttrBoolean:
			if a.Bool == nil {
				f.add(ptr, "oui ou non attendu")
			}
		case AttrEnum:
			ok := false
			if a.Text != nil {
				for _, o := range d.Options {
					if o.Value == *a.Text {
						ok = true
					}
				}
			}
			if !ok {
				f.add(ptr, "valeur non prévue dans la liste de choix")
			}
		}
	}
	return f.err("attributs invalides")
}

// PublishProblems liste ce qui manque pour publier ; vide = publiable.
func (p *Product) PublishProblems(defs []*AttributeDef) []apperr.Field {
	var f fieldErrs
	if strings.TrimSpace(p.Name[DefaultLanguage]) == "" {
		f.add("/name", "le nom en français est obligatoire")
	}
	if p.CategoryID == "" {
		f.add("/categoryId", "la catégorie est obligatoire")
	}
	if p.PriceMode == PriceFixed && p.PriceAmount == nil {
		f.add("/priceAmount", "le prix hors taxes est obligatoire (ou passez le produit « sur devis »)")
	}
	have := map[string]bool{}
	for _, a := range p.Attributes {
		have[a.Key] = true
	}
	for _, d := range defs {
		if d.Required && !have[d.Key] {
			f.add("/attributes", "attribut obligatoire manquant : « "+d.Key+" » ("+d.Label[DefaultLanguage]+")")
		}
	}
	return f
}
