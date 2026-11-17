// Contenu pédagogique de la boutique, séparé de la logique. Les textes vivent dans `guide/i18n/{en,fr,es}.js`
// sous les clés `guide.<visite>.<étape>.title|text`. Une cible absente (mobile, panier vide…) est simplement sautée.
// Ajouter une page = une entrée ici + ses textes dans les trois langues.

const step = (tour, id, o = {}) => ({ id, titleKey: `guide.${tour}.${id}.title`, textKey: `guide.${tour}.${id}.text`, ...o });
const help = { id: "help", target: "guide-help", placement: "bottom", titleKey: "guide.common.help.title", textKey: "guide.common.help.text" };

const tour = (id, path, match, steps, extra = {}) => ({ id, path, match, steps: steps.map((s) => (s.titleKey ? s : step(id, s.id, s))), ...extra });
const is = (p) => (pathname) => pathname === p;

export const TOURS = [
  tour("home", "/", is("/"), [
    { id: "intro" },
    { id: "search", target: "hero-search", placement: "bottom" },
    { id: "language", target: "language", placement: "bottom" },
    { id: "cart", target: "cart", placement: "bottom" },
    { id: "categories", target: "home-categories", placement: "top" },
    { id: "sectors", target: "home-sectors", placement: "top" },
    help,
  ]),
  tour("category", "/category", is("/category"), [
    { id: "intro" },
    { id: "filters", target: "catalog-filters", placement: "right" },
    { id: "results", target: "catalog-list", placement: "top" },
    help,
  ]),
  // Pas de lancement depuis le profil : la fiche produit a besoin d'un produit (chemin volontairement différent).
  tour("product", "/category", is("/card"), [
    { id: "anatomy", scene: "filter-anatomy" },
    { id: "gallery", target: "product-gallery", placement: "right" },
    { id: "buy", target: "product-buy", placement: "left" },
    { id: "details", target: "product-details", placement: "top" },
    { id: "reviews", target: "product-reviews", placement: "top" },
    { id: "related", target: "product-related", placement: "top" },
    help,
  ]),
  tour("basket", "/basket", is("/basket"), [
    { id: "lines", target: "cart-lines", placement: "bottom" },
    { id: "shipping", target: "cart-shipping", placement: "top" },
    { id: "promo", target: "cart-promo", placement: "top" },
    { id: "summary", target: "cart-summary", placement: "left" },
    { id: "saved", target: "cart-saved", placement: "top" },
    help,
  ]),
  tour("quote", "/quote", is("/quote"), [
    { id: "intro" },
    { id: "contact", target: "quote-contact", placement: "bottom" },
    { id: "items", target: "quote-items", placement: "top" },
    { id: "aside", target: "quote-aside", placement: "left" },
    help,
  ]),
  tour("contact", "/contact", is("/contact"), [
    { id: "form", target: "contact-form", placement: "right" },
    { id: "details", target: "contact-details", placement: "left" },
    help,
  ]),
  tour("profile", "/profil", is("/profil"), [
    { id: "info", target: "profile-info", placement: "bottom" },
    { id: "orders", target: "profile-orders", placement: "top" },
    { id: "guide", target: "profile-guide", placement: "top" },
    help,
  ]),
  tour("blog", "/blog", is("/blog"), [
    { id: "intro" },
    { id: "nav", target: "blog-nav", placement: "bottom" },
    { id: "latest", target: "blog-latest", placement: "top" },
    help,
  ]),
  tour("post", "/blog", (p) => /^\/blog\/[^/]+$/.test(p), [
    { id: "breadcrumb", target: "post-breadcrumb", placement: "bottom" },
    { id: "toc", target: "post-toc", placement: "right" },
    { id: "related", target: "post-related", placement: "top" },
    help,
  ]),
  tour("brands", "/brands", is("/brands"), [
    { id: "list", target: "brands-list", placement: "top" },
    help,
  ]),
  tour("legal", "/terms", (p) => p === "/terms" || p === "/refund" || p === "/privacy", [
    { id: "intro" },
    help,
  ]),
];
