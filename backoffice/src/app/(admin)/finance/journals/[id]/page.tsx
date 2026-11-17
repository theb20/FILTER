import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { journals } from "@/lib/finance";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { JournalFields } from "@/components/finance/forms";
import { updateJournal } from "../../actions";

export default async function JournalPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:admin")) redirect("/denied");
  const j = (await journals(`/finance/journals/${id}`)).find((x) => x.id === id);
  if (!j) redirect("/finance/journals");
  return (
    <div className="max-w-2xl">
      <PageHeader title={`${j.code} — ${j.label}`} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Journaux", href: "/finance/journals" }, { label: j.code }]} />
      <Card><ActionForm action={updateJournal.bind(null, id)} submit="Enregistrer"><JournalFields value={j} /></ActionForm></Card>
    </div>
  );
}
