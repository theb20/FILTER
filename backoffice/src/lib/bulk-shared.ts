// Briques communes à l'édition en masse (tableur) des catégories et des produits : pures, donc testables.
export type BulkCol = { id: string; title: string; width?: number; kind?: "text" | "int" | "check"; readOnly?: boolean; aliases?: string[] };
export type BulkRow = Record<string, unknown> & { _k: string; _id?: string; _orig?: string; _msg?: string; _ok?: boolean; del?: boolean };
export type BulkResult = { k: string; ok: boolean; error?: string; id?: string; deleted?: boolean };

/** Clé de comparaison : minuscules, sans accents ni ponctuation (« Nom (FR) » = « nom fr »). */
export const norm = (s: unknown) => String(s ?? "").normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase().replace(/[^a-z0-9]+/g, "");

/** Associe les en-têtes d'une feuille Excel aux colonnes (par titre, identifiant ou alias). Renvoie, pour chaque colonne de la feuille, l'identifiant de colonne ou null. */
export function matchHeaders(headers: unknown[], cols: BulkCol[]): (string | null)[] {
  const byName = new Map<string, string>();
  for (const c of cols) for (const n of [c.id, c.title, ...(c.aliases ?? [])]) byName.set(norm(n), c.id);
  return headers.map((h) => byName.get(norm(h)) ?? null);
}

export const text = (v: unknown) => (v === null || v === undefined ? "" : String(v).trim());

/** Nombre entier ≥ 0 ou null ; NaN si la saisie n'est pas un entier. */
export function toInt(v: unknown): number | null {
  const s = text(v).replace(/[\s  ]/g, "");
  if (s === "") return null;
  return /^\d+$/.test(s) ? Number(s) : Number.NaN;
}

/** Mode de prix saisi en clair (« Sur devis », « fixe »…) → valeur de l'API ; null si non reconnu. */
export function parsePriceMode(v: unknown): "fixed" | "on_quote" | null {
  const n = norm(v);
  if (n === "") return "fixed";
  if (["fixed", "fixe", "prixfixe", "fix"].includes(n)) return "fixed";
  if (["onquote", "surdevis", "devis"].includes(n)) return "on_quote";
  return null;
}
export function parseFulfillment(v: unknown): "stock" | "made_to_order" | null {
  const n = norm(v);
  if (n === "") return "stock";
  if (["stock", "enstock"].includes(n)) return "stock";
  if (["madetoorder", "surcommande", "fabrication", "surmesure"].includes(n)) return "made_to_order";
  return null;
}

/** Textes par langue : le français est obligatoire, les autres sont omis s'ils sont vides. */
export function localized(fr: unknown, en: unknown, es: unknown): Record<string, string> {
  const o: Record<string, string> = { fr: text(fr) };
  if (text(en)) o.en = text(en);
  if (text(es)) o.es = text(es);
  return o;
}
export const optLocalized = (fr: unknown, en: unknown, es: unknown) => (text(fr) || text(en) || text(es) ? localized(fr, en, es) : null);
