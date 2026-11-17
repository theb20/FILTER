import "server-only";
import type { FieldDef } from "./content-kinds";
import { adminGet } from "./admin";

export type Doc = Record<string, unknown> & { id: string; createdAt: string; updatedAt: string };
export type DocPage = { items: Doc[]; page: { page: number; pageSize: number; total: number; totalPages: number } };
export type RefOptions = Record<string, { value: string; label: string }[]>;

const LANGS = ["fr", "en", "es"] as const;

/** Convertit le formulaire en corps JSON du contrat. `partial` : les champs vides sont envoyés vides (le service les efface). */
export function formToBody(fields: FieldDef[], form: FormData, only?: string[] | null): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const f of fields) {
    if (f.ro || f.gated || (only && !only.includes(f.name))) continue; // « gated » : réglé par une route dédiée (publication)
    const t = f.type;
    const raw = (k: string) => String(form.get(k) ?? "").trim();
    if (t === "loc") {
      const v: Record<string, string> = {};
      for (const l of LANGS) if (raw(`${f.name}_${l}`)) v[l] = raw(`${f.name}_${l}`);
      if (Object.keys(v).length) out[f.name] = v;
    } else if (t === "bool") {
      out[f.name] = form.get(f.name) === "on";
    } else if (t === "int" || t === "num") {
      if (raw(f.name) !== "") out[f.name] = Number(raw(f.name));
    } else if (t === "ts") {
      if (raw(f.name)) out[f.name] = new Date(`${raw(f.name)}:00Z`).toISOString();
    } else if (t === "strs" || t === "uuids") {
      const l = raw(f.name).split(/[,\n]/).map((s) => s.trim()).filter(Boolean);
      if (l.length) out[f.name] = l;
    } else if (t === "items") {
      // lignes de devis : jamais modifiées depuis le backoffice
    } else if (raw(f.name)) {
      out[f.name] = raw(f.name);
    } else if (only) {
      out[f.name] = ""; // modification partielle : un champ vidé est effacé
    }
  }
  return out;
}

/** Valeurs proposées pour les champs qui désignent un autre élément (auteur, catégorie du blog). */
export async function loadRefs(fields: FieldDef[], returnTo: string): Promise<RefOptions> {
  const refs: RefOptions = {};
  for (const f of fields) {
    if (!f.ref || refs[f.ref]) continue;
    const src = f.ref === "author" ? "/admin/authors" : f.ref === "blog_category" ? "/admin/blog-categories" : null;
    if (!src) continue;
    const r = await adminGet<DocPage>(src, returnTo, { query: { pageSize: 100 } });
    refs[f.ref] = r.items.map((d) => ({ value: f.ref === "author" ? d.id : String(d.slug), label: label(d.name ?? d.title) }));
  }
  return refs;
}

/** Texte lisible d'une valeur « par langue » (français d'abord). */
export function label(v: unknown): string {
  if (v && typeof v === "object") {
    const m = v as Record<string, string>;
    return m.fr ?? Object.values(m)[0] ?? "";
  }
  return v === undefined || v === null ? "" : String(v);
}
