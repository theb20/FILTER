package application

import (
	"math"
)

// QuoteTotals : montants d'une réponse à un devis, en unités entières de la devise (F CFA : pas de décimale), hors taxes puis avec TVA.
// Une ligne « optionnelle » est proposée au client mais n'entre pas dans le total. Les arrondis se font ligne par ligne.
type QuoteTotals struct{ Subtotal, DiscountAmount, VATAmount, Total, Remaining int64 }

func quoteLineTotal(m map[string]any) (int64, bool) {
	price, priced := m["unitPrice"].(float64)
	if !priced {
		return 0, false
	}
	qty, _ := m["quantity"].(float64)
	disc, _ := m["discountPct"].(float64)
	return int64(math.Round(qty * price * (1 - disc/100))), true
}

// ComputeQuote calcule les totaux : sous-total des lignes, remise globale, livraison et « autres frais », TVA sur (sous-total − remise + livraison + autres), puis reste à payer après acompte.
func ComputeQuote(d map[string]any) QuoteTotals {
	var sub int64
	lines, _ := d["lines"].([]any)
	for _, x := range lines {
		m, _ := x.(map[string]any)
		if opt, _ := m["optional"].(bool); opt {
			continue
		}
		if t, ok := quoteLineTotal(m); ok {
			sub += t
		}
	}
	pct, _ := d["discountPct"].(float64)
	disc := int64(math.Round(float64(sub) * pct / 100))
	ship, _ := d["shipping"].(float64)
	other, _ := d["other"].(float64)
	taxable := sub - disc + int64(ship) + int64(other)
	vat, _ := d["vatRate"].(float64)
	vatAmount := int64(math.Round(float64(taxable) * vat / 100))
	total := taxable + vatAmount
	acompte, _ := d["acompte"].(float64)
	return QuoteTotals{Subtotal: sub, DiscountAmount: disc, VATAmount: vatAmount, Total: total, Remaining: total - int64(acompte)}
}

// quoteRules : les totaux sont toujours recalculés ici (jamais crus sur parole) ; un devis « envoyé » doit être complet.
func quoteRules(d map[string]any) error {
	t := ComputeQuote(d)
	d["subtotal"], d["discountAmount"], d["vatAmount"], d["total"] = float64(t.Subtotal), float64(t.DiscountAmount), float64(t.VATAmount), float64(t.Total)
	if d["status"] != "sent" {
		return nil
	}
	lines, _ := d["lines"].([]any)
	priced := 0
	for _, x := range lines {
		m, _ := x.(map[string]any)
		if opt, _ := m["optional"].(bool); opt {
			continue
		}
		if _, ok := m["unitPrice"].(float64); !ok {
			return invalid("lines", "chaque ligne doit avoir un prix avant l'envoi (ou être marquée « optionnelle »)")
		}
		priced++
	}
	if priced == 0 {
		return invalid("lines", "un devis envoyé doit contenir au moins une ligne chiffrée")
	}
	if _, ok := d["validUntil"]; !ok {
		return invalid("validUntil", "indiquez la date de validité du devis avant l'envoi")
	}
	if _, ok := d["number"]; !ok {
		return invalid("number", "le numéro du devis est obligatoire avant l'envoi")
	}
	return nil
}
