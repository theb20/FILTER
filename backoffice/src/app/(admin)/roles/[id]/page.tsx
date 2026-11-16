import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, Card, inputCls, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { PermissionPicker } from "@/components/permission-picker";
import type { Permission, Role } from "@/lib/types";
import { deleteRole, updateRole } from "../actions";

export default async function RoleDetailPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, role, permissions] = await Promise.all([currentStaff(), adminGet<Role>(`/admin/roles/${id}`, `/roles/${id}`), adminGet<Permission[]>("/admin/permissions", `/roles/${id}`)]);
  const editable = can(me?.permissions ?? [], "admin:write") && !role.system;
  return (
    <div className="max-w-3xl space-y-5">
      <PageHeader title={role.name} subtitle={role.description ?? undefined} action={role.system ? <Badge tone="info">Rôle système : non modifiable</Badge> : undefined} />
      {ok === "created" && <Notice kind="ok">Rôle créé.</Notice>}
      {ok === "updated" && <Notice kind="ok">Modifications enregistrées. Les membres concernés les voient à leur prochain renouvellement de session (15 min au plus).</Notice>}
      <Card>
        {editable ? (
          <ActionForm action={updateRole.bind(null, role.id)} submit="Enregistrer">
            <label className="block text-sm font-medium text-night">Nom<input name="name" defaultValue={role.name} required className={inputCls} /></label>
            <label className="block text-sm font-medium text-night">Description<input name="description" defaultValue={role.description ?? ""} className={inputCls} /></label>
            <PermissionPicker permissions={permissions} selected={role.permissions} />
          </ActionForm>
        ) : (
          <PermissionPicker permissions={permissions} selected={role.permissions} readOnly />
        )}
      </Card>
      {editable && (
        <Card>
          <h2 className="mb-3 text-lg font-semibold text-night">Supprimer ce rôle</h2>
          <p className="mb-3 text-sm text-slate">Impossible tant que des membres du personnel portent ce rôle.</p>
          <ActionButton action={deleteRole.bind(null, role.id)} label="Supprimer le rôle" variant="danger" confirm={`Supprimer le rôle « ${role.name} » ?`} />
        </Card>
      )}
    </div>
  );
}
