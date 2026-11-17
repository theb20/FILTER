import { describe, expect, it } from "vitest";
import { BLOG_KINDS } from "../data/blog.js";
import { FIXTURE_BLOG_ARTICLES as BLOG_ARTICLES, FIXTURE_BLOG_CATEGORIES as BLOG_CATEGORIES, FIXTURE_BLOG_TAGS as BLOG_TAGS } from "./__fixtures__/fixtures.js";
import { buildBlogQuery, filterArticles, paginate, parseBlogParams, pickArticles, relatedArticles, sortByDateDesc } from "./blogFilters.js";

const allowed = { categories: BLOG_CATEGORIES, kinds: BLOG_KINDS, tags: BLOG_TAGS.map((t) => t.slug) };
const sp = (q) => new URLSearchParams(q);

describe("parseBlogParams", () => {
  it("lit des paramètres valides", () => {
    expect(parseBlogParams(sp("cat=oil&kind=advice&tag=stockage&page=2"), allowed)).toEqual({ cat: "oil", kind: "advice", tag: "stockage", page: 2 });
  });
  it("ignore les valeurs inconnues et les pages invalides", () => {
    expect(parseBlogParams(sp("cat=nope&kind=x&tag=y&page=-3"), allowed)).toEqual({ cat: null, kind: null, tag: null, page: 1 });
    expect(parseBlogParams(sp("page=abc"), allowed).page).toBe(1);
  });
});

describe("buildBlogQuery", () => {
  const cur = { cat: "oil", kind: null, tag: null, page: 3 };
  it("repasse à la page 1 quand un filtre change", () => expect(buildBlogQuery(cur, { kind: "advice" })).toBe("?cat=oil&kind=advice"));
  it("conserve la page quand on la change explicitement", () => expect(buildBlogQuery(cur, { page: 2 })).toBe("?cat=oil&page=2"));
  it("renvoie une chaîne vide sans filtre", () => expect(buildBlogQuery(cur, { cat: null })).toBe(""));
});

describe("filtrage", () => {
  it("sans filtre : tous les articles", () => expect(filterArticles(BLOG_ARTICLES, { cat: null, kind: null, tag: null })).toHaveLength(BLOG_ARTICLES.length));
  it("filtre par catégorie", () => {
    const r = filterArticles(BLOG_ARTICLES, { cat: "oil", kind: null, tag: null });
    expect(r.length).toBeGreaterThan(0);
    expect(r.every((a) => a.category === "oil")).toBe(true);
  });
  it("combine catégorie, type et étiquette", () => {
    const r = filterArticles(BLOG_ARTICLES, { cat: "maintenance", kind: "advice", tag: "stockage" });
    expect(r.every((a) => a.category === "maintenance" && a.kind === "advice" && a.tags.includes("stockage"))).toBe(true);
  });
  it("peut ne rien renvoyer", () => expect(filterArticles(BLOG_ARTICLES, { cat: "fuel", kind: "news", tag: null })).toEqual([]));
  it("trie du plus récent au plus ancien", () => {
    const dates = sortByDateDesc(BLOG_ARTICLES).map((a) => a.date);
    expect(dates).toEqual([...dates].sort().reverse());
  });
});

describe("paginate", () => {
  const items = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11];
  it("découpe en pages", () => {
    expect(paginate(items, 1, 5)).toMatchObject({ items: [1, 2, 3, 4, 5], page: 1, pages: 3, total: 11 });
    expect(paginate(items, 3, 5).items).toEqual([11]);
  });
  it("ramène la page dans les bornes", () => {
    expect(paginate(items, 99, 5).page).toBe(3);
    expect(paginate(items, 0, 5).page).toBe(1);
  });
  it("liste vide : 1 page", () => expect(paginate([], 1, 5)).toMatchObject({ items: [], pages: 1, page: 1 }));
});

describe("données et liens", () => {
  it("les slugs sont uniques", () => expect(new Set(BLOG_ARTICLES.map((a) => a.slug)).size).toBe(BLOG_ARTICLES.length));
  it("les articles liés excluent l'article courant", () => {
    const a = BLOG_ARTICLES[0];
    const r = relatedArticles(a, BLOG_ARTICLES, 3);
    expect(r.length).toBeLessThanOrEqual(3);
    expect(r.some((x) => x.slug === a.slug)).toBe(false);
  });
  it("pickArticles ignore les slugs inconnus", () => expect(pickArticles(BLOG_ARTICLES, [BLOG_ARTICLES[0].slug, "inconnu"])).toHaveLength(1));
});

import { adjacentArticles } from "./blogFilters.js";
describe("adjacentArticles", () => {
  const sorted = sortByDateDesc(BLOG_ARTICLES);
  it("le plus récent n'a pas de suivant", () => expect(adjacentArticles(sorted[0], BLOG_ARTICLES)).toEqual({ newer: null, older: sorted[1] }));
  it("le plus ancien n'a pas de précédent", () => expect(adjacentArticles(sorted.at(-1), BLOG_ARTICLES).older).toBeNull());
  it("un article intermédiaire a les deux voisins", () => expect(adjacentArticles(sorted[3], BLOG_ARTICLES)).toEqual({ newer: sorted[2], older: sorted[4] }));
  it("slug inconnu : aucun voisin", () => expect(adjacentArticles({ slug: "x" }, BLOG_ARTICLES)).toEqual({ newer: null, older: null }));
});
