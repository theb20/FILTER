import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, inputCls, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { PermissionPicker } from "@/components/permission-picker";
import type { Permission } from "@/lib/types";
import { createRole } from "../actions";

export default async function NewRolePage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "admin:write")) redirect("/denied");
  const permissions = await adminGet<Permission[]>("/admin/permissions", "/roles/new");
  return (
    <div className="max-w-3xl">
      <PageHeader breadcrumb={[{ label: "Rôles", href: "/roles" }, { label: "Nouveau" }]} title="Créer un rôle" subtitle="Vous ne pouvez accorder que des permissions que vous possédez vous-même." />
      <Card>
        <ActionForm action={createRole} submit="Créer le rôle">
          <label className="block text-sm font-medium text-night">Nom<input name="name" required className={inputCls} /></label>
          <label className="block text-sm font-medium text-night">Description <span className="font-normal text-slate">(facultatif)</span><input name="description" className={inputCls} /></label>
          <PermissionPicker permissions={permissions} selected={[]} />
        </ActionForm>
      </Card>
    </div>
  );
}
