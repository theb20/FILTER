import type { GuideStep } from "../types";

// Étapes réutilisées : une cible absente (droit manquant, page vide) est simplement sautée.
export const header = (text: string, title = "Cette page"): GuideStep => ({ id: "header", target: "page-header", placement: "bottom", title, text });
export const action = (title: string, text: string): GuideStep => ({ id: "action", target: "page-action", placement: "bottom", title, text });
export const filters = (text: string): GuideStep => ({ id: "filters", target: "filters", placement: "bottom", title: "Rechercher et filtrer", text });
export const table = (title: string, text: string): GuideStep => ({ id: "table", target: "table", placement: "top", title, text });
export const pagination: GuideStep = { id: "pagination", target: "pagination", placement: "top", title: "Parcourir les pages", text: "Les listes sont découpées par 20 lignes. Vos filtres sont conservés quand vous changez de page." };
export const form = (title: string, text: string): GuideStep => ({ id: "form", target: "form", placement: "auto", title, text });
export const submit = (text: string): GuideStep => ({ id: "submit", target: "form-submit", placement: "top", title: "Enregistrer", text });
export const danger = (text: string): GuideStep => ({ id: "danger", target: "danger-zone", placement: "top", title: "Zone sensible", text });
export const help: GuideStep = { id: "help", target: "guide-help", placement: "bottom", title: "Relancer cette visite", text: "Ce bouton rouvre la visite de la page à tout moment. Pour désactiver le guide : Mon compte." };
export const intro = (title: string, text: string): GuideStep => ({ id: "intro", title, text });
