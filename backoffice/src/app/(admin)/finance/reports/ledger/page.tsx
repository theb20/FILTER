import Link from "next/link";
import { Download } from "lucide-react";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, defaultRange, fiscalYears } from "@/lib/finance";
import { dateFr, fmtBalance, fmtMoney, today } from "@/lib/money";
import { EmptyState, FilterBar, PageHeader, Pagination, Table, btnGhostCls, selectCls } from "@/components/ui/ui";
import type { GeneralLedger } from "@/lib/types";

export const metadata = { title: "Grand livre" };

export default async function LedgerPage({ searchParams }: { searchParams: Promise<{ accountId?: string; from?: string; to?: string; page?: string }> }) {
  const sp = await searchParams;
  const [{ me }, accounts, years] = await Promise.all([currentStaff(), allAccounts("/finance/reports/ledger"), fiscalYears("/finance/reports/ledger")]);
  const range = defaultRange(years, today());
  const from = sp.from || range?.from;
  const to = sp.to || range?.to;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const data = sp.accountId ? await adminGet<GeneralLedger>("/admin/finance/reports/general-ledger", "/finance/reports/ledger", { query: { accountId: sp.accountId, from, to, page: n, pageSize: 50 } }) : null;
  const canExport = can(me?.permissions ?? [], "finance:export");
  const link = (p: number) => `/finance/reports/ledger?${new URLSearchParams({ accountId: sp.accountId ?? "", ...(from ? { from } : {}), ...(to ? { to } : {}), page: String(p) })}`;
  return (
    <div>
      <PageHeader title="Grand livre" subtitle="Mouvements validés d'un compte, avec le solde cumulé (débit − crédit)." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Grand livre" }]}
        action={canExport && data ? <Link href={`/finance/export?kind=ledger&accountId=${sp.accountId}${from ? `&from=${from}` : ""}${to ? `&to=${to}` : ""}`} className={btnGhostCls}><Download className="h-4 w-4" aria-hidden />Exporter (CSV)</Link> : undefined} />
      <FilterBar>
        <select name="accountId" required defaultValue={sp.accountId ?? ""} aria-label="Compte" className={`${selectCls} w-full sm:w-80`}>
          <option value="" disabled>— Choisir un compte —</option>
          {accounts.map((a) => <option key={a.id} value={a.id}>{a.code} — {a.label}</option>)}
        </select>
        <label className="text-sm text-slate">Du <input type="date" name="from" defaultValue={from} className={selectCls} /></label>
        <label className="text-sm text-slate">Au <input type="date" name="to" defaultValue={to} className={selectCls} /></label>
      </FilterBar>
      {!data ? <EmptyState title="Choisissez un compte" hint="Le grand livre liste les écritures validées du compte sur la période." /> : (
        <>
          <p className="mb-3 text-sm text-slate"><span className="font-mono">{data.account.code}</span> {data.account.label} · solde d’ouverture <strong className="text-night">{fmtBalance(data.openingBalance, data.currency)}</strong> · solde de clôture <strong className="text-night">{fmtBalance(data.closingBalance, data.currency)}</strong></p>
          <Table head={["Date", "Numéro", "Libellé", "Débit", "Crédit", "Solde"]} empty={data.items.length === 0 ? "Aucun mouvement validé sur cette période." : undefined}>
            {data.items.map((l, i) => (
              <tr key={`${l.entryId}-${i}`} className="hover:bg-page">
                <td className="whitespace-nowrap px-4 py-2.5 text-slate">{dateFr(l.accountingDate)}</td>
                <td className="whitespace-nowrap px-4 py-2.5 font-mono text-xs"><Link href={`/finance/entries/${l.entryId}`} className="text-link hover:underline">{l.entryNumber}</Link></td>
                <td className="px-4 py-2.5 text-night">{l.label}{l.memo && <span className="block text-xs text-slate">{l.memo}</span>}</td>
                <td className="px-4 py-2.5 text-right tabular-nums">{l.debit ? fmtMoney(l.debit, data.currency) : ""}</td>
                <td className="px-4 py-2.5 text-right tabular-nums">{l.credit ? fmtMoney(l.credit, data.currency) : ""}</td>
                <td className="px-4 py-2.5 text-right tabular-nums font-medium text-night">{fmtBalance(l.balance, data.currency)}</td>
              </tr>
            ))}
          </Table>
          <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="lignes" href={link} />
        </>
      )}
    </div>
  );
}
