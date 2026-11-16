import "server-only";
import { adminGet } from "./admin";
import type { AttributeDef, AttributeDefPage, Category, CategoryPage, Localized } from "./types";

/** Langues gérées par le site. Le français est obligatoire ; les autres se replient dessus quand elles sont vides. */
export const LANGS = [
  { code: "fr", label: "Français", required: true },
  { code: "en", label: "Anglais", required: false },
  { code: "es", label: "Espagnol", required: false },
] as const;

/** Lit les champs `<prefix>_fr`, `<prefix>_en`… d'un formulaire ; les langues vides (sauf fr) sont omises. */
export function readLocalized(form: FormData, prefix: string): Localized {
  const out: Localized = {};
  for (const l of LANGS) {
    const v = String(form.get(`${prefix}_${l.code}`) ?? "").trim();
    if (v || l.required) out[l.code] = v;
  }
  return out;
}

/** Même chose pour un texte facultatif : null si aucune langue n'est renseignée. */
export function readOptionalLocalized(form: FormData, prefix: string): Localized | null {
  const o = readLocalized(form, prefix);
  return Object.values(o).some(Boolean) ? o : null;
}

export const t = (l: Localized | null | undefined) => l?.fr ?? Object.values(l ?? {})[0] ?? "—";

/** Toutes les catégories (la liste est paginée à 100). */
export async function allCategories(returnTo: string): Promise<Category[]> {
  const out: Category[] = [];
  for (let page = 1; page <= 50; page++) {
    const r = await adminGet<CategoryPage>("/admin/categories", returnTo, { query: { page, pageSize: 100 } });
    out.push(...r.items);
    if (page >= r.page.totalPages) break;
  }
  return out;
}

export type TreeNode = Category & { children: TreeNode[] };

/** Arbre trié (ordre manuel puis identifiant d'URL). Une catégorie dont le parent est absent est traitée comme racine. */
export function buildTree(cats: Category[]): TreeNode[] {
  const nodes = new Map<string, TreeNode>(cats.map((c) => [c.id, { ...c, children: [] }]));
  const roots: TreeNode[] = [];
  for (const n of nodes.values()) {
    const p = n.parentId ? nodes.get(n.parentId) : undefined;
    (p ? p.children : roots).push(n);
  }
  const sort = (xs: TreeNode[]) => {
    xs.sort((a, b) => a.sortOrder - b.sortOrder || a.slug.localeCompare(b.slug));
    xs.forEach((x) => sort(x.children));
  };
  sort(roots);
  return roots;
}

export function flatten(tree: TreeNode[], depth = 0): (TreeNode & { level: number })[] {
  return tree.flatMap((n) => [{ ...n, level: depth }, ...flatten(n.children, depth + 1)]);
}

/** Identifiants d'une catégorie et de tous ses descendants (ils ne peuvent pas devenir son parent). */
export function subtreeIds(cats: Category[], id: string): Set<string> {
  const ids = new Set([id]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const c of cats) if (c.parentId && ids.has(c.parentId) && !ids.has(c.id)) { ids.add(c.id); grew = true; }
  }
  return ids;
}

export async function attributesOf(categoryId: string, returnTo: string): Promise<AttributeDef[]> {
  const r = await adminGet<AttributeDefPage>("/admin/attribute-defs", returnTo, { query: { categoryId, pageSize: 100 } });
  return r.items;
}

/** Définitions applicables à une catégorie : les siennes et celles de ses ancêtres, de la racine vers la catégorie. */
export async function attributeDefsFor(categoryId: string, cats: Category[], returnTo: string): Promise<AttributeDef[]> {
  const byId = new Map(cats.map((c) => [c.id, c]));
  const chain: string[] = [];
  for (let c = byId.get(categoryId); c && chain.length < 50; c = c.parentId ? byId.get(c.parentId) : undefined) chain.unshift(c.id);
  const lists = await Promise.all(chain.map((id) => attributesOf(id, returnTo)));
  return lists.flat();
}
