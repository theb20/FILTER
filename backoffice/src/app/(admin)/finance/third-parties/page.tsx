import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table, selectCls } from "@/components/ui/ui";
import type { FinThirdPartyPage } from "@/lib/types";

export const metadata = { title: "Tiers" };

export default async function ThirdPartiesPage({ searchParams }: { searchParams: Promise<{ q?: string; page?: string; kind?: string; ok?: string }> }) {
  const sp = await searchParams;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const kind = sp.kind === "customer" || sp.kind === "supplier" ? sp.kind : undefined;
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<FinThirdPartyPage>("/admin/finance/third-parties", "/finance/third-parties", { query: { q: sp.q, page: n, pageSize: 20, kind } })]);
  const canWrite = can(me?.permissions ?? [], "finance:write");
  const keep = { ...(sp.q ? { q: sp.q } : {}), ...(kind ? { kind } : {}) };
  return (
    <div>
      <PageHeader title="Tiers" subtitle="Fournisseurs et clients des opérations, par référence (aucune donnée personnelle n’est copiée)." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Tiers" }]}
        action={canWrite && <LinkButton href="/finance/third-parties/new">Ajouter un tiers</LinkButton>} />
      {sp.ok && <Notice kind="ok">Tiers enregistré.</Notice>}
      <FilterBar reset={{ href: "/finance/third-parties", show: Object.keys(keep).length > 0 }}>
        <SearchInput defaultValue={sp.q} placeholder="Nom ou référence…" className="w-full sm:w-64" />
        <select name="kind" defaultValue={kind ?? ""} aria-label="Nature" className={selectCls}><option value="">Tous</option><option value="supplier">Fournisseurs</option><option value="customer">Clients</option></select>
      </FilterBar>
      <Table head={["Nom", "Nature", "Référence", "État"]} empty={data.items.length === 0 ? "Aucun tiers pour l’instant." : undefined}>
        {data.items.map((t) => (
          <tr key={t.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night">{canWrite ? <Link href={`/finance/third-parties/${t.id}`} className="hover:underline">{t.displayName}</Link> : t.displayName}</td>
            <td className="px-4 py-3 text-slate">{t.kind === "supplier" ? "Fournisseur" : "Client"}</td>
            <td className="px-4 py-3 font-mono text-xs text-slate">{t.externalRef ?? "—"}</td>
            <td className="px-4 py-3">{t.active ? <Badge tone="ok">Actif</Badge> : <Badge tone="off">Inactif</Badge>}</td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="tiers" href={(p) => `/finance/third-parties?${new URLSearchParams({ ...keep, page: String(p) })}`} />
    </div>
  );
}
