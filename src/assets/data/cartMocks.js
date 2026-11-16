// @ts-check
// ─────────────────────────────────────────────────────────────────────────────
// DONNÉES DE DÉMONSTRATION DU PANIER — aucune valeur réelle.
// Ce fichier est le SEUL endroit qui simule le backend (stock, caractéristiques, remises,
// panier initial…). À remplacer par des appels API ; le reste du code n'en dépend que via
// lib/cartLines.js. Tous les montants sont en XOF.
// ─────────────────────────────────────────────────────────────────────────────

/**
 * @typedef {{ minQty: number, percent: number }} DiscountTier
 * @typedef {{
 *   stock: number,
 *   shipping: "24h" | "48h" | "order",
 *   specs: { key: "dimensions" | "thread" | "micron", value: string }[],
 *   compat: string[],
 *   discounts: DiscountTier[],
 *   image: string | null,
 * }} CartProductMock
 */

/** Seuil à partir duquel on affiche « Plus que X en stock ». */
export const LOW_STOCK_THRESHOLD = 5;

/** @type {DiscountTier[]} */
const STANDARD_TIERS = [
  { minQty: 10, percent: 5 },
  { minQty: 50, percent: 10 },
];

/** Données complémentaires par id de produit (voir data/demoProducts.js). @type {Record<number, CartProductMock>} */
export const CART_PRODUCT_MOCKS = {
  1: { stock: 24, shipping: "24h", image: null, discounts: [{ minQty: 10, percent: 5 }],
    specs: [{ key: "dimensions", value: "Ø 93 × 210 mm" }, { key: "thread", value: "M22×1,5" }, { key: "micron", value: "10 µm" }],
    compat: ["compat.loader", "compat.press"] },
  2: { stock: 3, shipping: "48h", image: null, discounts: [],
    specs: [{ key: "dimensions", value: "Ø 78 × 160 mm" }, { key: "thread", value: "3/4\"-16 UNF" }, { key: "micron", value: "25 µm" }],
    compat: ["compat.pump"] },
  3: { stock: 150, shipping: "24h", image: null, discounts: STANDARD_TIERS,
    specs: [{ key: "dimensions", value: "Ø 76 × 123 mm" }, { key: "thread", value: "M20×1,5" }, { key: "micron", value: "20 µm" }],
    compat: ["compat.genset", "compat.truck", "compat.loader"] },
  4: { stock: 0, shipping: "order", image: null, discounts: [],
    specs: [{ key: "dimensions", value: "Ø 90 × 142 mm" }, { key: "thread", value: "M24×1,5" }, { key: "micron", value: "15 µm" }],
    compat: ["compat.truck"] },
  5: { stock: 8, shipping: "24h", image: null, discounts: STANDARD_TIERS,
    specs: [{ key: "dimensions", value: "280 × 215 × 58 mm" }, { key: "thread", value: "—" }, { key: "micron", value: "5 µm" }],
    compat: ["compat.genset", "compat.compressor"] },
  6: { stock: 2, shipping: "48h", image: null, discounts: [],
    specs: [{ key: "dimensions", value: "Ø 150 × 320 mm" }, { key: "thread", value: "G 1\"" }, { key: "micron", value: "0,01 µm" }],
    compat: ["compat.compressor"] },
  7: { stock: 500, shipping: "24h", image: null, discounts: STANDARD_TIERS,
    specs: [{ key: "dimensions", value: "Ø 60 × 100 mm" }, { key: "thread", value: "M16×1,5" }, { key: "micron", value: "40 µm" }],
    compat: ["compat.pump", "compat.crane"] },
  8: { stock: 40, shipping: "24h", image: null, discounts: [{ minQty: 20, percent: 8 }],
    specs: [{ key: "dimensions", value: "Ø 100 × 250 mm" }, { key: "thread", value: "M30×2" }, { key: "micron", value: "10 µm" }],
    compat: ["compat.press", "compat.crane"] },
};

/** Panier présent au tout premier chargement (démo). Une fois modifié, le panier persisté prend le relais. */
export const DEMO_INITIAL_CART = [
  { id: 1, quantity: 1 },
  { id: 3, quantity: 4 },
];

// ── Récapitulatif (étape 2) ──────────────────────────────────────────────────
// Les prix du catalogue sont HORS TAXES (HT) : la TVA s'ajoute au récapitulatif.

/** TVA applicable (fictive pour la démo ; la TVA ivoirienne standard est de 18 %). */
export const VAT_RATE = 0.18;

/** Seuil de livraison gratuite, en XOF HT après remise (valeur FICTIVE). */
export const FREE_SHIPPING_THRESHOLD = 150000;

/**
 * Zones de livraison. fee : frais HT en XOF ; days : délai [min, max] en jours ouvrés ;
 * freeEligible : la livraison gratuite s'applique-t-elle à cette zone ?
 * @typedef {{ id: string, fee: number, days: [number, number], freeEligible: boolean }} ShippingZone
 * @type {ShippingZone[]}
 */
export const SHIPPING_ZONES = [
  { id: "abidjan", fee: 3000, days: [1, 2], freeEligible: true },
  { id: "interior", fee: 8000, days: [3, 5], freeEligible: true },
  { id: "international", fee: 35000, days: [7, 14], freeEligible: false },
];
export const DEFAULT_ZONE = "abidjan";

/**
 * Codes promo de démonstration.
 *  - BIENVENUE10 : valide (−10 %)
 *  - REMISE5000  : valide (−5 000 F CFA) à partir de 50 000 F CFA
 *  - ETE2025     : expiré
 * @typedef {{ code: string, type: "percent" | "fixed", value: number, expiresAt: string, minSubtotal?: number }} PromoCode
 * @type {PromoCode[]}
 */
export const PROMO_CODES = [
  { code: "BIENVENUE10", type: "percent", value: 10, expiresAt: "2099-12-31" },
  { code: "REMISE5000", type: "fixed", value: 5000, expiresAt: "2099-12-31", minSubtotal: 50000 },
  { code: "ETE2025", type: "percent", value: 15, expiresAt: "2025-08-31" },
];

/**
 * Seuils d'accès à la demande de devis (valeurs FICTIVES). Le devis est proposé dès que
 * l'UN des deux seuils est atteint : nombre d'articles OU montant HT du sous-total.
 */
export const QUOTE_THRESHOLDS = { minItems: 10, minAmount: 200000 };
