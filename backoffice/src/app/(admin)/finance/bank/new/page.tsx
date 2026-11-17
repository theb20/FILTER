import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, financeSettings } from "@/lib/finance";
import { Card, EmptyState, LinkButton, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { BankFields } from "@/components/finance/bank-forms";
import { createBankAccount } from "../../actions";

export const metadata = { title: "Ajouter un compte bancaire" };

export default async function NewBankPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "treasury:write")) redirect("/denied");
  const [accounts, settings] = await Promise.all([allAccounts("/finance/bank/new", true), financeSettings("/finance/bank/new")]);
  return (
    <div className="max-w-3xl">
      <PageHeader title="Ajouter un compte bancaire" breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Comptes bancaires", href: "/finance/bank" }, { label: "Nouveau" }]} />
      {accounts.length === 0 ? <EmptyState title="Aucun compte actif" hint="Créez d’abord le compte du plan comptable (banque, caisse) à lier." action={<LinkButton href="/finance/accounts/new">Ajouter un compte</LinkButton>} /> :
        <Card><ActionForm action={createBankAccount} submit="Créer le compte"><BankFields accounts={accounts} currency={settings.baseCurrency} /></ActionForm></Card>}
    </div>
  );
}
