import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { dateFr, fmtBalance, fmtMoney } from "@/lib/money";
import { Badge, LinkButton, Notice, PageHeader, Table } from "@/components/ui/ui";
import type { FinBankAccount } from "@/lib/types";

export const metadata = { title: "Comptes bancaires" };

export default async function BankPage({ searchParams }: { searchParams: Promise<{ ok?: string }> }) {
  const { ok } = await searchParams;
  const [{ me }, data] = await Promise.all([currentStaff(), adminGet<{ items: FinBankAccount[] }>("/admin/finance/bank-accounts", "/finance/bank")]);
  const write = can(me?.permissions ?? [], "treasury:write");
  return (
    <div>
      <PageHeader title="Comptes bancaires" subtitle="Soldes du grand livre et des relevés importés. L’écart entre les deux se résorbe par le rapprochement." breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Comptes bancaires" }]}
        action={write && <LinkButton href="/finance/bank/new">Ajouter un compte</LinkButton>} />
      {ok && <Notice kind="ok">Compte enregistré.</Notice>}
      <Table head={["Compte", "Banque", "Compte comptable", "Grand livre", "Relevés", "Écart", "Dernier relevé", "État"]} empty={data.items.length === 0 ? "Aucun compte bancaire pour l’instant." : undefined} emptyHint="Ajoutez un compte, liez-le à un compte du plan comptable, puis importez un relevé.">
        {data.items.map((b) => {
          const diff = b.ledgerBalance - b.statementBalance;
          return (
            <tr key={b.id} className="hover:bg-page">
              <td className="px-4 py-3 font-medium text-night"><Link href={`/finance/bank/${b.id}`} className="hover:underline">{b.name}</Link>{b.numberLast4 && <span className="ml-2 font-mono text-xs font-normal text-slate">•••• {b.numberLast4}</span>}</td>
              <td className="px-4 py-3 text-slate">{b.bankName}</td>
              <td className="px-4 py-3 text-slate"><span className="font-mono text-xs">{b.accountCode}</span></td>
              <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums">{fmtBalance(b.ledgerBalance, b.currency)}</td>
              <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums">{b.lastStatementDate ? fmtBalance(b.statementBalance, b.currency) : "—"}</td>
              <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums">{b.lastStatementDate ? (diff === 0 ? <span className="text-[#007a52]">0</span> : <span className="font-medium text-night">{fmtMoney(diff, b.currency, { sign: true })}</span>) : "—"}</td>
              <td className="px-4 py-3 text-slate">{b.lastStatementDate ? dateFr(b.lastStatementDate) : "Aucun relevé"}</td>
              <td className="px-4 py-3">{b.active ? <Badge tone="ok">Actif</Badge> : <Badge tone="off">Inactif</Badge>}</td>
            </tr>
          );
        })}
      </Table>
    </div>
  );
}
