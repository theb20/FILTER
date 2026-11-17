// @ts-check
import { useSyncExternalStore } from "react";
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { clampQty, itemCount } from "../lib/cartMath.js";
import { MAX_QTY } from "../lib/cartLines.js";

/**
 * @typedef {import("../lib/cartLines.js").CartEntry} CartEntry
 * @typedef {{
 *   items: CartEntry[],
 *   saved: CartEntry[],
 *   promoCode: string | null,
 *   zone: string,
 *   note: string,
 *   applyPromo: (code: string) => void,
 *   clearPromo: () => void,
 *   setZone: (zone: string) => void,
 *   setNote: (note: string) => void,
 *   addItem: (product: { id: string, price: number }, quantity?: number) => void,
 *   setQuantity: (id: string, quantity: number) => void,
 *   removeItem: (id: string) => { entry: CartEntry, index: number } | null,
 *   restoreItem: (entry: CartEntry, index: number) => void,
 *   saveForLater: (id: string) => void,
 *   moveToCart: (id: string) => void,
 *   removeSaved: (id: string) => void,
 * }} CartState
 */

// Clé versionnée : si la structure persistée change, incrémenter la version et adapter `migrate`.
export const CART_STORAGE_KEY = "ff_cart_v3"; // v3 : les produits viennent de l'API (identifiants UUID) ; abandonne les paniers à identifiants numériques
export const CART_STORAGE_VERSION = 3;

/** @param {{ id: string, price: number }} product @param {number} quantity @returns {CartEntry} */
const makeEntry = (product, quantity) => ({ id: product.id, quantity: clampQty(quantity, MAX_QTY), priceAtAdd: product.price });

export const useCartStore = create(
  persist(
    /** @returns {CartState} */
    (set, get) => ({
      items: [],
      saved: [],
      promoCode: null,
      zone: "",
      note: "",

      applyPromo: (code) => set({ promoCode: code.trim().toUpperCase() }),
      clearPromo: () => set({ promoCode: null }),
      setZone: (zone) => set({ zone }),
      setNote: (note) => set({ note: note.slice(0, 200) }),

      addItem: (product, quantity = 1) =>
        set((s) => {
          const id = product.id;
          const existing = s.items.find((i) => i.id === id);
          if (existing) {
            return { items: s.items.map((i) => (i.id === id ? { ...i, quantity: clampQty(i.quantity + quantity, MAX_QTY) } : i)) };
          }
          return { items: [...s.items, makeEntry(product, quantity)] };
        }),

      setQuantity: (id, quantity) =>
        set((s) => ({ items: s.items.map((i) => (i.id === id ? { ...i, quantity: clampQty(quantity, MAX_QTY) } : i)) })),

      removeItem: (id) => {
        const index = get().items.findIndex((i) => i.id === id);
        if (index === -1) return null;
        const entry = get().items[index];
        set((s) => ({ items: s.items.filter((i) => i.id !== id) }));
        return { entry, index };
      },

      restoreItem: (entry, index) =>
        set((s) => {
          if (s.items.some((i) => i.id === entry.id)) return s;
          const items = [...s.items];
          items.splice(Math.min(index, items.length), 0, entry);
          return { items };
        }),

      saveForLater: (id) =>
        set((s) => {
          const entry = s.items.find((i) => i.id === id);
          if (!entry) return s;
          return {
            items: s.items.filter((i) => i.id !== id),
            saved: s.saved.some((i) => i.id === id) ? s.saved : [...s.saved, entry],
          };
        }),

      moveToCart: (id) =>
        set((s) => {
          const entry = s.saved.find((i) => i.id === id);
          if (!entry) return s;
          const saved = s.saved.filter((i) => i.id !== id);
          const existing = s.items.find((i) => i.id === id);
          if (existing) return { saved, items: s.items.map((i) => (i.id === id ? { ...i, quantity: clampQty(i.quantity + entry.quantity, MAX_QTY) } : i)) };
          return { saved, items: [...s.items, entry] };
        }),

      removeSaved: (id) => set((s) => ({ saved: s.saved.filter((i) => i.id !== id) })),
    }),
    {
      name: CART_STORAGE_KEY,
      version: CART_STORAGE_VERSION,
      // Champs ajoutés à l'étape 2 (promoCode, zone, note) : la fusion avec l'état initial les complète
      // pour un panier déjà persisté, pas de changement de version nécessaire.
      partialize: (s) => ({ items: s.items, saved: s.saved, promoCode: s.promoCode, zone: s.zone, note: s.note }),
      // Ex. : if (version === 1) { …transformer l'ancienne structure… } — rien à migrer pour l'instant.
      migrate: (persisted) => persisted,
    },
  ),
);

/**
 * `true` une fois le panier relu depuis le stockage. Permet d'afficher un squelette plutôt
 * qu'un panier vide qui se remplit (utile si le stockage devient asynchrone).
 */
export function useCartHydrated() {
  return useSyncExternalStore(
    (cb) => useCartStore.persist.onFinishHydration(cb),
    () => useCartStore.persist.hasHydrated(),
    () => false,
  );
}

/** Nombre total d'unités dans le panier (compteur du header). */
export const useCartCount = () => useCartStore((s) => itemCount(s.items));
