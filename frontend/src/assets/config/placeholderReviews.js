// AVIS FICTIFS (emplacements d'attente) : affichés sur l'accueil pour garnir les 3 cartes tant qu'il y a moins de 3 vrais avis.
// Chaque vrai avis publié (Boutique › Avis clients, sans produit) remplace un avis fictif. Ces textes ne sont PAS de vrais clients.
const BY_LANG = {
  fr: [
    { text: "Livraison rapide et références croisées exactes : nous avons retrouvé nos filtres en quelques minutes.", author: "Awa K.", company: "Maintenance industrielle", rating: 5 },
    { text: "Un interlocuteur réactif pour notre devis, et des prix clairs hors taxes. Très pratique pour nos achats récurrents.", author: "Jean-Marc T.", company: "Transport et logistique", rating: 5 },
    { text: "Le catalogue est facile à parcourir et les fiches techniques sont complètes. Je recommande.", author: "Mariam D.", company: "Atelier mécanique", rating: 4 },
  ],
  en: [
    { text: "Fast delivery and accurate cross-references: we found our filters within minutes.", author: "Awa K.", company: "Industrial maintenance", rating: 5 },
    { text: "A responsive contact for our quote and clear prices excluding tax. Very handy for our recurring purchases.", author: "Jean-Marc T.", company: "Transport & logistics", rating: 5 },
    { text: "The catalogue is easy to browse and the datasheets are complete. Recommended.", author: "Mariam D.", company: "Mechanical workshop", rating: 4 },
  ],
  es: [
    { text: "Entrega rápida y referencias cruzadas exactas: encontramos nuestros filtros en minutos.", author: "Awa K.", company: "Mantenimiento industrial", rating: 5 },
    { text: "Un interlocutor ágil para nuestro presupuesto y precios claros sin impuestos. Muy práctico para compras recurrentes.", author: "Jean-Marc T.", company: "Transporte y logística", rating: 5 },
    { text: "El catálogo es fácil de explorar y las fichas técnicas están completas. Lo recomiendo.", author: "Mariam D.", company: "Taller mecánico", rating: 4 },
  ],
};

export const REVIEW_SLOTS = 3;

/** Complète les vrais avis avec des avis fictifs jusqu'à 3 cartes : chaque vrai avis en retire un. */
export function withPlaceholders(real, lang) {
  const fill = (BY_LANG[lang] ?? BY_LANG.fr).slice(0, Math.max(0, REVIEW_SLOTS - real.length)).map((r, i) => ({ ...r, id: `placeholder-${i}`, placeholder: true }));
  return [...real, ...fill];
}
