import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, financeSettings, journals } from "@/lib/finance";
import { Card, EmptyState, LinkButton, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { RecurringFields } from "@/components/finance/op-forms";
import type { FinCategory, FinThirdPartyPage } from "@/lib/types";
import { createRecurring } from "../../actions";

export const metadata = { title: "Nouveau modèle récurrent" };

export default async function NewRecurringPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:write")) redirect("/denied");
  const here = "/finance/recurring/new";
  const [settings, accounts, jrn, cats, tps] = await Promise.all([financeSettings(here), allAccounts(here, true), journals(here, true), adminGet<{ items: FinCategory[] }>("/admin/finance/operation-categories", here, { query: { active: "true" } }), adminGet<FinThirdPartyPage>("/admin/finance/third-parties", here, { query: { pageSize: 100 } })]);
  return (
    <div className="max-w-3xl">
      <PageHeader title="Nouveau modèle récurrent" breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Opérations récurrentes", href: "/finance/recurring" }, { label: "Nouveau" }]} />
      {cats.items.length === 0 || jrn.length === 0 ? <EmptyState title="Configuration incomplète" hint="Il faut au moins une catégorie et un journal." action={<LinkButton href="/finance/categories/new">Créer une catégorie</LinkButton>} /> :
        <Card><ActionForm action={createRecurring} submit="Créer le modèle"><RecurringFields currency={settings.baseCurrency} categories={cats.items} accounts={accounts} thirdParties={tps.items.filter((t) => t.active)} journals={jrn} /></ActionForm></Card>}
    </div>
  );
}
