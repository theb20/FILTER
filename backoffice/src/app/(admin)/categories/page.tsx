import Link from "next/link";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories, buildTree, flatten, t } from "@/lib/catalog";
import { LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";

export const metadata = { title: "Catégories" };

export default async function CategoriesPage({ searchParams }: { searchParams: Promise<{ ok?: string }> }) {
  const { ok } = await searchParams;
  const [{ me }, cats] = await Promise.all([currentStaff(), allCategories("/categories")]);
  const rows = flatten(buildTree(cats));
  const canWrite = can(me?.permissions ?? [], "catalog:write");
  return (
    <div>
      <PageHeader title="Catégories" subtitle="L'arbre du catalogue. Les attributs d'une catégorie sont hérités par ses sous-catégories." action={canWrite && <LinkButton href="/categories/new">Ajouter une catégorie</LinkButton>} />
      {ok === "deleted" && <Notice kind="ok">Catégorie supprimée.</Notice>}
      <Table head={["Catégorie", "Identifiant d'URL", "Produits", ""]} empty={rows.length === 0 ? "Aucune catégorie pour l'instant." : undefined}>
        {rows.map((c) => (
          <tr key={c.id} className="hover:bg-page">
            <td className="py-3 pr-4 font-medium text-night" style={{ paddingLeft: `${16 + c.level * 24}px` }}>
              {c.level > 0 && <span aria-hidden className="mr-2 text-slate/50">└</span>}
              <Link href={`/categories/${c.id}`} className="hover:underline">{t(c.name)}</Link>
            </td>
            <td className="px-4 py-3 text-slate">{c.slug}</td>
            <td className="px-4 py-3 text-slate">{c.productCount}</td>
            <td className="px-4 py-3 text-right"><Link href={`/categories/${c.id}`} className="text-link hover:underline">Ouvrir</Link></td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
