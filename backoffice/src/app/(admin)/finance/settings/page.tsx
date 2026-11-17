import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { financeSettings } from "@/lib/finance";
import { fmtMoney, toInput } from "@/lib/money";
import { Card, inputCls, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { putSettings } from "../actions";

export const metadata = { title: "Réglages de la finance" };

export default async function FinanceSettingsPage() {
  const [{ me }, s] = await Promise.all([currentStaff(), financeSettings("/finance/settings")]);
  const admin = can(me?.permissions ?? [], "finance:admin");
  return (
    <div className="max-w-2xl space-y-5">
      <PageHeader title="Réglages de la finance" breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Réglages" }]} />
      <Card title="Devise et double validation">
        {admin ? (
          <ActionForm action={putSettings} submit="Enregistrer">
            <label className="block text-sm font-medium text-night">Devise de la comptabilité
              <input name="baseCurrency" required pattern="[A-Z]{3}" defaultValue={s.baseCurrency} className={inputCls} />
              <span className="mt-1 block text-xs font-normal text-slate">Code ISO sur 3 lettres. Ne peut plus changer dès qu’une écriture existe.</span>
            </label>
            <label className="block text-sm font-medium text-night">Seuil de double validation
              <input name="fourEyesMinAmount" inputMode="decimal" defaultValue={toInput(s.fourEyesMinAmount, s.baseCurrency)} placeholder="0 = toujours" className={inputCls} />
              <span className="mt-1 block text-xs font-normal text-slate">À partir de ce montant, une AUTRE personne que l’auteur doit valider l’écriture. Vide ou 0 : toujours (recommandé tant que le seuil n’est pas arrêté avec votre direction).</span>
            </label>
          </ActionForm>
        ) : (
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between"><dt className="text-slate">Devise</dt><dd className="font-medium text-night">{s.baseCurrency}</dd></div>
            <div className="flex justify-between"><dt className="text-slate">Seuil de double validation</dt><dd className="font-medium text-night">{s.fourEyesMinAmount === 0 ? "Toujours" : fmtMoney(s.fourEyesMinAmount, s.baseCurrency)}</dd></div>
          </dl>
        )}
      </Card>
      <p className="text-sm text-slate">Toute modification est inscrite au journal d’audit. La modification des réglages demande la permission « Administrer — finance ».</p>
    </div>
  );
}
