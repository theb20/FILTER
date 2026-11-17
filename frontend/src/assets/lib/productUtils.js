// @ts-check
// Fonctions pures de la fiche produit (testées dans productUtils.test.js).

/**
 * Statistiques d'avis : moyenne (1 décimale), nombre et répartition 5→1 étoiles.
 * @param {{ rating: number }[]} reviews
 */
export function ratingStats(reviews) {
  const count = reviews.length;
  const distribution = [5, 4, 3, 2, 1].map((stars) => {
    const n = reviews.filter((r) => r.rating === stars).length;
    return { stars, count: n, percent: count ? Math.round((n / count) * 100) : 0 };
  });
  const average = count ? Math.round((reviews.reduce((s, r) => s + r.rating, 0) / count) * 10) / 10 : 0;
  return { average, count, distribution };
}

/**
 * Produits associés : même catégorie d'abord, puis les autres, hors produit courant.
 * @template {{ id: number, category: string }} P
 * @param {P} product
 * @param {P[]} all
 * @param {number} n
 */
export function relatedProducts(product, all, n) {
  const others = all.filter((p) => p.id !== product.id);
  return [...others.filter((p) => p.category === product.category), ...others.filter((p) => p.category !== product.category)].slice(0, n);
}

/**
 * Lignes du tableau des prix dégressifs : prix unitaire remisé à chaque palier.
 * @param {number} price
 * @param {{ minQty: number, percent: number }[]} tiers
 */
export function tierRows(price, tiers) {
  return [...tiers]
    .sort((a, b) => a.minQty - b.minQty)
    .map((t) => ({ minQty: t.minQty, percent: t.percent, unit: Math.round(price * (1 - t.percent / 100)) }));
}
