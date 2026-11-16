import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, LinkButton, PageHeader, Table } from "@/components/ui/ui";
import { dateFr, STATUS_LABEL, STATUS_TONE, type Role, type StaffPage } from "@/lib/types";

export default async function StaffListPage({ searchParams }: { searchParams: Promise<{ q?: string; page?: string }> }) {
  const sp = await searchParams;
  const page = Math.max(1, Number(sp.page) || 1);
  const q = sp.q?.trim() ?? "";
  const [{ me }, data, roles] = await Promise.all([
    currentStaff(),
    adminGet<StaffPage>("/admin/staff-users", "/staff", { query: { q, page, pageSize: 20 } }),
    adminGet<Role[]>("/admin/roles", "/staff"),
  ]);
  const roleName = new Map(roles.map((r) => [r.id, r.name]));
  const canWrite = can(me?.permissions ?? [], "admin:write");
  const href = (p: number) => `/staff?${new URLSearchParams({ ...(q ? { q } : {}), page: String(p) })}`;

  return (
    <div>
      <PageHeader title="Personnel" subtitle={`${data.page.total} membre${data.page.total > 1 ? "s" : ""}`} action={canWrite && <LinkButton href="/staff/new">Inviter un membre</LinkButton>} />
      <form className="mb-4 flex gap-2" role="search">
        <input name="q" defaultValue={q} placeholder="Rechercher par nom ou e-mail" className="w-full max-w-sm rounded border border-line bg-white px-3 py-2 text-sm focus:border-link focus:outline-none" />
        <button className="rounded border border-line bg-white px-4 py-2 text-sm font-medium hover:bg-page">Rechercher</button>
      </form>
      <Table head={["Nom", "Adresse e-mail", "Rôles", "Statut", "2FA", "Dernière connexion"]} empty={data.items.length === 0 ? "Aucun membre ne correspond." : undefined}>
        {data.items.map((u) => (
          <tr key={u.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/staff/${u.id}`} className="hover:underline">{u.name}</Link></td>
            <td className="px-4 py-3 text-slate">{u.email}</td>
            <td className="px-4 py-3 text-slate">{u.roleIds.map((id) => roleName.get(id) ?? "—").join(", ")}</td>
            <td className="px-4 py-3"><Badge tone={STATUS_TONE[u.status]}>{STATUS_LABEL[u.status]}</Badge></td>
            <td className="px-4 py-3">{u.twoFactorEnabled ? <Badge tone="ok">Activée</Badge> : <Badge>Non</Badge>}</td>
            <td className="px-4 py-3 text-slate">{dateFr(u.lastLoginAt)}</td>
          </tr>
        ))}
      </Table>
      {data.page.totalPages > 1 && (
        <nav className="mt-4 flex items-center justify-between text-sm" aria-label="Pagination">
          {page > 1 ? <Link href={href(page - 1)} className="text-link hover:underline">← Précédent</Link> : <span />}
          <span className="text-slate">Page {data.page.page} sur {data.page.totalPages}</span>
          {page < data.page.totalPages ? <Link href={href(page + 1)} className="text-link hover:underline">Suivant →</Link> : <span />}
        </nav>
      )}
    </div>
  );
}
