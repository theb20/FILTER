import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories, buildTree, flatten } from "@/lib/catalog";
import type { BulkCol } from "@/lib/bulk-shared";
import { PageHeader } from "@/components/ui/ui";
import BulkGrid from "@/components/bulk/bulk-grid";
import { saveCategories } from "./actions";

export const metadata = { title: "Catégories — édition en masse" };

const COLS: BulkCol[] = [
  { id: "slug", title: "Identifiant d'URL", width: 180, aliases: ["slug"] },
  { id: "parent", title: "Parent (identifiant)", width: 180, aliases: ["parent", "categorie parente"] },
  { id: "nameFr", title: "Nom (FR)", width: 200, aliases: ["nom", "name"] },
  { id: "nameEn", title: "Nom (EN)", width: 180 },
  { id: "nameEs", title: "Nom (ES)", width: 180 },
  { id: "imageUrl", title: "Image (facultative)", width: 240, aliases: ["image", "image url"] },
  { id: "descFr", title: "Description (FR)", width: 220, aliases: ["description"] },
  { id: "descEn", title: "Description (EN)", width: 200 },
  { id: "descEs", title: "Description (ES)", width: 200 },
  { id: "productCount", title: "Produits", width: 90, kind: "int", readOnly: true },
  { id: "del", title: "Supprimer", width: 100, kind: "check" },
];

export default async function CategoriesBulkPage() {
  const [{ me }, cats] = await Promise.all([currentStaff(), allCategories("/categories/bulk")]);
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  const slugOf = new Map(cats.map((c) => [c.id, c.slug]));
  const initial = flatten(buildTree(cats)).map((c) => ({
    _id: c.id, slug: c.slug, parent: c.parentId ? slugOf.get(c.parentId) ?? "" : "",
    nameFr: c.name.fr ?? "", nameEn: c.name.en ?? "", nameEs: c.name.es ?? "",
    imageUrl: c.imageUrl ?? "", descFr: c.description?.fr ?? "", descEn: c.description?.en ?? "", descEs: c.description?.es ?? "",
    productCount: c.productCount, del: false,
  }));
  return (
    <div>
      <PageHeader title="Catégories — édition en masse" subtitle="Un tableur pour créer, modifier, déplacer et supprimer plusieurs catégories d'un coup. Collez vos lignes depuis Excel ou importez un classeur .xlsx." breadcrumb={[{ label: "Catégories", href: "/categories" }, { label: "Édition en masse" }]} />
      <BulkGrid cols={COLS} initial={initial} keyCol="slug" save={saveCategories} fileName="categories.xlsx" noun="catégorie(s)" />
    </div>
  );
}
