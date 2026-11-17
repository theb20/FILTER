import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader, inputCls } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { createFiscalYear } from "../../actions";

export const metadata = { title: "Ouvrir un exercice" };

export default async function NewYearPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:admin")) redirect("/denied");
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ouvrir un exercice" subtitle="Les dates sont les vôtres (à confirmer avec votre expert-comptable). Les périodes sont les mois calendaires compris entre ces deux dates." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Exercices", href: "/finance/periods" }, { label: "Nouveau" }]} />
      <Card>
        <ActionForm action={createFiscalYear} submit="Ouvrir l'exercice">
          <label className="block text-sm font-medium text-night">Libellé<input name="label" required maxLength={100} placeholder="Ex. 2026" className={inputCls} /></label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-medium text-night">Début<input name="startDate" type="date" required className={inputCls} /></label>
            <label className="block text-sm font-medium text-night">Fin<input name="endDate" type="date" required className={inputCls} /></label>
          </div>
          <p className="text-xs text-slate">Un exercice ne dépasse pas 24 mois et ne peut pas chevaucher un autre exercice.</p>
        </ActionForm>
      </Card>
    </div>
  );
}
