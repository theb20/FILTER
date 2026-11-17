import Link from "next/link";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { journals } from "@/lib/finance";
import { Badge, LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import { JOURNAL_KIND_LABEL } from "@/lib/types";

export const metadata = { title: "Journaux" };

export default async function JournalsPage({ searchParams }: { searchParams: Promise<{ ok?: string }> }) {
  const { ok } = await searchParams;
  const [{ me }, list] = await Promise.all([currentStaff(), journals("/finance/journals")]);
  const admin = can(me?.permissions ?? [], "finance:admin");
  return (
    <div>
      <PageHeader title="Journaux" subtitle="Les journaux regroupent les écritures (ventes, achats, banque…). Aucun journal n'est préinstallé." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Journaux" }]}
        action={admin && <LinkButton href="/finance/journals/new">Ajouter un journal</LinkButton>} />
      {ok && <Notice kind="ok">Journal enregistré.</Notice>}
      <Table head={["Code", "Libellé", "Nature", "État", ""]} empty={list.length === 0 ? "Aucun journal pour l'instant." : undefined} emptyHint="Créez les journaux définis avec votre expert-comptable.">
        {list.map((j) => (
          <tr key={j.id} className="hover:bg-page">
            <td className="px-4 py-3 font-mono text-xs text-night">{j.code}</td>
            <td className="px-4 py-3 text-night">{j.label}</td>
            <td className="px-4 py-3 text-slate">{JOURNAL_KIND_LABEL[j.kind]}</td>
            <td className="px-4 py-3">{j.active ? <Badge tone="ok">Actif</Badge> : <Badge tone="off">Inactif</Badge>}</td>
            <td className="px-4 py-3 text-right">{admin && <Link href={`/finance/journals/${j.id}`} className="text-link hover:underline">Modifier</Link>}</td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
