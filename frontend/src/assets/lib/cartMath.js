// @ts-check
// Calculs du panier : FONCTIONS PURES (aucun état, aucun accès au DOM), testées dans cartMath.test.js.
// Tous les montants sont des entiers en XOF (le franc CFA n'a pas de centime).

/** @typedef {{ minQty: number, percent: number }} DiscountTier */

/**
 * Borne une quantité : entier ≥ 1 et ≤ stock. Un article en rupture (stock 0) reste à 1 minimum.
 * @param {number} qty
 * @param {number} stock
 */
export function clampQty(qty, stock) {
  const n = Number.isFinite(qty) ? Math.floor(qty) : 1;
  const max = Math.max(1, Math.floor(stock));
  return Math.min(Math.max(1, n), max);
}

/**
 * Pourcentage de remise applicable à cette quantité (palier le plus élevé atteint, 0 sinon).
 * @param {number} qty
 * @param {DiscountTier[]} tiers
 */
export function discountPercent(qty, tiers) {
  return tiers.reduce((best, t) => (qty >= t.minQty && t.percent > best ? t.percent : best), 0);
}

/**
 * Prochain palier de remise non encore atteint, ou null.
 * @param {number} qty
 * @param {DiscountTier[]} tiers
 * @returns {DiscountTier | null}
 */
export function nextTier(qty, tiers) {
  const upcoming = tiers.filter((t) => t.minQty > qty).sort((a, b) => a.minQty - b.minQty);
  return upcoming[0] ?? null;
}

/**
 * Prix unitaire après remise, arrondi au franc.
 * @param {number} price
 * @param {number} percent
 */
export function discountedUnitPrice(price, percent) {
  return Math.round(price * (1 - percent / 100));
}

/**
 * Détail d'une ligne : prix unitaire remisé, total, économie réalisée.
 * @param {number} price prix unitaire catalogue
 * @param {number} qty
 * @param {DiscountTier[]} tiers
 */
export function priceLine(price, qty, tiers) {
  const percent = discountPercent(qty, tiers);
  const unit = discountedUnitPrice(price, percent);
  const total = unit * qty;
  const fullTotal = price * qty;
  return { percent, unit, total, fullTotal, savings: fullTotal - total };
}

/**
 * @param {{ total: number }[]} lines
 */
export function subtotal(lines) {
  return lines.reduce((sum, l) => sum + l.total, 0);
}

/**
 * Nombre total d'unités (pour le compteur du header).
 * @param {{ quantity: number }[]} items
 */
export function itemCount(items) {
  return items.reduce((sum, i) => sum + i.quantity, 0);
}

/**
 * Statut de stock : "out" (rupture), "low" (≤ seuil) ou "in".
 * @param {number} stock
 * @param {number} lowThreshold
 * @returns {"out" | "low" | "in"}
 */
export function stockStatus(stock, lowThreshold) {
  if (stock <= 0) return "out";
  return stock <= lowThreshold ? "low" : "in";
}

/**
 * Le prix catalogue a-t-il changé depuis l'ajout au panier ?
 * @param {number} priceAtAdd
 * @param {number} currentPrice
 */
export function hasPriceChanged(priceAtAdd, currentPrice) {
  return priceAtAdd !== currentPrice;
}
