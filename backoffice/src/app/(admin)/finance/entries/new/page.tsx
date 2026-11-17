import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, financeSettings, journals } from "@/lib/finance";
import { today } from "@/lib/money";
import { Card, EmptyState, LinkButton, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { EntryEditor } from "@/components/finance/entry-editor";
import { createEntry } from "../../actions";

export const metadata = { title: "Saisir une écriture" };

export default async function NewEntryPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:write")) redirect("/denied");
  const [accounts, jrn, settings] = await Promise.all([allAccounts("/finance/entries/new", true), journals("/finance/entries/new", true), financeSettings("/finance/entries/new")]);
  return (
    <div className="max-w-5xl">
      <PageHeader title="Saisir une écriture" subtitle="L'écriture est enregistrée en brouillon. Une autre personne la valide : elle devient alors définitive et reçoit son numéro." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Écritures", href: "/finance/entries" }, { label: "Nouvelle" }]} />
      {accounts.length === 0 || jrn.length === 0 ? (
        <EmptyState title="Configuration incomplète" hint="Il faut au moins un journal et des comptes actifs. Voir la vue d'ensemble Finance." action={<LinkButton href="/finance">Vue d’ensemble</LinkButton>} />
      ) : (
        <Card>
          <ActionForm action={createEntry} submit="Enregistrer le brouillon">
            <EntryEditor accounts={accounts} journals={jrn} currency={settings.baseCurrency} defaultDate={today()} />
          </ActionForm>
        </Card>
      )}
    </div>
  );
}
