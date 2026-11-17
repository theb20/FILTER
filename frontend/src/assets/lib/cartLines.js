// @ts-check
// Jointure entre les articles du panier (id + quantité), les produits lus dans l'API et leurs remises.
import { clampQty, hasPriceChanged, nextTier, priceLine } from "./cartMath.js";

/**
 * @typedef {{ id: string, quantity: number, priceAtAdd: number }} CartEntry
 * @typedef {import("./catalog.js").CatalogProduct & { volumeTiers?: { minQuantity: number, percent: number }[], attributes?: { label: string, unit: string | null, value: string | number | boolean }[], compat?: string[] }} LineProduct
 */

/** Tant que le stock n'est pas géré par le backend, la quantité n'est bornée que par cette limite technique. */
export const MAX_QTY = 9999;

/** @param {string} availability @returns {"in" | "low" | "out" | "request" | "made"} */
export function lineStatus(availability) {
  if (availability === "in_stock") return "in";
  if (availability === "low_stock") return "low";
  if (availability === "out_of_stock") return "out";
  if (availability === "made_to_order") return "made";
  return "request";
}

/**
 * Modèle d'affichage d'une ligne, ou null si le produit n'est pas (ou plus) publié.
 * @param {CartEntry} entry
 * @param {LineProduct | undefined} product
 */
export function buildLine(entry, product) {
  if (!product) return null;
  const quantity = clampQty(entry.quantity, MAX_QTY);
  const tiers = (product.volumeTiers ?? []).map((t) => ({ minQty: t.minQuantity, percent: t.percent }));
  return {
    id: product.id,
    name: product.name,
    reference: product.reference,
    brand: product.brand,
    quantity,
    stock: MAX_QTY,
    status: lineStatus(product.availability),
    leadTimeDays: product.leadTimeDays,
    onQuote: product.onQuote,
    specs: (product.attributes ?? []).slice(0, 4).map((a) => ({ label: a.label, value: `${a.value}${a.unit ? ` ${a.unit}` : ""}` })),
    compat: product.compat ?? [],
    image: product.image,
    tiers,
    nextTier: nextTier(quantity, tiers),
    price: product.price,
    priceAtAdd: entry.priceAtAdd,
    priceChanged: !product.onQuote && hasPriceChanged(entry.priceAtAdd, product.price),
    ...priceLine(product.price, quantity, tiers),
  };
}

/**
 * @param {CartEntry[]} entries
 * @param {Record<string, LineProduct>} products produits déjà chargés, par identifiant
 */
export const buildLines = (entries, products) => entries.map((e) => buildLine(e, products[e.id])).filter((l) => l !== null);
