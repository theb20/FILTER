import { notFound, redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { kindBySlug } from "@/lib/content-kinds";
import { label, loadRefs, type Doc } from "@/lib/content";
import { Badge, Card, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { ContentFields, ReadOnlyValue } from "@/components/content-fields";
import { deleteItem, setPublished, updateItem } from "../../actions";

export default async function ContentItemPage({ params, searchParams }: { params: Promise<{ collection: string; id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { collection, id } = await params;
  const { ok } = await searchParams;
  const k = kindBySlug(collection);
  if (!k || !k.ops.includes("get")) notFound();
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], k.perm.read)) redirect("/denied");
  const canWrite = can(me?.permissions ?? [], k.perm.write);
  const [doc, refs] = await Promise.all([adminGet<Doc>(`/admin/${collection}/${id}`, `/content/${collection}/${id}`), loadRefs(k.fields, `/content/${collection}/${id}`)]);
  const editable = k.fields.filter((f) => !f.ro && f.type !== "items" && (!k.updateFields || k.updateFields.includes(f.name)));
  const readOnly = k.fields.filter((f) => f.ro || f.type === "items");
  const title = label(doc[k.summary[0]]) || k.single;
  const canPublish = can(me?.permissions ?? [], "content:publish");
  const isLive = k.publish ? doc[k.publish.field] === k.publish.on : false;
  return (
    <div className="max-w-3xl space-y-5">
      <PageHeader breadcrumb={[{ label: k.title, href: `/content/${collection}` }, { label: "Détail" }]} title={title} subtitle={`Reçu / créé le ${new Date(doc.createdAt).toLocaleString("fr-FR")}`} />
      {ok === "created" && <Notice kind="ok">Élément créé.</Notice>}
      {ok === "updated" && <Notice kind="ok">Modifications enregistrées.</Notice>}
      {ok === "published" && <Notice kind="ok">Publié sur le site.</Notice>}
      {ok === "unpublished" && <Notice kind="ok">Retiré du site.</Notice>}
      {k.publish && (
        <Card guide="content-publish" title="Publication" description="Seuls les contenus publiés apparaissent sur le site. La publication exige la permission « Publier — contenu ».">
          <div className="flex flex-wrap items-center gap-3">
            <Badge tone={isLive ? "ok" : "off"}>{isLive ? "Publié" : "Brouillon"}</Badge>
            {canPublish ? (
              <ActionButton action={setPublished.bind(null, collection, id, !isLive)} label={isLive ? "Retirer du site" : "Publier"} variant={isLive ? "ghost" : undefined} confirm={isLive ? undefined : "Rendre ce contenu visible sur le site ?"} />
            ) : (
              <p className="text-sm text-slate">Vous n’avez pas la permission de publier.</p>
            )}
          </div>
        </Card>
      )}
      {readOnly.length > 0 && (
        <Card guide="content-received" title="Contenu reçu">
          <dl className="space-y-3 text-sm">
            {readOnly.map((f) => <div key={f.name}><dt className="text-xs font-semibold uppercase tracking-wide text-slate">{f.label}</dt><dd className="mt-0.5 text-night"><ReadOnlyValue f={f} value={doc[f.name]} /></dd></div>)}
          </dl>
        </Card>
      )}
      {k.ops.includes("update") && (
        <Card title={readOnly.length ? "Traitement" : undefined}>
          {canWrite ? <ActionForm action={updateItem.bind(null, collection, id)} submit="Enregistrer"><ContentFields fields={editable} value={doc} refs={refs} /></ActionForm> : <p className="text-sm text-slate">Lecture seule.</p>}
        </Card>
      )}
      {canWrite && k.ops.includes("delete") && (
        <Card guide="danger-zone">
          <h2 className="mb-3 text-lg font-semibold text-night">Supprimer</h2>
          <p className="mb-3 text-sm text-slate">La suppression est définitive.</p>
          <ActionButton action={deleteItem.bind(null, collection, id)} label="Supprimer" variant="danger" confirm="Supprimer définitivement cet élément ?" />
        </Card>
      )}
    </div>
  );
}
