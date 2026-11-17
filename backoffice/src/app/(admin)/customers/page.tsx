import Link from "next/link";
import { adminGet } from "@/lib/admin";
import { Badge, FilterBar, PageHeader, Pagination, SearchInput, Table, selectCls } from "@/components/ui/ui";
import { COMPANY_LABEL, COMPANY_TONE, customerName, dateFr, type CustomerPage } from "@/lib/types";

type SP = { q?: string; kind?: string; status?: string; page?: string };
export const metadata = { title: "Clients" };

export default async function CustomersPage({ searchParams }: { searchParams: Promise<SP> }) {
  const sp = await searchParams;
  const page = Math.max(1, Number(sp.page) || 1);
  const q = sp.q?.trim() ?? "";
  const kind = sp.kind === "individual" || sp.kind === "company" ? sp.kind : "";
  const status = sp.status === "active" || sp.status === "suspended" ? sp.status : "";
  const data = await adminGet<CustomerPage>("/admin/customers", "/customers", { query: { q, kind, status, page, pageSize: 20 } });
  const href = (p: number) => `/customers?${new URLSearchParams({ ...(q && { q }), ...(kind && { kind }), ...(status && { status }), page: String(p) })}`;

  return (
    <div>
      <PageHeader title="Clients" subtitle={`${data.page.total} compte${data.page.total > 1 ? "s" : ""}`} />
      <FilterBar reset={{ href: "/customers", show: !!(q || kind || status) }}>
        <SearchInput defaultValue={q} placeholder="Nom, e-mail ou raison sociale" className="w-full sm:w-72" />
        <select name="kind" defaultValue={kind} className={selectCls} aria-label="Type de compte">
          <option value="">Tous les types</option><option value="individual">Particuliers</option><option value="company">Entreprises</option>
        </select>
        <select name="status" defaultValue={status} className={selectCls} aria-label="Statut">
          <option value="">Tous les statuts</option><option value="active">Actifs</option><option value="suspended">Suspendus</option>
        </select>
      </FilterBar>
      <Table head={["Client", "Adresse e-mail", "Type", "Entreprise", "Statut", "Inscrit le"]} empty={data.items.length === 0 ? "Aucun client ne correspond." : undefined} emptyHint="Modifiez la recherche ou les filtres.">
        {data.items.map((c) => (
          <tr key={c.id} className="hover:bg-page">
            <td className="px-4 py-3 font-medium text-night"><Link href={`/customers/${c.id}`} className="hover:underline">{customerName(c)}</Link></td>
            <td className="px-4 py-3 text-slate">{c.email}</td>
            <td className="px-4 py-3 text-slate">{c.kind === "company" ? "Entreprise" : "Particulier"}</td>
            <td className="px-4 py-3 text-slate">{c.company ? <>{c.company.legalName} <Badge tone={COMPANY_TONE[c.company.status]}>{COMPANY_LABEL[c.company.status]}</Badge></> : "—"}</td>
            <td className="px-4 py-3"><Badge tone={c.status === "active" ? "ok" : "off"}>{c.status === "active" ? "Actif" : "Suspendu"}</Badge></td>
            <td className="px-4 py-3 text-slate">{dateFr(c.createdAt)}</td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="clients" href={href} />
    </div>
  );
}
