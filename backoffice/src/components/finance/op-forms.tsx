import { inputCls } from "@/components/ui/ui";
import { toInput } from "@/lib/money";
import type { FinAccount, FinCategory, FinJournal, FinOperation, FinRecurring, FinThirdParty } from "@/lib/types";

const sel = inputCls;
const Acc = ({ accounts }: { accounts: FinAccount[] }) => <>{accounts.map((a) => <option key={a.id} value={a.id}>{a.code} — {a.label}</option>)}</>;

/** Champs communs d'une opération (dépense ou revenu). Aucun compte, taux ni taxe n'est prérempli : tout est choisi par l'utilisateur. */
export function OperationFields({ kind, currency, categories, accounts, thirdParties, value, withDuplicate = true }: { kind: "expense" | "income"; currency: string; categories: FinCategory[]; accounts: FinAccount[]; thirdParties: FinThirdParty[]; value?: FinOperation; withDuplicate?: boolean }) {
  const exp = kind === "expense";
  return (
    <>
      <input type="hidden" name="kind" value={kind} />
      <input type="hidden" name="currency" value={currency} />
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm font-medium text-night">Date de l’opération<input name="operationDate" type="date" required defaultValue={value?.operationDate} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night">Échéance de paiement <span className="font-normal text-slate">(facultatif)</span><input name="dueDate" type="date" defaultValue={value?.dueDate ?? ""} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night sm:col-span-2">Libellé<input name="label" required maxLength={300} defaultValue={value?.label} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night">{exp ? "Fournisseur" : "Client"} <span className="font-normal text-slate">(facultatif)</span>
          <select name="thirdPartyId" defaultValue={value?.thirdPartyId ?? ""} className={sel}><option value="">— Aucun —</option>{thirdParties.map((t) => <option key={t.id} value={t.id}>{t.displayName}</option>)}</select>
        </label>
        <label className="block text-sm font-medium text-night">Référence de la pièce <span className="font-normal text-slate">(n° de facture…)</span><input name="reference" maxLength={100} defaultValue={value?.reference ?? ""} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night">Catégorie
          <select name="categoryId" required defaultValue={value?.categoryId ?? ""} className={sel}><option value="" disabled>— Choisir —</option>{categories.map((c) => <option key={c.id} value={c.id}>{c.label} ({c.accountCode})</option>)}</select>
          <span className="mt-1 block text-xs font-normal text-slate">Le compte de {exp ? "charge" : "produit"} est celui de la catégorie.</span>
        </label>
        {exp && (
          <label className="block text-sm font-medium text-night">Nature <span className="font-normal text-slate">(reprise de la catégorie si vide)</span>
            <select name="nature" defaultValue={value?.nature ?? ""} className={sel}><option value="">— Selon la catégorie —</option><option value="fixed">Fixe</option><option value="variable">Variable</option></select>
          </label>
        )}
      </div>

      <fieldset className="space-y-3 rounded-lg border border-line p-4">
        <legend className="px-1 text-sm font-semibold text-night">Montants ({currency})</legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-sm font-medium text-night">Montant hors taxes<input name="amountNet" required inputMode="decimal" defaultValue={value ? toInput(value.amountNet, currency) : ""} className={`${inputCls} text-right tabular-nums`} /></label>
          <label className="block text-sm font-medium text-night">Taxe <span className="font-normal text-slate">(montant, 0 si aucune)</span><input name="taxAmount" inputMode="decimal" defaultValue={value ? toInput(value.taxAmount, currency) : ""} className={`${inputCls} text-right tabular-nums`} /></label>
          <label className="block text-sm font-medium text-night sm:col-span-2">Compte de taxe <span className="font-normal text-slate">(obligatoire si la taxe est non nulle)</span>
            <select name="taxAccountId" defaultValue={value?.taxAccountId ?? ""} className={sel}><option value="">— Aucun —</option><Acc accounts={accounts} /></select>
            <span className="mt-1 block text-xs font-normal text-slate">Aucun taux de TVA n’est supposé : saisissez le montant de la taxe et choisissez son compte (règles à valider avec votre expert-comptable).</span>
          </label>
        </div>
      </fieldset>

      <fieldset className="space-y-3 rounded-lg border border-line p-4">
        <legend className="px-1 text-sm font-semibold text-night">Règlement</legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-sm font-medium text-night">Mode
            <select name="settlementMode" required defaultValue={value?.settlementMode ?? ""} className={sel}>
              <option value="" disabled>— Choisir —</option>
              <option value="immediate">Réglée immédiatement (banque ou caisse)</option>
              <option value="on_account">En compte : à régler plus tard ({exp ? "dette" : "créance"})</option>
            </select>
          </label>
          <label className="block text-sm font-medium text-night">Compte de contrepartie
            <select name="counterpartyAccountId" required defaultValue={value?.counterpartyAccountId ?? ""} className={sel}><option value="" disabled>— Choisir —</option><Acc accounts={accounts} /></select>
            <span className="mt-1 block text-xs font-normal text-slate">{exp ? "Compte crédité du montant toutes taxes comprises : fournisseur, banque ou caisse." : "Compte débité du montant toutes taxes comprises : client, banque ou caisse."}</span>
          </label>
        </div>
      </fieldset>

      {withDuplicate && (
        <label className="flex items-start gap-2 text-sm text-slate">
          <input type="checkbox" name="duplicateConfirmed" className="mt-0.5" />
          <span>Ce n’est pas un doublon <span className="text-xs">(à cocher seulement si le service signale une opération identique : même tiers, même référence, mêmes montants ; la confirmation est conservée)</span></span>
        </label>
      )}
    </>
  );
}

export function RecurringFields({ currency, categories, accounts, thirdParties, journals, value }: { currency: string; categories: FinCategory[]; accounts: FinAccount[]; thirdParties: FinThirdParty[]; journals: FinJournal[]; value?: FinRecurring }) {
  return (
    <>
      <input type="hidden" name="currency" value={currency} />
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm font-medium text-night">Type
          <select name="kind" required defaultValue={value?.kind ?? ""} disabled={!!value} className={sel}><option value="" disabled>— Choisir —</option><option value="expense">Dépense</option><option value="income">Revenu</option></select>
          {value && <input type="hidden" name="kind" value={value.kind} />}
        </label>
        <label className="block text-sm font-medium text-night">Catégorie
          <select name="categoryId" required defaultValue={value?.categoryId ?? ""} className={sel}><option value="" disabled>— Choisir —</option>{categories.map((c) => <option key={c.id} value={c.id}>{c.kind === "expense" ? "Dépense" : "Revenu"} · {c.label}</option>)}</select>
        </label>
        <label className="block text-sm font-medium text-night sm:col-span-2">Libellé<input name="label" required maxLength={300} defaultValue={value?.label} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night">Tiers <span className="font-normal text-slate">(facultatif)</span>
          <select name="thirdPartyId" defaultValue={value?.thirdPartyId ?? ""} className={sel}><option value="">— Aucun —</option>{thirdParties.map((t) => <option key={t.id} value={t.id}>{t.displayName}</option>)}</select>
        </label>
        <label className="block text-sm font-medium text-night">Journal des écritures
          <select name="journalId" required defaultValue={value?.journalId ?? ""} className={sel}><option value="" disabled>— Choisir —</option>{journals.map((j) => <option key={j.id} value={j.id}>{j.code} — {j.label}</option>)}</select>
        </label>
        <label className="block text-sm font-medium text-night">Montant hors taxes ({currency})<input name="amountNet" required inputMode="decimal" defaultValue={value ? toInput(value.amountNet, currency) : ""} className={`${inputCls} text-right tabular-nums`} /></label>
        <label className="block text-sm font-medium text-night">Taxe ({currency})<input name="taxAmount" inputMode="decimal" defaultValue={value ? toInput(value.taxAmount, currency) : ""} className={`${inputCls} text-right tabular-nums`} /></label>
        <label className="block text-sm font-medium text-night">Compte de taxe <span className="font-normal text-slate">(si taxe)</span>
          <select name="taxAccountId" defaultValue={value?.taxAccountId ?? ""} className={sel}><option value="">— Aucun —</option><Acc accounts={accounts} /></select>
        </label>
        <label className="block text-sm font-medium text-night">Mode de règlement
          <select name="settlementMode" required defaultValue={value?.settlementMode ?? ""} className={sel}><option value="" disabled>— Choisir —</option><option value="immediate">Réglée immédiatement</option><option value="on_account">En compte</option></select>
        </label>
        <label className="block text-sm font-medium text-night">Compte de contrepartie
          <select name="counterpartyAccountId" required defaultValue={value?.counterpartyAccountId ?? ""} className={sel}><option value="" disabled>— Choisir —</option><Acc accounts={accounts} /></select>
        </label>
        <label className="block text-sm font-medium text-night">Nature <span className="font-normal text-slate">(dépenses)</span>
          <select name="nature" defaultValue={value?.nature ?? ""} className={sel}><option value="">— Selon la catégorie —</option><option value="fixed">Fixe</option><option value="variable">Variable</option></select>
        </label>
        <label className="block text-sm font-medium text-night">Jour du mois <span className="font-normal text-slate">(1 à 28)</span><input name="dayOfMonth" type="number" min={1} max={28} required defaultValue={value?.dayOfMonth ?? ""} className={inputCls} /></label>
        <label className="block text-sm font-medium text-night">Début<input name="startDate" type="date" required defaultValue={value?.startDate} readOnly={!!value} className={`${inputCls} ${value ? "bg-page" : ""}`} /></label>
        <label className="block text-sm font-medium text-night">Fin <span className="font-normal text-slate">(facultatif)</span><input name="endDate" type="date" defaultValue={value?.endDate ?? ""} className={inputCls} /></label>
      </div>
      {value && <label className="flex items-center gap-2 text-sm text-night"><input type="checkbox" name="active" defaultChecked={value.active} /> Modèle actif <span className="text-slate">(un modèle inactif ne génère plus d’opérations)</span></label>}
    </>
  );
}
