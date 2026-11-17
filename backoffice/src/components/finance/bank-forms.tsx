"use client";

import { startTransition, useActionState } from "react";
import { btnCls, btnGhostCls, inputCls, Notice } from "@/components/ui/ui";
import { toInput } from "@/lib/money";
import type { FinAccount, FinBankAccount } from "@/lib/types";
import { importStatement } from "@/app/(admin)/finance/actions";

export function BankFields({ accounts, currency, value }: { accounts: FinAccount[]; currency: string; value?: FinBankAccount }) {
  return (
    <>
      <input type="hidden" name="currency" value={currency} />
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm font-medium text-night">Nom du compte<input name="name" required maxLength={120} defaultValue={value?.name} placeholder="Ex. Compte courant" className={inputCls} /></label>
        <label className="block text-sm font-medium text-night">Banque<input name="bankName" required maxLength={120} defaultValue={value?.bankName} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night sm:col-span-2">Compte du plan comptable lié
          <select name="accountId" required defaultValue={value?.accountId ?? ""} className={inputCls}><option value="" disabled>— Choisir —</option>{accounts.map((a) => <option key={a.id} value={a.id}>{a.code} — {a.label}</option>)}</select>
          <span className="mt-1 block text-xs font-normal text-slate">Un compte comptable = un compte bancaire. Il ne pourra plus changer dès qu’un relevé sera importé.</span>
        </label>
        <label className="block text-sm font-medium text-night">Date d’ouverture<input name="openingDate" type="date" required defaultValue={value?.openingDate} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night">Solde du relevé à cette date ({currency})<input name="openingBalance" inputMode="decimal" defaultValue={value ? toInput(value.openingBalance, currency) : ""} placeholder="0" className={`${inputCls} text-right tabular-nums`} /></label>
        <label className="block text-sm font-medium text-night sm:col-span-2">Numéro de compte <span className="font-normal text-slate">(facultatif)</span>
          <input name="number" autoComplete="off" maxLength={60} placeholder={value?.hasNumber ? `Enregistré : •••• ${value.numberLast4} (laisser vide pour le conserver)` : ""} className={inputCls} />
          <span className="mt-1 block text-xs font-normal text-slate">Chiffré avant d’être enregistré ; jamais réaffiché (seuls les 4 derniers caractères le seront).</span>
        </label>
      </div>
      {value && <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="active" defaultChecked={value.active} /> Compte actif</label>}
    </>
  );
}

/** Import d'un relevé : fichier CSV, « Simuler » d'abord, puis « Importer » (tout ou rien). */
export function StatementImportForm({ accountId, currency }: { accountId: string; currency: string }) {
  const [state, run, pending] = useActionState(importStatement.bind(null, accountId, currency), undefined);
  const go = (mode: "simulate" | "import") => (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    const form = e.currentTarget.form;
    if (!form) return;
    const data = new FormData(form);
    data.set("mode", mode);
    startTransition(() => run(data));
  };
  return (
    <form className="space-y-4">
      {state?.error && <Notice kind="error">{state.error}</Notice>}
      {state?.ok && <Notice kind="ok">{state.ok}</Notice>}
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm font-medium text-night">Fichier CSV<input type="file" name="file" accept=".csv,.txt,text/csv" className="mt-1.5 block w-full text-sm" /></label>
        <label className="block text-sm font-medium text-night">Solde de clôture du relevé <span className="font-normal text-slate">(facultatif)</span><input name="closingBalance" inputMode="decimal" className={`${inputCls} text-right tabular-nums`} /></label>
      </div>
      <details className="text-sm text-slate">
        <summary className="cursor-pointer text-link">Ou coller le contenu du fichier</summary>
        <textarea name="csv" rows={6} placeholder={"date;libelle;reference;montant\n2026-02-03;Virement client;FAC-1;1250000\n2026-02-04;Frais bancaires;;-2500"} className={`${inputCls} mt-2 font-mono text-sm`} />
      </details>
      <p className="text-xs text-slate">Colonnes : date ; libellé ; référence ; montant. Montant signé (négatif = sortie d’argent). Dates AAAA-MM-JJ ou JJ/MM/AAAA. Les lignes déjà importées (relevés qui se recouvrent) sont ignorées ; une ligne en erreur refuse tout l’import.</p>
      <div className="flex flex-wrap gap-3">
        <button type="button" disabled={pending} onClick={go("simulate")} className={btnGhostCls}>{pending ? "Patientez…" : "Simuler"}</button>
        <button type="button" disabled={pending} onClick={go("import")} className={btnCls}>Importer</button>
      </div>
    </form>
  );
}
