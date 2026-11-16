// @ts-check
// ─────────────────────────────────────────────────────────────────────────────
// DONNÉES DE DÉMONSTRATION DU BLOG — articles, auteurs, dates et contenus FICTIFS.
// Seul fichier qui simule le backend du blog : à remplacer par l'API (voir fetchBlogArticles).
// Les articles sont rédigés en français uniquement ; l'interface, elle, est traduite.
// Les images sont des visuels de démonstration déjà présents dans public/img/background.
// ─────────────────────────────────────────────────────────────────────────────

/**
 * @typedef {"hydraulic" | "oil" | "air" | "fuel" | "maintenance" | "guides" | "news"} BlogCategory
 * @typedef {"advice" | "comparison" | "news"} BlogKind
 * @typedef {{ id: string, heading: string, paragraphs: string[] }} BlogSection
 * @typedef {{
 *   slug: string, title: string, excerpt: string, category: BlogCategory, kind: BlogKind,
 *   author: string, date: string, readMin: number, image: string, tags: string[],
 *   products: number[], guide?: "steps" | "checklist", dossier?: boolean, sections: BlogSection[],
 * }} BlogArticle
 */

/** Catégories (onglets). @type {BlogCategory[]} */
export const BLOG_CATEGORIES = ["hydraulic", "oil", "air", "fuel", "maintenance", "guides", "news"];
/** Types d'article (pastilles). @type {BlogKind[]} */
export const BLOG_KINDS = ["advice", "comparison", "news"];

const IMG = (n) => `/img/background/bg_card_${n}.png`;

/** Texte de démonstration commun (le même squelette sert à tous les articles fictifs). */
const demoSections = (topic) => [
  {
    id: "contexte",
    heading: "Pourquoi c'est important",
    paragraphs: [
      `Contenu fictif de démonstration sur le thème « ${topic} ». Ce paragraphe illustre la mise en page d'un article : il ne contient aucune information technique réelle.`,
      "Avant toute intervention, référez-vous toujours aux préconisations du constructeur de votre machine.",
    ],
  },
  {
    id: "methode",
    heading: "La méthode, étape par étape",
    paragraphs: [
      "Deuxième paragraphe fictif. Dans la version définitive, cette section détaillera la démarche à suivre, avec les points de contrôle et les outils nécessaires.",
      "Un élément de liste, un schéma ou un tableau comparatif pourront venir enrichir cette partie.",
    ],
  },
  {
    id: "conclusion",
    heading: "À retenir",
    paragraphs: ["Conclusion fictive : résumé des points clés et renvoi vers les produits cités ci-dessous."],
  },
];

/** @type {BlogArticle[]} */
export const BLOG_ARTICLES = [
  { slug: "quand-changer-filtre-hydraulique", title: "Quand changer un filtre hydraulique ?", excerpt: "Les signes qui ne trompent pas et les intervalles de remplacement à surveiller pour protéger vos circuits.", category: "hydraulic", kind: "advice", author: "Équipe technique (démo)", date: "2026-11-10", readMin: 6, image: IMG(1), tags: ["engins-chantier", "stockage"], products: [1, 2, 8, 6], sections: demoSections("changer un filtre hydraulique") },
  { slug: "huile-moteur-comparatif-filtres", title: "Filtres à huile : le comparatif de la rédaction", excerpt: "Cartouche ou à visser, média cellulose ou synthétique : ce qui change vraiment pour votre moteur.", category: "oil", kind: "comparison", author: "Aminata K. (démo)", date: "2026-11-06", readMin: 8, image: IMG(3), tags: ["camions", "equivalences"], products: [3, 4, 7, 5], sections: demoSections("comparatif de filtres à huile") },
  { slug: "filtre-air-compresseur-entretien", title: "Entretenir le filtre à air d'un compresseur à vis", excerpt: "Un entretien régulier du filtre à air évite l'encrassement et préserve le rendement de votre compresseur.", category: "air", kind: "advice", author: "Équipe technique (démo)", date: "2026-11-02", readMin: 5, image: IMG(5), tags: ["compresseurs", "securite"], products: [5, 6, 7, 3], sections: demoSections("filtre à air de compresseur") },
  { slug: "gasoil-qualite-filtration", title: "Qualité du gasoil : pourquoi la filtration fait la différence", excerpt: "Eau, particules, impuretés : comment un bon préfiltre protège votre système d'injection.", category: "fuel", kind: "advice", author: "Yao B. (démo)", date: "2026-10-28", readMin: 7, image: IMG(4), tags: ["groupes-electrogenes", "stockage"], products: [3, 4, 7, 8], sections: demoSections("qualité du gasoil") },
  { slug: "checklist-entretien-groupe-electrogene", title: "Check-list d'entretien d'un groupe électrogène", excerpt: "Tous les points de contrôle à valider avant la saison de forte charge, filtre par filtre.", category: "maintenance", author: "Équipe technique (démo)", date: "2026-10-24", readMin: 4, image: IMG(6), tags: ["groupes-electrogenes", "securite"], products: [3, 5, 7, 1], guide: "checklist", sections: demoSections("entretien d'un groupe électrogène") },
  { slug: "guide-achat-filtres-engins", title: "Guide d'achat : choisir ses filtres pour engins de chantier", excerpt: "Références, équivalences, qualité : les critères pour acheter malin sans risquer sa garantie.", category: "guides", kind: "advice", author: "Aminata K. (démo)", date: "2026-10-20", readMin: 9, image: IMG(2), tags: ["engins-chantier", "equivalences"], products: [1, 2, 8, 3], dossier: true, sections: demoSections("choisir ses filtres pour engins") },
  { slug: "reglementation-filtration-actualite", title: "Actualité : ce qui change dans la filtration industrielle", excerpt: "Tour d'horizon (fictif) des évolutions du marché et des normes que les professionnels doivent suivre.", category: "news", kind: "news", author: "Rédaction (démo)", date: "2026-10-15", readMin: 3, image: IMG(7), tags: ["securite"], products: [6, 8, 2, 1], sections: demoSections("actualité de la filtration") },
  { slug: "comparatif-marques-equivalences", title: "Équivalences de marques : comment s'y retrouver", excerpt: "Une référence introuvable ? Les bonnes pratiques pour identifier un filtre équivalent fiable.", category: "guides", kind: "comparison", author: "Yao B. (démo)", date: "2026-10-09", readMin: 6, image: IMG(8), tags: ["equivalences", "engins-chantier"], products: [2, 4, 5, 7], sections: demoSections("équivalences de marques") },
  { slug: "pas-a-pas-vidange-hydraulique", title: "Pas à pas : réaliser une vidange hydraulique", excerpt: "La procédure complète, de la préparation au remplissage, pour une vidange propre et sûre.", category: "maintenance", kind: "advice", author: "Équipe technique (démo)", date: "2026-10-03", readMin: 10, image: IMG(9), tags: ["engins-chantier", "securite"], products: [1, 2, 8, 6], guide: "steps", dossier: true, sections: demoSections("vidange hydraulique") },
  { slug: "stocker-ses-filtres", title: "Stocker ses filtres de rechange sans les abîmer", excerpt: "Température, humidité, emballage : les bons réflexes pour garder vos pièces en parfait état.", category: "maintenance", kind: "advice", author: "Aminata K. (démo)", date: "2026-09-28", readMin: 4, image: IMG(5), tags: ["stockage"], products: [7, 3, 5, 1], guide: "checklist", sections: demoSections("stockage des filtres") },
  { slug: "pas-a-pas-filtre-air-tracteur", title: "Pas à pas : remplacer le filtre à air d'un tracteur", excerpt: "Un remplacement simple en quelques minutes, à condition de suivre les bonnes étapes.", category: "air", kind: "advice", author: "Yao B. (démo)", date: "2026-09-21", readMin: 5, image: IMG(3), tags: ["machines-agricoles"], products: [5, 7, 3, 2], guide: "steps", sections: demoSections("filtre à air de tracteur") },
  { slug: "dossier-filtration-fine-hydraulique", title: "Dossier : la filtration fine des circuits hydrauliques", excerpt: "Pourquoi le micronnage conditionne la durée de vie des pompes et des distributeurs.", category: "hydraulic", kind: "comparison", author: "Équipe technique (démo)", date: "2026-09-14", readMin: 12, image: IMG(6), tags: ["engins-chantier", "compresseurs"], products: [1, 6, 8, 2], dossier: true, sections: demoSections("filtration fine hydraulique") },
  { slug: "poids-lourds-huile-intervalles", title: "Poids lourds : les intervalles de vidange en question", excerpt: "Conditions d'utilisation, qualité d'huile, type de filtre : adapter ses intervalles à la réalité du terrain.", category: "oil", kind: "advice", author: "Yao B. (démo)", date: "2026-09-07", readMin: 7, image: IMG(1), tags: ["camions"], products: [3, 4, 7, 8], sections: demoSections("intervalles de vidange poids lourds") },
  { slug: "actualite-pieces-disponibilite", title: "Actualité : disponibilité des pièces d'ici la fin d'année", excerpt: "Les références les plus demandées et les délais à anticiper pour vos commandes.", category: "news", kind: "news", author: "Rédaction (démo)", date: "2026-08-30", readMin: 3, image: IMG(8), tags: ["stockage", "equivalences"], products: [7, 3, 5, 1], sections: demoSections("disponibilité des pièces") },
];

/** Étiquettes « Explorer par sujet » (libellés en français : ce sont des contenus). */
export const BLOG_TAGS = [
  { slug: "engins-chantier", label: "Engins de chantier" },
  { slug: "camions", label: "Camions et poids lourds" },
  { slug: "groupes-electrogenes", label: "Groupes électrogènes" },
  { slug: "machines-agricoles", label: "Machines agricoles" },
  { slug: "compresseurs", label: "Compresseurs" },
  { slug: "equivalences", label: "Équivalences de marques" },
  { slug: "stockage", label: "Stockage" },
  { slug: "securite", label: "Sécurité" },
];

/** Sélections éditoriales (slugs d'articles). */
export const BLOG_HEADLINE = "quand-changer-filtre-hydraulique";
export const BLOG_MOSAIC = ["quand-changer-filtre-hydraulique", "huile-moteur-comparatif-filtres", "filtre-air-compresseur-entretien", "gasoil-qualite-filtration"];
export const BLOG_TOPICS = ["guide-achat-filtres-engins", "stocker-ses-filtres", "comparatif-marques-equivalences", "poids-lourds-huile-intervalles"];
export const BLOG_GUIDES = ["checklist-entretien-groupe-electrogene", "pas-a-pas-vidange-hydraulique", "stocker-ses-filtres", "pas-a-pas-filtre-air-tracteur"];
export const BLOG_DOSSIERS = ["dossier-filtration-fine-hydraulique", "pas-a-pas-vidange-hydraulique", "guide-achat-filtres-engins"];
export const BLOG_POPULAR = ["huile-moteur-comparatif-filtres", "quand-changer-filtre-hydraulique", "guide-achat-filtres-engins", "checklist-entretien-groupe-electrogene"];
/** Produits « cités dans nos articles » (ids de data/demoProducts.js). */
export const BLOG_CITED_PRODUCTS = [1, 3, 5, 8];

/**
 * POINT D'ENTRÉE API : remplacer par un vrai appel (ex. axios.get("/api/blog/articles")).
 * La page gère déjà les états chargement / erreur autour de cette promesse.
 * @returns {Promise<BlogArticle[]>}
 */
export function fetchBlogArticles() {
  return new Promise((resolve) => setTimeout(() => resolve(BLOG_ARTICLES), 250));
}

/** Biographie fictive des auteurs (démonstration). */
export const BLOG_AUTHOR_BIO = "Auteur fictif de démonstration : cette biographie illustre le bloc « À propos de l'auteur » et sera remplacée par les vraies fiches des rédacteurs.";
