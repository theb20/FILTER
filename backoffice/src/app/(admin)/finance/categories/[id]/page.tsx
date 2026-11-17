import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts } from "@/lib/finance";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { CategoryFields } from "@/components/finance/misc-forms";
import type { FinCategory } from "@/lib/types";
import { updateCategory } from "../../actions";

export default async function CategoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:admin")) redirect("/denied");
  const here = `/finance/categories/${id}`;
  const [list, accounts] = await Promise.all([adminGet<{ items: FinCategory[] }>("/admin/finance/operation-categories", here), allAccounts(here, true)]);
  const c = list.items.find((x) => x.id === id);
  if (!c) redirect("/finance/categories");
  return (
    <div className="max-w-2xl">
      <PageHeader title={c.label} subtitle="Changer le compte n’affecte que les écritures générées à partir de maintenant." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Catégories", href: "/finance/categories" }, { label: c.label }]} />
      <Card><ActionForm action={updateCategory.bind(null, id)} submit="Enregistrer"><CategoryFields accounts={accounts} value={c} /></ActionForm></Card>
    </div>
  );
}
