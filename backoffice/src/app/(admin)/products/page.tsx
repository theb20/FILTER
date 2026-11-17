import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories, t } from "@/lib/catalog";
import { Badge, FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table, selectCls } from "@/components/ui/ui";
import { PRODUCT_STATUS_LABEL, PRODUCT_STATUS_TONE, xof, type BrandPage, type ProductPage, type ProductStatus } from "@/lib/types";

type SP = { q?: string; page?: string; status?: string; categoryId?: string; brandId?: string; ok?: string };

export const metadata = { title: "Produits" };

export default async function ProductsPage({ searchParams }: { searchParams: Promise<SP> }) {
  const sp = await searchParams;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const status = sp.status && sp.status in PRODUCT_STATUS_LABEL ? sp.status : undefined;
  const [{ me }, data, cats, brands] = await Promise.all([
    currentStaff(),
    adminGet<ProductPage>("/admin/products", "/products", { query: { q: sp.q, page: n, pageSize: 20, status, categoryId: sp.categoryId || undefined, brandId: sp.brandId || undefined } }),
    allCategories("/products"),
    adminGet<BrandPage>("/admin/brands", "/products", { query: { pageSize: 100 } }),
  ]);
  const canWrite = can(me?.permissions ?? [], "catalog:write");
  const catName = new Map(cats.map((c) => [c.id, t(c.name)]));
  const brandName = new Map(brands.items.map((b) => [b.id, b.name]));
  const keep = { ...(sp.q ? { q: sp.q } : {}), ...(status ? { status } : {}), ...(sp.categoryId ? { categoryId: sp.categoryId } : {}), ...(sp.brandId ? { brandId: sp.brandId } : {}) };
  const link = (p: number) => `/products?${new URLSearchParams({ ...keep, page: String(p) })}`;
  const sel = selectCls;
  return (
    <div>
      <PageHeader title="Produits" subtitle="Le catalogue. Un produit est publié sur le site seulement une fois sa fiche complète." action={canWrite && <LinkButton href="/products/new">Ajouter un produit</LinkButton>} />
      {sp.ok === "deleted" && <Notice kind="ok">Produit supprimé.</Notice>}
      <FilterBar reset={{ href: "/products", show: Object.keys(keep).length > 0 }}>
        <SearchInput defaultValue={sp.q} placeholder="SKU, nom, référence…" className="w-full sm:w-64" />
        <select name="status" defaultValue={status ?? ""} className={sel}>
          <option value="">Tous les statuts</option>
          {(Object.keys(PRODUCT_STATUS_LABEL) as ProductStatus[]).map((s) => <option key={s} value={s}>{PRODUCT_STATUS_LABEL[s]}</option>)}
        </select>
        <select name="categoryId" defaultValue={sp.categoryId ?? ""} className={sel}>
          <option value="">Toutes les catégories</option>
          {cats.map((c) => <option key={c.id} value={c.id}>{t(c.name)}</option>)}
        </select>
        <select name="brandId" defaultValue={sp.brandId ?? ""} className={sel}>
          <option value="">Toutes les marques</option>
          {brands.items.map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
        </select>
      </FilterBar>
      <Table head={["Produit", "SKU", "Catégorie", "Marque", "Prix", "Statut"]} empty={data.items.length === 0 ? (Object.keys(keep).length ? "Aucun produit ne correspond." : "Aucun produit pour l'instant.") : undefined}>
        {data.items.map((p) => (
          <tr key={p.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/products/${p.id}`} className="hover:underline">{t(p.name)}</Link></td>
            <td className="px-4 py-3 text-slate">{p.sku}</td>
            <td className="px-4 py-3 text-slate">{catName.get(p.categoryId) ?? "—"}</td>
            <td className="px-4 py-3 text-slate">{p.brandId ? brandName.get(p.brandId) ?? "—" : "—"}</td>
            <td className="px-4 py-3 text-slate">{p.priceMode === "on_quote" ? "Sur devis" : xof(p.priceAmount)}</td>
            <td className="px-4 py-3"><Badge tone={PRODUCT_STATUS_TONE[p.status]}>{PRODUCT_STATUS_LABEL[p.status]}</Badge></td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="produits" href={link} />
    </div>
  );
}
