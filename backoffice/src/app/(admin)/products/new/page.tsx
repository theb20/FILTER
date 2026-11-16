import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories, attributeDefsFor, buildTree, flatten, t } from "@/lib/catalog";
import { Card, PageHeader, btnCls, inputCls } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { ProductForm } from "@/components/product-form";
import type { BrandPage, SectorPage } from "@/lib/types";
import { createProduct } from "../actions";

export default async function NewProductPage({ searchParams }: { searchParams: Promise<{ categoryId?: string }> }) {
  const { categoryId } = await searchParams;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  const here = `/products/new${categoryId ? `?categoryId=${categoryId}` : ""}`;
  const cats = await allCategories(here);
  const cat = cats.find((c) => c.id === categoryId);

  if (!cat) {
    const rows = flatten(buildTree(cats));
    return (
      <div className="max-w-xl">
        <PageHeader title="Ajouter un produit" subtitle="Étape 1 : choisissez la catégorie. Ses attributs déterminent les champs de la fiche." />
        <Card>
          {rows.length === 0 ? (
            <p className="text-sm text-slate">Aucune catégorie : créez d'abord une catégorie dans le menu « Catégories ».</p>
          ) : (
            <form className="space-y-4">
              <label className="block text-sm font-medium text-night">Catégorie
                <select name="categoryId" required defaultValue="" className={inputCls}>
                  <option value="" disabled>— Choisir —</option>
                  {rows.map((r) => <option key={r.id} value={r.id}>{"  ".repeat(r.level)}{r.level > 0 ? "└ " : ""}{t(r.name)}</option>)}
                </select>
              </label>
              <button className={btnCls}>Continuer</button>
            </form>
          )}
        </Card>
      </div>
    );
  }
  const [defs, brands, sectors] = await Promise.all([
    attributeDefsFor(cat.id, cats, here),
    adminGet<BrandPage>("/admin/brands", here, { query: { pageSize: 100 } }),
    adminGet<SectorPage>("/admin/sectors", here, { query: { pageSize: 100 } }),
  ]);
  return (
    <div className="max-w-3xl">
      <PageHeader title="Ajouter un produit" subtitle="Le produit est créé en brouillon ; il sera publié quand sa fiche sera complète." />
      <Card>
        <ActionForm action={createProduct} submit="Créer le produit (brouillon)">
          <ProductForm categoryId={cat.id} categoryName={t(cat.name)} defs={defs} brands={brands.items} sectors={sectors.items} />
        </ActionForm>
      </Card>
    </div>
  );
}
