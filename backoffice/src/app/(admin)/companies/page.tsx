import Link from "next/link";
import { adminGet } from "@/lib/admin";
import { Badge, PageHeader, Table } from "@/components/ui/ui";
import { COMPANY_LABEL, COMPANY_TONE, dateFr, type CompanyPage } from "@/lib/types";

const TABS = [["pending", "En attente"], ["approved", "Approuvées"], ["rejected", "Rejetées"]] as const;

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
      <nav className="mb-4 flex gap-1 border-b border-line text-sm" aria-label="Statut">
        {TABS.map(([s, label]) => (
          <Link key={s} href={href(1, s)} aria-current={s === status ? "page" : undefined} className={`-mb-px border-b-2 px-4 py-2 ${s === status ? "border-night font-medium text-night" : "border-transparent text-slate hover:text-night"}`}>{label}</Link>
        ))}
      </nav>
      <form className="mb-4 flex gap-2" role="search">
        <input type="hidden" name="status" value={status} />
        <input name="q" defaultValue={q} placeholder="Raison sociale ou n° fiscal" className="w-full max-w-sm rounded border border-line bg-white px-3 py-2 text-sm focus:border-link focus:outline-none" />
        <button className="rounded border border-line bg-white px-4 py-2 text-sm font-medium hover:bg-page">Rechercher</button>
      </form>
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
