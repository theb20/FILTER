import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { AccountFields } from "@/components/finance/forms";
import { createAccount } from "../../actions";

export const metadata = { title: "Ajouter un compte" };

export default async function NewAccountPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:admin")) redirect("/denied");
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ajouter un compte" breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Plan comptable", href: "/finance/accounts" }, { label: "Nouveau" }]} />
      <Card><ActionForm action={createAccount} submit="Créer le compte"><AccountFields /></ActionForm></Card>
    </div>
  );
}
