// Généré depuis la spécification des collections de contenu : même source que le contrat, le service et le gateway.
// Les écrans « contenu » du backoffice (liste, création, détail) sont pilotés par ces descriptions.

export type FieldType = string; // str | text | slug | code | loc | int | num | bool | url | email | ts | uuid | enum:a|b | strs | uuids | items
export type FieldDef = { name: string; type: FieldType; label: string; req: boolean; labels?: Record<string, string>; help?: string; long?: boolean; ro?: boolean; gated?: boolean; ref?: string; min?: number; max?: number };
export type KindDef = {
  slug: string; kind: string; title: string; single: string; perm: { read: string; write: string }; summary: string[];
  ops: ("list" | "get" | "create" | "update" | "delete")[]; updateFields: string[] | null; fields: FieldDef[];
  publish: { field: string; on: string | boolean; off: string | boolean } | null;
};
export type SettingsDef = { key: string; slug: string; title: string; perm: { read: string; write: string }; fields: FieldDef[] };

export const CONTENT_KINDS: KindDef[] = [
 {
  "slug": "faq",
  "kind": "faq",
  "title": "Questions fréquentes",
  "single": "Question",
  "perm": {
   "read": "content:read",
   "write": "content:write"
  },
  "summary": [
   "question"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": null,
  "fields": [
   {
    "name": "question",
    "type": "loc",
    "label": "Question",
    "req": true
   },
   {
    "name": "answer",
    "type": "loc",
    "label": "Réponse",
    "req": true,
    "long": true
   },
   {
    "name": "sortOrder",
    "type": "int",
    "label": "Ordre d'affichage",
    "req": false,
    "min": 0
   },
   {
    "name": "published",
    "type": "bool",
    "label": "Publiée sur le site",
    "req": false
   }
  ]
 },
 {
  "slug": "pages",
  "kind": "page",
  "title": "Pages (légales et éditoriales)",
  "single": "Page",
  "perm": {
   "read": "content:read",
   "write": "content:write"
  },
  "summary": [
   "title",
   "slug"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": {
   "field": "published",
   "on": true,
   "off": false
  },
  "updateFields": null,
  "fields": [
   {
    "name": "slug",
    "type": "slug",
    "label": "Adresse (slug)",
    "req": true,
    "help": "Ex. confidentialite, cgv, mentions-legales"
   },
   {
    "name": "kind",
    "type": "enum:legal|editorial",
    "label": "Type",
    "req": true,
    "labels": {
     "legal": "Légale",
     "editorial": "Éditoriale"
    }
   },
   {
    "name": "title",
    "type": "loc",
    "label": "Titre",
    "req": true
   },
   {
    "name": "body",
    "type": "loc",
    "label": "Contenu (Markdown)",
    "req": true,
    "long": true
   },
   {
    "name": "published",
    "type": "bool",
    "label": "Publiée sur le site",
    "req": false,
    "gated": true
   }
  ]
 },
 {
  "slug": "authors",
  "kind": "author",
  "title": "Auteurs du blog",
  "single": "Auteur",
  "perm": {
   "read": "content:read",
   "write": "content:write"
  },
  "summary": [
   "name"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": null,
  "fields": [
   {
    "name": "name",
    "type": "str",
    "label": "Nom",
    "req": true
   },
   {
    "name": "role",
    "type": "loc",
    "label": "Fonction",
    "req": false
   },
   {
    "name": "bio",
    "type": "loc",
    "label": "Biographie",
    "req": false,
    "long": true
   },
   {
    "name": "avatarUrl",
    "type": "url",
    "label": "Photo (URL)",
    "req": false
   }
  ]
 },
 {
  "slug": "blog-categories",
  "kind": "blog_category",
  "title": "Catégories du blog",
  "single": "Catégorie",
  "perm": {
   "read": "content:read",
   "write": "content:write"
  },
  "summary": [
   "name",
   "slug"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": null,
  "fields": [
   {
    "name": "slug",
    "type": "slug",
    "label": "Adresse (slug)",
    "req": true
   },
   {
    "name": "name",
    "type": "loc",
    "label": "Nom",
    "req": true
   },
   {
    "name": "sortOrder",
    "type": "int",
    "label": "Ordre d'affichage",
    "req": false,
    "min": 0
   }
  ]
 },
 {
  "slug": "blog-tags",
  "kind": "blog_tag",
  "title": "Étiquettes du blog",
  "single": "Étiquette",
  "perm": {
   "read": "content:read",
   "write": "content:write"
  },
  "summary": [
   "name",
   "slug"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": null,
  "fields": [
   {
    "name": "slug",
    "type": "slug",
    "label": "Adresse (slug)",
    "req": true
   },
   {
    "name": "name",
    "type": "loc",
    "label": "Nom",
    "req": true
   }
  ]
 },
 {
  "slug": "articles",
  "kind": "article",
  "title": "Articles du blog",
  "single": "Article",
  "perm": {
   "read": "content:read",
   "write": "content:write"
  },
  "summary": [
   "title",
   "slug",
   "status"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": {
   "field": "status",
   "on": "published",
   "off": "draft"
  },
  "updateFields": null,
  "fields": [
   {
    "name": "slug",
    "type": "slug",
    "label": "Adresse (slug)",
    "req": true
   },
   {
    "name": "title",
    "type": "loc",
    "label": "Titre",
    "req": true
   },
   {
    "name": "excerpt",
    "type": "loc",
    "label": "Résumé",
    "req": false,
    "long": true
   },
   {
    "name": "body",
    "type": "loc",
    "label": "Contenu (Markdown : « ## » pour un intertitre)",
    "req": true,
    "long": true
   },
   {
    "name": "status",
    "type": "enum:draft|published",
    "label": "Statut",
    "req": true,
    "labels": {
     "draft": "Brouillon",
     "published": "Publié"
    },
    "gated": true
   },
   {
    "name": "kind",
    "type": "enum:advice|comparison|news",
    "label": "Type",
    "req": true,
    "labels": {
     "advice": "Conseil",
     "comparison": "Comparatif",
     "news": "Actualité"
    }
   },
   {
    "name": "categorySlug",
    "type": "str",
    "label": "Catégorie du blog (slug)",
    "req": false,
    "ref": "blog_category"
   },
   {
    "name": "authorId",
    "type": "uuid",
    "label": "Auteur",
    "req": false,
    "ref": "author"
   },
   {
    "name": "tags",
    "type": "strs",
    "label": "Étiquettes (slugs, séparés par une virgule)",
    "req": false
   },
   {
    "name": "imageUrl",
    "type": "url",
    "label": "Image (URL)",
    "req": false
   },
   {
    "name": "readMin",
    "type": "int",
    "label": "Temps de lecture (minutes)",
    "req": false,
    "min": 1,
    "max": 240
   },
   {
    "name": "guide",
    "type": "enum:steps|checklist",
    "label": "Format de guide",
    "req": false,
    "labels": {
     "steps": "Étapes",
     "checklist": "Liste de contrôle"
    }
   },
   {
    "name": "dossier",
    "type": "bool",
    "label": "Dossier à la une",
    "req": false
   },
   {
    "name": "featured",
    "type": "bool",
    "label": "Mis en avant",
    "req": false
   },
   {
    "name": "productIds",
    "type": "uuids",
    "label": "Produits cités (identifiants, séparés par une virgule)",
    "req": false
   },
   {
    "name": "publishedAt",
    "type": "ts",
    "label": "Date de publication",
    "req": false
   }
  ]
 },
 {
  "slug": "shipping-zones",
  "kind": "shipping_zone",
  "title": "Zones de livraison",
  "single": "Zone",
  "perm": {
   "read": "shipping:read",
   "write": "shipping:write"
  },
  "summary": [
   "name",
   "code"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": null,
  "fields": [
   {
    "name": "code",
    "type": "slug",
    "label": "Code",
    "req": true
   },
   {
    "name": "name",
    "type": "loc",
    "label": "Nom",
    "req": true
   },
   {
    "name": "fee",
    "type": "int",
    "label": "Frais de livraison (F CFA HT)",
    "req": true,
    "min": 0
   },
   {
    "name": "daysMin",
    "type": "int",
    "label": "Délai minimal (jours ouvrés)",
    "req": true,
    "min": 0,
    "max": 365
   },
   {
    "name": "daysMax",
    "type": "int",
    "label": "Délai maximal (jours ouvrés)",
    "req": true,
    "min": 0,
    "max": 365
   },
   {
    "name": "freeEligible",
    "type": "bool",
    "label": "Éligible à la livraison offerte",
    "req": false
   },
   {
    "name": "active",
    "type": "bool",
    "label": "Active",
    "req": false
   },
   {
    "name": "sortOrder",
    "type": "int",
    "label": "Ordre d'affichage",
    "req": false,
    "min": 0
   }
  ]
 },
 {
  "slug": "promo-codes",
  "kind": "promo_code",
  "title": "Codes promotionnels",
  "single": "Code",
  "perm": {
   "read": "pricing:read",
   "write": "pricing:write"
  },
  "summary": [
   "code",
   "type",
   "value"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": null,
  "fields": [
   {
    "name": "code",
    "type": "code",
    "label": "Code",
    "req": true,
    "help": "Majuscules, chiffres, tirets"
   },
   {
    "name": "type",
    "type": "enum:percent|fixed",
    "label": "Type de remise",
    "req": true,
    "labels": {
     "percent": "Pourcentage",
     "fixed": "Montant fixe (F CFA)"
    }
   },
   {
    "name": "value",
    "type": "int",
    "label": "Valeur (% ou F CFA)",
    "req": true,
    "min": 1
   },
   {
    "name": "minSubtotal",
    "type": "int",
    "label": "Sous-total minimal (F CFA HT)",
    "req": false,
    "min": 0
   },
   {
    "name": "startsAt",
    "type": "ts",
    "label": "Début de validité",
    "req": false
   },
   {
    "name": "expiresAt",
    "type": "ts",
    "label": "Fin de validité",
    "req": false
   },
   {
    "name": "active",
    "type": "bool",
    "label": "Actif",
    "req": false
   }
  ]
 },
 {
  "slug": "reviews",
  "kind": "review",
  "title": "Avis clients",
  "single": "Avis",
  "perm": {
   "read": "reviews:read",
   "write": "reviews:moderate"
  },
  "summary": [
   "title",
   "author",
   "rating",
   "status"
  ],
  "ops": [
   "list",
   "get",
   "create",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": null,
  "fields": [
   {
    "name": "productId",
    "type": "uuid",
    "label": "Produit concerné (identifiant)",
    "req": false,
    "help": "Laisser vide pour un témoignage affiché sur la page d'accueil.",
    "ref": "product"
   },
   {
    "name": "rating",
    "type": "int",
    "label": "Note (1 à 5)",
    "req": true,
    "min": 1,
    "max": 5
   },
   {
    "name": "title",
    "type": "str",
    "label": "Titre",
    "req": true
   },
   {
    "name": "text",
    "type": "text",
    "label": "Avis",
    "req": true,
    "long": true
   },
   {
    "name": "author",
    "type": "str",
    "label": "Auteur",
    "req": true
   },
   {
    "name": "company",
    "type": "str",
    "label": "Entreprise",
    "req": false
   },
   {
    "name": "verified",
    "type": "bool",
    "label": "Achat vérifié",
    "req": false
   },
   {
    "name": "status",
    "type": "enum:pending|published|rejected",
    "label": "Statut",
    "req": true,
    "labels": {
     "pending": "À modérer",
     "published": "Publié",
     "rejected": "Refusé"
    }
   }
  ]
 },
 {
  "slug": "contact-messages",
  "kind": "contact_message",
  "title": "Messages de contact",
  "single": "Message",
  "perm": {
   "read": "support:read",
   "write": "support:write"
  },
  "summary": [
   "name",
   "subject",
   "status"
  ],
  "ops": [
   "list",
   "get",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": [
   "status",
   "note"
  ],
  "fields": [
   {
    "name": "name",
    "type": "str",
    "label": "Nom",
    "req": true,
    "ro": true
   },
   {
    "name": "email",
    "type": "email",
    "label": "Adresse e-mail",
    "req": true,
    "ro": true
   },
   {
    "name": "company",
    "type": "str",
    "label": "Entreprise",
    "req": false,
    "ro": true
   },
   {
    "name": "phone",
    "type": "str",
    "label": "Téléphone",
    "req": false,
    "ro": true
   },
   {
    "name": "subject",
    "type": "str",
    "label": "Sujet",
    "req": false,
    "ro": true
   },
   {
    "name": "message",
    "type": "text",
    "label": "Message",
    "req": true,
    "long": true,
    "ro": true
   },
   {
    "name": "status",
    "type": "enum:new|read|handled",
    "label": "Statut",
    "req": true,
    "labels": {
     "new": "Nouveau",
     "read": "Lu",
     "handled": "Traité"
    }
   },
   {
    "name": "note",
    "type": "text",
    "label": "Note interne",
    "req": false,
    "long": true
   }
  ]
 },
 {
  "slug": "newsletter-subscribers",
  "kind": "newsletter_subscriber",
  "title": "Abonnés à la lettre d'information",
  "single": "Abonné",
  "perm": {
   "read": "support:read",
   "write": "support:write"
  },
  "summary": [
   "email",
   "status"
  ],
  "ops": [
   "list",
   "get",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": [
   "status"
  ],
  "fields": [
   {
    "name": "email",
    "type": "email",
    "label": "Adresse e-mail",
    "req": true,
    "ro": true
   },
   {
    "name": "locale",
    "type": "str",
    "label": "Langue",
    "req": false,
    "ro": true
   },
   {
    "name": "status",
    "type": "enum:subscribed|unsubscribed",
    "label": "Statut",
    "req": true,
    "labels": {
     "subscribed": "Abonné",
     "unsubscribed": "Désabonné"
    }
   }
  ]
 },
 {
  "slug": "quote-requests",
  "kind": "quote_request",
  "title": "Demandes de devis",
  "single": "Demande",
  "perm": {
   "read": "quotes:read",
   "write": "quotes:write"
  },
  "summary": [
   "name",
   "company",
   "status"
  ],
  "ops": [
   "list",
   "get",
   "update",
   "delete"
  ],
  "publish": null,
  "updateFields": [
   "status",
   "note"
  ],
  "fields": [
   {
    "name": "name",
    "type": "str",
    "label": "Nom",
    "req": true,
    "ro": true
   },
   {
    "name": "email",
    "type": "email",
    "label": "Adresse e-mail",
    "req": true,
    "ro": true
   },
   {
    "name": "company",
    "type": "str",
    "label": "Entreprise",
    "req": false,
    "ro": true
   },
   {
    "name": "phone",
    "type": "str",
    "label": "Téléphone",
    "req": false,
    "ro": true
   },
   {
    "name": "message",
    "type": "text",
    "label": "Remarques",
    "req": false,
    "long": true,
    "ro": true
   },
   {
    "name": "items",
    "type": "items",
    "label": "Articles demandés",
    "req": false,
    "ro": true
   },
   {
    "name": "status",
    "type": "enum:new|in_progress|quoted|closed",
    "label": "Statut",
    "req": true,
    "labels": {
     "new": "Nouvelle",
     "in_progress": "En cours",
     "quoted": "Devis envoyé",
     "closed": "Clôturée"
    }
   },
   {
    "name": "note",
    "type": "text",
    "label": "Note interne",
    "req": false,
    "long": true
   }
  ]
 }
];

export const CONTENT_SETTINGS: SettingsDef[] = [
 {
  "key": "site",
  "slug": "site-settings",
  "title": "Informations du site",
  "perm": {
   "read": "content:read",
   "write": "content:write"
  },
  "fields": [
   {
    "name": "name",
    "type": "str",
    "label": "Nom du site",
    "req": true
   },
   {
    "name": "publisher",
    "type": "str",
    "label": "Éditeur (mentions légales)",
    "req": true
   },
   {
    "name": "phone",
    "type": "str",
    "label": "Téléphone",
    "req": false
   },
   {
    "name": "email",
    "type": "email",
    "label": "E-mail de contact",
    "req": false
   },
   {
    "name": "address",
    "type": "str",
    "label": "Adresse du siège",
    "req": false
   },
   {
    "name": "linkedinUrl",
    "type": "url",
    "label": "Page LinkedIn",
    "req": false
   },
   {
    "name": "facebookUrl",
    "type": "url",
    "label": "Page Facebook",
    "req": false
   },
   {
    "name": "youtubeUrl",
    "type": "url",
    "label": "Chaîne YouTube",
    "req": false
   },
   {
    "name": "reviewsUrl",
    "type": "url",
    "label": "Page d'avis (ex. Trustpilot)",
    "req": false
   },
   {
    "name": "regionalCountry",
    "type": "str",
    "label": "Site régional : pays",
    "req": false
   },
   {
    "name": "regionalUrl",
    "type": "url",
    "label": "Site régional : adresse",
    "req": false
   },
   {
    "name": "crossReferencesClaim",
    "type": "str",
    "label": "Chiffre affiché : références croisées (ex. +5000)",
    "req": false
   },
   {
    "name": "stockValueClaim",
    "type": "str",
    "label": "Chiffre affiché : valeur du stock",
    "req": false
   },
   {
    "name": "foundedYear",
    "type": "str",
    "label": "Année de création",
    "req": false
   },
   {
    "name": "freeShippingClaim",
    "type": "str",
    "label": "Bandeau : montant de livraison offerte (texte)",
    "req": false
   }
  ]
 },
 {
  "key": "commerce",
  "slug": "commerce-settings",
  "title": "Règles commerciales",
  "perm": {
   "read": "pricing:read",
   "write": "pricing:write"
  },
  "fields": [
   {
    "name": "vatRate",
    "type": "num",
    "label": "Taux de TVA (%)",
    "req": false,
    "help": "Vide = prix affichés hors taxes uniquement",
    "min": 0,
    "max": 100
   },
   {
    "name": "freeShippingThreshold",
    "type": "int",
    "label": "Seuil de livraison offerte (F CFA HT)",
    "req": false,
    "min": 0
   },
   {
    "name": "quoteMinItems",
    "type": "int",
    "label": "Devis : nombre d'articles minimal",
    "req": false,
    "min": 1
   },
   {
    "name": "quoteMinAmount",
    "type": "int",
    "label": "Devis : montant HT minimal (F CFA)",
    "req": false,
    "min": 1
   },
   {
    "name": "lowStockThreshold",
    "type": "int",
    "label": "Seuil « stock faible »",
    "req": false,
    "min": 0
   },
   {
    "name": "defaultZone",
    "type": "str",
    "label": "Zone de livraison par défaut (code)",
    "req": false
   }
  ]
 }
];

export const kindBySlug = (slug: string) => CONTENT_KINDS.find((k) => k.slug === slug);
export const settingsByKey = (key: string) => CONTENT_SETTINGS.find((s) => s.key === key);
