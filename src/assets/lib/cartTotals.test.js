import { describe, expect, it } from "vitest";
import { addBusinessDays, computeTotals, deliveryWindow, promoDiscount, quoteEligibility, validatePromo } from "./cartTotals.js";

const PROMOS = [
  { code: "PCT10", type: "percent", value: 10, expiresAt: "2099-12-31" },
  { code: "FIX5000", type: "fixed", value: 5000, expiresAt: "2099-12-31", minSubtotal: 50000 },
  { code: "OLD", type: "percent", value: 15, expiresAt: "2025-08-31" },
];
const NOW = new Date("2026-11-16T10:00:00Z"); // lundi
const ABJ = { id: "abidjan", fee: 3000, days: [1, 2], freeEligible: true };
const INTL = { id: "international", fee: 35000, days: [7, 14], freeEligible: false };
const base = { promo: null, zone: ABJ, freeThreshold: 150000, vatRate: 0.18 };

describe("validatePromo", () => {
  it("accepte un code valide, sans tenir compte de la casse ni des espaces", () => {
    expect(validatePromo("  pct10 ", 100000, NOW, PROMOS)).toEqual({ status: "valid", promo: PROMOS[0] });
  });
  it("refuse un code inconnu", () => expect(validatePromo("NOPE", 100000, NOW, PROMOS).status).toBe("invalid"));
  it("refuse un code vide", () => expect(validatePromo("", 100000, NOW, PROMOS).status).toBe("invalid"));
  it("détecte un code expiré", () => expect(validatePromo("OLD", 100000, NOW, PROMOS).status).toBe("expired"));
  it("valable jusqu'à la fin du jour d'expiration", () => {
    const p = [{ ...PROMOS[0], expiresAt: "2026-11-16" }];
    expect(validatePromo("PCT10", 1000, new Date("2026-11-16T23:00:00Z"), p).status).toBe("valid");
    expect(validatePromo("PCT10", 1000, new Date("2026-11-17T00:30:00Z"), p).status).toBe("expired");
  });
  it("exige le minimum d'achat", () => {
    expect(validatePromo("FIX5000", 49999, NOW, PROMOS).status).toBe("minimum");
    expect(validatePromo("FIX5000", 50000, NOW, PROMOS).status).toBe("valid");
  });
});

describe("promoDiscount", () => {
  it("pourcentage arrondi", () => expect(promoDiscount(PROMOS[0], 12345)).toBe(1235));
  it("montant fixe plafonné au sous-total", () => {
    expect(promoDiscount(PROMOS[1], 80000)).toBe(5000);
    expect(promoDiscount({ ...PROMOS[1], minSubtotal: 0 }, 3000)).toBe(3000);
  });
  it("0 sans code", () => expect(promoDiscount(null, 80000)).toBe(0));
});

describe("jours ouvrés et livraison", () => {
  it("saute le week-end", () => {
    expect(addBusinessDays(new Date("2026-11-20T10:00:00Z"), 1).getDay()).toBe(1); // vendredi → lundi
    expect(addBusinessDays(NOW, 5).toISOString().slice(0, 10)).toBe("2026-11-23");
  });
  it("calcule la fenêtre d'une zone", () => {
    const w = deliveryWindow(NOW, ABJ);
    expect(w.from.toISOString().slice(0, 10)).toBe("2026-11-17");
    expect(w.to.toISOString().slice(0, 10)).toBe("2026-11-18");
  });
});

describe("computeTotals", () => {
  it("panier vide : tout à 0", () => {
    expect(computeTotals({ ...base, subtotal: 0 })).toMatchObject({ shipping: 0, ht: 0, vat: 0, ttc: 0, freeShipping: false });
  });
  it("sous le seuil : frais de zone + TVA 18 %", () => {
    const t = computeTotals({ ...base, subtotal: 100000 });
    expect(t).toMatchObject({ shipping: 3000, ht: 103000, vat: 18540, ttc: 121540, freeShipping: false, remainingForFree: 50000 });
    expect(t.progress).toBeCloseTo(2 / 3, 6);
  });
  it("livraison gratuite dès le seuil", () => {
    const t = computeTotals({ ...base, subtotal: 150000 });
    expect(t).toMatchObject({ shipping: 0, freeShipping: true, remainingForFree: 0, progress: 1, ht: 150000, vat: 27000, ttc: 177000 });
  });
  it("le seuil s'apprécie après remise", () => {
    const t = computeTotals({ ...base, subtotal: 160000, promo: PROMOS[0] }); // 144 000 après −10 %
    expect(t).toMatchObject({ discount: 16000, afterDiscount: 144000, freeShipping: false, shipping: 3000, remainingForFree: 6000 });
  });
  it("pas de livraison gratuite à l'international", () => {
    const t = computeTotals({ ...base, zone: INTL, subtotal: 500000 });
    expect(t).toMatchObject({ shipping: 35000, freeShipping: false, remainingForFree: 0, progress: 0 });
  });
  it("arrondit la TVA au franc", () => {
    expect(computeTotals({ ...base, subtotal: 10001, zone: { ...ABJ, fee: 0 } }).vat).toBe(1800); // 1800,18
  });
});

describe("quoteEligibility", () => {
  const TH = { minItems: 10, minAmount: 200000 };
  it("non éligible sous les deux seuils, avec ce qu'il manque", () => {
    expect(quoteEligibility(5, 137000, TH)).toEqual({ eligible: false, remainingItems: 5, remainingAmount: 63000 });
  });
  it("éligible dès le nombre d'articles", () => expect(quoteEligibility(10, 1000, TH).eligible).toBe(true));
  it("éligible dès le montant", () => expect(quoteEligibility(1, 200000, TH).eligible).toBe(true));
  it("panier vide : non éligible", () => expect(quoteEligibility(0, 0, TH)).toMatchObject({ eligible: false, remainingItems: 10, remainingAmount: 200000 }));
  it("pas de reste négatif", () => expect(quoteEligibility(25, 900000, TH)).toMatchObject({ remainingItems: 0, remainingAmount: 0 }));
});
