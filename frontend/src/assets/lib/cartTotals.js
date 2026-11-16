// @ts-check
// Totaux du récapitulatif : FONCTIONS PURES (testées dans cartTotals.test.js).
// Montants entiers en XOF. Les prix du panier sont HORS TAXES (HT).

/**
 * @typedef {import("../data/commerce.js").PromoCode} PromoCode
 * @typedef {import("../data/commerce.js").ShippingZone} ShippingZone
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
 *   subtotal: number, promo: PromoCode | null, zone: ShippingZone | null,
 *   freeThreshold: number | null, vatRate: number | null
 * }} input
 */
export function computeTotals({ subtotal, promo, zone, freeThreshold, vatRate }) {
  const discount = promoDiscount(promo, subtotal);
  const afterDiscount = subtotal - discount;
  const hasItems = subtotal > 0;
  // Règle non renseignée (null) : on ne l'applique pas et on n'invente aucune valeur.
  const freeKnown = zone !== null && zone.freeEligible && freeThreshold !== null;
  const freeShipping = hasItems && freeKnown && afterDiscount >= freeThreshold;
  const shipping = !hasItems || freeShipping || zone === null ? 0 : zone.fee;
  const ht = afterDiscount + shipping;
  const vat = vatRate === null ? 0 : Math.round(ht * vatRate);
  const ttc = ht + vat;
  const remainingForFree = freeKnown ? Math.max(0, freeThreshold - afterDiscount) : 0;
  const progress = freeKnown && freeThreshold > 0 ? Math.min(1, afterDiscount / freeThreshold) : 0;
  return { subtotal, discount, afterDiscount, shipping, freeShipping, ht, vat, ttc, remainingForFree, progress };
}

/**
 * La demande de devis est-elle accessible ? Oui dès que l'un des seuils est atteint
 * (nombre d'articles OU montant HT). Renvoie aussi ce qu'il manque pour y accéder.
 * @param {number} count nombre total d'unités dans le panier
 * @param {number} subtotal sous-total HT (avant remise)
 * @param {{ minItems: number | null, minAmount: number | null }} thresholds seuil null = non renseigné (jamais atteint)
 */
export function quoteEligibility(count, subtotal, thresholds) {
  const { minItems, minAmount } = thresholds;
  const eligible = (minItems !== null && count >= minItems) || (minAmount !== null && subtotal >= minAmount);
  return {
    eligible,
    remainingItems: minItems === null ? 0 : Math.max(0, minItems - count),
    remainingAmount: minAmount === null ? 0 : Math.max(0, minAmount - subtotal),
  };
}
