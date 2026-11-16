import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { t } from "@/lib/catalog";
import { Card, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { SectorFields } from "@/components/catalog-forms";
import type { Sector } from "@/lib/types";
import { deleteSector, updateSector } from "../actions";

export default async function SectorPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, sector] = await Promise.all([currentStaff(), adminGet<Sector>(`/admin/sectors/${id}`, `/sectors/${id}`)]);
  const canWrite = can(me?.permissions ?? [], "catalog:write");
  return (
    <div className="max-w-2xl space-y-5">
      <PageHeader title={t(sector.name)} />
      {ok === "created" && <Notice kind="ok">Secteur créé.</Notice>}
      {ok === "updated" && <Notice kind="ok">Modifications enregistrées.</Notice>}
      <Card>
        {canWrite ? <ActionForm action={updateSector.bind(null, sector.id)} submit="Enregistrer"><SectorFields value={sector} /></ActionForm> : <p className="text-sm text-slate">{t(sector.name)} · {sector.slug}</p>}
      </Card>
      {canWrite && (
        <Card>
          <h2 className="mb-3 text-lg font-semibold text-night">Supprimer ce secteur</h2>
          <ActionButton action={deleteSector.bind(null, sector.id)} label="Supprimer le secteur" variant="danger" confirm={`Supprimer le secteur « ${t(sector.name)} » ?`} />
        </Card>
      )}
    </div>
  );
}
