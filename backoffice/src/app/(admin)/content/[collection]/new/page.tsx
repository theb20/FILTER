import { notFound, redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { kindBySlug } from "@/lib/content-kinds";
import { loadRefs } from "@/lib/content";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { ContentFields } from "@/components/content-fields";
import { createItem } from "../../actions";

export default async function NewContentPage({ params }: { params: Promise<{ collection: string }> }) {
  const { collection } = await params;
  const k = kindBySlug(collection);
  if (!k || !k.ops.includes("create")) notFound();
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], k.perm.write)) redirect("/denied");
  const refs = await loadRefs(k.fields, `/content/${collection}/new`);
  return (
    <div className="max-w-3xl">
      <PageHeader breadcrumb={[{ label: k.title, href: `/content/${collection}` }, { label: "Nouveau" }]} title={`Ajouter : ${k.single.toLowerCase()}`} />
      <Card><ActionForm action={createItem.bind(null, collection)} submit="Créer"><ContentFields fields={k.fields} refs={refs} /></ActionForm></Card>
    </div>
  );
}
