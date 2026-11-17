import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { ThirdPartyFields } from "@/components/finance/misc-forms";
import type { FinThirdPartyPage } from "@/lib/types";
import { updateThirdParty } from "../../actions";

export default async function ThirdPartyPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "finance:write")) redirect("/denied");
  const all = await adminGet<FinThirdPartyPage>("/admin/finance/third-parties", `/finance/third-parties/${id}`, { query: { pageSize: 100 } });
  const t = all.items.find((x) => x.id === id);
  if (!t) redirect("/finance/third-parties");
  return (
    <div className="max-w-2xl">
      <PageHeader title={t.displayName} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Tiers", href: "/finance/third-parties" }, { label: t.displayName }]} />
      <Card><ActionForm action={updateThirdParty.bind(null, id)} submit="Enregistrer"><ThirdPartyFields value={t} /></ActionForm></Card>
    </div>
  );
}
