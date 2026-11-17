import Link from "next/link";
import { adminGet } from "@/lib/admin";
import { Badge, FilterBar, PageHeader, Table, selectCls } from "@/components/ui/ui";
import { dateFr, type AuditPage } from "@/lib/types";

const ACTIONS: Record<string, string> = {
  "staff.invite": "Invitation d'un membre", "staff.update": "Modification d'un membre", "staff.deactivate": "Désactivation d'un membre", "staff.reactivate": "Réactivation d'un membre",
  "staff.resend_invitation": "Renvoi d'invitation", "staff.reset_two_factor": "Réinitialisation du 2FA", "staff.change_password": "Changement de mot de passe",
  "settings.update": "Modification des réglages d'identité", "role.create": "Création d'un rôle", "role.update": "Modification d'un rôle", "role.delete": "Suppression d'un rôle",
};

export const metadata = { title: "Journal d'audit" };

export default async function AuditPage({ searchParams }: { searchParams: Promise<{ action?: string; entity?: string; cursor?: string }> }) {
  const sp = await searchParams;
  const data = await adminGet<AuditPage>("/admin/audit-log", "/audit", { query: { limit: 25, action: sp.action, entity: sp.entity, cursor: sp.cursor } });
  const filter = (k: string, v: string) => `/audit?${new URLSearchParams({ ...(sp.entity && k !== "entity" ? { entity: sp.entity } : {}), ...(sp.action && k !== "action" ? { action: sp.action } : {}), ...(v ? { [k]: v } : {}) })}`;
  return (
    <div>
      <PageHeader title="Journal d'audit" subtitle="Historique en ajout seul : aucune ligne ne peut être modifiée ni supprimée." />
      <FilterBar reset={{ href: "/audit", show: !!(sp.action || sp.entity) }}>
        <select name="entity" defaultValue={sp.entity ?? ""} className={selectCls}>
          <option value="">Toutes les entités</option><option value="staff_user">Personnel</option><option value="role">Rôles</option><option value="identity_settings">Réglages</option>
        </select>
        <select name="action" defaultValue={sp.action ?? ""} className={selectCls}>
          <option value="">Toutes les actions</option>
          {Object.entries(ACTIONS).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
        </select>
      </FilterBar>
      <Table head={["Date", "Action", "Cible", "Auteur", "Détail"]} empty={data.items.length === 0 ? "Aucune entrée." : undefined}>
        {data.items.map((e) => (
          <tr key={e.id} className="align-top">
            <td className="whitespace-nowrap px-4 py-3 text-slate">{dateFr(e.at)}</td>
            <td className="px-4 py-3 font-medium text-night">{ACTIONS[e.action] ?? e.action}</td>
            <td className="px-4 py-3 text-slate"><Badge>{e.entity}</Badge> <span className="font-mono text-xs">{e.entityId?.slice(0, 8)}</span></td>
            <td className="px-4 py-3 text-slate"><span className="font-mono text-xs">{e.actorId?.slice(0, 8) ?? e.actorType}</span>{e.ip && <span className="block text-xs">{e.ip}</span>}</td>
            <td className="px-4 py-3">
              {(e.before || e.after) && (
                <details className="text-xs text-slate">
                  <summary className="cursor-pointer text-link">Voir</summary>
                  {e.before && <pre className="mt-1 max-w-xs overflow-x-auto rounded bg-page p-2">avant : {JSON.stringify(e.before, null, 1)}</pre>}
                  {e.after && <pre className="mt-1 max-w-xs overflow-x-auto rounded bg-page p-2">après : {JSON.stringify(e.after, null, 1)}</pre>}
                </details>
              )}
            </td>
          </tr>
        ))}
      </Table>
      {data.nextCursor && <div className="mt-4 text-right"><Link href={`${filter("cursor", data.nextCursor)}`} className="inline-flex items-center rounded-lg border border-line bg-white px-3 py-1.5 text-sm font-medium text-night hover:bg-page">Entrées plus anciennes →</Link></div>}
    </div>
  );
}
