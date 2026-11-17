import { inputCls } from "@/components/ui/ui";
import type { FinAccount, FinCategory, FinThirdParty } from "@/lib/types";

export function ThirdPartyFields({ value }: { value?: FinThirdParty }) {
  return (
    <>
      <label className="block text-sm font-medium text-night">Nature
        <select name="kind" required defaultValue={value?.kind ?? ""} disabled={!!value} className={inputCls}><option value="" disabled>— Choisir —</option><option value="supplier">Fournisseur</option><option value="customer">Client</option></select>
        {value && <input type="hidden" name="kind" value={value.kind} />}
        {value && <span className="mt-1 block text-xs font-normal text-slate">La nature ne peut pas changer.</span>}
      </label>
      <label className="block text-sm font-medium text-night">Nom<input name="displayName" required maxLength={200} defaultValue={value?.displayName} className={inputCls} /></label>
      <label className="block text-sm font-medium text-night">Référence d’identité <span className="font-normal text-slate">(facultatif)</span>
        <input name="externalRef" maxLength={100} defaultValue={value?.externalRef ?? ""} className={inputCls} />
        <span className="mt-1 block text-xs font-normal text-slate">Identifiant du client dans la plateforme : la finance garde cette référence, jamais une copie de ses données personnelles.</span>
      </label>
      {value && <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="active" defaultChecked={value.active} /> Actif</label>}
    </>
  );
}

export function CategoryFields({ accounts, value }: { accounts: FinAccount[]; value?: FinCategory }) {
  return (
    <>
      <label className="block text-sm font-medium text-night">Type
        <select name="kind" required defaultValue={value?.kind ?? ""} disabled={!!value} className={inputCls}><option value="" disabled>— Choisir —</option><option value="expense">Dépense</option><option value="income">Revenu</option></select>
        {value && <input type="hidden" name="kind" value={value.kind} />}
      </label>
      <label className="block text-sm font-medium text-night">Libellé<input name="label" required maxLength={120} defaultValue={value?.label} className={inputCls} /></label>
      <label className="block text-sm font-medium text-night">Compte du plan comptable
        <select name="accountId" required defaultValue={value?.accountId ?? ""} className={inputCls}><option value="" disabled>— Choisir —</option>{accounts.map((a) => <option key={a.id} value={a.id}>{a.code} — {a.label}</option>)}</select>
        <span className="mt-1 block text-xs font-normal text-slate">Compte débité (dépense) ou crédité (revenu) par les écritures générées. Aucune correspondance n’est préremplie : elle se définit avec votre expert-comptable.</span>
      </label>
      <label className="block text-sm font-medium text-night">Nature <span className="font-normal text-slate">(dépenses : fixe ou variable)</span>
        <select name="nature" defaultValue={value?.nature ?? ""} className={inputCls}><option value="">— Non précisée —</option><option value="fixed">Fixe</option><option value="variable">Variable</option></select>
      </label>
      {value && <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="active" defaultChecked={value.active} /> Active</label>}
    </>
  );
}
