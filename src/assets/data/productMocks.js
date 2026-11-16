// @ts-check
// ─────────────────────────────────────────────────────────────────────────────
// DONNÉES DE DÉMONSTRATION DE LA FICHE PRODUIT — avis, noms et entreprises FICTIFS.
// À remplacer par l'API. Les stock, caractéristiques, remises et compatibilités viennent de cartMocks.js.
// ─────────────────────────────────────────────────────────────────────────────

/**
 * @typedef {{ id: string, rating: number, title: string, text: string, author: string, company: string, date: string, verified: boolean }} DemoReview
 */

/** Avis fictifs (identiques pour tous les produits en démonstration). @type {DemoReview[]} */
export const DEMO_REVIEWS = [
  { id: "r1", rating: 5, title: "Avis de démonstration n° 1", text: "Texte fictif : la livraison est arrivée dans les délais annoncés et la référence correspondait à la machine.", author: "Client exemple A.", company: "Société exemple SARL", date: "2026-10-30", verified: true },
  { id: "r2", rating: 5, title: "Avis de démonstration n° 2", text: "Texte fictif : bon rapport qualité-prix, l'équivalence indiquée était exacte.", author: "Client exemple B.", company: "Atelier exemple", date: "2026-10-12", verified: true },
  { id: "r3", rating: 4, title: "Avis de démonstration n° 3", text: "Texte fictif : produit conforme, emballage à améliorer.", author: "Client exemple C.", company: "Transport exemple", date: "2026-09-27", verified: true },
  { id: "r4", rating: 5, title: "Avis de démonstration n° 4", text: "Texte fictif : réponse rapide du support pour confirmer la compatibilité.", author: "Client exemple D.", company: "Chantier exemple SA", date: "2026-09-05", verified: false },
  { id: "r5", rating: 3, title: "Avis de démonstration n° 5", text: "Texte fictif : délai un peu plus long que prévu en dehors d'Abidjan.", author: "Client exemple E.", company: "Ferme exemple", date: "2026-08-19", verified: true },
  { id: "r6", rating: 4, title: "Avis de démonstration n° 6", text: "Texte fictif : je recommande, je recommanderai pour la prochaine révision.", author: "Client exemple F.", company: "Garage exemple", date: "2026-08-02", verified: true },
];

/** Nombre d'avis affichés avant « Voir tous les avis ». */
export const REVIEWS_PREVIEW = 3;
