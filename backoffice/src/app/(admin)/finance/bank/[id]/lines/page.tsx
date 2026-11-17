import Link from "next/link";
import { adminGet } from "@/lib/admin";
import { dateFr, fmtMoney } from "@/lib/money";
import { Badge, FilterBar, PageHeader, Pagination, SearchInput, Table, selectCls } from "@/components/ui/ui";
import { LINE_STATUS_LABEL, LINE_STATUS_TONE, type BankLinePage, type FinBankAccount } from "@/lib/types";

export const metadata = { title: "Mouvements du relevé" };
type SP = { q?: string; page?: string; status?: string; from?: string; to?: string };

export default async function BankLinesPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<SP> }) {
  const { id } = await params;
  const sp = await searchParams;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const status = sp.status && sp.status in LINE_STATUS_LABEL ? sp.status : undefined;
  const here = `/finance/bank/${id}/lines`;
  const [b, data] = await Promise.all([adminGet<FinBankAccount>(`/admin/finance/bank-accounts/${id}`, here), adminGet<BankLinePage>(`/admin/finance/bank-accounts/${id}/lines`, here, { query: { q: sp.q, page: n, pageSize: 25, status, from: sp.from || undefined, to: sp.to || undefined } })]);
  const keep = { ...(sp.q ? { q: sp.q } : {}), ...(status ? { status } : {}), ...(sp.from ? { from: sp.from } : {}), ...(sp.to ? { to: sp.to } : {}) };
  return (
    <div>
      <PageHeader title="Mouvements du relevé" subtitle={`${b.name} · lignes importées, avec leur état de rapprochement.`} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Comptes bancaires", href: "/finance/bank" }, { label: b.name, href: `/finance/bank/${id}` }, { label: "Mouvements" }]} />
      <FilterBar reset={{ href: here, show: Object.keys(keep).length > 0 }}>
        <SearchInput defaultValue={sp.q} placeholder="Libellé ou référence…" className="w-full sm:w-64" />
        <select name="status" defaultValue={status ?? ""} aria-label="État" className={selectCls}><option value="">Tous les états</option>{Object.entries(LINE_STATUS_LABEL).map(([k, v]) => <option key={k} value={k}>{v}</option>)}</select>
        <input type="date" name="from" defaultValue={sp.from} aria-label="Du" className={selectCls} /><input type="date" name="to" defaultValue={sp.to} aria-label="Au" className={selectCls} />
      </FilterBar>
      <Table head={["Date", "Libellé", "Référence", "Montant", "Rapproché", "Reste", "État", ""]} empty={data.items.length === 0 ? "Aucun mouvement." : undefined} emptyHint="Importez un relevé depuis la fiche du compte.">
        {data.items.map((l) => (
          <tr key={l.id} className="hover:bg-page">
            <td className="whitespace-nowrap px-4 py-3 text-slate">{dateFr(l.lineDate)}</td>
            <td className="px-4 py-3 text-night">{l.label}</td>
            <td className="px-4 py-3 text-slate">{l.reference ?? "—"}</td>
            <td className={`whitespace-nowrap px-4 py-3 text-right tabular-nums font-medium ${l.amount < 0 ? "text-night" : "text-[#007a52]"}`}>{fmtMoney(l.amount, b.currency, { sign: true })}</td>
            <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums text-slate">{fmtMoney(l.matched, b.currency)}</td>
            <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums text-slate">{l.remaining ? fmtMoney(l.remaining, b.currency) : "—"}</td>
            <td className="px-4 py-3"><Badge tone={LINE_STATUS_TONE[l.status]}>{LINE_STATUS_LABEL[l.status]}</Badge></td>
            <td className="px-4 py-3 text-right">{l.status !== "matched" && <Link href={`/finance/bank/${id}/reconcile?line=${l.id}`} className="text-link hover:underline">Rapprocher</Link>}</td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="mouvements" href={(p) => `${here}?${new URLSearchParams({ ...keep, page: String(p) })}`} />
    </div>
  );
}
