import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { t } from "@/lib/catalog";
import { FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table } from "@/components/ui/ui";
import type { SectorPage } from "@/lib/types";

export const metadata = { title: "Secteurs" };

export default async function SectorsPage({ searchParams }: { searchParams: Promise<{ q?: string; page?: string; ok?: string }> }) {
  const { q, page, ok } = await searchParams;
  const n = Math.max(1, Number.parseInt(page ?? "1", 10) || 1);
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<SectorPage>("/admin/sectors", "/sectors", { query: { q, page: n, pageSize: 20 } })]);
  const canWrite = can(me?.permissions ?? [], "catalog:write");
  const link = (p: number) => `/sectors?${new URLSearchParams({ ...(q ? { q } : {}), page: String(p) })}`;
  return (
    <div>
      <PageHeader title="Secteurs d'activité" subtitle="Les secteurs clients (agroalimentaire, industrie…)." action={canWrite && <LinkButton href="/sectors/new">Ajouter un secteur</LinkButton>} />
      {ok === "deleted" && <Notice kind="ok">Secteur supprimé.</Notice>}
      <FilterBar reset={{ href: "/sectors", show: !!q }}>
        <SearchInput defaultValue={q} placeholder="Rechercher un secteur…" className="w-full sm:w-80" />
      </FilterBar>
      <Table head={["Secteur", "Identifiant d'URL", ""]} empty={data.items.length === 0 ? (q ? "Aucun secteur ne correspond." : "Aucun secteur pour l'instant.") : undefined}>
        {data.items.map((s) => (
          <tr key={s.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/sectors/${s.id}`} className="hover:underline">{t(s.name)}</Link></td>
            <td className="px-4 py-3 text-slate">{s.slug}</td>
            <td className="px-4 py-3 text-right"><Link href={`/sectors/${s.id}`} className="text-link hover:underline">Ouvrir</Link></td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="secteurs" href={link} />
    </div>
  );
}
