// Informations du site. Les valeurs vides sont des EMPLACEMENTS « à compléter » :
// l'interface les affiche comme tels, rien n'est inventé.
export const SITE = {
  name: "FilterFinder",
  publisher: "Holding Ledan Et Associé (HLEA)",
  phone: "",
  email: "",
  address: "", // adresse du siège
  linkedinUrl: "",
  reviewsUrl: "", // page d'avis (ex. Trustpilot)
  currency: "USD",
  // Site régional proposé par le bandeau de géolocalisation. Vide = bandeau masqué.
  // ex. { country: "France", url: "https://..." }
  regionalSite: null,
};

// Affirmations commerciales fournies par le cahier des charges mais non vérifiées :
// à renseigner (ou laisser vides pour afficher un emplacement).
export const CLAIMS = {
  crossReferences: "", // ex. "1,000,000+"
  stockValue: "", // ex. "$100M"
  foundedYear: "", // ex. "1968"
  freeShippingThreshold: "", // ex. "$500"
};

export const DEMO_NOTICE = "Demo mode: this action is not connected yet.";
