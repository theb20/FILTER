// FIXTURES DE TEST UNIQUEMENT — jamais importées par l'application (les données réelles viendront de l'API).
// Valeurs volontairement neutres et génériques : elles ne prétendent à aucune réalité commerciale.

export const FIXTURE_PRODUCTS = [
  { id: 1, brand: "Marque 1", reference: "REF-1", category: "cat-a", price: 1000, inStock: true, crossRefs: [] },
  { id: 2, brand: "Marque 1", reference: "REF-2", category: "cat-a", price: 2000, inStock: true, crossRefs: [] },
  { id: 3, brand: "Marque 2", reference: "REF-3", category: "cat-b", price: 3000, inStock: true, crossRefs: [] },
  { id: 4, brand: "Marque 2", reference: "REF-4", category: "cat-b", price: 4000, inStock: false, crossRefs: [] },
  { id: 5, brand: "Marque 3", reference: "REF-5", category: "cat-c", price: 5000, inStock: true, crossRefs: [] },
  { id: 6, brand: "Marque 3", reference: "REF-6", category: "cat-c", price: 6000, inStock: true, crossRefs: [] },
];

export const FIXTURE_REVIEWS = [
  { id: "r1", rating: 5 }, { id: "r2", rating: 4 }, { id: "r3", rating: 4 }, { id: "r4", rating: 1 },
];

export const FIXTURE_BLOG_CATEGORIES = ["oil", "maintenance", "fuel"];
export const FIXTURE_BLOG_TAGS = [{ slug: "stockage", label: "Stockage" }, { slug: "choix", label: "Choix" }];

const article = (n, category, kind, date, tags) => ({
  slug: `article-${n}`, title: `Article ${n}`, excerpt: "", category, kind, author: "Auteur", date, readMin: 1, image: "", tags, products: [], sections: [],
});

export const FIXTURE_BLOG_ARTICLES = [
  article(1, "oil", "advice", "2026-06-01", ["stockage"]),
  article(2, "oil", "comparison", "2026-05-01", ["choix"]),
  article(3, "maintenance", "advice", "2026-04-01", ["stockage"]),
  article(4, "maintenance", "news", "2026-03-01", []),
  article(5, "fuel", "advice", "2026-02-01", ["choix"]),
  article(6, "fuel", "comparison", "2026-01-01", []),
];
