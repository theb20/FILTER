import "server-only";

/** Montants des sources externes : micro-unités de la devise (1 unité = 1 000 000 micros). Jamais de virgule flottante en base. */
export const MICROS = 1_000_000;

/** « 0,005 USD » — montant lisible à partir des micro-unités. `max` : décimales affichées au plus. */
export function fromMicros(micros: number, currency: string, max = 6): string {
  const n = new Intl.NumberFormat("fr-FR", { minimumFractionDigits: 0, maximumFractionDigits: max }).format(micros / MICROS);
  return `${n} ${currency}`;
}

/** Saisie en unités de la devise (« 0,005 » ou « 5 ») → micro-unités. NaN si la saisie n'est pas un nombre positif. */
export function toMicros(raw: string): number {
  const v = raw.trim().replace(/\s| | /g, "").replace(",", ".");
  if (v === "" || !/^\d+(\.\d+)?$/.test(v)) return Number.NaN;
  return Math.round(Number(v) * MICROS);
}

export const SOURCE_KIND_LABEL: Record<string, string> = {
  web_search_api: "Recherche web", manufacturer_api: "API constructeur", distributor_feed: "Flux distributeur",
};
export const HEALTH_LABEL: Record<string, string> = {
  healthy: "En service", degraded: "Dégradée", down: "Hors service", suspended: "Suspendue (budget)", disabled: "Désactivée",
};
export const HEALTH_TONE: Record<string, "ok" | "warn" | "danger" | "off"> = {
  healthy: "ok", degraded: "warn", down: "danger", suspended: "warn", disabled: "off",
};
/** Pourquoi un appel a échoué, en clair. */
export const ERROR_KIND_LABEL: Record<string, string> = {
  secret_missing: "Clé d'API absente de l'environnement du service. Sur le serveur : export BRAVE_SEARCH_API_KEY=… puis ./dev.sh up",
  http_4xx: "La source a refusé l'appel : paramètre non accepté (pays non couvert, par exemple), clé invalide ou plan insuffisant",
  http_5xx: "La source est en erreur de son côté",
  quota: "La source limite le débit : trop de requêtes (429)",
  timeout: "La source n'a pas répondu dans le délai imparti",
  network: "Source injoignable (réseau)",
  decode: "Réponse de la source illisible",
  error: "Appel en échec",
  unknown: "Échec sans cause identifiée",
};

/** Valeurs proposées pour Brave Search (docs/PROPOSITION-MOTEUR.md) : crédit de 5 $, 0,005 $ par appel. */
export const BRAVE_PRESET = {
  kind: "web_search_api",
  name: "Brave Search",
  baseUrl: "https://api.search.brave.com/res/v1/web/search",
  priority: 0,
  timeoutMs: 3000,
  maxRetries: 2,
  secretRef: "BRAVE_SEARCH_API_KEY",
  costPerCall: "0,005",
  currency: "USD",
  monthlyBudget: "5",
  dailyCap: "0,333333",
  alertThresholdPct: 80,
} as const;
