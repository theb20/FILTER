import type { GuideTour } from "../types";

export const dashboardTour: GuideTour = {
  id: "dashboard",
  title: "Tableau de bord",
  summary: "Repérez l'essentiel : indicateurs, actions en attente et raccourcis.",
  path: "/dashboard",
  auto: true,
  match: (p) => p === "/dashboard",
  steps: [
    { id: "intro", title: "Bienvenue dans le backoffice", text: "Cette visite de moins d'une minute présente la page d'accueil. Vous pouvez la quitter à tout moment avec Échap, et la relancer depuis le bouton « Découvrir cette section »." },
    { id: "nav", target: "nav", placement: "right", title: "Le menu principal", text: "Les pages sont regroupées par métier. Vous ne voyez que celles que vos permissions autorisent : si une page manque, c'est un droit à demander à un administrateur." },
    { id: "stats", target: "dashboard-stats", placement: "bottom", title: "Les indicateurs", text: "Chaque carte résume un chiffre clé et mène à la liste correspondante. Une carte jaune signale quelque chose à traiter." },
    { id: "todo", target: "dashboard-todo", placement: "top", title: "À traiter", text: "Les décisions qui attendent quelqu'un : entreprises à valider, produits à relire, brouillons à compléter. Un clic ouvre directement la liste filtrée." },
    { id: "shortcuts", target: "dashboard-shortcuts", placement: "left", title: "Raccourcis", text: "Les actions les plus fréquentes, selon vos droits : ajouter un produit, inviter un membre, consulter le journal d'audit." },
    { id: "help", target: "guide-help", placement: "bottom", title: "Besoin d'un rappel ?", text: "Ce bouton relance la visite de la page en cours. Pour désactiver le guide ou les animations, rendez-vous dans « Mon compte »." },
  ],
};
