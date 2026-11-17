import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import type { Role } from "@/lib/types";

export const metadata = { title: "Rôles et permissions" };

export default async function RolesPage({ searchParams }: { searchParams: Promise<{ ok?: string }> }) {
  const { ok } = await searchParams;
  const [{ me }, roles] = await Promise.all([currentStaff(), adminGet<Role[]>("/admin/roles", "/roles")]);
  const canWrite = can(me?.permissions ?? [], "admin:write");
  return (
    <div>
      <PageHeader title="Rôles et permissions" subtitle="Un rôle regroupe des permissions ; un membre peut en cumuler plusieurs." action={canWrite && <LinkButton href="/roles/new">Créer un rôle</LinkButton>} />
      {ok === "deleted" && <Notice kind="ok">Rôle supprimé.</Notice>}
      <Table head={["Rôle", "Description", "Permissions", ""]}>
        {roles.map((r) => (
          <tr key={r.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/roles/${r.id}`} className="hover:underline">{r.name}</Link> {r.system && <Badge tone="info">Système</Badge>}</td>
            <td className="px-4 py-3 text-slate">{r.description ?? "—"}</td>
            <td className="px-4 py-3 text-slate">{r.permissions.includes("*") ? "Toutes" : r.permissions.length}</td>
            <td className="px-4 py-3 text-right"><Link href={`/roles/${r.id}`} className="text-link hover:underline">Ouvrir</Link></td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
