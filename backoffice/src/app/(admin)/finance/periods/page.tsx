import Link from "next/link";
import { can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { fiscalYears } from "@/lib/finance";
import { dateFr } from "@/lib/money";
import { Badge, LinkButton, PageHeader, Table } from "@/components/ui/ui";
import { YEAR_STATUS_LABEL } from "@/lib/types";

export const metadata = { title: "Exercices et périodes" };

export default async function PeriodsPage() {
  const [{ me }, years] = await Promise.all([currentStaff(), fiscalYears("/finance/periods")]);
  const admin = can(me?.permissions ?? [], "finance:admin");
  return (
    <div>
      <PageHeader title="Exercices et périodes" subtitle="Chaque exercice est découpé en périodes mensuelles. Une période clôturée refuse toute écriture." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Exercices" }]}
        action={admin && <LinkButton href="/finance/periods/new">Ouvrir un exercice</LinkButton>} />
      <Table head={["Exercice", "Du", "Au", "Périodes closes", "État", ""]} empty={years.length === 0 ? "Aucun exercice pour l'instant." : undefined} emptyHint="Sans exercice, aucune écriture ne peut être saisie.">
        {years.map((y) => {
          const closed = y.periods.filter((p) => p.status === "closed").length;
          return (
            <tr key={y.id} className="hover:bg-page">
              <td className="px-4 py-3 font-medium text-night"><Link href={`/finance/periods/${y.id}`} className="hover:underline">{y.label}</Link></td>
              <td className="px-4 py-3 text-slate">{dateFr(y.startDate)}</td>
              <td className="px-4 py-3 text-slate">{dateFr(y.endDate)}</td>
              <td className="px-4 py-3 text-slate">{closed} / {y.periods.length}</td>
              <td className="px-4 py-3"><Badge tone={y.status === "open" ? "ok" : y.status === "closed" ? "off" : "warn"}>{YEAR_STATUS_LABEL[y.status]}</Badge></td>
              <td className="px-4 py-3 text-right"><Link href={`/finance/periods/${y.id}`} className="text-link hover:underline">Ouvrir</Link></td>
            </tr>
          );
        })}
      </Table>
    </div>
  );
}
