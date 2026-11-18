import type { GuideStep, GuideTour } from "../types";
import { action, danger, filters, form, header, help, intro, pagination, submit, table } from "./shared";

// Section « Boutique », « Contenu du site » et « Messages reçus » : des collections décrites par `lib/content-kinds.ts`.
// Les pages ont toutes la même forme (liste, création, fiche) ; seuls les textes changent, d'où ces constructeurs.

type Kind = {
  slug: string; title: string; perm: string;
  /** À quoi sert la collection. */
  about: string;
  /** Ce qu'il faut savoir en remplissant la fiche. */
  fields: string;
  /** Collection reçue (messages, abonnés, demandes) : on ne crée rien, on traite. */
  received?: boolean;
};

const listTour = (k: Kind): GuideTour => ({
  id: `content-${k.slug}`, title: k.title, summary: k.about, path: `/content/${k.slug}`, permission: k.perm,
  match: (p) => p === `/content/${k.slug}`,
  steps: [
    intro(k.title, k.about),
    ...(k.received ? [] : [action("Ajouter", "Crée un nouvel élément. Le bouton n'apparaît que si vous avez le droit d'écrire dans cette section.")]),
    filters("Recherche dans la liste. « Réinitialiser » efface la recherche."),
    table("La liste", k.received ? "Un clic sur la première colonne ouvre l'élément pour le lire et le traiter. Le badge indique son statut." : "Un clic sur la première colonne ouvre la fiche pour la modifier. « Modifié le » montre la dernière mise à jour."),
    pagination, help,
  ],
});

const formTour = (k: Kind): GuideTour => ({
  id: `content-${k.slug}-form`, title: `Fiche : ${k.title}`, summary: k.fields, path: `/content/${k.slug}`, permission: k.perm,
  match: (p) => p === `/content/${k.slug}/new` || new RegExp(`^/content/${k.slug}/(?!new$)[^/]+$`).test(p),
  steps: [
    header(k.received ? "L'élément reçu, avec sa date de réception." : "Créer ou modifier un élément.", k.received ? "L'élément" : "La fiche"),
    { id: "publish", target: "content-publish", placement: "bottom", title: "Publication", text: "Seuls les contenus publiés apparaissent sur le site. Publier exige la permission « Publier — contenu » ; sans elle, vous pouvez rédiger mais pas mettre en ligne." },
    { id: "received", target: "content-received", placement: "bottom", title: "Contenu reçu", text: "Ce que le visiteur a envoyé. Ces champs sont en lecture seule." },
    form(k.received ? "Traitement" : "Le formulaire", k.fields),
    submit("Les changements sont enregistrés immédiatement."),
    danger("La suppression est définitive et demande une confirmation."),
    help,
  ] satisfies GuideStep[],
});

const KINDS: Kind[] = [
  { slug: "shipping-zones", title: "Zones de livraison", perm: "shipping:read", about: "Les zones desservies, leurs frais et leurs délais.", fields: "Frais de livraison, délai minimum et maximum en jours, éligibilité à la livraison offerte, zone active ou non, ordre d'affichage." },
  { slug: "promo-codes", title: "Codes promotionnels", perm: "pricing:read", about: "Les codes de réduction que les clients saisissent au panier.", fields: "Type et valeur de la remise, montant minimum de panier, dates de début et de fin, code actif ou non." },
  { slug: "reviews", title: "Avis clients", perm: "reviews:read", about: "Les avis déposés sur les produits, à modérer avant affichage.", fields: "Produit, note, texte, auteur, entreprise, avis vérifié ou non, et statut de modération. Modérer exige la permission « Modérer les avis »." },
  { slug: "quote-requests", title: "Demandes de devis", perm: "quotes:read", received: true, about: "Les demandes de devis envoyées depuis le site.", fields: "Seuls le statut et la note interne sont modifiables : la demande du client reste intacte." },
  { slug: "faq", title: "Questions fréquentes", perm: "content:read", about: "La foire aux questions affichée sur le site.", fields: "La question et la réponse existent dans chaque langue. L'ordre d'affichage et « Publiée sur le site » contrôlent leur apparition." },
  { slug: "hero-cards", title: "Cartes du hero (accueil)", perm: "content:read", about: "Les grandes cartes avec image en haut de la page d'accueil du site.", fields: "Titre, texte et bouton (par langue), adresse de l'image, lien au clic, couleurs du panneau et du texte, ordre d'affichage et « Affichée sur le site ». Sans carte publiée, le site affiche ses cartes par défaut." },
  { slug: "home-partners", title: "Partenaires (accueil)", perm: "content:read", about: "Les logos de la section « Nos partenaires clés » de l'accueil.", fields: "Nom, adresse du logo, lien facultatif vers le site du partenaire, ordre d'affichage et « Affiché sur le site ». Sans partenaire publié, le site affiche les logos des marques." },
  { slug: "pages", title: "Pages légales", perm: "content:read", about: "Les pages légales et éditoriales : confidentialité, CGV, mentions légales…", fields: "L'adresse (slug) forme l'URL de la page ; le contenu s'écrit en Markdown, dans chaque langue. Le type distingue page légale et éditoriale." },
  { slug: "articles", title: "Articles du blog", perm: "content:read", about: "Les articles du blog, leur auteur, leur catégorie et leurs produits liés.", fields: "Titre, extrait et corps par langue, catégorie, auteur, étiquettes, image, temps de lecture, mise en avant, produits liés. Le statut de publication est réservé à ceux qui peuvent publier." },
  { slug: "blog-categories", title: "Catégories du blog", perm: "content:read", about: "Les rubriques qui classent les articles.", fields: "Adresse (slug), nom par langue et ordre d'affichage." },
  { slug: "blog-tags", title: "Étiquettes du blog", perm: "content:read", about: "Les étiquettes libres qui complètent les catégories.", fields: "Adresse (slug) et nom par langue." },
  { slug: "authors", title: "Auteurs du blog", perm: "content:read", about: "Les personnes affichées comme auteurs des articles.", fields: "Nom, fonction, biographie et photo." },
  { slug: "contact-messages", title: "Messages de contact", perm: "support:read", received: true, about: "Les messages envoyés depuis le formulaire de contact du site.", fields: "Changez le statut et ajoutez une note interne pour suivre le traitement. Le message du visiteur reste en lecture seule." },
  { slug: "newsletter-subscribers", title: "Lettre d'information", perm: "support:read", received: true, about: "Les abonnés à la lettre d'information.", fields: "Seul le statut de l'abonné est modifiable (par exemple pour le désabonner)." },
];

const settingsTour = (id: string, title: string, perm: string, about: string, fields: string): GuideTour => ({
  id: `content-settings-${id}`, title, summary: about, path: `/content/settings/${id}`, permission: perm,
  match: (p) => p === `/content/settings/${id}`,
  steps: [
    header(about, title),
    form("Les réglages", fields),
    submit("Les valeurs s'appliquent au site dès l'enregistrement. Sans droit d'écriture, la page est en lecture seule."),
    help,
  ],
});

export const shopTours: GuideTour[] = [
  ...KINDS.flatMap((k) => [listTour(k), formTour(k)]),
  settingsTour("commerce", "Règles commerciales", "pricing:read", "Les règles de prix et de commande appliquées par la boutique.", "Taux de TVA, seuil de livraison offerte, conditions d'une demande de devis (nombre d'articles, montant) et seuil de stock bas."),
  settingsTour("home", "Textes de l'accueil", "content:read", "Les textes de la section Secteurs de l'accueil.", "Petit titre, titre et texte d'introduction, dans chaque langue. Un champ vide garde le texte par défaut du site.", ),
  settingsTour("site", "Informations du site", "content:read", "Les informations de l'entreprise affichées sur le site.", "Raison sociale, coordonnées, réseaux sociaux, année de fondation et chiffres clés mis en avant. Ces informations s'affichent sur le site."),
];

export const quotesTour: GuideTour = {
  id: "quotes", title: "Devis", summary: "Toutes les demandes de devis reçues.", path: "/quotes", permission: "quotes:read",
  match: (p) => p === "/quotes",
  steps: [
    intro("Toutes les demandes de devis", "Chaque demande envoyée depuis le site arrive ici, de la plus récente à la plus ancienne."),
    { id: "stats", target: "quotes-stats", placement: "bottom", title: "Les statuts d'un coup d'œil", text: "Cliquez une tuile pour ne voir que les demandes Nouvelles, En cours, Devis envoyé ou Clôturées." },
    filters("Cherchez par nom, e-mail, entreprise ou référence client."),
    table("La liste", "Particulier ou entreprise, objet, nombre d'articles et statut. « Ouvrir » affiche la demande complète."),
    pagination, help,
  ],
};
export const quoteDetailTour: GuideTour = {
  id: "quote-detail", title: "Détail d'un devis", summary: "Lire la demande et la traiter.", path: "/quotes", permission: "quotes:read",
  match: (p) => /^\/quotes\/[^/]+$/.test(p),
  steps: [
    header("L'objet de la demande et son statut actuel.", "La demande"),
    { id: "studio", target: "quote-studio", placement: "top", title: "Le studio de devis", text: "Chiffrez chaque ligne (prix, remise, délai, notes), réglez les conditions, relisez l'aperçu tel que le client le verra, puis envoyez. Les prix fixes du catalogue sont repris automatiquement ; les notes internes ne sont jamais visibles du client." },
    { id: "process", target: "quote-process", placement: "top", title: "Suivi de la demande", text: "Le statut est celui que le client voit. Il passe tout seul à « En cours » au premier brouillon et à « Devis envoyé » à l'envoi. La demande du client n'est jamais modifiée." },
    danger("La suppression est définitive et demande une confirmation."),
    help,
  ],
};
