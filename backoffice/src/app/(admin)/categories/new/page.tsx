import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories, buildTree, flatten } from "@/lib/catalog";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { CategoryFields } from "@/components/catalog-forms";
import { ParentSelect } from "@/components/parent-select";
import { createCategory } from "../actions";

export default async function NewCategoryPage({ searchParams }: { searchParams: Promise<{ parent?: string }> }) {
  const { parent } = await searchParams;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  const rows = flatten(buildTree(await allCategories("/categories/new")));
  return (
    <div className="max-w-2xl">
      <PageHeader title="Ajouter une catégorie" />
      <Card>
        <ActionForm action={createCategory} submit="Créer la catégorie">
          <ParentSelect rows={rows} value={parent} />
          <CategoryFields />
        </ActionForm>
      </Card>
    </div>
  );
}
