import type { GuideTour } from "../types";
import { action, danger, filters, form, header, help, intro, pagination, submit, table } from "./shared";

export const productsTour: GuideTour = {
  id: "products", title: "Produits", summary: "Le catalogue : chercher, filtrer et suivre l'état de publication.", path: "/products", permission: "catalog:read",
  match: (p) => p === "/products",
  steps: [
    intro("Le catalogue produits", "Un produit n'est visible sur le site qu'une fois publié. Cette page montre où en est chaque fiche."),
    action("Ajouter un produit", "Démarre la création d'une fiche. Le bouton n'apparaît que si vous avez le droit d'écrire dans le catalogue."),
    filters("Combinez SKU, nom ou référence avec le statut, la catégorie et la marque. Idéal pour retrouver tous les produits « À valider »."),
    { ...table("Statuts et prix", "Brouillon, À valider, Publié, Archivé : le badge de la dernière colonne indique l'étape. « Sur devis » signale un produit sans prix affiché."), scene: "product-lifecycle" },
    pagination, help,
  ],
};

export const productNewTour: GuideTour = {
  id: "product-new", title: "Nouveau produit", summary: "Créer une fiche en deux étapes.", path: "/products/new", permission: "catalog:read",
  match: (p) => p === "/products/new",
  steps: [
    header("La création se fait en deux étapes : d'abord la catégorie, puis la fiche complète.", "Deux étapes"),
    form("Le formulaire", "Étape 1 : choisissez la catégorie, car ses attributs (et ceux de ses parents) déterminent les champs de la fiche. Étape 2 : remplissez-la."),
    submit("Le produit est créé en brouillon. Il ne sera publié que lorsque sa fiche sera complète et validée."),
    help,
  ],
};

export const productDetailTour: GuideTour = {
  id: "product-detail", title: "Fiche produit", summary: "Cycle de vie, modification, historique.", path: "/products", permission: "catalog:read",
  match: (p) => /^\/products\/(?!(?:new|bulk)$)[^/]+$/.test(p),
  steps: [
    header("Le nom du produit et son statut actuel.", "La fiche"),
    { id: "life", scene: "product-lifecycle", target: "product-lifecycle", placement: "bottom", title: "Le cycle de vie", text: "Brouillon → À valider → Publié → Archivé. Soumettre à validation passe la fiche en relecture ; publier la rend visible sur le site ; archiver la retire sans la perdre." },
    { id: "sheet", target: "product-sheet", placement: "top", title: "La fiche", text: "Prix, références, dimensions, attributs de la catégorie, secteurs et références croisées. Chaque enregistrement crée une version." },
    { id: "history", target: "product-history", placement: "top", title: "L'historique", text: "Chaque modification est listée avec son auteur et les champs changés. Rien n'est perdu : on peut toujours savoir qui a changé quoi." },
    danger("Un produit jamais publié peut être supprimé ; un produit déjà publié est archivé, pour garder la trace de ses ventes."),
    help,
  ],
};

export const categoriesTour: GuideTour = {
  id: "categories", title: "Catégories", summary: "L'arbre du catalogue et l'héritage des attributs.", path: "/categories", permission: "catalog:read",
  match: (p) => p === "/categories",
  steps: [
    { ...intro("L'arbre des catégories", "Les catégories s'emboîtent. Une sous-catégorie hérite des attributs de ses parents : définissez une caractéristique une fois, au bon niveau."), scene: "category-tree" },
    action("Ajouter une catégorie", "Crée une catégorie à la racine ; vous pourrez la placer sous une autre depuis sa fiche."),
    table("L'arbre", "Le retrait et le symbole └ montrent la profondeur. La colonne Produits compte les produits rattachés."),
    help,
  ],
};

export const categoryNewTour: GuideTour = {
  id: "category-new", title: "Nouvelle catégorie", summary: "Créer une catégorie.", path: "/categories/new", permission: "catalog:read",
  match: (p) => p === "/categories/new",
  steps: [
    header("Le nom existe dans chaque langue du site ; l'identifiant d'URL sert à l'adresse de la page.", "Créer une catégorie"),
    form("Les champs", "Le parent, s'il y en a un, détermine les attributs que la catégorie héritera."),
    submit("Après la création, ajoutez ses attributs depuis sa fiche."),
    help,
  ],
};

export const categoryDetailTour: GuideTour = {
  id: "category-detail", title: "Fiche catégorie", summary: "Informations, déplacement, attributs hérités.", path: "/categories", permission: "catalog:read",
  match: (p) => /^\/categories\/(?!(?:new|bulk)$)[^/]+$/.test(p),
  steps: [
    header("Le chemin complet de la catégorie dans l'arbre.", "Où suis-je ?"),
    { id: "info", target: "category-info", placement: "bottom", title: "Informations", text: "Nom, description et identifiant d'URL, dans chaque langue." },
    { id: "move", target: "category-move", placement: "top", title: "Déplacer", text: "Rattache la catégorie à un autre parent. Ses attributs hérités changent en conséquence." },
    { id: "children", target: "category-children", placement: "top", title: "Sous-catégories", text: "Les catégories directement en dessous. Cliquez pour descendre dans l'arbre." },
    { id: "own", target: "category-attributes", placement: "top", title: "Attributs propres", text: "Les caractéristiques de cette catégorie (type, unité, filtre, obligatoire). Elles se retrouveront dans la fiche de ses produits et de ses sous-catégories." },
    { id: "inherited", scene: "category-tree", target: "category-inherited", placement: "top", title: "Attributs hérités", text: "Venus des catégories parentes. Ils se modifient chez le parent, pas ici." },
    danger("La suppression est impossible tant que la catégorie a des sous-catégories, des attributs ou des produits."),
    help,
  ],
};

export const attributeTour: GuideTour = {
  id: "attribute", title: "Attributs", summary: "Définir une caractéristique de produit.", path: "/categories", permission: "catalog:read",
  match: (p) => p === "/attributes/new" || /^\/attributes\/[^/]+$/.test(p),
  steps: [
    header("Un attribut appartient à une catégorie ; ses sous-catégories en héritent.", "Un attribut"),
    form("Le formulaire", "Choisissez le type (texte, nombre, oui/non, liste de choix), l'unité, et si l'attribut est filtrable ou obligatoire. Les attributs filtrables alimentent les filtres du site."),
    submit("L'attribut apparaît aussitôt dans les fiches produit de la catégorie."),
    danger("La suppression d'un attribut demande une confirmation."),
    help,
  ],
};

const listOf = (id: string, title: string, noun: string, path: string): GuideTour => ({
  id, title, summary: `Gérer les ${noun}.`, path, permission: "catalog:read", match: (p) => p === path,
  steps: [
    intro(title, `Les ${noun} servent à classer les produits et à alimenter les filtres du site.`),
    action("Ajouter", `Crée une entrée. Le bouton n'apparaît que si vous avez le droit d'écrire dans le catalogue.`),
    filters("Recherche par nom."),
    table("La liste", "L'identifiant d'URL est la partie de l'adresse du site. « Ouvrir » mène à la fiche pour modifier ou supprimer."),
    pagination, help,
  ],
});
export const brandsTour = listOf("brands", "Marques", "marques", "/brands");
export const sectorsTour = listOf("sectors", "Secteurs", "secteurs d'activité", "/sectors");

const formOf = (id: string, title: string, base: string, what: string): GuideTour => ({
  id, title, summary: `Créer ou modifier ${what}.`, path: base, permission: "catalog:read",
  match: (p) => p === `${base}/new` || new RegExp(`^${base}/(?!new$)[^/]+$`).test(p),
  steps: [
    header(`Créer ou modifier ${what}.`, title),
    form("Le formulaire", "Renseignez les champs puis enregistrez. Un message vous indique tout de suite si quelque chose est refusé, sans perdre votre saisie."),
    submit("Les changements sont enregistrés immédiatement."),
    danger("La suppression demande une confirmation."),
    help,
  ],
});
export const brandFormTour = formOf("brand-form", "Fiche marque", "/brands", "une marque");
export const sectorFormTour = formOf("sector-form", "Fiche secteur", "/sectors", "un secteur");

export const categoriesBulkTour: GuideTour = {
  id: "categories-bulk", title: "Catégories en masse", summary: "Créer et modifier plusieurs catégories dans un tableur.", path: "/categories/bulk", permission: "catalog:write",
  match: (p) => p === "/categories/bulk",
  steps: [
    { id: "grid", target: "bulk-grid", placement: "top", title: "Un tableur, comme Excel", text: "Cliquez une cellule et tapez, collez des lignes copiées depuis Excel (Ctrl/⌘+V), tirez la poignée pour recopier. Jaune : modifiée · bleu : nouvelle." },
    { id: "toolbar", target: "bulk-toolbar", placement: "bottom", title: "Enregistrer, importer, exporter", text: "« Enregistrer » envoie toutes les lignes modifiées d'un coup ; la colonne « État » dit ce qui a réussi ou pas. Importez un classeur .xlsx (la première ligne porte les titres) ou exportez la grille." },
    { id: "delete", target: "bulk-grid", placement: "top", title: "Supprimer", text: "Cochez « Supprimer » sur une ligne existante : elle est supprimée à l'enregistrement, après confirmation." },
    help,
  ],
};

export const productsBulkTour: GuideTour = {
  id: "products-bulk", title: "Produits en masse", summary: "Créer et modifier plusieurs produits dans un tableur.", path: "/products/bulk", permission: "catalog:write",
  match: (p) => p === "/products/bulk",
  steps: [
    { id: "grid", target: "bulk-grid", placement: "top", title: "Un tableur, comme Excel", text: "Cliquez une cellule et tapez, collez des lignes copiées depuis Excel (Ctrl/⌘+V), tirez la poignée pour recopier. Jaune : modifiée · bleu : nouvelle." },
    { id: "toolbar", target: "bulk-toolbar", placement: "bottom", title: "Enregistrer, importer, exporter", text: "« Enregistrer » envoie toutes les lignes modifiées d'un coup ; la colonne « État » dit ce qui a réussi ou pas. Importez un classeur .xlsx (la première ligne porte les titres) ou exportez la grille." },
    { id: "delete", target: "bulk-grid", placement: "top", title: "Supprimer", text: "Cochez « Supprimer » sur une ligne existante : elle est supprimée à l'enregistrement, après confirmation." },
    help,
  ],
};
