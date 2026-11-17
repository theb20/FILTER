import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { redirect } from "next/navigation";
import { Card, inputCls, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import type { Role } from "@/lib/types";
import { createStaff } from "../actions";

export default async function NewStaffPage() {
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "admin:write")) redirect("/denied");
  const roles = await adminGet<Role[]>("/admin/roles", "/staff/new");
  return (
    <div className="max-w-xl">
      <PageHeader breadcrumb={[{ label: "Personnel", href: "/staff" }, { label: "Nouveau" }]} title="Inviter un membre" subtitle="La personne reçoit un lien valable 72 h pour choisir son mot de passe, puis enregistrer sa double authentification." />
      <Card>
        <ActionForm action={createStaff} submit="Envoyer l'invitation">
          <label className="block text-sm font-medium text-night">Nom<input name="name" required className={inputCls} /></label>
          <label className="block text-sm font-medium text-night">Adresse e-mail<input name="email" type="email" required className={inputCls} /></label>
          <fieldset>
            <legend className="text-sm font-medium text-night">Rôles</legend>
            <div className="mt-2 space-y-2">
              {roles.map((r) => (
                <label key={r.id} className="flex items-start gap-2 text-sm">
                  <input type="checkbox" name="roleIds" value={r.id} className="mt-1 h-4 w-4 accent-night" />
                  <span><span className="font-medium text-night">{r.name}</span>{r.description && <span className="block text-xs text-slate">{r.description}</span>}</span>
                </label>
              ))}
            </div>
          </fieldset>
        </ActionForm>
      </Card>
    </div>
  );
}
