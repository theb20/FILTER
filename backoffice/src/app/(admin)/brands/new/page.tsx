import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { BrandFields } from "@/components/catalog-forms";
import { createBrand } from "../actions";

export default async function NewBrandPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  return (
    <div className="max-w-2xl">
      <PageHeader breadcrumb={[{ label: "Marques", href: "/brands" }, { label: "Nouveau" }]} title="Ajouter une marque" />
      <Card><ActionForm action={createBrand} submit="Créer la marque"><BrandFields /></ActionForm></Card>
    </div>
  );
}
