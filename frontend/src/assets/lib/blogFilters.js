// @ts-check
// Filtrage, pagination et URL du blog : FONCTIONS PURES (testées dans blogFilters.test.js).

/** @typedef {import("../data/blogMocks.js").BlogArticle} BlogArticle */
/** @typedef {{ cat: string | null, kind: string | null, tag: string | null, page: number }} BlogParams */

export const PAGE_SIZE = 5; // 1 grande carte + 4 articles en liste

/**
 * Lit les paramètres d'URL en ignorant toute valeur inconnue.
 * @param {URLSearchParams} sp
 * @param {{ categories: string[], kinds: string[], tags: string[] }} allowed
 * @returns {BlogParams}
 */
export function parseBlogParams(sp, allowed) {
  const pick = (key, list) => {
    const v = sp.get(key);
    return v && list.includes(v) ? v : null;
  };
  const page = parseInt(sp.get("page") ?? "1", 10);
  return {
    cat: pick("cat", allowed.categories),
    kind: pick("kind", allowed.kinds),
    tag: pick("tag", allowed.tags),
    page: Number.isFinite(page) && page > 0 ? page : 1,
  };
}

/**
 * Construit la query string : applique `patch`, repasse à la page 1 si un filtre change.
 * @param {BlogParams} current
 * @param {Partial<BlogParams>} patch
 */
export function buildBlogQuery(current, patch) {
  const next = { ...current, ...patch };
  if (!("page" in patch)) next.page = 1;
  const sp = new URLSearchParams();
  if (next.cat) sp.set("cat", next.cat);
  if (next.kind) sp.set("kind", next.kind);
  if (next.tag) sp.set("tag", next.tag);
  if (next.page > 1) sp.set("page", String(next.page));
  const s = sp.toString();
  return s ? `?${s}` : "";
}

/** @param {BlogArticle[]} list */
export const sortByDateDesc = (list) => [...list].sort((a, b) => b.date.localeCompare(a.date));

/**
 * @param {BlogArticle[]} list
 * @param {{ cat: string | null, kind: string | null, tag: string | null }} f
 */
export function filterArticles(list, f) {
  return list.filter(
    (a) => (!f.cat || a.category === f.cat) && (!f.kind || a.kind === f.kind) && (!f.tag || a.tags.includes(f.tag)),
  );
}

/**
 * Pagination : la page demandée est ramenée dans [1, pages].
 * @template T
 * @param {T[]} items
 * @param {number} page
 * @param {number} size
 */
export function paginate(items, page, size) {
  const pages = Math.max(1, Math.ceil(items.length / size));
  const current = Math.min(Math.max(1, page), pages);
  return { items: items.slice((current - 1) * size, current * size), page: current, pages, total: items.length };
}

/**
 * Articles liés : même catégorie ou étiquettes communes, hors article courant, les plus proches d'abord.
 * @param {BlogArticle} article
 * @param {BlogArticle[]} all
 * @param {number} n
 */
export function relatedArticles(article, all, n) {
  const score = (a) => (a.category === article.category ? 2 : 0) + a.tags.filter((t) => article.tags.includes(t)).length;
  return sortByDateDesc(all.filter((a) => a.slug !== article.slug && score(a) > 0))
    .sort((a, b) => score(b) - score(a))
    .slice(0, n);
}

/** @param {BlogArticle[]} all @param {string[]} slugs */
export const pickArticles = (all, slugs) => slugs.map((s) => all.find((a) => a.slug === s)).filter((a) => a !== undefined);

/**
 * Article précédent (plus ancien) et suivant (plus récent) dans l'ordre chronologique.
 * @param {BlogArticle} article
 * @param {BlogArticle[]} all
 */
export function adjacentArticles(article, all) {
  const sorted = sortByDateDesc(all); // du plus récent au plus ancien
  const i = sorted.findIndex((a) => a.slug === article.slug);
  if (i === -1) return { newer: null, older: null };
  return { newer: sorted[i - 1] ?? null, older: sorted[i + 1] ?? null };
}
