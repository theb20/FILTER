package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/theb20/FILTER/backend/pkg/apperr"
	pb "github.com/theb20/FILTER/backend/pkg/gen/sourcing/v1"
	"github.com/theb20/FILTER/backend/pkg/jwtx"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
)

// Plafonds JOURNALIERS de la recherche web (décision du 17 nov. 2026). Ils s'appliquent ENSEMBLE : le premier atteint
// refuse l'appel. Ils s'ajoutent au budget de la source, qui reste la dernière barrière contre une facture imprévue.
// Une recherche ordinaire dans le catalogue ne passe JAMAIS par ici : seul un geste explicite du visiteur déclenche
// un appel payant.
const (
	webPerVisitorDay  = 3
	webPerCustomerDay = 10
	webPerNetworkDay  = 30
	webDay            = 24 * time.Hour
	webHitsMax        = 10
	webImagesMax      = 12
)

// networkPepper : secret du serveur, jamais stocké avec les données. Il protège les empreintes d'adresses réseau :
// sans lui, l'espace des adresses IPv4 est assez petit pour retrouver une adresse par essais. En développement, il est
// tiré au hasard au démarrage (les compteurs repartent donc à zéro à chaque redémarrage).
var networkPepper = func() []byte {
	if v := strings.TrimSpace(os.Getenv("SEARCH_NETWORK_PEPPER")); v != "" {
		return []byte(v)
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}()

// dayKey : clé dérivée du secret et de la DATE (UTC). Elle change chaque jour, donc l'empreinte d'hier ne peut pas
// être rapprochée de celle d'aujourd'hui.
func dayKey(now time.Time) []byte {
	m := hmac.New(sha256.New, networkPepper)
	m.Write([]byte(now.UTC().Format("2006-01-02")))
	return m.Sum(nil)
}

// subjectKey : empreinte tronquée (16 octets) du sujet compté. Ni l'adresse réseau, ni l'identifiant de session,
// ni l'identifiant du client n'apparaissent jamais en clair, même en mémoire.
func subjectKey(kind, value string, now time.Time) string {
	m := hmac.New(sha256.New, dayKey(now))
	m.Write([]byte(kind + "\x00" + value))
	return "web:" + kind + ":" + hex.EncodeToString(m.Sum(nil)[:16])
}

// networkOf : l'adresse est ramenée à son préfixe /64 en IPv6 (un client dispose sinon de milliards d'adresses, et la
// limite serait contournable). Elle n'est lue que pour calculer l'empreinte.
func networkOf(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return "inconnue"
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String()
}

// customerID renvoie l'identifiant du client si la requête porte un jeton valide. La route est publique : un jeton
// absent ou invalide n'est pas une erreur, le visiteur est simplement traité comme anonyme.
func (h *handler) customerID(ctx context.Context) string {
	tok := infoFrom(ctx).Bearer
	if tok == "" || h.srv.keys == nil {
		return ""
	}
	c, err := h.srv.keys.Verify(ctx, jwtx.AudCustomer, tok)
	if err != nil || c == nil {
		return ""
	}
	return c.Subject
}

// webLimitReached : 429 explicite, avec le délai avant la prochaine recherche possible.
func webLimitReached(retry time.Duration) error {
	e := apperr.RateLimited(retry)
	e.Message = "limite de recherches web atteinte pour aujourd'hui"
	return e
}

// allowWebCall applique les plafonds du jour AVANT tout appel payant, et renvoie ce qu'il reste au demandeur.
// Les deux routes payantes (pages et photos) partagent ces compteurs : ouvrir une fiche compte comme une recherche.
func (h *handler) allowWebCall(ctx context.Context, visitorID string) (int, error) {
	now := h.srv.now()
	// 1. Plafond du demandeur : 10 par jour pour un client connecté, 3 pour un visiteur anonyme.
	limit, subject := webPerVisitorDay, subjectKey("visitor", strings.TrimSpace(visitorID), now)
	if id := h.customerID(ctx); id != "" {
		limit, subject = webPerCustomerDay, subjectKey("customer", id, now)
	}
	ok, remaining, retry := h.srv.limiter.Allow(subject, limit, webDay, now)
	if !ok {
		return 0, webLimitReached(retry)
	}
	// 2. Plafond de l'adresse réseau : il couvre le cas d'un visiteur qui changerait d'identifiant de session.
	if ok, _, retry := h.srv.limiter.Allow(subjectKey("network", networkOf(infoFrom(ctx).IP), now), webPerNetworkDay, webDay, now); !ok {
		return 0, webLimitReached(retry)
	}
	return remaining, nil
}

// SearchWeb interroge une source externe À LA DEMANDE du visiteur, en respectant les plafonds du jour et le budget
// de la source. Les échecs de la source ne sont pas des erreurs HTTP : la réponse porte un `status` que le site
// explique au visiteur, car une recherche web est un complément, jamais le cœur du service.
func (h *handler) SearchWeb(ctx context.Context, p api.SearchWebParams) (api.SearchWebRes, error) {
	q := strings.TrimSpace(p.Q.Or(""))
	if len([]rune(q)) < 2 {
		return nil, apperr.Validation("recherche trop courte", apperr.Field{Pointer: "/q", Code: apperr.CodeValidation, Message: "saisissez au moins deux caractères"})
	}
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	remaining, err := h.allowWebCall(ctx, p.XVisitorID.Or(""))
	if err != nil {
		return nil, err
	}

	ctx, cancel := h.rpc(ctx)
	defer cancel()
	src, err := h.firstWebSource(ctx, c, "/web/search")
	if err != nil {
		return nil, err
	}

	res, err := c.CallSource(ctx, &pb.CallSourceRequest{
		SourceId: src.GetId(), SearchId: uuid.Must(uuid.NewV7()).String(), Query: q,
		Language: strings.ToLower(infoFrom(ctx).Locale),
	})
	if err != nil {
		return nil, err
	}
	out := &api.WebSearchResult{Query: q, Source: src.GetName(), Hits: []api.WebSearchHit{}, Remaining: int32(remaining)}
	switch res.GetStatus() {
	case pb.CallStatus_CALL_STATUS_OK:
		out.Status = api.WebSearchResultStatusOk
		for _, hit := range res.GetHits() {
			if len(out.Hits) >= webHitsMax {
				break
			}
			out.Hits = append(out.Hits, webHit(hit))
		}
	case pb.CallStatus_CALL_STATUS_BUDGET_EXHAUSTED:
		out.Status = api.WebSearchResultStatusBudget
	default:
		out.Status = api.WebSearchResultStatusUnavailable
	}
	return out, nil
}

// firstWebSource : la source active la plus prioritaire dont l'adresse contient `path` (« /web/search » pour les
// pages, « /images/search » pour les photos). Deux sources distinctes : chacune a son budget et ses compteurs.
func (h *handler) firstWebSource(ctx context.Context, c pb.SourcingServiceClient, path string) (*pb.Source, error) {
	list, err := c.ListSources(ctx, &pb.ListSourcesRequest{})
	if err != nil {
		return nil, err
	}
	usable := make([]*pb.Source, 0, 2)
	for _, s := range list.GetItems() {
		if s.GetKind() == pb.SourceKind_SOURCE_KIND_WEB_SEARCH_API && s.GetEnabled() && !s.GetSuspended() && strings.Contains(s.GetBaseUrl(), path) {
			usable = append(usable, s)
		}
	}
	if len(usable) == 0 {
		return nil, apperr.NotFound("aucune source active pour cette recherche")
	}
	sort.SliceStable(usable, func(i, j int) bool { return usable[i].GetPriority() < usable[j].GetPriority() })
	return usable[0], nil
}

// webHit : vue publique d'un résultat. Le détail brut de la source n'est jamais recopié tel quel ; seuls le titre,
// l'adresse, un extrait et, s'ils existent, la marque et la référence sont exposés.
func webHit(hit *pb.RawHit) api.WebSearchHit {
	out := api.WebSearchHit{Title: hit.GetTitle()}
	if u := hit.GetUrl(); u != "" {
		out.URL = api.NewOptString(u)
	}
	if b := hit.GetBrand(); b != "" {
		out.Brand = api.NewOptString(b)
	}
	if r := hit.GetReference(); r != "" {
		out.Reference = api.NewOptString(r)
	}
	var payload struct {
		Description   string   `json:"description"`
		ExtraSnippets []string `json:"extra_snippets"`
		Thumbnail     struct {
			Src string `json:"src"`
		} `json:"thumbnail"`
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
		MetaURL struct {
			Hostname string `json:"hostname"`
		} `json:"meta_url"`
	}
	if json.Unmarshal([]byte(hit.GetPayloadJson()), &payload) == nil {
		if d := plainText(payload.Description); d != "" {
			out.Description = api.NewOptString(trimRunes(d, 300))
		}
		// La vignette n'est affichée que si la source la sert en https : jamais de « data: », de « javascript: » ni de http.
		if img := strings.TrimSpace(payload.Thumbnail.Src); strings.HasPrefix(img, "https://") && len(img) <= 2000 {
			out.Image = api.NewOptString(img)
		}
		if site := plainText(firstNonEmpty(payload.Profile.Name, payload.MetaURL.Hostname)); site != "" {
			out.Site = api.NewOptString(trimRunes(site, 80))
		}
		for _, sn := range payload.ExtraSnippets {
			if len(out.Snippets) >= 6 {
				break
			}
			if t := plainText(sn); t != "" {
				out.Snippets = append(out.Snippets, trimRunes(t, 300))
			}
		}
	}
	out.Title = plainText(out.Title)
	return out
}

// plainText : les moteurs renvoient des extraits balisés (<strong>…</strong>, entités HTML). Le site les affiche comme
// du TEXTE : on retire les balises et on décode les entités ici, pour ne jamais injecter de HTML venu d'un tiers.
func plainText(s string) string {
	// 1. Décodage des entités, en boucle : certains moteurs encodent deux fois (« &amp;amp; » pour « & »).
	for i := 0; i < 3; i++ {
		d := html.UnescapeString(s)
		if d == s {
			break
		}
		s = d
	}
	// 2. Retrait des balises, APRÈS décodage : une balise encodée deux fois ne peut pas réapparaître ensuite.
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	// 3. Espaces : une seule espace entre les mots, aucun saut de ligne.
	return strings.Join(strings.Fields(b.String()), " ")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func trimRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}

// SearchWebImages cherche les photos d'un produit pour sa fiche sur le site. L'appel est payant : il passe par les
// mêmes plafonds que la recherche web, et par une source DISTINCTE (le point d'entrée « images » de la source).
func (h *handler) SearchWebImages(ctx context.Context, p api.SearchWebImagesParams) (api.SearchWebImagesRes, error) {
	q := strings.TrimSpace(p.Q.Or(""))
	if len([]rune(q)) < 2 {
		return nil, apperr.Validation("recherche trop courte", apperr.Field{Pointer: "/q", Code: apperr.CodeValidation, Message: "saisissez au moins deux caractères"})
	}
	c, err := h.sourcing()
	if err != nil {
		return nil, err
	}
	remaining, err := h.allowWebCall(ctx, p.XVisitorID.Or(""))
	if err != nil {
		return nil, err
	}
	ctx, cancel := h.rpc(ctx)
	defer cancel()
	src, err := h.firstWebSource(ctx, c, "/images/search")
	if err != nil {
		return nil, err
	}
	res, err := c.CallSource(ctx, &pb.CallSourceRequest{
		SourceId: src.GetId(), SearchId: uuid.Must(uuid.NewV7()).String(), Query: q,
		Language: strings.ToLower(infoFrom(ctx).Locale),
	})
	if err != nil {
		return nil, err
	}
	out := &api.WebImageResult{Query: q, Source: src.GetName(), Images: []api.WebImage{}, Remaining: int32(remaining)}
	switch res.GetStatus() {
	case pb.CallStatus_CALL_STATUS_OK:
		out.Status = api.WebImageResultStatusOk
		for _, hit := range res.GetHits() {
			if len(out.Images) >= webImagesMax {
				break
			}
			if img, ok := webImage(hit); ok {
				out.Images = append(out.Images, img)
			}
		}
	case pb.CallStatus_CALL_STATUS_BUDGET_EXHAUSTED:
		out.Status = api.WebImageResultStatusBudget
	default:
		out.Status = api.WebImageResultStatusUnavailable
	}
	return out, nil
}

// webImage : vue publique d'une photo. Seule la vignette servie en https par la source est retenue ; l'image d'origine
// du site tiers n'est jamais chargée par le navigateur du visiteur.
func webImage(hit *pb.RawHit) (api.WebImage, bool) {
	var payload struct {
		Source    string `json:"source"`
		Thumbnail struct {
			Src string `json:"src"`
		} `json:"thumbnail"`
	}
	if json.Unmarshal([]byte(hit.GetPayloadJson()), &payload) != nil {
		return api.WebImage{}, false
	}
	src := strings.TrimSpace(payload.Thumbnail.Src)
	if !strings.HasPrefix(src, "https://") || len(src) > 2000 {
		return api.WebImage{}, false
	}
	out := api.WebImage{URL: src}
	if t := plainText(hit.GetTitle()); t != "" {
		out.Title = api.NewOptString(trimRunes(t, 200))
	}
	if u := hit.GetUrl(); strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		out.PageUrl = api.NewOptString(u)
	}
	if s := plainText(payload.Source); s != "" {
		out.Site = api.NewOptString(trimRunes(s, 80))
	}
	return out, true
}
