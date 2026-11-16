import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { t } from "@/lib/catalog";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { AttributeFields } from "@/components/attribute-fields";
import type { Category } from "@/lib/types";
import { createAttribute } from "../actions";

export default async function NewAttributePage({ searchParams }: { searchParams: Promise<{ categoryId?: string }> }) {
  const { categoryId } = await searchParams;
  if (!categoryId) redirect("/categories");
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  const cat = await adminGet<Category>(`/admin/categories/${categoryId}`, `/attributes/new?categoryId=${categoryId}`);
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ajouter un attribut" subtitle={`Catégorie : ${t(cat.name)} — les sous-catégories en héritent.`} />
      <Card><ActionForm action={createAttribute} submit="Créer l'attribut"><AttributeFields categoryId={cat.id} /></ActionForm></Card>
    </div>
  );
}
