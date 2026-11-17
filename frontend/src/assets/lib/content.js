// @ts-check
// Contenu éditorial lu dans l'API (FAQ, pages, avis, blog) : tout se gère au backoffice, rien n'est écrit dans le site.
import { cached, getJson } from "./catalog.js";

/** @returns {Promise<{ id: string, question: string, answer: string }[]>} */
export const fetchFaq = () => cached("faq", () => getJson("/faq"));

/** Page légale ou éditoriale publiée, ou null si elle n'existe pas (encore). @param {string} slug */
export const fetchPage = (slug) =>
  cached(`page:${slug}`, async () => {
    try {
      return await getJson(`/pages/${encodeURIComponent(slug)}`);
    } catch (e) {
      if (/** @type {any} */ (e).status === 404) return null;
      throw e;
    }
  });

/** @typedef {{ id: string, rating: number, title: string, text: string, author: string, company?: string, verified: boolean, date: string }} Review */
/** @param {string} productId @returns {Promise<{ items: Review[], count: number, average: number }>} */
export const fetchProductReviews = (productId) => cached(`reviews:${productId}`, () => getJson(`/products/${productId}/reviews`));
/** Témoignages affichés sur l'accueil. @returns {Promise<{ items: Review[], count: number, average: number }>} */
export const fetchSiteReviews = () => cached("reviews:site", () => getJson("/reviews"));

/** @typedef {{ slug: string, title: string, excerpt: string, category: string, categoryName: string, kind: "advice" | "comparison" | "news", author: string, authorBio: string, date: string, readMin: number, image: string, tags: string[], products: string[], guide?: "steps" | "checklist", dossier: boolean, featured: boolean, sections: { id: string, heading: string, paragraphs: string[] }[] }} BlogArticle */

/** Corps Markdown simplifié (« ## » = intertitre, ligne vide = nouveau paragraphe) → sections d'affichage. @param {string} body */
export function parseBody(body) {
  /** @type {{ id: string, heading: string, paragraphs: string[] }[]} */
  const sections = [];
  let cur = { id: "intro", heading: "", paragraphs: /** @type {string[]} */ ([]) };
  for (const block of body.replace(/\r/g, "").split(/\n{2,}/)) {
    const m = block.match(/^#{2,3}\s+(.+?)\s*(?:\n([\s\S]*))?$/);
    if (m) {
      if (cur.heading || cur.paragraphs.length) sections.push(cur);
      cur = { id: `s${sections.length + 1}`, heading: m[1], paragraphs: m[2] ? [m[2].trim()] : [] };
    } else if (block.trim()) {
      cur.paragraphs.push(block.trim());
    }
  }
  if (cur.heading || cur.paragraphs.length) sections.push(cur);
  return sections;
}

/** @param {any} a article public de l'API @param {Map<string, string>} [catNames] @returns {BlogArticle} */
export function adaptArticle(a, catNames = new Map()) {
  return {
    slug: a.slug, title: a.title, excerpt: a.excerpt ?? "", category: a.categorySlug ?? "", categoryName: catNames.get(a.categorySlug ?? "") ?? a.categorySlug ?? "", kind: a.kind, author: a.author?.name ?? "", authorBio: a.author?.bio ?? "",
    date: String(a.publishedAt).slice(0, 10), readMin: a.readMin ?? 0, image: a.imageUrl ?? "", tags: a.tags ?? [], products: a.productIds ?? [], guide: a.guide,
    dossier: Boolean(a.dossier), featured: Boolean(a.featured), sections: parseBody(a.body ?? ""),
  };
}

/** Tous les articles publiés, du plus récent au plus ancien. @returns {Promise<BlogArticle[]>} */
export const fetchBlogArticles = () =>
  cached("blog:articles", async () => {
    const names = new Map((await fetchBlogCategories().catch(() => [])).map((c) => [c.slug, c.name]));
    const out = [];
    for (let page = 1; page < 50; page++) {
      const d = await getJson(`/blog/articles?page=${page}&pageSize=100`);
      out.push(...d.items.map((/** @type {any} */ a) => adaptArticle(a, names)));
      if (page >= d.page.totalPages) break;
    }
    return out;
  });

/** @returns {Promise<{ slug: string, name: string }[]>} */
export const fetchBlogCategories = () => cached("blog:categories", () => getJson("/blog/categories"));
/** @returns {Promise<{ slug: string, name: string }[]>} */
export const fetchBlogTags = () => cached("blog:tags", () => getJson("/blog/tags"));
