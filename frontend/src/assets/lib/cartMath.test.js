import { describe, expect, it } from "vitest";
import { clampQty, discountPercent, discountedUnitPrice, hasPriceChanged, itemCount, nextTier, priceLine, stockStatus, subtotal } from "./cartMath.js";
import { convertFromXof, formatMoney } from "../config/currency.js";

const TIERS = [{ minQty: 10, percent: 5 }, { minQty: 50, percent: 10 }];
const flat = (s) => s.replace(/\s/g, " "); // espaces insécables → espace simple

describe("clampQty", () => {
  it("impose un minimum de 1", () => {
    expect(clampQty(0, 10)).toBe(1);
    expect(clampQty(-3, 10)).toBe(1);
    expect(clampQty(NaN, 10)).toBe(1);
  });
  it("plafonne au stock", () => expect(clampQty(99, 8)).toBe(8));
  it("arrondit à l'entier inférieur", () => expect(clampQty(3.9, 10)).toBe(3));
  it("garde 1 si le stock est nul", () => expect(clampQty(5, 0)).toBe(1));
});

describe("remises par quantité", () => {
  it("aucune remise sous le premier palier", () => expect(discountPercent(9, TIERS)).toBe(0));
  it("applique le palier atteint", () => {
    expect(discountPercent(10, TIERS)).toBe(5);
    expect(discountPercent(49, TIERS)).toBe(5);
    expect(discountPercent(50, TIERS)).toBe(10);
  });
  it("0 sans palier", () => expect(discountPercent(100, [])).toBe(0));
  it("trouve le prochain palier", () => {
    expect(nextTier(4, TIERS)).toEqual({ minQty: 10, percent: 5 });
    expect(nextTier(10, TIERS)).toEqual({ minQty: 50, percent: 10 });
    expect(nextTier(50, TIERS)).toBeNull();
  });
  it("arrondit le prix unitaire au franc", () => {
    expect(discountedUnitPrice(15000, 5)).toBe(14250);
    expect(discountedUnitPrice(11001, 5)).toBe(10451);
  });
});

describe("priceLine", () => {
  it("sans remise", () => {
    expect(priceLine(15000, 4, TIERS)).toEqual({ percent: 0, unit: 15000, total: 60000, fullTotal: 60000, savings: 0 });
  });
  it("avec remise", () => {
    expect(priceLine(15000, 10, TIERS)).toEqual({ percent: 5, unit: 14250, total: 142500, fullTotal: 150000, savings: 7500 });
  });
});

describe("totaux et compteurs", () => {
  it("somme les lignes", () => expect(subtotal([{ total: 100 }, { total: 250 }])).toBe(350));
  it("0 pour un panier vide", () => expect(subtotal([])).toBe(0));
  it("compte les unités", () => expect(itemCount([{ quantity: 1 }, { quantity: 4 }])).toBe(5));
});

describe("stock et prix", () => {
  it("statuts de stock", () => {
    expect(stockStatus(0, 5)).toBe("out");
    expect(stockStatus(5, 5)).toBe("low");
    expect(stockStatus(6, 5)).toBe("in");
  });
  it("détecte un changement de prix", () => {
    expect(hasPriceChanged(100, 100)).toBe(false);
    expect(hasPriceChanged(100, 120)).toBe(true);
  });
});

describe("devises", () => {
  it("formate le XOF sans décimale", () => expect(flat(formatMoney(77000, "XOF", "fr-FR"))).toBe("77 000 F CFA"));
  it("arrondit le XOF au franc", () => expect(flat(formatMoney(1234.6, "XOF", "fr-FR"))).toBe("1 235 F CFA"));
  it("convertit en EUR à la parité fixe 655,957", () => expect(convertFromXof(655957, "EUR")).toBeCloseTo(1000, 6));
  it("formate l'EUR avec 2 décimales", () => expect(formatMoney(655957, "EUR", "en-US")).toBe("€1,000.00"));
});
