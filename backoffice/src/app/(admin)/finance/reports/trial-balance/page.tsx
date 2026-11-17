import Link from "next/link";
import { Download } from "lucide-react";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { defaultRange, fiscalYears } from "@/lib/finance";
import { fmtBalance, fmtMoney, today } from "@/lib/money";
import { Badge, EmptyState, FilterBar, PageHeader, Table, btnGhostCls, selectCls } from "@/components/ui/ui";
import type { TrialBalance } from "@/lib/types";

export const metadata = { title: "Balance" };

export default async function TrialBalancePage({ searchParams }: { searchParams: Promise<{ from?: string; to?: string }> }) {
  const sp = await searchParams;
  const [{ me }, years] = await Promise.all([currentStaff(), fiscalYears("/finance/reports/trial-balance")]);
  const range = defaultRange(years, today());
  const from = sp.from || range?.from;
  const to = sp.to || range?.to;
  const data = from && to ? await adminGet<TrialBalance>("/admin/finance/reports/trial-balance", `/finance/reports/trial-balance?from=${from}&to=${to}`, { query: { from, to } }) : null;
  const canExport = can(me?.permissions ?? [], "finance:export");
  return (
    <div>
      <PageHeader title="Balance des comptes" subtitle="Calculée sur les écritures VALIDÉES uniquement (les brouillons sont exclus). Soldes en débit (D) ou crédit (C)." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Balance" }]}
        action={canExport && from && to ? <Link href={`/finance/export?kind=trial-balance&from=${from}&to=${to}`} className={btnGhostCls}><Download className="h-4 w-4" aria-hidden />Exporter (CSV)</Link> : undefined} />
      <FilterBar>
        <label className="text-sm text-slate">Du <input type="date" name="from" defaultValue={from} className={selectCls} /></label>
        <label className="text-sm text-slate">Au <input type="date" name="to" defaultValue={to} className={selectCls} /></label>
      </FilterBar>
      {!data ? <EmptyState title="Aucun exercice défini" hint="Ouvrez un exercice pour consulter la balance." /> : (
        <>
          <Table head={["Compte", "Libellé", "Solde d'ouverture", "Débit", "Crédit", "Solde de clôture"]} empty={data.rows.length === 0 ? "Aucune écriture validée sur cette période." : undefined}>
            {data.rows.map((r) => (
              <tr key={r.accountId} className="hover:bg-page">
                <td className="px-4 py-2.5 font-mono text-xs"><Link href={`/finance/reports/ledger?accountId=${r.accountId}&from=${data.from}&to=${data.to}`} className="text-link hover:underline">{r.code}</Link></td>
                <td className="px-4 py-2.5 text-night">{r.label}</td>
                <td className="px-4 py-2.5 text-right tabular-nums text-slate">{fmtBalance(r.openingBalance, data.currency)}</td>
                <td className="px-4 py-2.5 text-right tabular-nums">{fmtMoney(r.debit, data.currency)}</td>
                <td className="px-4 py-2.5 text-right tabular-nums">{fmtMoney(r.credit, data.currency)}</td>
                <td className="px-4 py-2.5 text-right tabular-nums font-medium text-night">{fmtBalance(r.closingBalance, data.currency)}</td>
              </tr>
            ))}
          </Table>
          {data.rows.length > 0 && (
            <div className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line bg-white p-4 text-sm shadow-card">
              <span className="font-medium text-night">Totaux : débit {fmtMoney(data.totalDebit, data.currency)} · crédit {fmtMoney(data.totalCredit, data.currency)}</span>
              {data.balanced ? <Badge tone="ok">Balance équilibrée</Badge> : <Badge tone="danger">Balance déséquilibrée : à signaler immédiatement</Badge>}
            </div>
          )}
        </>
      )}
    </div>
  );
}
