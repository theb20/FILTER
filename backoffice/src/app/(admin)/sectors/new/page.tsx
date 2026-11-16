import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { SectorFields } from "@/components/catalog-forms";
import { createSector } from "../actions";

export default async function NewSectorPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ajouter un secteur" />
      <Card><ActionForm action={createSector} submit="Créer le secteur"><SectorFields /></ActionForm></Card>
    </div>
  );
}
