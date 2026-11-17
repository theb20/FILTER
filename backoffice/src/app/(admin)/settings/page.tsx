import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, PageHeader } from "@/components/ui/ui";
import { SettingsForm, type IdentitySettings } from "./form";

export const metadata = { title: "Réglages d'identité" };

export default async function SettingsPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "admin:read")) redirect("/denied");
  const s = await adminGet<IdentitySettings>("/admin/identity-settings", "/settings");
  const canWrite = can(me?.permissions ?? [], "admin:write");
  return (
    <div className="max-w-3xl">
      <PageHeader title="Réglages d'identité" subtitle="Durées des sessions et des liens, ralentissement des connexions, conservation des données. Chaque modification est tracée dans le journal d'audit." />
      <Card guide="settings-form">
        <SettingsForm s={s} canWrite={canWrite} />
      </Card>
    </div>
  );
}
