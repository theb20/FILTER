package server

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	pb "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	"github.com/theb20/FILTER/backend/services/gateway/internal/api"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mkdoc(t *testing.T, kind string, m map[string]any) *pb.Document {
	t.Helper()
	s, err := structpb.NewStruct(m)
	if err != nil {
		t.Fatal(err)
	}
	return &pb.Document{Id: uuid.NewString(), Kind: kind, Data: s, Version: 1, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now()}
}

// Un document complet doit traverser le contrat dans les deux sens sans perte (type, date, liste, texte par langue).
func TestDocumentsTraversentLeContrat(t *testing.T) {
	art := map[string]any{"slug": "choisir-un-filtre", "title": map[string]any{"fr": "Choisir", "en": "Choose"}, "body": map[string]any{"fr": "## Titre\n\nTexte"}, "status": "published", "kind": "advice",
		"categorySlug": "guides", "authorId": uuid.NewString(), "tags": []any{"huile", "air"}, "imageUrl": "https://exemple.org/a.png", "readMin": 7.0, "guide": "steps", "dossier": true,
		"productIds": []any{uuid.NewString()}, "publishedAt": "2026-11-17T08:30:00Z"}
	a, err := decodeDoc[api.CmsArticle](mkdoc(t, "article", art))
	if err != nil {
		t.Fatalf("article : %v", err)
	}
	if a.ReadMin.Or(0) != 7 || len(a.Tags) != 2 || a.Title["en"] != "Choose" || !a.PublishedAt.Set {
		t.Fatalf("article décodé : %+v", a)
	}
	// et dans l'autre sens : le corps typé redevient l'objet du service
	st, err := inputStruct(a)
	if err != nil {
		t.Fatal(err)
	}
	back := st.AsMap()
	if back["readMin"] != 7.0 || back["slug"] != "choisir-un-filtre" || back["status"] != "published" {
		t.Fatalf("aller-retour : %v", back)
	}
	for _, c := range []struct {
		kind string
		m    map[string]any
		into func(*pb.Document) error
	}{
		{"quote_request", map[string]any{"name": "A", "email": "a@b.co", "message": "m", "status": "new", "items": []any{map[string]any{"reference": "S1", "quantity": 3.0}}},
			func(d *pb.Document) error { _, e := decodeDoc[api.CmsQuoteRequest](d); return e }},
		{"promo_code", map[string]any{"code": "BONJOUR10", "type": "percent", "value": 10.0, "startsAt": "2026-01-01T00:00:00Z", "active": true},
			func(d *pb.Document) error { _, e := decodeDoc[api.CmsPromoCode](d); return e }},
		{"shipping_zone", map[string]any{"code": "abj", "name": map[string]any{"fr": "Abidjan"}, "fee": 5000.0, "daysMin": 1.0, "daysMax": 3.0, "freeEligible": true, "active": true},
			func(d *pb.Document) error { _, e := decodeDoc[api.CmsShippingZone](d); return e }},
		{"review", map[string]any{"productId": uuid.NewString(), "rating": 5.0, "title": "t", "text": "x", "author": "a", "status": "published", "verified": true},
			func(d *pb.Document) error { _, e := decodeDoc[api.CmsReview](d); return e }},
	} {
		if err := c.into(mkdoc(t, c.kind, c.m)); err != nil {
			t.Errorf("%s : %v", c.kind, err)
		}
	}
	// réglages : champs absents = vides
	var s api.CmsCommerceSettings
	if err := json.Unmarshal([]byte(`{"vatRate":18,"freeShippingThreshold":500000}`), &s); err != nil || s.VatRate.Or(0) != 18 {
		t.Fatalf("réglages : %v %+v", err, s)
	}
}
