import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { BrandFields } from "@/components/catalog-forms";
import type { Brand } from "@/lib/types";
import { deleteBrand, updateBrand } from "../actions";

export default async function BrandPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, brand] = await Promise.all([currentStaff(), adminGet<Brand>(`/admin/brands/${id}`, `/brands/${id}`)]);
  const canWrite = can(me?.permissions ?? [], "catalog:write");
  return (
    <div className="max-w-2xl space-y-5">
      <PageHeader title={brand.name} />
      {ok === "created" && <Notice kind="ok">Marque créée.</Notice>}
      {ok === "updated" && <Notice kind="ok">Modifications enregistrées.</Notice>}
      <Card>
        {canWrite ? <ActionForm action={updateBrand.bind(null, brand.id)} submit="Enregistrer"><BrandFields value={brand} /></ActionForm> : <p className="text-sm text-slate">{brand.name} · {brand.slug}</p>}
      </Card>
      {canWrite && (
        <Card>
          <h2 className="mb-3 text-lg font-semibold text-night">Supprimer cette marque</h2>
          <p className="mb-3 text-sm text-slate">Impossible tant que des produits portent cette marque.</p>
          <ActionButton action={deleteBrand.bind(null, brand.id)} label="Supprimer la marque" variant="danger" confirm={`Supprimer la marque « ${brand.name} » ?`} />
        </Card>
      )}
    </div>
  );
}
