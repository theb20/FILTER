import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table } from "@/components/ui/ui";
import type { BrandPage } from "@/lib/types";

export const metadata = { title: "Marques" };

export default async function BrandsPage({ searchParams }: { searchParams: Promise<{ q?: string; page?: string; ok?: string }> }) {
  const { q, page, ok } = await searchParams;
  const n = Math.max(1, Number.parseInt(page ?? "1", 10) || 1);
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<BrandPage>("/admin/brands", "/brands", { query: { q, page: n, pageSize: 20 } })]);
  const canWrite = can(me?.permissions ?? [], "catalog:write");
  const link = (p: number) => `/brands?${new URLSearchParams({ ...(q ? { q } : {}), page: String(p) })}`;
  return (
    <div>
      <PageHeader title="Marques" subtitle="Les marques des produits vendus." action={canWrite && <LinkButton href="/brands/new">Ajouter une marque</LinkButton>} />
      {ok === "deleted" && <Notice kind="ok">Marque supprimée.</Notice>}
      <FilterBar reset={{ href: "/brands", show: !!q }}>
        <SearchInput defaultValue={q} placeholder="Rechercher une marque…" className="w-full sm:w-80" />
      </FilterBar>
      <Table head={["Marque", "Identifiant d'URL", ""]} empty={data.items.length === 0 ? (q ? "Aucune marque ne correspond." : "Aucune marque pour l'instant.") : undefined}>
        {data.items.map((b) => (
          <tr key={b.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/brands/${b.id}`} className="hover:underline">{b.name}</Link></td>
            <td className="px-4 py-3 text-slate">{b.slug}</td>
            <td className="px-4 py-3 text-right"><Link href={`/brands/${b.id}`} className="text-link hover:underline">Ouvrir</Link></td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="marques" href={link} />
    </div>
  );
}
