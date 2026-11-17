import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts } from "@/lib/finance";
import { dateFr, fmtBalance, fmtMoney } from "@/lib/money";
import { Badge, Card, LinkButton, Notice, PageHeader, StatCard, Table } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { BankFields, StatementImportForm } from "@/components/finance/bank-forms";
import { dateFr as dateTimeFr, type FinBankAccount, type FinStatement, type ReconSummary } from "@/lib/types";
import { updateBankAccount } from "../../actions";

export default async function BankAccountPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const here = `/finance/bank/${id}`;
  const [{ me }, b, sum, sts] = await Promise.all([currentStaff(), adminGet<FinBankAccount>(`/admin/finance/bank-accounts/${id}`, here), adminGet<ReconSummary>(`/admin/finance/bank-accounts/${id}/reconciliation-summary`, here), adminGet<{ items: FinStatement[] }>(`/admin/finance/bank-accounts/${id}/statements`, here)]);
  const perms = me?.permissions ?? [];
  const write = can(perms, "treasury:write");
  const accounts = write ? await allAccounts(here, true) : [];
  const cur = b.currency;
  return (
    <div className="max-w-5xl space-y-5">
      <PageHeader title={b.name} subtitle={`${b.bankName} · compte comptable ${b.accountCode} — ${b.accountLabel}${b.numberLast4 ? ` · •••• ${b.numberLast4}` : ""}`} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Comptes bancaires", href: "/finance/bank" }, { label: b.name }]}
        action={<><LinkButton href={`/finance/bank/${id}/lines`} variant="ghost">Mouvements du relevé</LinkButton>{can(perms, "treasury:reconcile") && <LinkButton href={`/finance/bank/${id}/reconcile`}>Rapprocher</LinkButton>}</>} />
      {ok && <Notice kind="ok">{ok === "created" ? "Compte créé." : "Compte modifié."}</Notice>}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Solde du grand livre" value={fmtBalance(sum.ledgerBalance, cur)} hint="écritures validées" />
        <StatCard label="Solde du relevé" value={b.lastStatementDate ? fmtBalance(sum.statementBalance, cur) : "—"} hint={b.lastStatementDate ? `au ${dateFr(b.lastStatementDate)}` : "aucun relevé importé"} />
        <StatCard label="Écart" value={b.lastStatementDate ? fmtMoney(sum.difference, cur, { sign: true }) : "—"} hint="grand livre − relevé" tone={b.lastStatementDate && sum.difference !== 0 ? "alert" : "default"} />
        <StatCard label="Non rapprochés" value={`${sum.unmatchedStatementCount} / ${sum.unmatchedLedgerCount}`} hint="lignes de relevé / d’écritures" href={`/finance/bank/${id}/reconcile`} tone={sum.unmatchedStatementCount + sum.unmatchedLedgerCount > 0 ? "alert" : "default"} />
      </div>

      {b.lastStatementDate && (
        <Card title="Explication de l’écart" description="L’écart n’est jamais absorbé : il est expliqué par des éléments identifiables, ou il reste visible.">
          <dl className="grid gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
            <div className="flex justify-between"><dt className="text-slate">Écritures non encore au relevé</dt><dd className="font-medium tabular-nums text-night">{fmtMoney(sum.unmatchedLedger, cur, { sign: true })}</dd></div>
            <div className="flex justify-between"><dt className="text-slate">Mouvements du relevé non comptabilisés</dt><dd className="font-medium tabular-nums text-night">{fmtMoney(-sum.unmatchedStatement, cur, { sign: true })}</dd></div>
            <div className="flex justify-between"><dt className="text-slate">Écart d’ouverture (grand livre ≠ solde d’ouverture)</dt><dd className={`font-medium tabular-nums ${sum.openingGap === 0 ? "text-night" : "text-danger"}`}>{fmtMoney(sum.openingGap, cur, { sign: true })}</dd></div>
            <div className="flex justify-between"><dt className="text-slate">Contrôle de cohérence (doit être 0)</dt><dd className={`font-medium tabular-nums ${sum.unexplained === 0 ? "text-[#007a52]" : "text-danger"}`}>{fmtMoney(sum.unexplained, cur)}</dd></div>
          </dl>
          {sum.unexplained !== 0 && <p className="mt-3 text-sm text-danger">Anomalie de données : le contrôle de cohérence n’est pas à zéro. À signaler à l’administrateur.</p>}
          {sum.openingGap !== 0 && <p className="mt-3 text-sm text-[#7a5a00]">Le grand livre avant la date d’ouverture ne correspond pas au solde d’ouverture saisi : vérifiez l’écriture d’à-nouveau ou le solde d’ouverture.</p>}
        </Card>
      )}

      {write && b.active && (
        <Card title="Importer un relevé" description="Simulez d’abord : rien n’est écrit tant que vous n’avez pas cliqué sur « Importer ». Un relevé importé ne se modifie plus.">
          <StatementImportForm accountId={id} currency={cur} />
        </Card>
      )}

      <section>
        <h2 className="mb-3 text-lg font-semibold text-night">Relevés importés ({sts.items.length})</h2>
        <Table head={["Fichier", "Période", "Lignes", "Nouvelles", "Ignorées", "Solde de clôture annoncé", "Importé le"]} empty={sts.items.length === 0 ? "Aucun relevé importé." : undefined}>
          {sts.items.map((s) => (
            <tr key={s.id}>
              <td className="px-4 py-3 text-night">{s.filename}</td>
              <td className="whitespace-nowrap px-4 py-3 text-slate">{dateFr(s.periodStart)} → {dateFr(s.periodEnd)}</td>
              <td className="px-4 py-3 text-slate">{s.lineCount}</td>
              <td className="px-4 py-3 text-slate">{s.insertedCount}</td>
              <td className="px-4 py-3 text-slate">{s.skippedCount}</td>
              <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums text-slate">{s.closingBalance === null ? "—" : fmtBalance(s.closingBalance, cur)}</td>
              <td className="whitespace-nowrap px-4 py-3 text-slate">{dateTimeFr(s.importedAt)}</td>
            </tr>
          ))}
        </Table>
      </section>

      {write && (
        <Card title="Modifier le compte">
          <ActionForm action={updateBankAccount.bind(null, id)} submit="Enregistrer"><BankFields accounts={accounts} currency={cur} value={b} /></ActionForm>
        </Card>
      )}
      <p className="text-sm text-slate"><Link href={`/finance/reports/ledger?accountId=${b.accountId}`} className="text-link hover:underline">Voir le grand livre du compte {b.accountCode} →</Link> {!b.active && <Badge tone="off">Inactif</Badge>}</p>
    </div>
  );
}
