// @ts-check
// Informations du site et règles commerciales, lues dans l'API (renseignées au backoffice). Rien n'est codé en dur :
// une valeur absente reste vide (« à compléter »), jamais inventée. La dernière réponse est gardée dans le navigateur
// pour éviter un affichage « à compléter » au chargement (simple confort : l'API fait foi dès qu'elle répond).
import { useEffect } from "react";
import { create } from "zustand";
import { request } from "../lib/api.js";

const KEY = "ff_site_v1";

/** @typedef {{ name: string, publisher: string, phone: string, email: string, address: string, linkedinUrl: string, facebookUrl: string, youtubeUrl: string, reviewsUrl: string,
 *   regionalSite: { country: string, url: string } | null,
 *   claims: { crossReferences: string, stockValue: string, foundedYear: string, freeShippingThreshold: string } }} Site
 * @typedef {{ id: string, name: string, fee: number, days: [number, number], freeEligible: boolean }} Zone
 * @typedef {{ vatRate: number | null, freeShippingThreshold: number | null, quoteThresholds: { minItems: number | null, minAmount: number | null }, lowStockThreshold: number, defaultZone: string, zones: Zone[] }} Commerce
 */

/** @type {Site} */
const EMPTY_SITE = { name: "", publisher: "", phone: "", email: "", address: "", linkedinUrl: "", facebookUrl: "", youtubeUrl: "", reviewsUrl: "", regionalSite: null,
  claims: { crossReferences: "", stockValue: "", foundedYear: "", freeShippingThreshold: "" } };
/** @type {Commerce} */
const EMPTY_COMMERCE = { vatRate: null, freeShippingThreshold: null, quoteThresholds: { minItems: null, minAmount: null }, lowStockThreshold: 5, defaultZone: "", zones: [] };

/** @param {any} d @returns {Site} */
export function siteFromApi(d) {
  return {
    name: d.name ?? "", publisher: d.publisher ?? "", phone: d.phone ?? "", email: d.email ?? "", address: d.address ?? "",
    linkedinUrl: d.linkedinUrl ?? "", facebookUrl: d.facebookUrl ?? "", youtubeUrl: d.youtubeUrl ?? "", reviewsUrl: d.reviewsUrl ?? "",
    regionalSite: d.regionalCountry && d.regionalUrl ? { country: d.regionalCountry, url: d.regionalUrl } : null,
    claims: { crossReferences: d.crossReferencesClaim ?? "", stockValue: d.stockValueClaim ?? "", foundedYear: d.foundedYear ?? "", freeShippingThreshold: d.freeShippingClaim ?? "" },
  };
}

/** @param {any} d @param {any[]} zones @returns {Commerce} */
export function commerceFromApi(d, zones) {
  return {
    vatRate: typeof d.vatRate === "number" ? d.vatRate / 100 : null,
    freeShippingThreshold: typeof d.freeShippingThreshold === "number" ? d.freeShippingThreshold : null,
    quoteThresholds: { minItems: typeof d.quoteMinItems === "number" ? d.quoteMinItems : null, minAmount: typeof d.quoteMinAmount === "number" ? d.quoteMinAmount : null },
    lowStockThreshold: typeof d.lowStockThreshold === "number" ? d.lowStockThreshold : 5,
    defaultZone: d.defaultZone ?? "",
    zones: zones.map((z) => ({ id: z.code, name: z.name, fee: z.fee, days: /** @type {[number, number]} */ ([z.daysMin, z.daysMax]), freeEligible: z.freeEligible })),
  };
}

function cached() {
  try {
    const raw = localStorage.getItem(KEY);
    if (raw) return JSON.parse(raw);
  } catch {
    // stockage indisponible : on attend l'API
  }
  return null;
}

export const useSiteStore = create((set, get) => {
  const c = cached();
  return {
    site: c?.site ?? EMPTY_SITE,
    commerce: c?.commerce ?? EMPTY_COMMERCE,
    status: /** @type {"idle" | "loading" | "ready" | "error"} */ ("idle"),
    async load() {
      if (get().status === "loading" || get().status === "ready") return;
      set({ status: "loading" });
      const [s, c2, z] = await Promise.all([request("/site-settings"), request("/commerce-settings"), request("/shipping/zones")]);
      if (!s.ok || !c2.ok || !z.ok) return set({ status: "error" });
      const site = siteFromApi(s.data);
      const commerce = commerceFromApi(c2.data, z.data);
      set({ site, commerce, status: "ready" });
      try {
        localStorage.setItem(KEY, JSON.stringify({ site, commerce }));
      } catch {
        // préférence non mémorisée, sans conséquence
      }
    },
  };
});

/** Informations du site ; déclenche le chargement au premier usage. @returns {Site} */
export function useSite() {
  const load = useSiteStore((s) => s.load);
  useEffect(() => { load(); }, [load]);
  return useSiteStore((s) => s.site);
}

/** Règles commerciales (TVA, livraison, seuils de devis, zones). @returns {Commerce} */
export function useCommerce() {
  const load = useSiteStore((s) => s.load);
  useEffect(() => { load(); }, [load]);
  return useSiteStore((s) => s.commerce);
}
