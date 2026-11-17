import type { GuideTour } from "../types";
import { filters, header, help, intro, pagination, table } from "./shared";

export const customersTour: GuideTour = {
  id: "customers", title: "Clients", summary: "Retrouver un compte client, particulier ou entreprise.", path: "/customers", permission: "customers:read",
  match: (p) => p === "/customers",
  steps: [
    intro("Les clients", "Tous les comptes de la boutique, particuliers et entreprises. Cette page sert à retrouver un client et à ouvrir sa fiche."),
    filters("Cherchez par nom, e-mail ou raison sociale, puis affinez par type de compte et par statut. « Réinitialiser » efface tous les filtres."),
    table("La liste des clients", "Un clic sur le nom ouvre la fiche. Les entreprises affichent leur statut de validation à côté de leur raison sociale."),
    pagination, help,
  ],
};

export const customerDetailTour: GuideTour = {
  id: "customer-detail", title: "Fiche client", summary: "Consulter un client et suspendre ou réactiver son compte.", path: "/customers", permission: "customers:read",
  match: (p) => /^\/customers\/[^/]+$/.test(p),
  steps: [
    header("Le nom, l'adresse e-mail et le statut du compte. Le fil d'Ariane ramène à la liste.", "La fiche client"),
    { id: "actions", target: "customer-actions", placement: "top", title: "Suspendre ou réactiver", text: "Suspendre ferme immédiatement les sessions du client, sans supprimer ses données. L'action est réversible et demande une confirmation." },
    help,
  ],
};

export const companiesTour: GuideTour = {
  id: "companies", title: "Entreprises", summary: "Valider les comptes professionnels en attente.", path: "/companies", permission: "customers:read",
  match: (p) => p === "/companies",
  steps: [
    intro("Validation des entreprises", "Un compte professionnel doit être approuvé avant de bénéficier des conditions entreprises. Cette page est votre file de décisions."),
    { id: "tabs", target: "tabs", placement: "bottom", title: "Les trois statuts", text: "« En attente » est l'onglet de travail. Les onglets Approuvées et Rejetées servent à retrouver une décision passée." },
    filters("Retrouvez une entreprise par sa raison sociale ou son numéro fiscal."),
    table("La file", "Ouvrez une entreprise pour vérifier son numéro fiscal puis l'approuver ou la rejeter. La colonne Décision indique la date de la dernière décision."),
    pagination, help,
  ],
};

export const companyDetailTour: GuideTour = {
  id: "company-detail", title: "Fiche entreprise", summary: "Approuver ou rejeter une demande.", path: "/companies", permission: "customers:read",
  match: (p) => /^\/companies\/[^/]+$/.test(p),
  steps: [
    header("La raison sociale, le numéro fiscal, le pays et le statut de la demande.", "La demande"),
    { id: "approve", target: "company-approve", placement: "top", title: "Approuver", text: "Avant d'approuver, vérifiez le numéro fiscal et que l'adresse e-mail du titulaire est vérifiée. Une confirmation est demandée." },
    { id: "reject", target: "company-reject", placement: "top", title: "Rejeter", text: "Le rejet exige un motif, conservé avec la décision. Ces boutons n'apparaissent que si vous avez le droit de décider et que la demande est en attente." },
    help,
  ],
};
