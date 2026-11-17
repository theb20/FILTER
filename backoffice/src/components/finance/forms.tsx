"use client";

import { startTransition, useActionState } from "react";
import { btnCls, btnGhostCls, inputCls, Notice } from "@/components/ui/ui";
import { ACCOUNT_KIND_LABEL, JOURNAL_KIND_LABEL, type AccountKind, type FinAccount, type FinJournal } from "@/lib/types";
import { importAccounts } from "@/app/(admin)/finance/actions";

export function AccountFields({ value }: { value?: FinAccount }) {
  return (
    <>
      <label className="block text-sm font-medium text-night">Code
        <input name="code" required defaultValue={value?.code} readOnly={!!value} pattern="[A-Za-z0-9][A-Za-z0-9._\-]{0,19}" maxLength={20} className={`${inputCls} ${value ? "bg-page" : ""}`} />
        <span className="mt-1 block text-xs font-normal text-slate">{value ? "Le code ne peut pas changer." : "Lettres, chiffres, point, tiret ou souligné (20 caractères maximum). Ne pourra plus changer."}</span>
      </label>
      <label className="block text-sm font-medium text-night">Libellé<input name="label" required maxLength={200} defaultValue={value?.label} className={inputCls} /></label>
      <label className="block text-sm font-medium text-night">Nature
        <select name="kind" required defaultValue={value?.kind ?? ""} className={inputCls}>
          <option value="" disabled>— Choisir —</option>
          {(Object.keys(ACCOUNT_KIND_LABEL) as AccountKind[]).map((k) => <option key={k} value={k}>{ACCOUNT_KIND_LABEL[k]}</option>)}
        </select>
        {value && <span className="mt-1 block text-xs font-normal text-slate">La nature ne peut plus changer dès que le compte porte des écritures.</span>}
      </label>
      <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="active" defaultChecked={value?.active ?? true} /> Compte actif <span className="text-slate">(un compte inactif ne peut plus recevoir d’écritures)</span></label>
    </>
  );
}

export function JournalFields({ value }: { value?: FinJournal }) {
  return (
    <>
      <label className="block text-sm font-medium text-night">Code
        <input name="code" required defaultValue={value?.code} readOnly={!!value} pattern="[A-Z0-9]{1,10}" maxLength={10} className={`${inputCls} ${value ? "bg-page" : ""}`} />
        <span className="mt-1 block text-xs font-normal text-slate">Majuscules et chiffres (ex. VT, AC, BQ). Il figure dans les numéros d’écriture : {value ? "non modifiable." : "ne pourra plus changer."}</span>
      </label>
      <label className="block text-sm font-medium text-night">Libellé<input name="label" required maxLength={200} defaultValue={value?.label} className={inputCls} /></label>
      <label className="block text-sm font-medium text-night">Nature
        <select name="kind" required defaultValue={value?.kind ?? ""} className={inputCls}>
          <option value="" disabled>— Choisir —</option>
          {(Object.keys(JOURNAL_KIND_LABEL) as FinJournal["kind"][]).map((k) => <option key={k} value={k}>{JOURNAL_KIND_LABEL[k]}</option>)}
        </select>
      </label>
      <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="active" defaultChecked={value?.active ?? true} /> Journal actif</label>
    </>
  );
}

/** Import CSV du plan comptable : « Simuler » (par défaut) puis « Importer ». Le rapport s'affiche sous le formulaire. */
export function ImportForm() {
  const [state, run, pending] = useActionState(importAccounts, undefined);
  const submit = (mode: "simulate" | "import") => (e: React.MouseEvent<HTMLButtonElement>) => {
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
      <label className="block text-sm font-medium text-night">Contenu du fichier
        <textarea name="csv" rows={12} required placeholder={"code;libelle;nature\n512;Banque;asset\n706;Prestations de services;income"} className={`${inputCls} font-mono text-sm`} />
        <span className="mt-1 block text-xs font-normal text-slate">Une ligne par compte : code ; libellé ; nature (asset, liability, equity, income, expense, other). Séparateur « ; » ou « , ». Un code déjà présent est ignoré, jamais écrasé.</span>
      </label>
      <div className="flex flex-wrap gap-3">
        <button type="button" disabled={pending} onClick={submit("simulate")} className={btnGhostCls}>{pending ? "Patientez…" : "Simuler"}</button>
        <button type="button" disabled={pending} onClick={submit("import")} className={btnCls}>Importer</button>
      </div>
    </form>
  );
}
