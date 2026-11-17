// @ts-check
// BLOG : aucun article local. Articles, catégories, étiquettes et auteurs viennent de l'API (lib/content.js) et se gèrent
// au backoffice. Seuls les TYPES d'article (pastilles de filtre) sont une structure de l'interface.

/** @typedef {import("../lib/content.js").BlogArticle} BlogArticle */
/** @typedef {"advice" | "comparison" | "news"} BlogKind */

/** @type {BlogKind[]} */
export const BLOG_KINDS = ["advice", "comparison", "news"];
