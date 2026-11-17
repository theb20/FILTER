import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, financeSettings } from "@/lib/finance";
import { Card, EmptyState, LinkButton, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { OperationFields } from "@/components/finance/op-forms";
import type { FinCategory, FinThirdPartyPage } from "@/lib/types";
import { createOperation } from "../../actions";

export const metadata = { title: "Nouvelle opération" };

export default async function NewOperationPage({ searchParams }: { searchParams: Promise<{ kind?: string }> }) {
  const sp = await searchParams;
  const kind = sp.kind === "income" ? "income" : "expense";
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:write")) redirect("/denied");
  const here = `/finance/operations/new?kind=${kind}`;
  const [settings, accounts, cats, tps] = await Promise.all([
    financeSettings(here), allAccounts(here, true),
    adminGet<{ items: FinCategory[] }>("/admin/finance/operation-categories", here, { query: { kind, active: "true" } }),
    adminGet<FinThirdPartyPage>("/admin/finance/third-parties", here, { query: { kind: kind === "expense" ? "supplier" : "customer", pageSize: 100 } }),
  ]);
  const title = kind === "expense" ? "Nouvelle dépense" : "Nouveau revenu";
  return (
    <div className="max-w-3xl">
      <PageHeader title={title} subtitle="L'opération est enregistrée sans écriture. Vous générez ensuite l'écriture en brouillon, qu'une autre personne valide." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Dépenses et revenus", href: "/finance/operations" }, { label: title }]} />
      {cats.items.length === 0 || accounts.length === 0 ? (
        <EmptyState title="Configuration incomplète" hint="Il faut au moins une catégorie (avec son compte du plan comptable) et des comptes actifs. Les catégories se créent dans le menu Catégories." action={<LinkButton href="/finance/categories/new">Créer une catégorie</LinkButton>} />
      ) : (
        <Card><ActionForm action={createOperation} submit="Enregistrer"><OperationFields kind={kind} currency={settings.baseCurrency} categories={cats.items} accounts={accounts} thirdParties={tps.items.filter((t) => t.active)} /></ActionForm></Card>
      )}
    </div>
  );
}
