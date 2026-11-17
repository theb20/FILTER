import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { ThirdPartyFields } from "@/components/finance/misc-forms";
import { createThirdParty } from "../../actions";

export const metadata = { title: "Ajouter un tiers" };

export default async function NewThirdPartyPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:write")) redirect("/denied");
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ajouter un tiers" breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Tiers", href: "/finance/third-parties" }, { label: "Nouveau" }]} />
      <Card><ActionForm action={createThirdParty} submit="Créer le tiers"><ThirdPartyFields /></ActionForm></Card>
    </div>
  );
}
