import { describe, expect, it } from "vitest";
import { ratingStats, relatedProducts, tierRows } from "./productUtils.js";
import { DEMO_REVIEWS } from "../data/productMocks.js";
import { DEMO_PRODUCTS } from "../data/demoProducts.js";

describe("ratingStats", () => {
  it("calcule moyenne, nombre et répartition", () => {
    const s = ratingStats([{ rating: 5 }, { rating: 5 }, { rating: 4 }, { rating: 1 }]);
    expect(s.count).toBe(4);
    expect(s.average).toBe(3.8); // 15 / 4 = 3,75 → 3,8
    expect(s.distribution.find((d) => d.stars === 5)).toEqual({ stars: 5, count: 2, percent: 50 });
    expect(s.distribution.find((d) => d.stars === 2).count).toBe(0);
  });
  it("sans avis : tout à 0", () => expect(ratingStats([])).toMatchObject({ average: 0, count: 0 }));
  it("la répartition totalise le nombre d'avis", () => {
    const s = ratingStats(DEMO_REVIEWS);
    expect(s.distribution.reduce((n, d) => n + d.count, 0)).toBe(DEMO_REVIEWS.length);
  });
});

describe("relatedProducts", () => {
  it("exclut le produit courant et privilégie la catégorie", () => {
    const p = DEMO_PRODUCTS[0]; // hydraulic
    const r = relatedProducts(p, DEMO_PRODUCTS, 4);
    expect(r).toHaveLength(4);
    expect(r.some((x) => x.id === p.id)).toBe(false);
    expect(r[0].category).toBe(p.category);
  });
  it("respecte la limite", () => expect(relatedProducts(DEMO_PRODUCTS[0], DEMO_PRODUCTS, 2)).toHaveLength(2));
});

describe("tierRows", () => {
  it("trie par quantité et calcule le prix unitaire remisé", () => {
    expect(tierRows(15000, [{ minQty: 50, percent: 10 }, { minQty: 10, percent: 5 }])).toEqual([
      { minQty: 10, percent: 5, unit: 14250 },
      { minQty: 50, percent: 10, unit: 13500 },
    ]);
  });
  it("tableau vide sans palier", () => expect(tierRows(15000, [])).toEqual([]));
});
