// @ts-check
// ─────────────────────────────────────────────────────────────────────────────
// RÈGLES COMMERCIALES : aucune valeur inventée. TVA, zones et frais de livraison, seuil de livraison gratuite,
// codes promotionnels et seuils de devis seront fournis par le backend (réglages des services pricing, fulfillment,
// order, quote). Tant qu'ils ne sont pas renseignés, ils restent vides (`null` ou liste vide).
// ─────────────────────────────────────────────────────────────────────────────

/**
 * @typedef {{ code: string, type: "percent" | "fixed", value: number, expiresAt: string, minSubtotal?: number }} PromoCode
 * @typedef {{ id: string, fee: number, days: [number, number], freeEligible: boolean }} ShippingZone
 */

/** Seuil d'affichage « Plus que X en stock » : valeur technique d'interface, remplacée par le réglage du service fulfillment. */
export const LOW_STOCK_THRESHOLD = 5;

/** @type {number | null} TVA applicable (ex. 0.18) — à fournir. */
export const VAT_RATE = null;
/** @type {number | null} Seuil de livraison gratuite, en XOF HT — à fournir. */
export const FREE_SHIPPING_THRESHOLD = null;
/** @type {ShippingZone[]} */
export const SHIPPING_ZONES = [];
/** @type {string | null} */
export const DEFAULT_ZONE = null;
/** @type {PromoCode[]} */
export const PROMO_CODES = [];
/** @type {{ minItems: number | null, minAmount: number | null }} Seuils d'accès à la demande de devis — à fournir. */
export const QUOTE_THRESHOLDS = { minItems: null, minAmount: null };
/** Panier initial : toujours vide. @type {{ id: number, quantity: number }[]} */
export const INITIAL_CART = [];
