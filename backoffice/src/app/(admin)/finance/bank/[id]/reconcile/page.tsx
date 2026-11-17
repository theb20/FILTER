import Link from "next/link";
import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { dateFr, fmtMoney } from "@/lib/money";
import { Badge, Card, EmptyState, inputCls, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { LINE_STATUS_LABEL, LINE_STATUS_TONE, type BankLine, type BankLinePage, type FinBankAccount, type LedgerCandidate, type LedgerLinePage } from "@/lib/types";
import { reconcile, unreconcile } from "../../../actions";

export const metadata = { title: "Rapprochement bancaire" };

export default async function ReconcilePage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ line?: string; ok?: string }> }) {
  const { id } = await params;
  const sp = await searchParams;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "treasury:reconcile")) redirect("/denied");
  const here = `/finance/bank/${id}/reconcile${sp.line ? `?line=${sp.line}` : ""}`;
  const [b, open] = await Promise.all([adminGet<FinBankAccount>(`/admin/finance/bank-accounts/${id}`, here), adminGet<BankLinePage>(`/admin/finance/bank-accounts/${id}/lines`, here, { query: { pageSize: 50 } })]);
  const todo = open.items.filter((l) => l.status !== "matched");
  const selected: BankLine | undefined = todo.find((l) => l.id === sp.line) ?? open.items.find((l) => l.id === sp.line) ?? todo[0];
  const [cands, ledger] = selected
    ? await Promise.all([adminGet<{ items: LedgerCandidate[] }>(`/admin/finance/bank-statement-lines/${selected.id}/candidates`, here), adminGet<LedgerLinePage>(`/admin/finance/bank-accounts/${id}/ledger-lines`, here, { query: { status: "unmatched", pageSize: 50 } })])
    : [{ items: [] as LedgerCandidate[] }, { items: [] as LedgerCandidate[] }];
  const cur = b.currency;
  const suggested = new Set(cands.items.map((c) => c.entryLineId));
  const others = ledger.items.filter((c) => !suggested.has(c.entryLineId) && selected && (c.amount > 0) === (selected.amount > 0));

  const Match = ({ c, label }: { c: LedgerCandidate; label: string }) => (
    <li className="flex flex-wrap items-center justify-between gap-3 py-3 text-sm">
      <div className="min-w-0">
        <p className="font-medium text-night"><Link href={`/finance/entries/${c.entryId}`} className="hover:underline">{c.entryNumber}</Link> · {c.label}</p>
        <p className="text-xs text-slate">{dateFr(c.accountingDate)}{c.reference ? ` · ${c.reference}` : ""} · reste {fmtMoney(c.remaining, cur)}{c.exactAmount ? " · montant identique" : ""}</p>
      </div>
      {selected && (
        <ActionForm action={reconcile.bind(null, id, selected.id, c.entryLineId)} submit={label} className="flex items-center gap-2">
          <input type="hidden" name="currency" value={cur} />
          <input name="amount" inputMode="decimal" aria-label="Montant à rapprocher (vide = le maximum)" placeholder="max" className={`${inputCls} mt-0 w-28 text-right tabular-nums`} />
        </ActionForm>
      )}
    </li>
  );

  return (
    <div className="space-y-5">
      <PageHeader title="Rapprochement bancaire" subtitle="Choisissez une ligne du relevé, puis confirmez l’écriture correspondante. Les suggestions ne sont jamais appliquées seules ; ce qui n’est pas rapproché reste visible." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Comptes bancaires", href: "/finance/bank" }, { label: b.name, href: `/finance/bank/${id}` }, { label: "Rapprochement" }]} />
      {sp.ok === "matched" && <Notice kind="ok">Rapprochement enregistré.</Notice>}
      {sp.ok === "unmatched" && <Notice kind="ok">Rapprochement retiré : l’écart réapparaît.</Notice>}
      {todo.length === 0 && !selected ? <EmptyState title="Rien à rapprocher" hint="Toutes les lignes importées sont rapprochées, ou aucun relevé n’a encore été importé." /> : (
        <div className="grid gap-5 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
          <Card title={`Lignes du relevé à rapprocher (${todo.length})`}>
            <ul className="divide-y divide-line">
              {todo.map((l) => (
                <li key={l.id}>
                  <Link href={`/finance/bank/${id}/reconcile?line=${l.id}`} className={`block rounded-lg px-2 py-2.5 text-sm transition hover:bg-page ${selected?.id === l.id ? "bg-page ring-1 ring-link/30" : ""}`} aria-current={selected?.id === l.id ? "true" : undefined}>
                    <span className="flex items-center justify-between gap-3"><span className="min-w-0 truncate font-medium text-night">{l.label}</span><span className={`shrink-0 tabular-nums font-medium ${l.amount < 0 ? "text-night" : "text-[#007a52]"}`}>{fmtMoney(l.amount, cur, { sign: true })}</span></span>
                    <span className="text-xs text-slate">{dateFr(l.lineDate)}{l.reference ? ` · ${l.reference}` : ""} · reste {fmtMoney(l.remaining, cur)}</span>
                  </Link>
                </li>
              ))}
            </ul>
          </Card>

          <div className="space-y-5">
            {selected && (
              <Card title="Ligne sélectionnée" action={<Badge tone={LINE_STATUS_TONE[selected.status]}>{LINE_STATUS_LABEL[selected.status]}</Badge>}>
                <p className="text-sm text-night"><strong>{selected.label}</strong> · {dateFr(selected.lineDate)} · <span className="tabular-nums">{fmtMoney(selected.amount, cur, { sign: true })}</span></p>
                <p className="mt-1 text-sm text-slate">Rapproché {fmtMoney(selected.matched, cur)} · reste {fmtMoney(selected.remaining, cur)}</p>
                {selected.matches.length > 0 && (
                  <ul className="mt-3 divide-y divide-line text-sm">
                    {selected.matches.map((m) => (
                      <li key={m.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                        <span><Link href={`/finance/entries/${m.entryId}`} className="text-link hover:underline">{m.entryNumber}</Link> · {fmtMoney(m.amount, cur)}</span>
                        <ActionButton action={unreconcile.bind(null, id, selected.id, m.id)} label="Retirer" variant="ghost" confirm="Retirer ce rapprochement ? L’écart réapparaîtra." />
                      </li>
                    ))}
                  </ul>
                )}
              </Card>
            )}
            {selected && selected.remaining > 0 && (
              <>
                <Card title="Suggestions" description="Même sens, montant et date proches, mots communs. À confirmer une à une.">
                  {cands.items.length === 0 ? <p className="text-sm text-slate">Aucune suggestion.</p> : <ul className="divide-y divide-line">{cands.items.map((c) => <Match key={c.entryLineId} c={c} label="Rapprocher" />)}</ul>}
                </Card>
                {others.length > 0 && <Card title="Autres écritures non rapprochées de même sens"><ul className="divide-y divide-line">{others.map((c) => <Match key={c.entryLineId} c={c} label="Rapprocher" />)}</ul></Card>}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
