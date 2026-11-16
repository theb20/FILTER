import Link from "next/link";
import { adminGet } from "@/lib/admin";
import { Badge, PageHeader, Table } from "@/components/ui/ui";
import { COMPANY_LABEL, COMPANY_TONE, customerName, dateFr, type CustomerPage } from "@/lib/types";

type SP = { q?: string; kind?: string; status?: string; page?: string };
const sel = "rounded border border-line bg-white px-3 py-2 text-sm focus:border-link focus:outline-none";

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
      <form className="mb-4 flex flex-wrap gap-2" role="search">
        <input name="q" defaultValue={q} placeholder="Nom, e-mail ou raison sociale" className={`${sel} w-full max-w-xs`} />
        <select name="kind" defaultValue={kind} className={sel} aria-label="Type de compte">
          <option value="">Tous les types</option><option value="individual">Particuliers</option><option value="company">Entreprises</option>
        </select>
        <select name="status" defaultValue={status} className={sel} aria-label="Statut">
          <option value="">Tous les statuts</option><option value="active">Actifs</option><option value="suspended">Suspendus</option>
        </select>
        <button className="rounded border border-line bg-white px-4 py-2 text-sm font-medium hover:bg-page">Filtrer</button>
      </form>
      <Table head={["Client", "Adresse e-mail", "Type", "Entreprise", "Statut", "Inscrit le"]} empty={data.items.length === 0 ? "Aucun client ne correspond." : undefined}>
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
