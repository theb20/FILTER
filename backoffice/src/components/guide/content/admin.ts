import type { GuideTour } from "../types";
import { action, danger, filters, form, header, help, intro, pagination, submit, table } from "./shared";

export const staffTour: GuideTour = {
  id: "staff", title: "Personnel", summary: "Inviter et gérer les membres de l'équipe.", path: "/staff", permission: "admin:read",
  match: (p) => p === "/staff",
  steps: [
    intro("Le personnel", "Les comptes qui ont accès à ce backoffice. Chacun reçoit des rôles, qui déterminent ce qu'il peut voir et faire."),
    action("Inviter un membre", "Envoie un lien valable 72 h pour choisir un mot de passe et activer la double authentification."),
    filters("Cherchez par nom ou par adresse e-mail."),
    table("La liste", "Statut (Invité, Actif, Désactivé), rôles, double authentification et dernière connexion. Ouvrez un membre pour le modifier."),
    pagination, help,
  ],
};

export const staffNewTour: GuideTour = {
  id: "staff-new", title: "Inviter un membre", summary: "Créer un accès pour un collègue.", path: "/staff/new", permission: "admin:read",
  match: (p) => p === "/staff/new",
  steps: [
    header("La personne reçoit un e-mail d'invitation, valable 72 h.", "Invitation"),
    form("Le formulaire", "Indiquez son nom, son adresse e-mail et ses rôles : ils déterminent ce qu'il pourra voir et faire."),
    submit("L'invitation part immédiatement. Vous pourrez la renvoyer depuis la fiche du membre."),
    help,
  ],
};

export const staffDetailTour: GuideTour = {
  id: "staff-detail", title: "Fiche membre", summary: "Modifier les rôles, désactiver, réinitialiser la 2FA.", path: "/staff", permission: "admin:read",
  match: (p) => /^\/staff\/(?!new$)[^/]+$/.test(p),
  steps: [
    header("Le nom, l'adresse e-mail et le statut du compte.", "La fiche"),
    { id: "roles", scene: "roles-permissions", target: "staff-roles", placement: "top", title: "Informations et rôles", text: "Modifiez le nom et les rôles du membre. Ses permissions sont celles de tous ses rôles réunis." },
    { id: "actions", target: "staff-actions", placement: "top", title: "Actions sur le compte", text: "Renvoyer l'invitation, désactiver ou réactiver le compte, réinitialiser la double authentification en cas de téléphone perdu. Vous ne pouvez pas vous désactiver vous-même." },
    help,
  ],
};

export const rolesTour: GuideTour = {
  id: "roles", title: "Rôles et permissions", summary: "Regrouper des permissions en rôles.", path: "/roles", permission: "admin:read",
  match: (p) => p === "/roles",
  steps: [
    { ...intro("Rôles et permissions", "Une permission autorise une action précise. Un rôle en regroupe plusieurs, et un membre peut cumuler plusieurs rôles."), scene: "roles-permissions" },
    action("Créer un rôle", "Compose un nouveau rôle sur mesure."),
    table("Les rôles", "Les rôles « Système » sont fournis et non modifiables. La colonne Permissions indique le nombre de droits accordés, ou « Toutes »."),
    help,
  ],
};

export const roleFormTour: GuideTour = {
  id: "role-form", title: "Fiche rôle", summary: "Choisir les permissions d'un rôle.", path: "/roles", permission: "admin:read",
  match: (p) => p === "/roles/new" || /^\/roles\/(?!new$)[^/]+$/.test(p),
  steps: [
    header("Un nom, une description et la liste des permissions accordées.", "Un rôle"),
    { id: "perms", scene: "roles-permissions", target: "permissions", placement: "auto", title: "Les permissions", text: "Cochez les actions autorisées, regroupées par domaine. Lecture et écriture sont des droits distincts. Vous ne pouvez accorder que des permissions que vous possédez." },
    submit("Le changement touche tous les membres qui portent ce rôle."),
    danger("Impossible de supprimer un rôle tant que des membres du personnel le portent."),
    help,
  ],
};

export const auditTour: GuideTour = {
  id: "audit", title: "Journal d'audit", summary: "L'historique infalsifiable des actions sensibles.", path: "/audit", permission: "audit:read",
  match: (p) => p === "/audit",
  steps: [
    intro("Le journal d'audit", "Chaque action sensible y est consignée : qui, quand, quoi. Le journal est en ajout seul : aucune ligne ne peut être modifiée ni supprimée."),
    filters("Filtrez par entité (personnel, rôles, réglages) et par type d'action pour retrouver un événement précis."),
    table("Les entrées", "Date, action, cible, auteur et adresse IP. « Voir » déplie les valeurs avant et après la modification."),
    help,
  ],
};

export const settingsTour: GuideTour = {
  id: "settings", title: "Réglages d'identité", summary: "Sessions, liens et sécurité des connexions.", path: "/settings", permission: "admin:read",
  match: (p) => p === "/settings",
  steps: [
    header("Ces réglages s'appliquent à tous les comptes, pas seulement au vôtre.", "Réglages globaux"),
    { id: "form", target: "settings-form", placement: "auto", title: "Durées et protections", text: "Durée des sessions et des liens, ralentissement des connexions répétées, conservation des données. Sans droit d'écriture, les valeurs sont en lecture seule. Chaque modification est tracée dans le journal d'audit." },
    help,
  ],
};

export const accountTour: GuideTour = {
  id: "account", title: "Mon compte", summary: "Mot de passe, guide et sessions.", path: "/account",
  match: (p) => p === "/account",
  steps: [
    header("Vos informations, vos rôles et la sécurité de votre accès.", "Votre compte"),
    { id: "password", target: "account-password", placement: "bottom", title: "Mot de passe", text: "12 caractères minimum. Utilisez un mot de passe qui n'est employé nulle part ailleurs." },
    { id: "guide", target: "account-guide", placement: "top", title: "Le guide visuel", text: "Activez ou désactivez le guide et ses animations, relancez une visite ou marquez-les toutes comme non vues. Ces choix sont propres à votre compte." },
    { id: "sessions", target: "account-sessions", placement: "top", title: "Sessions", text: "« Me déconnecter partout » ferme toutes vos sessions, y compris celle-ci : utile si vous avez oublié de vous déconnecter d'un autre appareil." },
  ],
};
