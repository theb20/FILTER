import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, financeSettings, journals } from "@/lib/finance";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { RecurringFields } from "@/components/finance/op-forms";
import type { FinCategory, FinRecurring, FinThirdPartyPage } from "@/lib/types";
import { updateRecurring } from "../../actions";

export default async function RecurringPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:write")) redirect("/denied");
  const here = `/finance/recurring/${id}`;
  const [settings, accounts, jrn, cats, tps, list] = await Promise.all([financeSettings(here), allAccounts(here, true), journals(here, true), adminGet<{ items: FinCategory[] }>("/admin/finance/operation-categories", here), adminGet<FinThirdPartyPage>("/admin/finance/third-parties", here, { query: { pageSize: 100 } }), adminGet<{ items: FinRecurring[] }>("/admin/finance/recurring-templates", here)]);
  const r = list.items.find((x) => x.id === id);
  if (!r) redirect("/finance/recurring");
  return (
    <div className="max-w-3xl">
      <PageHeader title={r.label} subtitle="Les opérations déjà générées ne changent pas : la modification vaut pour les mois à venir." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Opérations récurrentes", href: "/finance/recurring" }, { label: r.label }]} />
      <Card><ActionForm action={updateRecurring.bind(null, id)} submit="Enregistrer"><RecurringFields currency={settings.baseCurrency} categories={cats.items} accounts={accounts} thirdParties={tps.items.filter((t) => t.active || t.id === r.thirdPartyId)} journals={jrn} value={r} /></ActionForm></Card>
    </div>
  );
}
