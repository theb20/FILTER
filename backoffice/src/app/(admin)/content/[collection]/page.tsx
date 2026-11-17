import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { kindBySlug } from "@/lib/content-kinds";
import { label, type DocPage } from "@/lib/content";
import { Badge, FilterBar, LinkButton, Notice, PageHeader, Pagination, SearchInput, Table } from "@/components/ui/ui";

type SP = { q?: string; page?: string; ok?: string };

export async function generateMetadata({ params }: { params: Promise<{ collection: string }> }) {
  const k = kindBySlug((await params).collection);
  return { title: k?.title ?? "Contenu" };
}

export default async function ContentListPage({ params, searchParams }: { params: Promise<{ collection: string }>; searchParams: Promise<SP> }) {
  const { collection } = await params;
  const k = kindBySlug(collection);
  if (!k || !k.ops.includes("list")) notFound();
  const { q, page, ok } = await searchParams;
  const n = Math.max(1, Number.parseInt(page ?? "1", 10) || 1);
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], k.perm.read)) redirect("/denied");
  const canWrite = can(me?.permissions ?? [], k.perm.write);
  const data = await adminGet<DocPage>(`/admin/${collection}`, `/content/${collection}`, { query: { q, page: n, pageSize: 20 } });
  const cols = k.summary.map((name) => k.fields.find((f) => f.name === name)!).filter(Boolean);
  const link = (p: number) => `/content/${collection}?${new URLSearchParams({ ...(q ? { q } : {}), page: String(p) })}`;
  const cell = (f: (typeof cols)[number], v: unknown) => {
    if (f.type.startsWith("enum:")) return v ? <Badge tone={v === "published" || v === "subscribed" || v === "handled" ? "ok" : v === "pending" || v === "new" ? "warn" : "off"}>{f.labels?.[String(v)] ?? String(v)}</Badge> : "—";
    if (f.type === "bool") return v ? "Oui" : "Non";
    return label(v) || "—";
  };
  return (
    <div>
      <PageHeader title={k.title} action={canWrite && k.ops.includes("create") && <LinkButton href={`/content/${collection}/new`}>Ajouter</LinkButton>} />
      {ok === "deleted" && <Notice kind="ok">Élément supprimé.</Notice>}
      <FilterBar reset={{ href: `/content/${collection}`, show: !!q }}>
        <SearchInput defaultValue={q} placeholder="Rechercher…" className="w-full sm:w-80" />
      </FilterBar>
      <Table head={[...cols.map((c) => c.label), "Modifié le"]} empty={data.items.length === 0 ? (q ? "Aucun résultat." : "Rien pour l'instant.") : undefined}>
        {data.items.map((d, i) => (
          <tr key={d.id} className="hover:bg-page">
            {cols.map((c, j) => (
              <td key={c.name} className={`px-4 py-3 ${j === 0 ? "font-medium text-night" : "text-slate"}`}>
                {j === 0 ? <Link href={`/content/${collection}/${d.id}`} className="hover:underline">{cell(c, d[c.name])}</Link> : cell(c, d[c.name])}
              </td>
            ))}
            <td className="px-4 py-3 text-slate" data-row={i}>{new Date(d.updatedAt).toLocaleDateString("fr-FR")}</td>
          </tr>
        ))}
      </Table>
      <Pagination page={data.page.page} totalPages={data.page.totalPages} total={data.page.total} noun="éléments" href={link} />
    </div>
  );
}
