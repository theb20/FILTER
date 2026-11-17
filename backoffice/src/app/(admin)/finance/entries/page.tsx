import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { journals } from "@/lib/finance";
import { dateFr, fmtMoney } from "@/lib/money";
import { Badge, FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table, selectCls } from "@/components/ui/ui";
import type { FinEntryPage } from "@/lib/types";

export const metadata = { title: "Écritures" };
type SP = { q?: string; page?: string; status?: string; journalId?: string; from?: string; to?: string; ok?: string };

export default async function EntriesPage({ searchParams }: { searchParams: Promise<SP> }) {
  const sp = await searchParams;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const status = sp.status === "draft" || sp.status === "posted" ? sp.status : undefined;
  const [{ me }, data, jrn] = await Promise.all([
    currentStaff(),
    adminGet<FinEntryPage>("/admin/finance/entries", "/finance/entries", { query: { q: sp.q, page: n, pageSize: 20, status, journalId: sp.journalId || undefined, from: sp.from || undefined, to: sp.to || undefined } }),
    journals("/finance/entries"),
  ]);
  const canWrite = can(me?.permissions ?? [], "finance:write");
  const keep = { ...(sp.q ? { q: sp.q } : {}), ...(status ? { status } : {}), ...(sp.journalId ? { journalId: sp.journalId } : {}), ...(sp.from ? { from: sp.from } : {}), ...(sp.to ? { to: sp.to } : {}) };
  const link = (p: number) => `/finance/entries?${new URLSearchParams({ ...keep, page: String(p) })}`;
  return (
    <div>
      <PageHeader title="Écritures" subtitle="Journal des écritures comptables. Une écriture validée est définitive." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Écritures" }]}
        action={canWrite && <LinkButton href="/finance/entries/new">Saisir une écriture</LinkButton>} />
      {sp.ok === "deleted" && <Notice kind="ok">Brouillon supprimé.</Notice>}
      <FilterBar reset={{ href: "/finance/entries", show: Object.keys(keep).length > 0 }}>
        <SearchInput defaultValue={sp.q} placeholder="Libellé, référence, numéro…" className="w-full sm:w-64" />
        <select name="status" defaultValue={status ?? ""} aria-label="Statut" className={selectCls}><option value="">Tous les statuts</option><option value="draft">Brouillons</option><option value="posted">Validées</option></select>
        <select name="journalId" defaultValue={sp.journalId ?? ""} aria-label="Journal" className={selectCls}><option value="">Tous les journaux</option>{jrn.map((j) => <option key={j.id} value={j.id}>{j.code}</option>)}</select>
        <input type="date" name="from" defaultValue={sp.from} aria-label="Du" className={selectCls} />
        <input type="date" name="to" defaultValue={sp.to} aria-label="Au" className={selectCls} />
      </FilterBar>
      <Table head={["Numéro", "Date", "Libellé", "Journal", "Montant", "Statut"]} empty={data.items.length === 0 ? (Object.keys(keep).length ? "Aucune écriture ne correspond." : "Aucune écriture pour l'instant.") : undefined}
        emptyHint={data.items.length === 0 && !Object.keys(keep).length ? "Les écritures apparaîtront ici dès la première saisie." : undefined}>
        {data.items.map((e) => (
          <tr key={e.id} className="hover:bg-page">
            <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-slate">{e.number ?? "—"}</td>
            <td className="whitespace-nowrap px-4 py-3 text-slate">{dateFr(e.accountingDate)}</td>
            <td className="px-4 py-3 font-medium text-night"><Link href={`/finance/entries/${e.id}`} className="hover:underline">{e.label}</Link>{e.origin === "reversal" && <span className="ml-2 text-xs font-normal text-slate">contre-passation</span>}</td>
            <td className="px-4 py-3 text-slate">{e.journalCode}</td>
            <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums text-night">{fmtMoney(e.totalDebit, e.currency)}</td>
            <td className="px-4 py-3">{e.status === "posted" ? <Badge tone="ok">Validée</Badge> : <Badge tone="warn">Brouillon</Badge>}</td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="écritures" href={link} />
    </div>
  );
}
