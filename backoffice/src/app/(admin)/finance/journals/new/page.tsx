import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { JournalFields } from "@/components/finance/forms";
import { createJournal } from "../../actions";

export const metadata = { title: "Ajouter un journal" };

export default async function NewJournalPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:admin")) redirect("/denied");
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ajouter un journal" breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Journaux", href: "/finance/journals" }, { label: "Nouveau" }]} />
      <Card><ActionForm action={createJournal} submit="Créer le journal"><JournalFields /></ActionForm></Card>
    </div>
  );
}
