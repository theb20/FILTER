import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts } from "@/lib/finance";
import { Card, EmptyState, LinkButton, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { CategoryFields } from "@/components/finance/misc-forms";
import { createCategory } from "../../actions";

export const metadata = { title: "Ajouter une catégorie" };

export default async function NewCategoryPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:admin")) redirect("/denied");
  const accounts = await allAccounts("/finance/categories/new", true);
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ajouter une catégorie" breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Catégories", href: "/finance/categories" }, { label: "Nouvelle" }]} />
      {accounts.length === 0 ? <EmptyState title="Aucun compte actif" hint="Créez ou importez d’abord le plan comptable." action={<LinkButton href="/finance/accounts/import">Importer le plan</LinkButton>} /> : <Card><ActionForm action={createCategory} submit="Créer la catégorie"><CategoryFields accounts={accounts} /></ActionForm></Card>}
    </div>
  );
}
