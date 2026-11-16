import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allCategories, attributeDefsFor, t } from "@/lib/catalog";
import { Badge, Card, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { ProductForm } from "@/components/product-form";
import { ACTION_LABEL, FIELD_LABEL, PRODUCT_STATUS_LABEL, PRODUCT_STATUS_TONE, dateFr, type BrandPage, type Product, type SectorPage } from "@/lib/types";
import { archiveProduct, deleteProduct, publishProduct, restoreProduct, submitReview, updateProduct } from "../actions";

const NOTICES: Record<string, string> = {
  created: "Produit créé en brouillon.", updated: "Modifications enregistrées.", review: "Fiche soumise à validation.",
  published: "Produit publié.", archived: "Produit archivé.", restored: "Produit restauré en brouillon.",
};

type VersionsRes = { items: { version: number; at: string; actorId: string | null; action: string; changes: Record<string, { before: unknown; after: unknown; origin: string }> }[] };

function show(v: unknown): string {
  if (v === null || v === undefined) return "—";
  if (typeof v === "object") return JSON.stringify(v);
  return String(v);
}

export default async function ProductPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const here = `/products/${id}`;
  const [{ me }, product, versions, brands, sectors, cats] = await Promise.all([
    currentStaff(),
    adminGet<Product>(`/admin/products/${id}`, here),
    adminGet<VersionsRes>(`/admin/products/${id}/versions`, here),
    adminGet<BrandPage>("/admin/brands", here, { query: { pageSize: 100 } }),
    adminGet<SectorPage>("/admin/sectors", here, { query: { pageSize: 100 } }),
    allCategories(here),
  ]);
  const perms = me?.permissions ?? [];
  const canWrite = can(perms, "catalog:write");
  const canPublish = can(perms, "catalog:publish");
  const cat = cats.find((c) => c.id === product.categoryId);
  const defs = await attributeDefsFor(product.categoryId, cats, here);
  const editable = canWrite && product.status !== "archived";
  const st = product.status;

  return (
    <div className="max-w-4xl space-y-5">
      <PageHeader title={t(product.name)} subtitle={`${product.sku} · ${cat ? t(cat.name) : "—"} · version ${product.version}`} action={<Badge tone={PRODUCT_STATUS_TONE[st]}>{PRODUCT_STATUS_LABEL[st]}</Badge>} />
      {ok && NOTICES[ok] && <Notice kind="ok">{NOTICES[ok]}</Notice>}

      {canWrite && (
        <Card>
          <h2 className="mb-3 text-lg font-semibold text-night">Cycle de vie</h2>
          <div className="flex flex-wrap items-start gap-3">
            {st === "draft" && <ActionButton action={submitReview.bind(null, id)} label="Soumettre à validation" variant="ghost" />}
            {(st === "draft" || st === "review") && canPublish && <ActionButton action={publishProduct.bind(null, id)} label="Publier" />}
            {(st === "draft" || st === "review") && !canPublish && <p className="max-w-sm text-sm text-slate">Seul un membre ayant la permission de publication peut publier.</p>}
            {st !== "archived" && <ActionButton action={archiveProduct.bind(null, id)} label="Archiver" variant="ghost" confirm="Archiver ce produit ? Il ne sera plus visible sur le site." />}
            {st === "archived" && <ActionButton action={restoreProduct.bind(null, id)} label="Restaurer en brouillon" variant="ghost" />}
          </div>
          <p className="mt-3 text-sm text-slate">Publication : nom en français, prix hors taxes (sauf « sur devis ») et attributs obligatoires de la catégorie.{product.publishedAt && ` Publié une première fois le ${dateFr(product.publishedAt)} : SKU et adresse figés.`}</p>
        </Card>
      )}

      <Card>
        <h2 className="mb-3 text-lg font-semibold text-night">Fiche</h2>
        {editable ? (
          <ActionForm action={updateProduct.bind(null, id)} submit="Enregistrer">
            <ProductForm product={product} categoryId={product.categoryId} categoryName={cat ? t(cat.name) : "—"} defs={defs} brands={brands.items} sectors={sectors.items} />
          </ActionForm>
        ) : (
          <p className="text-sm text-slate">{st === "archived" ? "Produit archivé : restaurez-le pour le modifier." : "Vous n'avez pas la permission de modifier ce produit."}</p>
        )}
      </Card>

      <section>
        <h2 className="mb-3 text-lg font-semibold text-night">Historique ({versions.items.length})</h2>
        <ol className="space-y-3">
          {versions.items.map((v) => (
            <li key={v.version} className="rounded-lg border border-line bg-white p-4 text-sm shadow-card">
              <p className="font-medium text-night">Version {v.version} · {ACTION_LABEL[v.action] ?? v.action} <span className="font-normal text-slate">· {dateFr(v.at)}</span></p>
              <ul className="mt-2 space-y-1 text-slate">
                {Object.entries(v.changes).map(([k, c]) => (
                  <li key={k}><span className="font-medium text-night">{FIELD_LABEL[k] ?? k}</span> : {v.action === "create" ? show(c.after) : <>{show(c.before)} → {show(c.after)}</>} <span className="text-xs">({c.origin === "import" ? "import" : "saisie manuelle"})</span></li>
                ))}
              </ul>
            </li>
          ))}
        </ol>
      </section>

      {canWrite && (
        <Card>
          <h2 className="mb-3 text-lg font-semibold text-night">Supprimer</h2>
          <p className="mb-3 text-sm text-slate">Un produit jamais publié est supprimé définitivement. Un produit déjà publié est archivé (il reste référencé).</p>
          <ActionButton action={deleteProduct.bind(null, id)} label={product.publishedAt ? "Archiver le produit" : "Supprimer le produit"} variant="danger" confirm={`${product.publishedAt ? "Archiver" : "Supprimer définitivement"} « ${t(product.name)} » ?`} />
        </Card>
      )}
    </div>
  );
}
