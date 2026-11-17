import { notFound, redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { settingsByKey } from "@/lib/content-kinds";
import { Card, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { ContentFields } from "@/components/content-fields";
import { saveSettings } from "../../actions";

export default async function ContentSettingsPage({ params }: { params: Promise<{ key: string }> }) {
  const { key } = await params;
  const s = settingsByKey(key);
  if (!s) notFound();
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], s.perm.read)) redirect("/denied");
  const canWrite = can(me?.permissions ?? [], s.perm.write);
  const value = await adminGet<Record<string, unknown>>(`/admin/${s.slug}`, `/content/settings/${key}`);
  return (
    <div className="max-w-3xl">
      <PageHeader title={s.title} subtitle="Ces informations s'affichent sur le site." />
      <Card>
        {canWrite ? <ActionForm action={saveSettings.bind(null, key)} submit="Enregistrer"><ContentFields fields={s.fields} value={value} refs={{}} /></ActionForm> : <p className="text-sm text-slate">Lecture seule.</p>}
      </Card>
    </div>
  );
}
