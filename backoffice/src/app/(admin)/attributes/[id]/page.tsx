import Link from "next/link";
import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { t } from "@/lib/catalog";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { AttributeFields } from "@/components/attribute-fields";
import type { AttributeDef, Category } from "@/lib/types";
import { deleteAttribute, updateAttribute } from "../actions";

export default async function AttributePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "catalog:write")) redirect("/denied");
  const attr = await adminGet<AttributeDef>(`/admin/attribute-defs/${id}`, `/attributes/${id}`);
  const cat = await adminGet<Category>(`/admin/categories/${attr.categoryId}`, `/attributes/${id}`);
  return (
    <div className="max-w-2xl space-y-5">
      <PageHeader breadcrumb={[{ label: "Catégories", href: "/categories" }, { label: "Détail" }]} title={t(attr.label)} subtitle={`Catégorie : ${t(cat.name)}`} action={<Link href={`/categories/${cat.id}`} className="text-sm text-link hover:underline">← Retour à la catégorie</Link>} />
      <Card><ActionForm action={updateAttribute.bind(null, id)} submit="Enregistrer"><AttributeFields categoryId={cat.id} value={attr} /></ActionForm></Card>
      <Card guide="danger-zone">
        <h2 className="mb-3 text-lg font-semibold text-night">Supprimer cet attribut</h2>
        <p className="mb-3 text-sm text-slate">Impossible tant que des produits portent une valeur pour cet attribut.</p>
        <ActionButton action={deleteAttribute.bind(null, id, cat.id)} label="Supprimer l'attribut" variant="danger" confirm={`Supprimer l'attribut « ${t(attr.label)} » ?`} />
      </Card>
    </div>
  );
}
