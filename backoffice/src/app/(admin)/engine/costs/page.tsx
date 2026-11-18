import Link from "next/link";
import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { fromMicros } from "@/lib/engine";
import { EmptyState, FilterBar, PageHeader, Table } from "@/components/ui/ui";
import type { SourceCostReport } from "@/lib/types";

export const metadata = { title: "Coûts des sources" };

const day = (d: Date) => d.toISOString().slice(0, 10);
/** Jour courant décalé de n jours (yyyy-mm-dd), UTC. */
function dayShift(n: number) { return day(new Date(Date.now() + n * 86400000)); }
const isDay = (v?: string) => !!v && /^\d{4}-\d{2}-\d{2}$/.test(v);

export default async function CostsPage({ searchParams }: { searchParams: Promise<{ from?: string; to?: string }> }) {
  const sp = await searchParams;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "engine:read")) redirect("/denied");
  const to = isDay(sp.to) ? sp.to! : dayShift(0);
  const from = isDay(sp.from) ? sp.from! : dayShift(-29);
  const report = await adminGet<SourceCostReport>("/admin/source-costs", "/engine/costs", { query: { from: `${from}T00:00:00Z`, to: `${to}T23:59:59Z` } });
  // Les devises ne sont jamais additionnées entre elles : un total par devise.
  const totals = new Map<string, number>();
  for (const l of report.lines) totals.set(l.currency, (totals.get(l.currency) ?? 0) + l.costMicros);

  return (
    <div>
      <PageHeader title="Coûts des sources" subtitle="Ce que les appels aux sources externes ont coûté sur la période, source par source." />
      <FilterBar reset={{ href: "/engine/costs", show: isDay(sp.from) || isDay(sp.to) }}>
        <label className="text-sm text-slate">Du <input type="date" name="from" defaultValue={from} className="ml-1 rounded-lg border border-line bg-white px-3 py-2 text-sm text-night" /></label>
        <label className="text-sm text-slate">au <input type="date" name="to" defaultValue={to} className="ml-1 rounded-lg border border-line bg-white px-3 py-2 text-sm text-night" /></label>
      </FilterBar>
      {report.lines.length === 0 ? (
        <EmptyState title="Aucun appel sur la période" hint="Les coûts apparaissent dès qu'une source externe est appelée." action={<Link href="/engine/sources" className="text-link hover:underline">Voir les sources</Link>} />
      ) : (
        <>
          <Table head={["Source", "Appels", "Depuis le cache", "En échec", "Coût", "Coût par recherche"]}>
            {report.lines.map((l) => (
              <tr key={l.sourceId + l.currency} className="hover:bg-page">
                <td className="px-4 py-3 font-medium text-night"><Link href={`/engine/sources/${l.sourceId}`} className="hover:underline">{l.sourceName}</Link></td>
                <td className="px-4 py-3 text-slate tabular-nums">{l.calls}</td>
                <td className="px-4 py-3 text-slate tabular-nums">{l.cachedCalls}</td>
                <td className="px-4 py-3 text-slate tabular-nums">{l.failedCalls}</td>
                <td className="px-4 py-3 font-medium text-night tabular-nums">{fromMicros(l.costMicros, l.currency)}</td>
                <td className="px-4 py-3 text-slate tabular-nums">{fromMicros(l.costPerSearchMicros, l.currency)}</td>
              </tr>
            ))}
          </Table>
          <div className="mt-4 flex flex-wrap items-center gap-4 rounded-xl border border-line bg-white p-4">
            <p className="text-sm font-semibold text-night">Total de la période</p>
            {[...totals].map(([cur, micros]) => <p key={cur} className="text-lg font-bold tabular-nums text-night">{fromMicros(micros, cur)}</p>)}
            <p className="ml-auto text-xs text-slate">{report.note}</p>
          </div>
        </>
      )}
    </div>
  );
}
