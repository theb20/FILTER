import Link from "next/link";
import { redirect } from "next/navigation";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { fromMicros, SOURCE_KIND_LABEL } from "@/lib/engine";
import { Badge, EmptyState, LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import type { SourcePage } from "@/lib/types";

export const metadata = { title: "Sources externes" };

export default async function SourcesPage({ searchParams }: { searchParams: Promise<{ ok?: string }> }) {
  const { ok } = await searchParams;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "engine:read")) redirect("/denied");
  const canWrite = can(me?.permissions ?? [], "engine:write");
  const data = await adminGet<SourcePage>("/admin/sources", "/engine/sources", { query: { pageSize: 100 } });
  const state = (s: SourcePage["items"][number]) =>
    !s.enabled ? { tone: "off" as const, label: "Désactivée" } : s.suspended ? { tone: "warn" as const, label: "Suspendue (budget)" } : { tone: "ok" as const, label: "Active" };
  return (
    <div>
      <PageHeader title="Sources externes" subtitle="Les services interrogés en plus du catalogue (recherche web, API constructeur, flux distributeur). Leur clé d'API n'est jamais enregistrée ici."
        action={canWrite && <LinkButton href="/engine/sources/new">Ajouter une source</LinkButton>} />
      {ok === "created" && <Notice kind="ok">Source créée. Réglez son budget, testez-la, puis activez-la.</Notice>}
      {data.items.length === 0 ? (
        <EmptyState title="Aucune source externe" hint="Ajoutez Brave Search pour que la recherche puisse aussi interroger le web. Le formulaire est prérempli."
          action={canWrite && <LinkButton href="/engine/sources/new">Ajouter une source</LinkButton>} />
      ) : (
        <Table head={["Source", "Type", "Coût par appel", "Priorité", "État", ""]}>
          {data.items.map((s) => {
            const st = state(s);
            return (
              <tr key={s.id} className="hover:bg-page">
                <td className="px-4 py-3"><Link href={`/engine/sources/${s.id}`} className="font-medium text-night hover:underline">{s.name}</Link><span className="block text-xs text-slate">{s.baseUrl}</span></td>
                <td className="px-4 py-3 text-slate">{SOURCE_KIND_LABEL[s.kind] ?? s.kind}</td>
                <td className="px-4 py-3 text-slate tabular-nums">{fromMicros(s.costPerCallMicros, s.currency)}</td>
                <td className="px-4 py-3 text-slate tabular-nums">{s.priority}</td>
                <td className="px-4 py-3"><Badge tone={st.tone}>{st.label}</Badge></td>
                <td className="px-4 py-3 text-right"><Link href={`/engine/sources/${s.id}`} className="text-link hover:underline">Ouvrir</Link></td>
              </tr>
            );
          })}
        </Table>
      )}
    </div>
  );
}
