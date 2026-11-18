import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories } from "@/lib/catalog";
import type { BulkCol } from "@/lib/bulk-shared";
import { Notice, PageHeader } from "@/components/ui/ui";
import BulkGrid from "@/components/bulk/bulk-grid";
import { PRODUCT_STATUS_LABEL, type BrandPage, type ProductPage } from "@/lib/types";
import { saveProducts } from "./actions";

export const metadata = { title: "Produits — édition en masse" };

const MAX = 1000;
const COLS: BulkCol[] = [
  { id: "sku", title: "SKU", width: 140 },
  { id: "nameFr", title: "Nom (FR)", width: 240, aliases: ["nom", "name"] },
  { id: "nameEn", title: "Nom (EN)", width: 200 },
  { id: "nameEs", title: "Nom (ES)", width: 200 },
  { id: "category", title: "Catégorie (identifiant)", width: 190, aliases: ["categorie", "category"] },
  { id: "brand", title: "Marque", width: 150, aliases: ["brand"] },
  { id: "manufacturerRef", title: "Réf. constructeur", width: 160, aliases: ["reference constructeur"] },
  { id: "priceMode", title: "Prix (Fixe / Sur devis)", width: 170, aliases: ["mode de prix", "price mode"] },
  { id: "priceAmount", title: "Prix HT (F CFA)", width: 140, kind: "int", aliases: ["prix", "prix ht", "price"] },
  { id: "fulfillmentMode", title: "Commande (Stock / Sur commande)", width: 220, aliases: ["mode de commande"] },
  { id: "leadTimeDays", title: "Délai (jours)", width: 120, kind: "int", aliases: ["delai"] },
  { id: "minOrderQuantity", title: "Qté minimale", width: 120, kind: "int" },
  { id: "saleUnit", title: "Unité de vente", width: 130 },
  { id: "gtin", title: "GTIN/EAN", width: 150 },
  { id: "descFr", title: "Description (FR)", width: 260, aliases: ["description"] },
  { id: "descEn", title: "Description (EN)", width: 220 },
  { id: "descEs", title: "Description (ES)", width: 220 },
  { id: "status", title: "Statut", width: 110, readOnly: true },
  { id: "del", title: "Supprimer", width: 100, kind: "check" },
];

export default async function ProductsBulkPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  const [cats, brands] = await Promise.all([allCategories("/products/bulk"), adminGet<BrandPage>("/admin/brands", "/products/bulk", { query: { pageSize: 100 } })]);
  const catSlug = new Map(cats.map((c) => [c.id, c.slug]));
  const brandName = new Map(brands.items.map((b) => [b.id, b.name]));
  const items: ProductPage["items"] = [];
  let total = 0;
  for (let page = 1; items.length < MAX; page++) {
    const r = await adminGet<ProductPage>("/admin/products", "/products/bulk", { query: { page, pageSize: 100 } });
    items.push(...r.items);
    total = r.page.total ?? items.length;
    if (page >= r.page.totalPages) break;
  }
  const initial = items.map((p) => ({
    _id: p.id, sku: p.sku, nameFr: p.name.fr ?? "", nameEn: p.name.en ?? "", nameEs: p.name.es ?? "",
    category: catSlug.get(p.categoryId) ?? "", brand: p.brandId ? brandName.get(p.brandId) ?? "" : "", manufacturerRef: p.manufacturerRef ?? "",
    priceMode: p.priceMode === "on_quote" ? "Sur devis" : "Fixe", priceAmount: p.priceAmount, fulfillmentMode: p.fulfillmentMode === "made_to_order" ? "Sur commande" : "Stock",
    leadTimeDays: p.leadTimeDays, minOrderQuantity: p.minOrderQuantity, saleUnit: p.saleUnit ?? "", gtin: p.gtin ?? "",
    descFr: p.description?.fr ?? "", descEn: p.description?.en ?? "", descEs: p.description?.es ?? "", status: PRODUCT_STATUS_LABEL[p.status], del: false,
  }));
  return (
    <div>
      <PageHeader title="Produits — édition en masse" subtitle="Un tableur pour créer et modifier plusieurs produits d'un coup (prix, noms, catégories, marques…). Collez vos lignes depuis Excel ou importez un classeur .xlsx." breadcrumb={[{ label: "Produits", href: "/products" }, { label: "Édition en masse" }]} />
      {total > items.length && <Notice kind="error">Seuls les {MAX} premiers produits sur {total} sont chargés. Utilisez la recherche de la liste pour cibler les autres.</Notice>}
      <BulkGrid cols={COLS} initial={initial} keyCol="sku" save={saveProducts} fileName="produits.xlsx" noun="produit(s)" />
    </div>
  );
}
