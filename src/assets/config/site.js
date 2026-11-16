// Informations du site. Les valeurs vides sont des EMPLACEMENTS « à compléter » :
// l'interface les affiche comme tels, rien n'est inventé.
export const SITE = {
  name: "Holding Ledan & Associe",
  publisher: "Holding Ledan Et Associé (HLEA)",
  phone: "+225 07 47 225 087",
  email: "ledan.associe@gmail.com",
  address: "RCI ABIDJAN Palmeraie Résidence KAHA, Appartement E3  ", // adresse du siège
  linkedinUrl: "https://www.linkedin.com/company/holding-ledan-et-associ%C3%A9/", // page LinkedIn
  facebookUrl: "", // page Facebook (vide = « à compléter » dans le blog)
  youtubeUrl: "", // chaîne YouTube
  reviewsUrl: "", // page d'avis (ex. Trustpilot)
  // Site régional proposé par le bandeau de géolocalisation. Vide = bandeau masqué.
  // ex. { country: "France", url: "https://..." }
  regionalSite: null,
};

// Affirmations commerciales fournies par le cahier des charges mais non vérifiées :
// à renseigner (ou laisser vides pour afficher un emplacement).
export const CLAIMS = {
  crossReferences: "+5000", // ex. "1,000,000+"
  stockValue: "1000", // ex. "$100M"
  foundedYear: "2011", // ex. "1968"
  freeShippingThreshold: "", // ex. "$500"
};

