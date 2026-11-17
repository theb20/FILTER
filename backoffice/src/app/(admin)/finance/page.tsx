import Link from "next/link";
import { ArrowRight, BookMarked, CalendarDays, FilePenLine, ShieldCheck } from "lucide-react";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, financeSettings, fiscalYears, journals } from "@/lib/finance";
import { fmtMoney, today } from "@/lib/money";
import { Card, EmptyState, LinkButton, PageHeader, StatCard } from "@/components/ui/ui";
import type { FinApprovalPage, FinEntryPage } from "@/lib/types";

export const metadata = { title: "Finance" };

const total = async <T extends { page: { total: number } }>(path: string, returnTo: string, query: Record<string, string | number | undefined>) =>
  (await adminGet<T>(path, returnTo, { query: { ...query, page: 1, pageSize: 1 } })).page.total;

/** Vue d'ensemble : uniquement des nombres lus dans la comptabilité. Aucun indicateur n'est calculé tant que des écritures validées n'existent pas. */
export default async function FinancePage() {
  const { me } = await currentStaff();
  const perms = me?.permissions ?? [];
  const [settings, accounts, jrn, years, drafts, posted, pending] = await Promise.all([
    financeSettings("/finance"),
    allAccounts("/finance"),
    journals("/finance"),
    fiscalYears("/finance"),
    total<FinEntryPage>("/admin/finance/entries", "/finance", { status: "draft" }),
    total<FinEntryPage>("/admin/finance/entries", "/finance", { status: "posted" }),
    total<FinApprovalPage>("/admin/finance/approvals", "/finance", { status: "pending" }),
  ]);
  const t = today();
  const current = years.find((y) => y.startDate <= t && t <= y.endDate);
  const period = current?.periods.find((p) => p.startDate <= t && t <= p.endDate);
  const admin = can(perms, "finance:admin");

  const steps = [
    { done: accounts.length > 0, label: "Plan comptable", hint: "Importez le plan fourni par votre expert-comptable, ou créez les comptes un à un.", href: "/finance/accounts/import", icon: BookMarked },
    { done: jrn.length > 0, label: "Journaux", hint: "Ventes, achats, banque, caisse, opérations diverses : selon l'organisation de votre expert-comptable.", href: "/finance/journals/new", icon: FilePenLine },
    { done: years.length > 0, label: "Exercice comptable", hint: "Définissez les dates de l'exercice : les périodes mensuelles en découlent.", href: "/finance/periods/new", icon: CalendarDays },
  ];
  const ready = steps.every((s) => s.done);

  return (
    <div>
      <PageHeader title="Finance" subtitle={`Comptabilité en ${settings.baseCurrency}. Les montants sont des entiers d'unités mineures ; aucune donnée d'exemple n'est affichée.`}
        action={ready && can(perms, "finance:write") ? <LinkButton href="/finance/entries/new">Saisir une écriture</LinkButton> : undefined} />

      {!ready && (
        <Card title="Mise en place" description="Rien n'est préconfiguré : le plan comptable, les journaux et l'exercice sont définis par vous, avec votre expert-comptable.">
          <ol className="space-y-3">
            {steps.map((s, i) => (
              <li key={s.label} className="flex items-start gap-3">
                <span className={`mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full text-xs font-bold ${s.done ? "bg-ok/15 text-[#007a52]" : "bg-page text-slate"}`}>{s.done ? "✓" : i + 1}</span>
                <div className="min-w-0 flex-1">
                  <p className="font-medium text-night">{s.label}</p>
                  {!s.done && <p className="text-sm text-slate">{s.hint}</p>}
                </div>
                {!s.done && admin && <Link href={s.href} className="inline-flex shrink-0 items-center gap-1 text-sm text-link hover:underline">Configurer <ArrowRight className="h-4 w-4" aria-hidden /></Link>}
              </li>
            ))}
          </ol>
          {!admin && <p className="mt-4 text-sm text-slate">La configuration demande la permission « Administrer — finance ».</p>}
        </Card>
      )}

      {ready && (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Écritures validées" value={posted} href="/finance/entries?status=posted" hint="immuables" />
            <StatCard label="Brouillons" value={drafts} href="/finance/entries?status=draft" hint="à valider ou supprimer" tone={drafts > 0 ? "alert" : "default"} />
            <StatCard label="Approbations en attente" value={pending} href="/finance/approvals" hint="double validation" tone={pending > 0 ? "alert" : "default"} />
            <StatCard label="Plan comptable" value={accounts.length} href="/finance/accounts" hint={`${jrn.length} journal(aux)`} />
          </div>
          <div className="mt-6 grid gap-4 lg:grid-cols-2">
            <Card title="Période en cours">
              {period && current ? (
                <dl className="space-y-1.5 text-sm">
                  <div className="flex justify-between"><dt className="text-slate">Exercice</dt><dd className="font-medium text-night">{current.label}</dd></div>
                  <div className="flex justify-between"><dt className="text-slate">Période</dt><dd className="font-medium text-night">{period.startDate} → {period.endDate}</dd></div>
                  <div className="flex justify-between"><dt className="text-slate">État</dt><dd className="font-medium text-night">{period.status === "open" ? "Ouverte" : "Clôturée"}</dd></div>
                </dl>
              ) : (
                <EmptyState compact title="Aucun exercice ne couvre la date du jour" hint="Ouvrez l'exercice correspondant pour saisir des écritures à cette date." />
              )}
            </Card>
            <Card title="Contrôles" description="Ce que le système garantit, et ce qu'il vous reste à faire.">
              <ul className="space-y-2 text-sm text-slate">
                <li className="flex gap-2"><ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-ok" aria-hidden />Une écriture validée ne se modifie ni ne se supprime : elle se contre-passe.</li>
                <li className="flex gap-2"><ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-ok" aria-hidden />{settings.fourEyesMinAmount === 0 ? "Une autre personne valide toujours chaque écriture." : `Une autre personne valide les écritures de ${fmtMoney(settings.fourEyesMinAmount, settings.baseCurrency)} et plus.`}</li>
                <li className="flex gap-2"><ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-ok" aria-hidden />La réouverture d’une période clôturée exige l’approbation d’une autre personne.</li>
              </ul>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}
