import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, inputCls, Notice, PageHeader, Pagination, Table } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { dateFr, type FinApprovalPage } from "@/lib/types";
import { decideApproval } from "../actions";

export const metadata = { title: "Approbations" };
const TYPE = { reopen_period: "Réouverture de période" } as const;
const TONE = { pending: "warn", approved: "ok", rejected: "off" } as const;
const LABEL = { pending: "En attente", approved: "Approuvée", rejected: "Rejetée" } as const;

export default async function ApprovalsPage({ searchParams }: { searchParams: Promise<{ page?: string; ok?: string }> }) {
  const sp = await searchParams;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<FinApprovalPage>("/admin/finance/approvals", "/finance/approvals", { query: { page: n, pageSize: 20 } })]);
  const canDecide = can(me?.permissions ?? [], "finance:close");
  return (
    <div>
      <PageHeader title="Approbations" subtitle="Les opérations sensibles demandent l'accord d'une autre personne que le demandeur, y compris pour le super-administrateur." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Approbations" }]} />
      {sp.ok && <Notice kind="ok">{sp.ok === "approved" ? "Demande approuvée : la période est rouverte." : "Demande rejetée."}</Notice>}
      <Table head={["Demande", "Objet", "Motif", "Demandée le", "État", "Décision"]} empty={data.items.length === 0 ? "Aucune approbation pour l'instant." : undefined} emptyHint="Une demande apparaît ici quand quelqu'un souhaite rouvrir une période clôturée.">
        {data.items.map((a) => (
          <tr key={a.id}>
            <td className="px-4 py-3 font-medium text-night">{TYPE[a.type]}</td>
            <td className="px-4 py-3 text-slate">{a.subjectLabel}</td>
            <td className="max-w-xs px-4 py-3 text-slate">{a.reason}{a.decisionNote && <span className="block text-xs">Note : {a.decisionNote}</span>}</td>
            <td className="whitespace-nowrap px-4 py-3 text-slate">{dateFr(a.requestedAt)}</td>
            <td className="px-4 py-3"><Badge tone={TONE[a.status]}>{LABEL[a.status]}</Badge></td>
            <td className="px-4 py-3">
              {a.status === "pending" && canDecide && (
                a.requestedBy === me?.id ? <span className="text-xs text-slate">Vous êtes le demandeur : une autre personne doit décider.</span> : (
                  <div className="flex flex-wrap items-start gap-2">
                    <ActionForm action={decideApproval.bind(null, a.id, true)} submit="Approuver" className="flex gap-2"><input name="note" maxLength={500} placeholder="Note (facultatif)" aria-label="Note" className={`${inputCls} mt-0 w-40`} /></ActionForm>
                    <ActionForm action={decideApproval.bind(null, a.id, false)} submit="Rejeter" className="flex gap-2"><input type="hidden" name="note" value="" /></ActionForm>
                  </div>
                )
              )}
            </td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="demandes" href={(p) => `/finance/approvals?page=${p}`} />
    </div>
  );
}
