import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, Card, inputCls, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { dateFr, STATUS_LABEL, STATUS_TONE, type Role, type StaffUser } from "@/lib/types";
import { deactivateStaff, reactivateStaff, resendInvitation, resetTwoFactor, updateStaff } from "../actions";

const OK: Record<string, string> = {
  invited: "Invitation envoyée.", updated: "Modifications enregistrées.", deactivated: "Membre désactivé : ses sessions ont été fermées.",
  reactivated: "Membre réactivé.", resent: "Nouvelle invitation envoyée (l'ancien lien ne fonctionne plus).", reset2fa: "Double authentification réinitialisée : la personne en enregistrera une nouvelle à sa prochaine connexion.",
};

export default async function StaffDetailPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, u, roles] = await Promise.all([currentStaff(), adminGet<StaffUser>(`/admin/staff-users/${id}`, `/staff/${id}`), adminGet<Role[]>("/admin/roles", `/staff/${id}`)]);
  const canWrite = can(me?.permissions ?? [], "admin:write");
  const isMe = me?.id === u.id;

  return (
    <div className="max-w-3xl space-y-5">
      <PageHeader title={u.name} subtitle={u.email} action={<Badge tone={STATUS_TONE[u.status]}>{STATUS_LABEL[u.status]}</Badge>} />
      {ok && OK[ok] && <Notice kind="ok">{OK[ok]}</Notice>}

      <Card>
        <dl className="grid gap-3 text-sm sm:grid-cols-3">
          <div><dt className="text-slate">Double authentification</dt><dd className="font-medium text-night">{u.twoFactorEnabled ? "Activée" : "Non enregistrée"}</dd></div>
          <div><dt className="text-slate">Dernière connexion</dt><dd className="font-medium text-night">{dateFr(u.lastLoginAt)}</dd></div>
          <div><dt className="text-slate">Identifiant</dt><dd className="truncate font-mono text-xs text-night">{u.id}</dd></div>
        </dl>
      </Card>

      <Card>
        <h2 className="mb-4 text-lg font-semibold text-night">Informations et rôles</h2>
        {canWrite ? (
          <ActionForm action={updateStaff.bind(null, u.id)} submit="Enregistrer">
            <input type="hidden" name="email" value={u.email} />
            <label className="block text-sm font-medium text-night">Nom<input name="name" defaultValue={u.name} required className={inputCls} /></label>
            <fieldset>
              <legend className="text-sm font-medium text-night">Rôles</legend>
              <div className="mt-2 space-y-2">
                {roles.map((r) => (
                  <label key={r.id} className="flex items-start gap-2 text-sm">
                    <input type="checkbox" name="roleIds" value={r.id} defaultChecked={u.roleIds.includes(r.id)} className="mt-1 h-4 w-4 accent-night" />
                    <span><span className="font-medium text-night">{r.name}</span>{r.description && <span className="block text-xs text-slate">{r.description}</span>}</span>
                  </label>
                ))}
              </div>
            </fieldset>
            <p className="text-xs text-slate">L&apos;adresse e-mail ne peut pas être modifiée. Quand les rôles d&apos;un autre membre changent, ses sessions sont fermées.</p>
          </ActionForm>
        ) : (
          <p className="text-sm text-slate">Rôles : {u.roleIds.map((rid) => roles.find((r) => r.id === rid)?.name ?? "—").join(", ")}</p>
        )}
      </Card>

      {canWrite && (
        <Card>
          <h2 className="mb-4 text-lg font-semibold text-night">Actions</h2>
          <div className="flex flex-wrap items-start gap-3">
            {u.status === "invited" && <ActionButton action={resendInvitation.bind(null, u.id)} label="Renvoyer l'invitation" variant="ghost" />}
            {u.status === "deactivated" ? (
              <ActionButton action={reactivateStaff.bind(null, u.id)} label="Réactiver" variant="ghost" />
            ) : (
              !isMe && <ActionButton action={deactivateStaff.bind(null, u.id)} label="Désactiver" variant="danger" confirm={`Désactiver ${u.name} ? Ses sessions seront fermées immédiatement.`} />
            )}
            {u.twoFactorEnabled && <ActionButton action={resetTwoFactor.bind(null, u.id)} label="Réinitialiser la double authentification" variant="danger" confirm={`Réinitialiser le 2FA de ${u.name} ? Ses sessions seront fermées et la personne sera prévenue.`} />}
          </div>
          {isMe && <p className="mt-3 text-xs text-slate">Vous ne pouvez pas vous désactiver vous-même.</p>}
        </Card>
      )}
    </div>
  );
}
