import { redirect } from "next/navigation";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { SourceFields } from "@/components/engine/source-fields";
import { createSource } from "../actions";

export const metadata = { title: "Nouvelle source" };

export default async function NewSourcePage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "engine:write")) redirect("/denied");
  return (
    <div className="max-w-3xl space-y-5">
      <PageHeader breadcrumb={[{ label: "Sources externes", href: "/engine/sources" }, { label: "Nouvelle" }]} title="Ajouter une source"
        subtitle="Le formulaire est prérempli pour Brave Search. La clé d'API n'est jamais saisie ici : seul le nom de la variable d'environnement qui la porte est enregistré." />
      <Card><ActionForm action={createSource} submit="Créer la source"><SourceFields /></ActionForm></Card>
      <Card title="Après la création">
        <ol className="list-decimal space-y-1.5 pl-5 text-sm text-slate">
          <li>Réglez le <strong className="text-night">budget</strong> de la source (sans budget, aucun appel n&apos;est autorisé).</li>
          <li><strong className="text-night">Testez</strong> la source : un appel réel est facturé, et le résultat dit si la clé est bien lue.</li>
          <li><strong className="text-night">Activez</strong> la source pour qu&apos;elle soit interrogée.</li>
        </ol>
      </Card>
    </div>
  );
}
