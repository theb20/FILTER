import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ImportForm } from "@/components/finance/forms";

export const metadata = { title: "Importer un plan comptable" };

export default async function ImportAccountsPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:admin")) redirect("/denied");
  return (
    <div className="max-w-3xl">
      <PageHeader title="Importer un plan comptable" subtitle="Collez le fichier de votre expert-comptable. Simulez d'abord : rien n'est écrit tant que vous n'avez pas cliqué sur « Importer »." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Plan comptable", href: "/finance/accounts" }, { label: "Importer" }]} />
      <Card><ImportForm /></Card>
    </div>
  );
}
