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
  settingsTour("site", "Informations du site", "content:read", "Les informations de l'entreprise affichées sur le site.", "Raison sociale, coordonnées, réseaux sociaux, année de fondation et chiffres clés mis en avant. Ces informations s'affichent sur le site."),
];
