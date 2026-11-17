import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { dateFr } from "@/lib/money";
import { Badge, inputCls, Notice, PageHeader, Table } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { YEAR_STATUS_LABEL, type FinYear } from "@/lib/types";
import { closePeriod, requestReopen } from "../../actions";

const NOTICES: Record<string, string> = { created: "Exercice ouvert.", closed: "Période clôturée.", requested: "Demande de réouverture envoyée : une autre personne doit l'approuver (menu Approbations)." };

export default async function YearPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, y] = await Promise.all([currentStaff(), adminGet<FinYear>(`/admin/finance/fiscal-years/${id}`, `/finance/periods/${id}`)]);
  const canClose = can(me?.permissions ?? [], "finance:close");
  return (
    <div className="space-y-5">
      <PageHeader title={`Exercice ${y.label}`} subtitle={`Du ${dateFr(y.startDate)} au ${dateFr(y.endDate)} · ${YEAR_STATUS_LABEL[y.status]}`} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Exercices", href: "/finance/periods" }, { label: y.label }]} />
      {ok && NOTICES[ok] && <Notice kind="ok">{NOTICES[ok]}</Notice>}
      <Table head={["N°", "Du", "Au", "État", "Action"]}>
        {y.periods.map((p) => (
          <tr key={p.id}>
            <td className="px-4 py-3 text-slate">{p.number}</td>
            <td className="px-4 py-3 text-night">{dateFr(p.startDate)}</td>
            <td className="px-4 py-3 text-night">{dateFr(p.endDate)}</td>
            <td className="px-4 py-3">{p.status === "open" ? <Badge tone="ok">Ouverte</Badge> : <Badge tone="off">Clôturée</Badge>}</td>
            <td className="px-4 py-3">
              {canClose && p.status === "open" && <ActionButton action={closePeriod.bind(null, y.id, p.id)} label="Clôturer" variant="ghost" confirm={`Clôturer la période ${p.startDate} → ${p.endDate} ? Plus aucune écriture ne pourra y être saisie.`} />}
              {canClose && p.status === "closed" && (
                <ActionForm action={requestReopen.bind(null, y.id, p.id)} submit="Demander la réouverture" className="flex flex-wrap items-start gap-2">
                  <input name="reason" required minLength={5} maxLength={500} placeholder="Motif de la réouverture" aria-label="Motif de la réouverture" className={`${inputCls} mt-0 w-64`} />
                </ActionForm>
              )}
            </td>
          </tr>
        ))}
      </Table>
      <p className="text-sm text-slate">Clôturer est refusé s’il reste des brouillons datés dans la période. Rouvrir une période clôturée exige l’approbation d’une autre personne.</p>
    </div>
  );
}
