import Link from "next/link";
import { adminGet } from "@/lib/admin";
import { dateFr, fmtBalance, fmtMoney } from "@/lib/money";
import { Card, EmptyState, FilterBar, LinkButton, PageHeader, StatCard, Table, selectCls } from "@/components/ui/ui";
import type { CashForecast } from "@/lib/types";

export const metadata = { title: "Prévisions de trésorerie" };

export default async function ForecastPage({ searchParams }: { searchParams: Promise<{ days?: string }> }) {
  const sp = await searchParams;
  const days = ["30", "60", "90", "180"].includes(sp.days ?? "") ? sp.days! : "90";
  const f = await adminGet<CashForecast>("/admin/finance/cash-forecast", `/finance/forecast?days=${days}`, { query: { horizonDays: days } });
  const cur = f.currency;
  const max = Math.max(1, ...f.buckets.flatMap((b) => [b.inflows, b.outflows]));
  return (
    <div className="space-y-5">
      <PageHeader title="Prévisions de trésorerie" subtitle="Projection hebdomadaire des flux CONNUS uniquement : aucune statistique, aucune hypothèse cachée." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Prévisions" }]} />
      <FilterBar>
        <label className="text-sm text-slate">Horizon <select name="days" defaultValue={days} className={selectCls}>{["30", "60", "90", "180"].map((d) => <option key={d} value={d}>{d} jours</option>)}</select></label>
      </FilterBar>
      {f.bankAccounts === 0 ? (
        <EmptyState title="Aucun compte bancaire actif" hint="La prévision part du solde de vos comptes bancaires du grand livre." action={<LinkButton href="/finance/bank/new">Ajouter un compte bancaire</LinkButton>} />
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Solde de départ" value={fmtBalance(f.openingBalance, cur)} hint={`au ${dateFr(f.asOf)} · ${f.bankAccounts} compte(s)`} />
            <StatCard label={`Solde projeté à ${f.horizonDays} jours`} value={fmtBalance(f.buckets.at(-1)?.projectedBalance ?? f.openingBalance, cur)} hint="flux connus datés" />
            <StatCard label="En retard" value={fmtMoney(f.overdueInflows - f.overdueOutflows, cur, { sign: true })} hint={`encaissements ${fmtMoney(f.overdueInflows, cur)} · décaissements ${fmtMoney(f.overdueOutflows, cur)}`} tone={f.overdueOutflows > 0 ? "alert" : "default"} />
            <StatCard label="Sans échéance" value={fmtMoney(f.undatedInflows - f.undatedOutflows, cur, { sign: true })} hint={`encaissements ${fmtMoney(f.undatedInflows, cur)} · décaissements ${fmtMoney(f.undatedOutflows, cur)}`} />
          </div>
          <Table head={["Semaine", "Encaissements", "Décaissements", "Solde projeté"]}>
            {f.buckets.map((b) => (
              <tr key={b.startDate}>
                <td className="whitespace-nowrap px-4 py-3 text-night">{dateFr(b.startDate)} → {dateFr(b.endDate)}</td>
                <td className="px-4 py-3"><div className="flex items-center justify-end gap-3"><span className="h-2 rounded bg-ok/60" style={{ width: `${Math.round((b.inflows / max) * 120)}px` }} aria-hidden /><span className="tabular-nums text-[#007a52]">{b.inflows ? fmtMoney(b.inflows, cur) : "—"}</span></div></td>
                <td className="px-4 py-3"><div className="flex items-center justify-end gap-3"><span className="h-2 rounded bg-danger/50" style={{ width: `${Math.round((b.outflows / max) * 120)}px` }} aria-hidden /><span className="tabular-nums text-night">{b.outflows ? fmtMoney(b.outflows, cur) : "—"}</span></div></td>
                <td className={`whitespace-nowrap px-4 py-3 text-right tabular-nums font-medium ${b.projectedBalance < 0 ? "text-danger" : "text-night"}`}>{fmtBalance(b.projectedBalance, cur)}</td>
              </tr>
            ))}
          </Table>
          <Card title="Hypothèses et limites" description="Ce que cette prévision contient, et ce qu’elle ne contient pas.">
            <ul className="list-disc space-y-1.5 pl-5 text-sm text-slate">
              {f.assumptions.map((a) => <li key={a}>{a}</li>)}
              {f.excludedDrafts > 0 && <li>{f.excludedDrafts} opération(s) non comptabilisée(s) (brouillons) exclue(s), pour {fmtMoney(f.excludedDraftsAmount, cur, { sign: true })} net : <Link href="/finance/operations?status=draft" className="text-link hover:underline">les voir</Link>.</li>}
            </ul>
          </Card>
        </>
      )}
    </div>
  );
}
