// @ts-check
// Catalogue : lecture des routes publiques du gateway (produits publiés, catégories, marques, secteurs) et adaptation
// des réponses au modèle d'affichage du site. AUCUNE donnée locale : tout vient du service catalog.
import { useEffect, useState } from "react";
import { request, getApiLang } from "./api.js";

/**
 * @typedef {{
 *   id: string, name: string, reference: string, brand: string, category: string, categoryName: string, categoryPath: string[],
 *   price: number, onQuote: boolean, image: string | null, availability: string, inStock: boolean, leadTimeDays: number | null,
 * }} CatalogProduct
 */

/** @param {any} s résumé produit de l'API @returns {CatalogProduct} */
export function adaptSummary(s) {
  const mode = s.price?.mode ?? "fixed";
  return {
    id: s.id,
    name: s.name,
    reference: s.sku,
    brand: s.brand?.name ?? "",
    category: s.category?.slug ?? "",
    categoryName: s.category?.name ?? "",
    categoryPath: (s.category?.path ?? []).map((/** @type {{slug: string}} */ c) => c.slug),
    price: Number(s.price?.unitPrice?.amount ?? 0),
    onQuote: mode === "on_quote",
    image: s.image?.url ?? null,
    availability: s.availability?.status ?? "on_request",
    inStock: s.availability?.status === "in_stock" || s.availability?.status === "low_stock",
    leadTimeDays: s.availability?.leadTimeDays ?? null,
  };
}

/** Fiche complète : le résumé plus description, caractéristiques, médias, références croisées, compatibilités. @param {any} p */
export function adaptProduct(p) {
  return {
    ...adaptSummary(p),
    description: p.description ?? "",
    attributes: /** @type {{key: string, label: string, unit: string | null, value: string | number | boolean}[]} */ (p.attributes ?? []),
    media: /** @type {{id: string, url: string, alt: string | null}[]} */ (p.media ?? []),
    crossRefs: /** @type {string[]} */ ((p.crossReferences ?? []).map((/** @type {{brand: string, number: string}} */ r) => `${r.brand} ${r.number}`)),
    sectors: /** @type {{id: string, slug: string, name: string, imageUrl: string | null}[]} */ (p.sectors ?? []),
    compat: /** @type {string[]} */ ((p.compatibilities ?? []).map((/** @type {{machine: {make: string, model: string}}} */ c) => `${c.machine.make} ${c.machine.model}`)),
    volumeTiers: /** @type {{minQuantity: number, percent: number}[]} */ (p.price?.volumeTiers ?? []),
  };
}

/** Cache mémoire, par langue, valable 60 s : évite de redemander à chaque navigation sans retarder les mises à jour du backoffice. @type {Map<string, { at: number, p: Promise<any> }>} */
const cache = new Map();
const TTL_MS = 60_000;
export function cached(/** @type {string} */ key, /** @type {() => Promise<any>} */ load) {
  const k = `${getApiLang()}|${key}`;
  const hit = cache.get(k);
  if (hit && Date.now() - hit.at < TTL_MS) return hit.p;
  const p = load().catch((e) => { cache.delete(k); throw e; });
  cache.set(k, { at: Date.now(), p });
  return p;
}
export const clearCatalogCache = () => cache.clear();

/** @param {string} path */
export async function getJson(path) {
  const r = await request(path);
  if (!r.ok) throw Object.assign(new Error(r.code ?? "ERROR"), { status: r.status });
  return r.data;
}

/** Toutes les pages d'une liste paginée (le catalogue est filtré côté site tant que le moteur de recherche n'est pas branché). @param {string} path */
async function getAll(path) {
  const out = [];
  for (let page = 1; page < 50; page++) {
    const d = await getJson(`${path}${path.includes("?") ? "&" : "?"}page=${page}&pageSize=100`);
    out.push(...d.items);
    if (page >= d.page.totalPages) break;
  }
  return out;
}

/** @returns {Promise<CatalogProduct[]>} */
export const fetchProducts = () => cached("products", async () => (await getAll("/products")).map(adaptSummary));
/** Identifiants des produits d'un secteur (filtre côté API : ?sectorId=). @param {string} sectorId @returns {Promise<Set<string>>} */
export const fetchProductIdsBySector = (sectorId) => cached(`sector-products:${sectorId}`, async () => new Set((await getAll(`/products?sectorId=${encodeURIComponent(sectorId)}`)).map((p) => p.id)));
/** @param {string} idOrSku */
export const fetchProduct = (idOrSku) => cached(`product:${idOrSku}`, async () => adaptProduct(await getJson(`/products/${encodeURIComponent(idOrSku)}`)));
/** @param {string} id @returns {Promise<CatalogProduct[]>} */
export const fetchRelated = (id) => cached(`related:${id}`, async () => (await getJson(`/products/${id}/related?limit=6`)).map(adaptSummary));
/** Identifiant de session anonyme : un nombre au hasard, gardé dans le navigateur, qui sert UNIQUEMENT à compter les
 *  recherches web du jour. Aucun lien avec une personne ; l'effacer remet simplement le compteur du navigateur à zéro. */
export function visitorId() {
  try {
    let v = localStorage.getItem("ff-visitor");
    if (!v) {
      v = crypto.randomUUID();
      localStorage.setItem("ff-visitor", v);
    }
    return v;
  } catch {
    return ""; // navigation privée ou stockage refusé : seul le plafond par adresse réseau s'applique
  }
}

/** Recherche sur le web (source externe), à la demande explicite du visiteur : chaque appel est facturé.
 *  @param {string} q @param {string} [token] jeton du client connecté (plafond journalier plus élevé)
 *  @returns {Promise<{ok: boolean, status: number, data: any, retryAfter?: number}>} */
export const searchWeb = (q, token) =>
  request(`/search/web?q=${encodeURIComponent(q)}`, { token, headers: { "X-Visitor-Id": visitorId() } });

/** Photos d'un produit trouvé sur le web (fiche détaillée). Appel facturé, décompté des mêmes plafonds. */
export const searchWebImages = (q, token) =>
  request(`/search/web/images?q=${encodeURIComponent(q)}`, { token, headers: { "X-Visitor-Id": visitorId() } });

/** Identifiant court et stable d'un résultat web, dérivé de son adresse : sert d'adresse à sa fiche sur le site. */
export function webHitId(url) {
  let h = 5381;
  for (const ch of String(url)) h = ((h * 33) ^ ch.codePointAt(0)) >>> 0;
  return h.toString(36);
}

/** Les résultats web ne sont pas conservés côté serveur : la fiche les relit dans l'onglet (effacé à sa fermeture). */
export function rememberWebHit(hit) {
  try {
    sessionStorage.setItem(`ff-web:${webHitId(hit.url)}`, JSON.stringify(hit));
  } catch { /* stockage refusé : la fiche se rouvrira depuis la liste */ }
}

/** Série de résultats d'une recherche : la fiche s'en sert pour proposer les autres produits trouvés. */
export function rememberWebResults(query, hits) {
  try {
    sessionStorage.setItem("ff-web:last", JSON.stringify({ query, hits }));
    hits.forEach(rememberWebHit);
  } catch { /* stockage refusé */ }
}
export function recallWebResults() {
  try {
    return JSON.parse(sessionStorage.getItem("ff-web:last") || "null");
  } catch {
    return null;
  }
}

/** Référence probable lue dans une désignation : le plus long mot qui mêle lettres et chiffres (ex. « SH 63029 », « 7572FC »).
 *  Sert à préremplir la demande de devis ; le client la vérifie toujours. */
export function guessReference(title = "") {
  const words = String(title).toUpperCase().match(/[A-Z0-9][A-Z0-9./-]{3,}/g) ?? [];
  const refs = words.filter((w) => /\d/.test(w) && w.length >= 4 && !/^(19|20)\d\d$/.test(w));
  return refs.sort((a, b) => b.length - a.length)[0] ?? "";
}
export function recallWebHit(id) {
  try {
    return JSON.parse(sessionStorage.getItem(`ff-web:${id}`) || "null");
  } catch {
    return null;
  }
}

/** Arbre des catégories (racines avec leurs enfants). @returns {Promise<any[]>} */
export const fetchCategories = () => cached("categories", () => getJson("/categories"));
/** @returns {Promise<{id: string, slug: string, name: string, logoUrl: string | null}[]>} */
export const fetchBrands = () => cached("brands", () => getAll("/brands"));
/** @returns {Promise<{id: string, slug: string, name: string, imageUrl: string | null}[]>} */
export const fetchSectors = () => cached("sectors", () => getJson("/sectors"));

/** Aplatit l'arbre : [{ slug, name, depth, path }]. @param {any[]} tree @param {string[]} [path] */
export function flattenCategories(tree, path = []) {
  /** @type {{slug: string, name: string, depth: number, path: string[]}[]} */
  const out = [];
  for (const c of tree) {
    out.push({ slug: c.slug, name: c.name, depth: path.length, path: [...path, c.slug] });
    out.push(...flattenCategories(c.children ?? [], [...path, c.slug]));
  }
  return out;
}

/**
 * Charge une donnée asynchrone. `status` : "loading" | "ready" | "error". Se relance quand `deps` change.
 * @template T
 * @param {() => Promise<T>} load
 * @param {any[]} deps
 * @returns {{ status: "loading" | "ready" | "error", data: T | null, error: any }}
 */
export function useAsync(load, deps) {
  const [state, setState] = useState(/** @type {{status: "loading" | "ready" | "error", data: T | null, error: any}} */ ({ status: "loading", data: null, error: null }));
  useEffect(() => {
    let live = true;
    setState((s) => (s.status === "loading" ? s : { ...s, status: "loading" }));
    load().then(
      (data) => live && setState({ status: "ready", data, error: null }),
      (error) => live && setState({ status: "error", data: null, error }),
    );
    return () => { live = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return state;
}

/**
 * Produits du panier : charge chaque fiche publiée par identifiant. Un produit retiré du catalogue est simplement absent
 * (la ligne disparaît du panier affiché). `ready` passe à true quand toutes les réponses sont arrivées.
 * @param {string[]} ids
 * @returns {{ products: Record<string, any>, ready: boolean }}
 */
export function useProductsById(ids) {
  const key = [...new Set(ids)].sort().join(",");
  const state = useAsync(async () => {
    const list = key ? key.split(",") : [];
    const found = await Promise.all(list.map((id) => fetchProduct(id).catch(() => null)));
    return Object.fromEntries(found.filter(Boolean).map((p) => [p.id, p]));
  }, [key, getApiLang()]);
  return { products: state.data ?? {}, ready: state.status !== "loading" };
}

/** Chiffres du catalogue, calculés par l'API (produits publiés, marques). @returns {Promise<{ products: number, brands: number }>} */
export const fetchStats = () =>
  cached("stats", async () => {
    const [p, b] = await Promise.all([getJson("/products?pageSize=1"), getJson("/brands?pageSize=1")]);
    return { products: p.page.total, brands: b.page.total };
  });

/** Derniers produits publiés (les plus récemment modifiés d'abord). @returns {Promise<CatalogProduct[]>} */
export const fetchLatest = () => cached("latest", async () => (await getJson("/products?pageSize=12")).items.map(adaptSummary));
