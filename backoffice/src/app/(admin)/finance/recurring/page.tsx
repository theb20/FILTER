import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { dateFr, fmtMoney, today } from "@/lib/money";
import { Badge, Card, inputCls, LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { financeSettings } from "@/lib/finance";
import type { FinRecurring } from "@/lib/types";
import { generateRecurring } from "../actions";

export const metadata = { title: "Opérations récurrentes" };

export default async function RecurringPage({ searchParams }: { searchParams: Promise<{ ok?: string }> }) {
  const { ok } = await searchParams;
  const [{ me }, data, settings] = await Promise.all([currentStaff(), adminGet<{ items: FinRecurring[] }>("/admin/finance/recurring-templates", "/finance/recurring"), financeSettings("/finance/recurring")]);
  const write = can(me?.permissions ?? [], "finance:write");
  return (
    <div className="space-y-5">
      <PageHeader title="Opérations récurrentes" subtitle="Un modèle crée chaque mois une opération en brouillon, sans écriture : vous la contrôlez puis la comptabilisez." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Opérations récurrentes" }]}
        action={write && <LinkButton href="/finance/recurring/new">Ajouter un modèle</LinkButton>} />
      {ok && <Notice kind="ok">Modèle enregistré.</Notice>}
      <Table head={["Modèle", "Type", "Catégorie", "Montant HT", "Jour", "Dernier mois généré", "État", ""]} empty={data.items.length === 0 ? "Aucun modèle pour l’instant." : undefined} emptyHint="Loyer, abonnements, salaires (montants) : créez le modèle une fois.">
        {data.items.map((r) => (
          <tr key={r.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night">{r.label}</td>
            <td className="px-4 py-3 text-slate">{r.kind === "expense" ? "Dépense" : "Revenu"}</td>
            <td className="px-4 py-3 text-slate">{r.categoryLabel}</td>
            <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums">{fmtMoney(r.amountNet, settings.baseCurrency)}</td>
            <td className="px-4 py-3 text-slate">{r.dayOfMonth}</td>
            <td className="px-4 py-3 text-slate">{r.lastGeneratedMonth ? dateFr(r.lastGeneratedMonth) : "—"}</td>
            <td className="px-4 py-3">{r.active ? <Badge tone="ok">Actif</Badge> : <Badge tone="off">Inactif</Badge>}</td>
            <td className="px-4 py-3 text-right">{write && <Link href={`/finance/recurring/${r.id}`} className="text-link hover:underline">Modifier</Link>}</td>
          </tr>
        ))}
      </Table>
      {write && data.items.some((r) => r.active) && (
        <Card title="Générer les opérations échues" description="Crée les opérations (brouillons) de chaque mois échu jusqu’à la date choisie. Sans risque de doublon : un mois déjà généré est ignoré.">
          <ActionForm action={generateRecurring} submit="Générer" className="flex flex-wrap items-end gap-3">
            <label className="block text-sm font-medium text-night">Jusqu’au<input name="upTo" type="date" required defaultValue={today()} className={`${inputCls} mt-1 w-48`} /></label>
          </ActionForm>
        </Card>
      )}
    </div>
  );
}
