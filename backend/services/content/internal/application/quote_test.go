package application

import "testing"

func line(ref string, qty, price, disc float64, extra map[string]any) any {
	m := map[string]any{"reference": ref, "quantity": qty}
	if price >= 0 {
		m["unitPrice"] = price
	}
	if disc > 0 {
		m["discountPct"] = disc
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestTotauxDuDevis(t *testing.T) {
	d := map[string]any{
		"lines":       []any{line("A", 2, 1000, 10, nil), line("B", 1, 500, 0, nil), line("C", 1, 9999, 0, map[string]any{"optional": true}), line("D", 3, -1, 0, nil)},
		"discountPct": 5.0, "shipping": 100.0, "vatRate": 18.0,
	}
	got := ComputeQuote(d)
	// A : 2×1000×0,9 = 1800 ; B : 500 ; C optionnelle et D sans prix : ignorées → 2300
	// remise 5 % = 115 ; (2300−115+100) = 2285 ; TVA 18 % = 411 (411,3) ; total = 2696
	want := QuoteTotals{Subtotal: 2300, DiscountAmount: 115, VATAmount: 411, Total: 2696, Remaining: 2696}
	if got != want {
		t.Fatalf("totaux = %+v, attendu %+v", got, want)
	}
}

func TestRegleDevisEnvoye(t *testing.T) {
	ok := func() map[string]any {
		return map[string]any{"status": "sent", "number": "DEV-1", "validUntil": "2030-01-01T00:00:00Z", "lines": []any{line("A", 1, 100, 0, nil)}}
	}
	if err := quoteRules(ok()); err != nil {
		t.Fatalf("devis complet refusé : %v", err)
	}
	d := ok()
	d["lines"] = []any{line("A", 1, -1, 0, nil)}
	if quoteRules(d) == nil {
		t.Fatal("ligne sans prix acceptée à l'envoi")
	}
	d = ok()
	delete(d, "validUntil")
	if quoteRules(d) == nil {
		t.Fatal("devis sans validité accepté à l'envoi")
	}
	d = ok()
	d["status"] = "draft"
	delete(d, "validUntil")
	d["lines"] = []any{line("A", 1, -1, 0, nil)}
	if err := quoteRules(d); err != nil {
		t.Fatalf("un brouillon incomplet doit être accepté : %v", err)
	}
	if d["total"].(float64) != 0 {
		t.Fatal("les totaux d'un brouillon doivent être recalculés")
	}
}

func TestAutresFraisEtAcompte(t *testing.T) {
	d := map[string]any{"lines": []any{line("A", 10, 1000, 0, nil)}, "other": 500.0, "shipping": 500.0, "vatRate": 18.0, "acompte": 3000.0}
	// 10 000 + 500 + 500 = 11 000 ; TVA 1 980 ; TTC 12 980 ; reste 9 980
	got := ComputeQuote(d)
	if got.Subtotal != 10000 || got.VATAmount != 1980 || got.Total != 12980 || got.Remaining != 9980 {
		t.Fatalf("totaux = %+v", got)
	}
}
