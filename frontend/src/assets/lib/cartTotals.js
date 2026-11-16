// @ts-check
// Totaux du récapitulatif : FONCTIONS PURES (testées dans cartTotals.test.js).
// Montants entiers en XOF. Les prix du panier sont HORS TAXES (HT).

/**
 * @typedef {import("../data/cartMocks.js").PromoCode} PromoCode
 * @typedef {import("../data/cartMocks.js").ShippingZone} ShippingZone
 * @typedef {{ status: "valid", promo: PromoCode }
 *   | { status: "invalid" | "expired" }
 *   | { status: "minimum", promo: PromoCode }} PromoResult
 */

/**
 * Valide un code promo : "valid", "invalid" (inconnu), "expired" ou "minimum" (panier trop bas).
 * Un code est valable jusqu'à la fin du jour d'expiration (UTC).
 * @param {string} rawCode
 * @param {number} subtotal sous-total HT
 * @param {Date} now
 * @param {PromoCode[]} promos
 * @returns {PromoResult}
 */
export function validatePromo(rawCode, subtotal, now, promos) {
  const code = rawCode.trim().toUpperCase();
  const promo = promos.find((p) => p.code === code);
  if (!promo) return { status: "invalid" };
  if (now.getTime() > new Date(`${promo.expiresAt}T23:59:59Z`).getTime()) return { status: "expired" };
  if (promo.minSubtotal && subtotal < promo.minSubtotal) return { status: "minimum", promo };
  return { status: "valid", promo };
}

/**
 * Montant de la remise (jamais supérieur au sous-total).
 * @param {PromoCode | null} promo
 * @param {number} subtotal
 */
export function promoDiscount(promo, subtotal) {
  if (!promo) return 0;
  const raw = promo.type === "percent" ? Math.round((subtotal * promo.value) / 100) : promo.value;
  return Math.min(raw, subtotal);
}

/**
 * Ajoute `n` jours ouvrés (lundi–vendredi) à une date.
 * @param {Date} from
 * @param {number} n
 */
export function addBusinessDays(from, n) {
  const d = new Date(from.getTime());
  let left = n;
  while (left > 0) {
    d.setDate(d.getDate() + 1);
    const day = d.getDay();
    if (day !== 0 && day !== 6) left -= 1;
  }
  return d;
}

/**
 * Fenêtre de livraison estimée pour une zone.
 * @param {Date} now
 * @param {ShippingZone} zone
 */
export function deliveryWindow(now, zone) {
  return { from: addBusinessDays(now, zone.days[0]), to: addBusinessDays(now, zone.days[1]) };
}

/**
 * Calcule tous les montants du récapitulatif.
 * @param {{
 *   subtotal: number, promo: PromoCode | null, zone: ShippingZone,
 *   freeThreshold: number, vatRate: number
 * }} input
 */
export function computeTotals({ subtotal, promo, zone, freeThreshold, vatRate }) {
  const discount = promoDiscount(promo, subtotal);
  const afterDiscount = subtotal - discount;
  const hasItems = subtotal > 0;
  const freeShipping = hasItems && zone.freeEligible && afterDiscount >= freeThreshold;
  const shipping = !hasItems || freeShipping ? 0 : zone.fee;
  const ht = afterDiscount + shipping;
  const vat = Math.round(ht * vatRate);
  const ttc = ht + vat;
  const remainingForFree = zone.freeEligible ? Math.max(0, freeThreshold - afterDiscount) : 0;
  const progress = zone.freeEligible && freeThreshold > 0 ? Math.min(1, afterDiscount / freeThreshold) : 0;
  return { subtotal, discount, afterDiscount, shipping, freeShipping, ht, vat, ttc, remainingForFree, progress };
}

/**
 * La demande de devis est-elle accessible ? Oui dès que l'un des seuils est atteint
 * (nombre d'articles OU montant HT). Renvoie aussi ce qu'il manque pour y accéder.
 * @param {number} count nombre total d'unités dans le panier
 * @param {number} subtotal sous-total HT (avant remise)
 * @param {{ minItems: number, minAmount: number }} thresholds
 */
export function quoteEligibility(count, subtotal, thresholds) {
  const eligible = count >= thresholds.minItems || subtotal >= thresholds.minAmount;
  return {
    eligible,
    remainingItems: Math.max(0, thresholds.minItems - count),
    remainingAmount: Math.max(0, thresholds.minAmount - subtotal),
  };
}
