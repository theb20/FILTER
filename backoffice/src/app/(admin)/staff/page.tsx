import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, FilterBar, LinkButton, PageHeader, Pagination, SearchInput, Table } from "@/components/ui/ui";
import { dateFr, STATUS_LABEL, STATUS_TONE, type Role, type StaffPage } from "@/lib/types";

export const metadata = { title: "Personnel" };

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
      <FilterBar reset={{ href: "/staff", show: !!q }}>
        <SearchInput defaultValue={q} placeholder="Rechercher par nom ou e-mail" className="w-full sm:w-80" />
      </FilterBar>
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
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="membres" href={href} />
    </div>
  );
}
