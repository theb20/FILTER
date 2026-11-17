import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { fmtMoney, dateFr } from "@/lib/money";
import { Badge, FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table, selectCls } from "@/components/ui/ui";
import { KIND_LABEL, OP_STATUS_LABEL, OP_STATUS_TONE, type FinOperationPage, type OpStatus } from "@/lib/types";

export const metadata = { title: "Dépenses et revenus" };
type SP = { q?: string; page?: string; kind?: string; status?: string; from?: string; to?: string; ok?: string };

export default async function OperationsPage({ searchParams }: { searchParams: Promise<SP> }) {
  const sp = await searchParams;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const kind = sp.kind === "expense" || sp.kind === "income" ? sp.kind : undefined;
  const status = sp.status && sp.status in OP_STATUS_LABEL ? sp.status : undefined;
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<FinOperationPage>("/admin/finance/operations", "/finance/operations", { query: { q: sp.q, page: n, pageSize: 20, kind, status, from: sp.from || undefined, to: sp.to || undefined } })]);
  const canWrite = can(me?.permissions ?? [], "finance:write");
  const keep = { ...(sp.q ? { q: sp.q } : {}), ...(kind ? { kind } : {}), ...(status ? { status } : {}), ...(sp.from ? { from: sp.from } : {}), ...(sp.to ? { to: sp.to } : {}) };
  const link = (p: number) => `/finance/operations?${new URLSearchParams({ ...keep, page: String(p) })}`;
  const filtered = Object.keys(keep).length > 0;
  return (
    <div>
      <PageHeader title="Dépenses et revenus" subtitle="Chaque opération génère une écriture en brouillon, validée ensuite par une autre personne." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Dépenses et revenus" }]}
        action={canWrite && <><LinkButton href="/finance/operations/new?kind=income" variant="ghost">Nouveau revenu</LinkButton><LinkButton href="/finance/operations/new?kind=expense">Nouvelle dépense</LinkButton></>} />
      {sp.ok === "deleted" && <Notice kind="ok">Opération supprimée.</Notice>}
      <FilterBar reset={{ href: "/finance/operations", show: filtered }}>
        <SearchInput defaultValue={sp.q} placeholder="Libellé, référence, tiers…" className="w-full sm:w-64" />
        <select name="kind" defaultValue={kind ?? ""} aria-label="Type" className={selectCls}><option value="">Dépenses et revenus</option><option value="expense">Dépenses</option><option value="income">Revenus</option></select>
        <select name="status" defaultValue={status ?? ""} aria-label="Statut" className={selectCls}><option value="">Tous les statuts</option>{(Object.keys(OP_STATUS_LABEL) as OpStatus[]).map((s) => <option key={s} value={s}>{OP_STATUS_LABEL[s]}</option>)}</select>
        <input type="date" name="from" defaultValue={sp.from} aria-label="Du" className={selectCls} /><input type="date" name="to" defaultValue={sp.to} aria-label="Au" className={selectCls} />
      </FilterBar>
      <Table head={["Date", "Opération", "Catégorie", "Tiers", "Hors taxes", "Toutes taxes", "Statut"]} empty={data.items.length === 0 ? (filtered ? "Aucune opération ne correspond." : "Aucune opération pour l'instant.") : undefined}
        emptyHint={data.items.length === 0 && !filtered ? "Enregistrez une dépense ou un revenu pour commencer." : undefined}>
        {data.items.map((o) => (
          <tr key={o.id} className="hover:bg-page">
            <td className="whitespace-nowrap px-4 py-3 text-slate">{dateFr(o.operationDate)}</td>
            <td className="px-4 py-3 font-medium text-night"><Link href={`/finance/operations/${o.id}`} className="hover:underline">{o.label}</Link><span className="ml-2 text-xs font-normal text-slate">{KIND_LABEL[o.kind]}{o.reference ? ` · ${o.reference}` : ""}</span></td>
            <td className="px-4 py-3 text-slate">{o.categoryLabel}</td>
            <td className="px-4 py-3 text-slate">{o.thirdPartyName ?? "—"}</td>
            <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums">{fmtMoney(o.amountNet, o.currency)}</td>
            <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums text-night">{fmtMoney(o.amountGross, o.currency)}</td>
            <td className="px-4 py-3"><Badge tone={OP_STATUS_TONE[o.status]}>{OP_STATUS_LABEL[o.status]}</Badge></td>
          </tr>
        ))}
      </Table>
      {data.items.length > 0 && <p className="mt-3 text-sm text-slate">Total de la sélection ({data.page.total} opération{data.page.total > 1 ? "s" : ""}) : <strong className="text-night">{fmtMoney(data.totalNet, data.currency)}</strong> hors taxes · taxes {fmtMoney(data.totalTax, data.currency)} · <strong className="text-night">{fmtMoney(data.totalGross, data.currency)}</strong> toutes taxes. Les brouillons sans écriture y sont inclus : filtrez sur « Comptabilisée » pour les seuls montants comptabilisés.</p>}
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="opérations" href={link} />
    </div>
  );
}
