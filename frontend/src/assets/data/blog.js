// @ts-check
// ─────────────────────────────────────────────────────────────────────────────
// BLOG : AUCUN article local. Les articles, auteurs, catégories et étiquettes viendront du service content.
// Tant qu'il n'existe pas, les collections restent vides et la page affiche un état vide honnête.
// ─────────────────────────────────────────────────────────────────────────────

/**
 * @typedef {string} BlogCategory
 * @typedef {"advice" | "comparison" | "news"} BlogKind
 * @typedef {{ id: string, heading: string, paragraphs: string[] }} BlogSection
 * @typedef {{
 *   slug: string, title: string, excerpt: string, category: BlogCategory, kind: BlogKind,
 *   author: string, date: string, readMin: number, image: string, tags: string[],
 *   products: number[], guide?: "steps" | "checklist", dossier?: boolean, sections: BlogSection[],
 * }} BlogArticle
 */

/** Types d'article (pastilles) : structure de l'interface, pas des données. @type {BlogKind[]} */
export const BLOG_KINDS = ["advice", "comparison", "news"];

/** @type {BlogCategory[]} */
export const BLOG_CATEGORIES = [];
/** @type {BlogArticle[]} */
export const BLOG_ARTICLES = [];
/** @type {string[]} */
export const BLOG_TAGS = [];
/** @type {string | null} */
export const BLOG_HEADLINE = null;
/** @type {string[]} */ export const BLOG_MOSAIC = [];
/** @type {string[]} */ export const BLOG_TOPICS = [];
/** @type {string[]} */ export const BLOG_GUIDES = [];
/** @type {string[]} */ export const BLOG_DOSSIERS = [];
/** @type {string[]} */ export const BLOG_POPULAR = [];
/** @type {number[]} */ export const BLOG_CITED_PRODUCTS = [];
/** @type {string | null} */
export const BLOG_AUTHOR_BIO = null;

/** Articles publiés. @returns {Promise<BlogArticle[]>} */
export async function fetchBlogArticles() {
  return BLOG_ARTICLES;
}
