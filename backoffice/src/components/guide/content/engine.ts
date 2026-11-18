import type { GuideTour } from "../types";
import { action, filters, form, header, help, intro, submit, table } from "./shared";

export const sourcesTour: GuideTour = {
  id: "engine-sources", title: "Sources externes", summary: "Les services interrogés en plus du catalogue.", path: "/engine/sources", permission: "engine:read",
  match: (p) => p === "/engine/sources",
  steps: [
    intro("Les sources externes", "Brave Search, une API constructeur, un flux distributeur : chaque source est appelée en plus du catalogue interne. Chaque appel a un coût, donc chaque source a un budget."),
    action("Ajouter une source", "Le formulaire est prérempli pour Brave Search : il ne reste qu'à vérifier le coût par appel et la devise."),
    table("La liste", "L'état dit si la source est active, désactivée, ou suspendue parce que son budget mensuel est épuisé."),
    help,
  ],
};

export const sourceNewTour: GuideTour = {
  id: "engine-source-new", title: "Nouvelle source", summary: "Déclarer une source externe.", path: "/engine/sources/new", permission: "engine:write",
  match: (p) => p === "/engine/sources/new",
  steps: [
    header("Les valeurs proposées sont celles de Brave Search.", "Ajouter une source"),
    form("La clé d'API ne se saisit pas ici", "On enregistre seulement le NOM de la variable d'environnement qui porte la clé (BRAVE_SEARCH_API_KEY). La clé elle-même reste sur le serveur : « export BRAVE_SEARCH_API_KEY=… » puis « ./dev.sh up ». Elle n'entre jamais en base de données."),
    submit("Après la création : réglez le budget, testez la source, puis activez-la."),
    help,
  ],
};

export const sourceDetailTour: GuideTour = {
  id: "engine-source", title: "Fiche d'une source", summary: "État, test, budget et paramètres.", path: "/engine/sources", permission: "engine:read",
  match: (p) => /^\/engine\/sources\/(?!new$)[^/]+$/.test(p),
  steps: [
    { id: "state", target: "source-state", placement: "bottom", title: "État et santé", text: "Une source désactivée n'est jamais appelée. Les chiffres montrent les appels des 24 dernières heures, le temps de réponse et la dernière erreur." },
    { id: "test", target: "source-test-card", placement: "top", title: "Tester", text: "Un vrai appel est lancé : il vérifie d'un coup la clé d'API, le réseau et le budget. Il est facturé par la source." },
    { id: "budget", target: "source-budget", placement: "top", title: "Budget", text: "Le service réserve le coût d'un appel avant de le lancer. Sans budget restant, l'appel n'a pas lieu : c'est ce qui empêche une facture surprise." },
    { id: "info", target: "source-info", placement: "top", title: "Paramètres", text: "Adresse de l'API, coût par appel, délai, priorité. La durée de cache reste vide tant que les conditions d'usage de la source ne l'autorisent pas." },
    { id: "raw", target: "source-raw", placement: "top", title: "Résultats bruts", text: "Conserver les réponses permet de rejouer une recherche, mais beaucoup de sources l'interdisent. À n'autoriser qu'avec leurs conditions d'usage sous les yeux." },
    help,
  ],
};

export const costsTour: GuideTour = {
  id: "engine-costs", title: "Coûts des sources", summary: "Ce que les appels externes ont coûté.", path: "/engine/costs", permission: "engine:read",
  match: (p) => p === "/engine/costs",
  steps: [
    intro("Les coûts", "Appels, appels servis par le cache, échecs et coût réel, source par source."),
    filters("Choisissez la période. Par défaut : les 30 derniers jours."),
    table("Le détail", "Les montants ne sont jamais additionnés entre devises : chaque devise a son total."),
    help,
  ],
};
