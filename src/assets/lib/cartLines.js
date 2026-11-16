// @ts-check
// Jointure entre les articles du store (id + quantité), le catalogue et les mocks du panier.
// C'est le seul endroit à adapter quand ces données viendront de l'API.
import { DEMO_PRODUCTS } from "../data/demoProducts.js";
import { CART_PRODUCT_MOCKS, LOW_STOCK_THRESHOLD } from "../data/cartMocks.js";
import { clampQty, hasPriceChanged, nextTier, priceLine, stockStatus } from "./cartMath.js";

/**
 * @typedef {{ id: number, quantity: number, priceAtAdd: number }} CartEntry
 */

/** @param {number} id */
export const getProduct = (id) => DEMO_PRODUCTS.find((p) => p.id === id);

/** Stock disponible d'un produit (0 si inconnu). @param {number} id */
export const getStock = (id) => CART_PRODUCT_MOCKS[id]?.stock ?? 0;

/**
 * Construit le modèle d'affichage d'une ligne, ou null si le produit n'existe plus.
 * @param {CartEntry} entry
 */
export function buildLine(entry) {
  const product = getProduct(entry.id);
  const mock = CART_PRODUCT_MOCKS[entry.id];
  if (!product || !mock) return null;
  const quantity = clampQty(entry.quantity, mock.stock);
  return {
    id: product.id,
    reference: product.reference,
    brand: product.brand,
    quantity,
    stock: mock.stock,
    status: stockStatus(mock.stock, LOW_STOCK_THRESHOLD),
    shipping: mock.shipping,
    specs: mock.specs,
    compat: mock.compat,
    image: mock.image,
    tiers: mock.discounts,
    nextTier: nextTier(quantity, mock.discounts),
    price: product.price,
    priceAtAdd: entry.priceAtAdd,
    priceChanged: hasPriceChanged(entry.priceAtAdd, product.price),
    ...priceLine(product.price, quantity, mock.discounts),
  };
}

/** @param {CartEntry[]} entries */
export const buildLines = (entries) => entries.map(buildLine).filter((l) => l !== null);
