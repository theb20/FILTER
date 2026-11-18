"use server";

import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { allCategories } from "@/lib/catalog";
import { localized, optLocalized, text, type BulkResult, type BulkRow } from "@/lib/bulk-shared";
import type { Category } from "@/lib/types";

type Cat = Pick<Category, "id" | "slug" | "parentId" | "sortOrder">;

/** Enregistre en série les lignes du tableur : suppressions, modifications (dont déplacement), puis créations (parents avant enfants). */
export async function saveCategories(rows: BulkRow[]): Promise<BulkResult[]> {
  const cats: Cat[] = await allCategories("/categories/bulk");
  const idBySlug = new Map(cats.map((c) => [c.slug.toLowerCase(), c.id]));
  const slugById = new Map(cats.map((c) => [c.id, c.slug.toLowerCase()]));
  const children = new Map<string | null, number>();
  for (const c of cats) children.set(c.parentId, (children.get(c.parentId) ?? 0) + 1);
  const out = new Map<string, BulkResult>();
  const fail = (r: BulkRow, error: string) => out.set(r._k, { k: r._k, ok: false, error });

  const body = (r: BulkRow) => ({ slug: text(r.slug), name: localized(r.nameFr, r.nameEn, r.nameEs), description: optLocalized(r.descFr, r.descEn, r.descEs), imageUrl: text(r.imageUrl) || null });
  const check = (r: BulkRow) => (!text(r.slug) ? "L'identifiant d'URL est obligatoire." : !text(r.nameFr) ? "Le nom en français est obligatoire." : null);

  // 1. Suppressions
  for (const r of rows.filter((x) => x.del && x._id)) {
    const res = await adminAct("DELETE", `/admin/categories/${r._id}`);
    if (res.ok) out.set(r._k, { k: r._k, ok: true, deleted: true }); else fail(r, errorMessage(res));
  }
  // 2. Modifications et déplacements
  for (const r of rows.filter((x) => !x.del && x._id)) {
    const bad = check(r);
    if (bad) { fail(r, bad); continue; }
    const res = await adminAct<Category>("PUT", `/admin/categories/${r._id}`, { body: body(r) });
    if (!res.ok) { fail(r, errorMessage(res)); continue; }
    const wanted = text(r.parent).toLowerCase();
    const current = slugById.get(cats.find((c) => c.id === r._id)?.parentId ?? "") ?? "";
    if (wanted !== current) {
      const parentId = wanted ? idBySlug.get(wanted) : null;
      if (wanted && !parentId) { fail(r, `Informations enregistrées, mais le parent « ${wanted} » est introuvable.`); continue; }
      const mv = await adminAct("POST", `/admin/categories/${r._id}/move`, { body: { parentId: parentId ?? null, position: children.get(parentId ?? null) ?? 0 } });
      if (!mv.ok) { fail(r, `Informations enregistrées, déplacement refusé : ${errorMessage(mv)}`); continue; }
    }
    out.set(r._k, { k: r._k, ok: true, id: r._id });
  }
  // 3. Créations : une catégorie n'est créée qu'une fois son parent connu (on boucle tant qu'il y a du progrès).
  let todo = rows.filter((x) => !x.del && !x._id);
  for (const r of todo) { const bad = check(r); if (bad) fail(r, bad); }
  todo = todo.filter((r) => !out.has(r._k));
  for (let progress = true; progress && todo.length > 0;) {
    progress = false;
    for (const r of [...todo]) {
      const parentSlug = text(r.parent).toLowerCase();
      if (parentSlug && !idBySlug.has(parentSlug)) continue;
      const res = await adminAct<Category>("POST", "/admin/categories", { body: { ...body(r), parentId: parentSlug ? idBySlug.get(parentSlug) : null } });
      todo = todo.filter((x) => x !== r);
      progress = true;
      if (res.ok) { idBySlug.set(text(r.slug).toLowerCase(), res.data.id); out.set(r._k, { k: r._k, ok: true, id: res.data.id }); } else fail(r, errorMessage(res));
    }
  }
  for (const r of todo) fail(r, `Le parent « ${text(r.parent)} » est introuvable (ni existant, ni dans les lignes à créer).`);

  revalidatePath("/categories");
  return rows.map((r) => out.get(r._k) ?? { k: r._k, ok: false, error: "Ignorée." });
}
