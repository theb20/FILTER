import Link from "next/link";
import { adminGet } from "@/lib/admin";
import { Badge, FilterBar, PageHeader, Pagination, SearchInput, Table } from "@/components/ui/ui";
import { COMPANY_LABEL, COMPANY_TONE, dateFr, type CompanyPage } from "@/lib/types";

const TABS = [["pending", "En attente"], ["approved", "Approuvées"], ["rejected", "Rejetées"]] as const;

export const metadata = { title: "Entreprises" };

export default async function CompaniesPage({ searchParams }: { searchParams: Promise<{ q?: string; status?: string; page?: string }> }) {
  const sp = await searchParams;
  const page = Math.max(1, Number(sp.page) || 1);
  const q = sp.q?.trim() ?? "";
  const status = TABS.some(([s]) => s === sp.status) ? sp.status! : "pending";
  const data = await adminGet<CompanyPage>("/admin/companies", "/companies", { query: { q, status, page, pageSize: 20 } });
  const href = (p: number, s = status) => `/companies?${new URLSearchParams({ ...(q && { q }), status: s, page: String(p) })}`;

  return (
    <div>
      <PageHeader title="Entreprises" subtitle="Validation des comptes professionnels" />
      <nav data-guide="tabs" className="mb-4 flex gap-1 border-b border-line text-sm" aria-label="Statut">
        {TABS.map(([s, label]) => (
          <Link key={s} href={href(1, s)} aria-current={s === status ? "page" : undefined} className={`-mb-px border-b-2 px-4 py-2.5 ${s === status ? "border-night font-medium text-night" : "border-transparent text-slate hover:text-night"}`}>{label}</Link>
        ))}
      </nav>
      <FilterBar reset={{ href: `/companies?status=${status}`, show: !!q }}>
        <input type="hidden" name="status" value={status} />
        <SearchInput defaultValue={q} placeholder="Raison sociale ou n° fiscal" className="w-full sm:w-80" />
      </FilterBar>
      <Table head={["Raison sociale", "N° fiscal", "Pays", "Statut", "Décision"]} empty={data.items.length === 0 ? "Aucune entreprise." : undefined}>
        {data.items.map((c) => (
          <tr key={c.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/companies/${c.id}`} className="hover:underline">{c.legalName}</Link></td>
            <td className="px-4 py-3 text-slate">{c.taxId}</td>
            <td className="px-4 py-3 text-slate">{c.country}</td>
            <td className="px-4 py-3"><Badge tone={COMPANY_TONE[c.status]}>{COMPANY_LABEL[c.status]}</Badge></td>
            <td className="px-4 py-3 text-slate">{dateFr(c.decidedAt)}</td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="entreprises" href={(p) => href(p)} />
    </div>
  );
}
