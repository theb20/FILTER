import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import type { FinCategory } from "@/lib/types";

export const metadata = { title: "Catégories de dépenses et revenus" };

export default async function CategoriesPage({ searchParams }: { searchParams: Promise<{ ok?: string }> }) {
  const { ok } = await searchParams;
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<{ items: FinCategory[] }>("/admin/finance/operation-categories", "/finance/categories")]);
  const admin = can(me?.permissions ?? [], "finance:admin");
  return (
    <div>
      <PageHeader title="Catégories" subtitle="Chaque catégorie correspond à un compte du plan comptable : c’est ainsi que les opérations génèrent leurs écritures. Aucune correspondance n’est préinstallée." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Catégories" }]}
        action={admin && <LinkButton href="/finance/categories/new">Ajouter une catégorie</LinkButton>} />
      {ok && <Notice kind="ok">Catégorie enregistrée.</Notice>}
      <Table head={["Catégorie", "Type", "Compte", "Nature", "État", ""]} empty={data.items.length === 0 ? "Aucune catégorie pour l’instant." : undefined} emptyHint="Créez les catégories de dépenses et de revenus avec votre expert-comptable.">
        {data.items.map((c) => (
          <tr key={c.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night">{c.label}</td>
            <td className="px-4 py-3 text-slate">{c.kind === "expense" ? "Dépense" : "Revenu"}</td>
            <td className="px-4 py-3 text-slate"><span className="font-mono text-xs">{c.accountCode}</span> {c.accountLabel}</td>
            <td className="px-4 py-3 text-slate">{c.nature === "fixed" ? "Fixe" : c.nature === "variable" ? "Variable" : "—"}</td>
            <td className="px-4 py-3">{c.active ? <Badge tone="ok">Active</Badge> : <Badge tone="off">Inactive</Badge>}</td>
            <td className="px-4 py-3 text-right">{admin && <Link href={`/finance/categories/${c.id}`} className="text-link hover:underline">Modifier</Link>}</td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
