import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories, attributesOf, buildTree, flatten, subtreeIds, t } from "@/lib/catalog";
import { Badge, Card, inputCls, LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { CategoryFields } from "@/components/catalog-forms";
import { ParentSelect } from "@/components/parent-select";
import { ATTRIBUTE_TYPE_LABEL, type AttributeDef, type Category } from "@/lib/types";
import { deleteCategory, moveCategory, updateCategory } from "../actions";

const NOTICES: Record<string, string> = {
  created: "Catégorie créée.",
  updated: "Modifications enregistrées.",
  moved: "Catégorie déplacée.",
  "attr-created": "Attribut créé.",
  "attr-updated": "Attribut modifié.",
  "attr-deleted": "Attribut supprimé.",
};

function AttrTable({ attrs, editable }: { attrs: AttributeDef[]; editable: boolean }) {
  return (
    <Table head={["Attribut", "Clé", "Type", "Options", ""]}>
      {attrs.map((a) => (
        <tr key={a.id} className="hover:bg-page">
          <td className="px-4 py-3 font-medium text-night">{t(a.label)}{a.required && <span className="ml-1 text-danger" title="Obligatoire">*</span>}</td>
          <td className="px-4 py-3 text-slate">{a.key}</td>
          <td className="px-4 py-3 text-slate">{ATTRIBUTE_TYPE_LABEL[a.type]}{a.unit ? ` (${a.unit})` : ""}</td>
          <td className="px-4 py-3">{a.filterable && <Badge tone="info">Filtre</Badge>}</td>
          <td className="px-4 py-3 text-right">{editable && <Link href={`/attributes/${a.id}`} className="text-link hover:underline">Modifier</Link>}</td>
        </tr>
      ))}
    </Table>
  );
}

export default async function CategoryPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const here = `/categories/${id}`;
  const [{ me }, cat, cats] = await Promise.all([currentStaff(), adminGet<Category>(`/admin/categories/${id}`, here), allCategories(here)]);
  const canWrite = can(me?.permissions ?? [], "catalog:write");
  const rows = flatten(buildTree(cats));
  const byId = new Map(cats.map((c) => [c.id, c]));
  const ancestors: Category[] = [];
  for (let p = cat.parentId ? byId.get(cat.parentId) : undefined; p && ancestors.length < 50; p = p.parentId ? byId.get(p.parentId) : undefined) ancestors.unshift(p);
  const own = await attributesOf(id, here);
  const inherited = (await Promise.all(ancestors.map(async (a) => ({ from: a, attrs: await attributesOf(a.id, here) })))).filter((x) => x.attrs.length > 0);
  const children = cats.filter((c) => c.parentId === id).sort((a, b) => a.sortOrder - b.sortOrder || a.slug.localeCompare(b.slug));
  const siblings = cats.filter((c) => c.parentId === cat.parentId).sort((a, b) => a.sortOrder - b.sortOrder || a.slug.localeCompare(b.slug));
  const position = Math.max(0, siblings.findIndex((c) => c.id === id));

  return (
    <div className="max-w-4xl space-y-5">
      <PageHeader title={t(cat.name)} subtitle={[...ancestors.map((a) => t(a.name)), t(cat.name)].join(" › ")} action={canWrite && <LinkButton href={`/categories/new?parent=${id}`} variant="ghost">Ajouter une sous-catégorie</LinkButton>} />
      {ok && NOTICES[ok] && <Notice kind="ok">{NOTICES[ok]}</Notice>}

      <Card>
        <h2 className="mb-3 text-lg font-semibold text-night">Informations</h2>
        {canWrite ? <ActionForm action={updateCategory.bind(null, id)} submit="Enregistrer"><CategoryFields value={cat} /></ActionForm> : <p className="text-sm text-slate">{cat.slug}</p>}
      </Card>

      {canWrite && (
        <Card>
          <h2 className="mb-1 text-lg font-semibold text-night">Déplacer</h2>
          <p className="mb-3 text-sm text-slate">Impossible de la placer sous elle-même ni sous l'une de ses sous-catégories, ni si cela crée un conflit de clés d'attributs.</p>
          <ActionForm action={moveCategory.bind(null, id)} submit="Déplacer">
            <ParentSelect rows={rows} value={cat.parentId} exclude={subtreeIds(cats, id)} label="Nouveau parent" />
            <label className="block text-sm font-medium text-night">Position parmi ses frères <span className="font-normal text-slate">(0 = en premier)</span><input name="position" type="number" min={0} defaultValue={position} className={inputCls} /></label>
          </ActionForm>
        </Card>
      )}

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-night">Sous-catégories ({children.length})</h2>
        </div>
        {children.length === 0 ? <p className="text-sm text-slate">Aucune sous-catégorie.</p> : (
          <ul className="flex flex-wrap gap-2">{children.map((c) => <li key={c.id}><Link href={`/categories/${c.id}`} className="inline-block rounded border border-line bg-white px-3 py-1.5 text-sm text-link hover:bg-page">{t(c.name)}</Link></li>)}</ul>
        )}
      </section>

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-night">Attributs de cette catégorie ({own.length})</h2>
          {canWrite && <LinkButton href={`/attributes/new?categoryId=${id}`}>Ajouter un attribut</LinkButton>}
        </div>
        {own.length === 0 ? <p className="text-sm text-slate">Aucun attribut propre à cette catégorie.</p> : <AttrTable attrs={own} editable={canWrite} />}
      </section>

      {inherited.map(({ from, attrs }) => (
        <section key={from.id}>
          <h2 className="mb-3 text-base font-semibold text-night">Hérités de « <Link href={`/categories/${from.id}`} className="text-link hover:underline">{t(from.name)}</Link> »</h2>
          <AttrTable attrs={attrs} editable={false} />
        </section>
      ))}

      {canWrite && (
        <Card>
          <h2 className="mb-3 text-lg font-semibold text-night">Supprimer cette catégorie</h2>
          <p className="mb-3 text-sm text-slate">Impossible tant qu'elle a des sous-catégories, des attributs ou des produits.</p>
          <ActionButton action={deleteCategory.bind(null, id)} label="Supprimer la catégorie" variant="danger" confirm={`Supprimer la catégorie « ${t(cat.name)} » ?`} />
        </Card>
      )}
    </div>
  );
}
