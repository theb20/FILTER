// @ts-check
// ─────────────────────────────────────────────────────────────────────────────
// CATALOGUE : AUCUNE donnée locale. Produits, catégories, stocks, caractéristiques, remises et avis viendront des
// services catalog / pricing / fulfillment (via le gateway). Tant qu'ils n'existent pas, ces collections restent
// VIDES et les pages affichent un état vide honnête. Aucune valeur n'est inventée.
// Les prix sont HORS TAXES, en XOF (devise de base, voir config/currency.js).
// ─────────────────────────────────────────────────────────────────────────────

/**
 * @typedef {{ minQty: number, percent: number }} DiscountTier
 * @typedef {{ id: number, brand: string, reference: string, category: string, price: number, inStock: boolean, crossRefs: string[] }} Product
 * @typedef {{
 *   stock: number, shipping: "24h" | "48h" | "order",
 *   specs: { key: "dimensions" | "thread" | "micron", value: string }[],
 *   compat: string[], discounts: DiscountTier[], image: string | null,
 * }} ProductExtras
 * @typedef {{ id: string, rating: number, title: string, text: string, author: string, company: string, date: string, verified: boolean }} Review
 */

/** @type {{ slug: string }[]} */
export const CATEGORIES = [];

/** @type {Product[]} */
export const PRODUCTS = [];

/** Données complémentaires par id de produit. @type {Record<number, ProductExtras>} */
export const PRODUCT_EXTRAS = {};

/** Avis clients RÉELS uniquement. @type {Review[]} */
export const REVIEWS = [];

/** Nombre d'avis affichés avant « Voir tous les avis ». */
export const REVIEWS_PREVIEW = 3;
