"use client";

import { useMemo, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { exponent, fmtMoney, parseMinor, toInput } from "@/lib/money";
import { btnGhostCls, inputCls } from "@/components/ui/ui";
import type { FinAccount, FinEntry, FinJournal } from "@/lib/types";

type Line = { accountId: string; debit: string; credit: string; memo: string };
const empty = (): Line => ({ accountId: "", debit: "", credit: "", memo: "" });

/** Saisie d'une écriture : lignes dynamiques, totaux et équilibre calculés en entiers (BigInt), jamais en flottant. */
export function EntryEditor({ accounts, journals, currency, entry, defaultDate }: { accounts: FinAccount[]; journals: FinJournal[]; currency: string; entry?: FinEntry; defaultDate: string }) {
  const [lines, setLines] = useState<Line[]>(
    entry ? entry.lines.map((l) => ({ accountId: l.accountId, debit: toInput(l.debit, currency), credit: toInput(l.credit, currency), memo: l.memo ?? "" })) : [empty(), empty()],
  );
  const set = (i: number, patch: Partial<Line>) => setLines((ls) => ls.map((l, k) => (k === i ? { ...l, ...patch } : l)));

  const t = useMemo(() => {
    let d = 0n, c = 0n, bad = false;
    for (const l of lines) {
      const pd = parseMinor(l.debit, currency), pc = parseMinor(l.credit, currency);
      if (pd === null || pc === null) bad = true;
      else { d += pd; c += pc; }
    }
    return { d, c, bad, diff: d - c };
  }, [lines, currency]);

  const balanced = !t.bad && t.d === t.c && t.d > 0n;
  const e = exponent(currency);

  return (
    <div className="space-y-5">
      <input type="hidden" name="currency" value={currency} />
      <input type="hidden" name="lines" value={JSON.stringify(lines)} />
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm font-medium text-night">Journal
          <select name="journalId" required defaultValue={entry?.journalId ?? ""} className={inputCls}>
            <option value="" disabled>— Choisir —</option>
            {journals.map((j) => <option key={j.id} value={j.id}>{j.code} — {j.label}</option>)}
          </select>
        </label>
        <label className="block text-sm font-medium text-night">Date comptable
          <input name="accountingDate" type="date" required defaultValue={entry?.accountingDate ?? defaultDate} className={inputCls} />
        </label>
        <label className="block text-sm font-medium text-night sm:col-span-2">Libellé
          <input name="label" required maxLength={300} defaultValue={entry?.label ?? ""} className={inputCls} />
        </label>
        <label className="block text-sm font-medium text-night sm:col-span-2">Référence de la pièce <span className="font-normal text-slate">(facultatif)</span>
          <input name="reference" maxLength={100} defaultValue={entry?.reference ?? ""} className={inputCls} />
        </label>
      </div>

      <div>
        <div className="overflow-x-auto rounded-xl border border-line">
          <table className="w-full min-w-[720px] text-sm">
            <thead className="bg-page/70 text-left text-xs uppercase tracking-wide text-slate">
              <tr><th className="px-3 py-2 font-semibold">Compte</th><th className="w-40 px-3 py-2 font-semibold">Débit ({currency})</th><th className="w-40 px-3 py-2 font-semibold">Crédit ({currency})</th><th className="px-3 py-2 font-semibold">Note</th><th className="w-10" /></tr>
            </thead>
            <tbody className="divide-y divide-line">
              {lines.map((l, i) => (
                <tr key={i}>
                  <td className="p-2">
                    <select aria-label={`Compte, ligne ${i + 1}`} value={l.accountId} onChange={(ev) => set(i, { accountId: ev.target.value })} className={inputCls.replace("mt-1.5", "mt-0")}>
                      <option value="">— Compte —</option>
                      {accounts.map((a) => <option key={a.id} value={a.id}>{a.code} — {a.label}</option>)}
                    </select>
                  </td>
                  <td className="p-2"><input aria-label={`Débit, ligne ${i + 1}`} inputMode="decimal" value={l.debit} onChange={(ev) => set(i, { debit: ev.target.value, ...(ev.target.value ? { credit: "" } : {}) })} className={`${inputCls.replace("mt-1.5", "mt-0")} text-right tabular-nums`} /></td>
                  <td className="p-2"><input aria-label={`Crédit, ligne ${i + 1}`} inputMode="decimal" value={l.credit} onChange={(ev) => set(i, { credit: ev.target.value, ...(ev.target.value ? { debit: "" } : {}) })} className={`${inputCls.replace("mt-1.5", "mt-0")} text-right tabular-nums`} /></td>
                  <td className="p-2"><input aria-label={`Note, ligne ${i + 1}`} value={l.memo} maxLength={300} onChange={(ev) => set(i, { memo: ev.target.value })} className={inputCls.replace("mt-1.5", "mt-0")} /></td>
                  <td className="p-2 text-center">
                    {lines.length > 2 && <button type="button" onClick={() => setLines((ls) => ls.filter((_, k) => k !== i))} aria-label={`Retirer la ligne ${i + 1}`} className="rounded p-1.5 text-slate hover:bg-danger/10 hover:text-danger"><Trash2 className="h-4 w-4" aria-hidden /></button>}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot className="border-t border-line bg-page/50 font-medium tabular-nums">
              <tr>
                <td className="px-3 py-2 text-right text-slate">Totaux</td>
                <td className="px-3 py-2 text-right">{t.bad ? "—" : fmtMoney(t.d, currency)}</td>
                <td className="px-3 py-2 text-right">{t.bad ? "—" : fmtMoney(t.c, currency)}</td>
                <td colSpan={2} className="px-3 py-2" />
              </tr>
            </tfoot>
          </table>
        </div>
        <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
          <button type="button" onClick={() => setLines((ls) => [...ls, empty()])} className={btnGhostCls}><Plus className="h-4 w-4" aria-hidden />Ajouter une ligne</button>
          <p role="status" className={`text-sm font-medium ${t.bad ? "text-danger" : balanced ? "text-[#007a52]" : "text-[#7a5a00]"}`}>
            {t.bad
              ? `Un montant est invalide (nombre positif, ${e === 0 ? "sans décimale" : `${e} décimales au plus`}).`
              : balanced
                ? "Écriture équilibrée : prête à être validée."
                : t.d === 0n && t.c === 0n
                  ? "Saisissez les montants."
                  : `Non équilibrée : écart de ${fmtMoney(t.diff < 0n ? -t.diff : t.diff, currency)} (${t.diff > 0n ? "débit en excès" : "crédit en excès"}). Un brouillon peut être enregistré ainsi, mais pas validé.`}
          </p>
        </div>
      </div>
    </div>
  );
}
