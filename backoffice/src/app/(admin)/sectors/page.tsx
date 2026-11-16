import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { t } from "@/lib/catalog";
import { LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import type { SectorPage } from "@/lib/types";

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
      <form className="mb-4"><input name="q" defaultValue={q} placeholder="Rechercher un secteur…" className="w-full max-w-sm rounded border border-line bg-white px-3 py-2 text-sm" /></form>
      <Table head={["Secteur", "Identifiant d'URL", ""]} empty={data.items.length === 0 ? (q ? "Aucun secteur ne correspond." : "Aucun secteur pour l'instant.") : undefined}>
        {data.items.map((s) => (
          <tr key={s.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/sectors/${s.id}`} className="hover:underline">{t(s.name)}</Link></td>
            <td className="px-4 py-3 text-slate">{s.slug}</td>
            <td className="px-4 py-3 text-right"><Link href={`/sectors/${s.id}`} className="text-link hover:underline">Ouvrir</Link></td>
          </tr>
        ))}
      </Table>
      {data.page.totalPages > 1 && (
        <p className="mt-4 flex items-center gap-4 text-sm text-slate">
          {n > 1 && <Link href={link(n - 1)} className="text-link hover:underline">← Précédent</Link>}
          <span>Page {data.page.page} / {data.page.totalPages}</span>
          {n < data.page.totalPages && <Link href={link(n + 1)} className="text-link hover:underline">Suivant →</Link>}
        </p>
      )}
    </div>
  );
}
