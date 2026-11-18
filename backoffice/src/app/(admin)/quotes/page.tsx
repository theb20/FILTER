import Link from "next/link";
import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import type { Doc, DocPage } from "@/lib/content";
import { QUOTE_STATUS, QUOTE_TONE, type QuoteStatus } from "@/lib/quotes";
import { Badge, FilterBar, Notice, PageHeader, Pagination, SearchInput, StatCard, Table } from "@/components/ui/ui";

export const metadata = { title: "Devis" };

type Status = QuoteStatus;

const PER_PAGE = 20;
const MAX = 1000;
type SP = { q?: string; status?: string; page?: string; ok?: string };
const str = (v: unknown) => (typeof v === "string" ? v : "");

export default async function QuotesPage({ searchParams }: { searchParams: Promise<SP> }) {
  const sp = await searchParams;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "quotes:read")) redirect("/denied");
  const status = sp.status && sp.status in QUOTE_STATUS ? (sp.status as Status) : undefined;
  const n = Math.max(1, Number.parseInt(sp.page ?? "1", 10) || 1);

  // Toutes les demandes (la recherche est faite par l'API), les filtres de statut et la pagination ici : on compte ainsi chaque statut.
  const all: Doc[] = [];
  let total = 0;
  for (let page = 1; all.length < MAX; page++) {
    const r = await adminGet<DocPage>("/admin/quote-requests", "/quotes", { query: { q: sp.q || undefined, page, pageSize: 100 } });
    all.push(...r.items);
    total = r.page.total;
    if (page >= r.page.totalPages) break;
  }
  all.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  const count = (s: Status) => all.filter((d) => d.status === s).length;
  const shown = status ? all.filter((d) => d.status === status) : all;
  const pages = Math.max(1, Math.ceil(shown.length / PER_PAGE));
  const rows = shown.slice((n - 1) * PER_PAGE, n * PER_PAGE);
  const keep = { ...(sp.q ? { q: sp.q } : {}), ...(status ? { status } : {}) };
  const link = (p: number) => `/quotes?${new URLSearchParams({ ...keep, page: String(p) })}`;
  const tile = (s: Status | undefined) => `/quotes${new URLSearchParams({ ...(sp.q ? { q: sp.q } : {}), ...(s ? { status: s } : {}) }).toString() ? `?${new URLSearchParams({ ...(sp.q ? { q: sp.q } : {}), ...(s ? { status: s } : {}) })}` : ""}`;
  const fmt = (d: string) => new Date(d).toLocaleDateString("fr-FR", { day: "2-digit", month: "short", year: "numeric" });

  return (
    <div>
      <PageHeader title="Devis" subtitle="Toutes les demandes de devis reçues depuis le site, de la plus récente à la plus ancienne." />
      {sp.ok === "deleted" && <Notice kind="ok">Demande supprimée.</Notice>}
      <div data-guide="quotes-stats" className="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-5">
        <StatCard label="Toutes" value={all.length} href={tile(undefined)} />
        {(Object.keys(QUOTE_STATUS) as Status[]).map((s) => <StatCard key={s} label={QUOTE_STATUS[s]} value={count(s)} href={tile(s)} tone={s === "new" && count(s) > 0 ? "alert" : "default"} />)}
      </div>
      <FilterBar reset={{ href: "/quotes", show: Object.keys(keep).length > 0 }}>
        <SearchInput defaultValue={sp.q} placeholder="Nom, e-mail, entreprise, référence…" className="w-full sm:w-80" />
        {status && <input type="hidden" name="status" value={status} />}
      </FilterBar>
      <Table head={["Reçue le", "Demandeur", "Entreprise", "Objet / référence", "Articles", "Statut", ""]} empty={rows.length === 0 ? (Object.keys(keep).length ? "Aucune demande ne correspond." : "Aucune demande de devis pour l'instant.") : undefined}>
        {rows.map((d) => {
          const s = (str(d.status) in QUOTE_STATUS ? d.status : "new") as Status;
          const items = Array.isArray(d.items) ? d.items.length : 0;
          return (
            <tr key={d.id} className="hover:bg-page">
              <td className="whitespace-nowrap px-4 py-3 text-slate">{fmt(d.createdAt)}</td>
              <td className="px-4 py-3"><Link href={`/quotes/${d.id}`} className="font-medium text-night hover:underline">{str(d.name) || "—"}</Link><span className="block text-xs text-slate">{str(d.email)}</span></td>
              <td className="px-4 py-3 text-slate">{str(d.company) || "Particulier"}</td>
              <td className="px-4 py-3 text-slate">{str(d.subject) || "—"}{str(d.reference) && <span className="block text-xs">Réf. {str(d.reference)}</span>}</td>
              <td className="px-4 py-3 text-slate">{items}</td>
              <td className="px-4 py-3"><Badge tone={QUOTE_TONE[s]}>{QUOTE_STATUS[s]}</Badge></td>
              <td className="px-4 py-3 text-right"><Link href={`/quotes/${d.id}`} className="text-link hover:underline">Ouvrir</Link></td>
            </tr>
          );
        })}
      </Table>
      <Pagination page={Math.min(n, pages)} totalPages={pages} total={shown.length} noun="devis" href={link} />
      {total > all.length && <p className="mt-3 text-xs text-slate">Les {MAX} demandes les plus récentes sont chargées sur {total}. Affinez avec la recherche.</p>}
    </div>
  );
}
