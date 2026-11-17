import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table, selectCls } from "@/components/ui/ui";
import { ACCOUNT_KIND_LABEL, type AccountKind, type FinAccountPage } from "@/lib/types";

export const metadata = { title: "Plan comptable" };
type SP = { q?: string; page?: string; kind?: string; ok?: string };

export default async function AccountsPage({ searchParams }: { searchParams: Promise<SP> }) {
  const sp = await searchParams;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);
  const kind = sp.kind && sp.kind in ACCOUNT_KIND_LABEL ? sp.kind : undefined;
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<FinAccountPage>("/admin/finance/accounts", "/finance/accounts", { query: { q: sp.q, page: n, pageSize: 50, kind } })]);
  const admin = can(me?.permissions ?? [], "finance:admin");
  const keep = { ...(sp.q ? { q: sp.q } : {}), ...(kind ? { kind } : {}) };
  const link = (p: number) => `/finance/accounts?${new URLSearchParams({ ...keep, page: String(p) })}`;
  return (
    <div>
      <PageHeader title="Plan comptable" subtitle="Les comptes utilisables dans les écritures. Aucun plan n'est préinstallé : il vient de votre expert-comptable." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Plan comptable" }]}
        action={admin && <><LinkButton href="/finance/accounts/import" variant="ghost">Importer (CSV)</LinkButton><LinkButton href="/finance/accounts/new">Ajouter un compte</LinkButton></>} />
      {sp.ok && <Notice kind="ok">Opération effectuée.</Notice>}
      <FilterBar reset={{ href: "/finance/accounts", show: Object.keys(keep).length > 0 }}>
        <SearchInput defaultValue={sp.q} placeholder="Code ou libellé…" className="w-full sm:w-64" />
        <select name="kind" defaultValue={kind ?? ""} aria-label="Nature" className={selectCls}><option value="">Toutes les natures</option>{(Object.keys(ACCOUNT_KIND_LABEL) as AccountKind[]).map((k) => <option key={k} value={k}>{ACCOUNT_KIND_LABEL[k]}</option>)}</select>
      </FilterBar>
      <Table head={["Code", "Libellé", "Nature", "État"]} empty={data.items.length === 0 ? (Object.keys(keep).length ? "Aucun compte ne correspond." : "Aucun compte pour l'instant.") : undefined}
        emptyHint={data.items.length === 0 && !Object.keys(keep).length ? "Importez le plan de votre expert-comptable ou ajoutez les comptes un à un." : undefined}>
        {data.items.map((a) => (
          <tr key={a.id} className="hover:bg-page">
            <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-night"><Link href={`/finance/accounts/${a.id}`} className="hover:underline">{a.code}</Link></td>
            <td className="px-4 py-3 text-night">{a.label}</td>
            <td className="px-4 py-3 text-slate">{ACCOUNT_KIND_LABEL[a.kind]}</td>
            <td className="px-4 py-3">{a.active ? <Badge tone="ok">Actif</Badge> : <Badge tone="off">Inactif</Badge>}</td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="comptes" href={link} />
    </div>
  );
}
