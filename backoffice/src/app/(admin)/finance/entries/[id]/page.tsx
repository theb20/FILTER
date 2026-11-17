import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, financeSettings, journals } from "@/lib/finance";
import { dateFr, fmtMoney, today } from "@/lib/money";
import { Badge, Card, inputCls, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { EntryEditor } from "@/components/finance/entry-editor";
import { dateFr as dateTimeFr } from "@/lib/types";
import type { FinEntry } from "@/lib/types";
import { deleteEntry, postEntry, reverseEntry, updateEntry } from "../../actions";

const NOTICES: Record<string, string> = { created: "Brouillon enregistré.", updated: "Brouillon modifié.", posted: "Écriture validée : elle est désormais définitive.", reversal: "Brouillon de contre-passation créé : il doit être validé par une autre personne." };

export default async function EntryPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const here = `/finance/entries/${id}`;
  const [{ me }, e, settings] = await Promise.all([currentStaff(), adminGet<FinEntry>(`/admin/finance/entries/${id}`, here), financeSettings(here)]);
  const perms = me?.permissions ?? [];
  const posted = e.status === "posted";
  const isAuthor = me?.id === e.createdBy;
  const needsOther = e.totalDebit >= settings.fourEyesMinAmount;
  const canPost = can(perms, "finance:approve") && !posted;
  const [accounts, jrn] = !posted && can(perms, "finance:write") ? await Promise.all([allAccounts(here, true), journals(here, true)]) : [[], []];
  const cur = e.currency;

  return (
    <div className="max-w-5xl space-y-5">
      <PageHeader title={e.number ?? "Brouillon"} subtitle={e.label} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Écritures", href: "/finance/entries" }, { label: e.number ?? "Brouillon" }]}
        action={posted ? <Badge tone="ok">Validée</Badge> : <Badge tone="warn">Brouillon</Badge>} />
      {ok && NOTICES[ok] && <Notice kind="ok">{NOTICES[ok]}</Notice>}

      <Card title="Écriture">
        <dl className="grid gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
          <div className="flex justify-between"><dt className="text-slate">Date comptable</dt><dd className="font-medium text-night">{dateFr(e.accountingDate)}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Journal</dt><dd className="font-medium text-night">{e.journalCode}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Référence</dt><dd className="font-medium text-night">{e.reference ?? "—"}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Origine</dt><dd className="font-medium text-night">{e.origin === "reversal" ? "Contre-passation" : e.origin === "operation" ? "Opération" : "Saisie manuelle"}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Saisie le</dt><dd className="font-medium text-night">{dateTimeFr(e.createdAt)}</dd></div>
          {posted && <div className="flex justify-between"><dt className="text-slate">Validée le</dt><dd className="font-medium text-night">{dateTimeFr(e.postedAt)}</dd></div>}
        </dl>
        {e.reversesEntryId && <p className="mt-3 text-sm text-slate">Contre-passe <Link className="text-link hover:underline" href={`/finance/entries/${e.reversesEntryId}`}>l’écriture d’origine</Link>. Motif : {e.reversalReason}</p>}
        {e.reversedByEntryId && <p className="mt-3 text-sm text-slate">Cette écriture a été contre-passée par <Link className="text-link hover:underline" href={`/finance/entries/${e.reversedByEntryId}`}>une écriture de régularisation</Link>.</p>}
      </Card>

      {(posted || !can(perms, "finance:write")) ? (
        <div className="overflow-x-auto rounded-xl border border-line bg-white shadow-card">
          <table className="w-full min-w-[640px] text-sm">
            <thead className="border-b border-line bg-page/70 text-left text-xs uppercase tracking-wide text-slate"><tr><th className="px-4 py-3 font-semibold">Compte</th><th className="px-4 py-3 font-semibold">Note</th><th className="px-4 py-3 text-right font-semibold">Débit</th><th className="px-4 py-3 text-right font-semibold">Crédit</th></tr></thead>
            <tbody className="divide-y divide-line">
              {e.lines.map((l) => (
                <tr key={l.id}><td className="px-4 py-3"><span className="font-mono text-xs text-slate">{l.accountCode}</span> <span className="text-night">{l.accountLabel}</span></td><td className="px-4 py-3 text-slate">{l.memo ?? ""}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{l.debit ? fmtMoney(l.debit, cur) : ""}</td><td className="px-4 py-3 text-right tabular-nums">{l.credit ? fmtMoney(l.credit, cur) : ""}</td></tr>
              ))}
            </tbody>
            <tfoot className="border-t border-line bg-page/50 font-medium"><tr><td colSpan={2} className="px-4 py-3 text-right text-slate">Totaux</td><td className="px-4 py-3 text-right tabular-nums">{fmtMoney(e.totalDebit, cur)}</td><td className="px-4 py-3 text-right tabular-nums">{fmtMoney(e.totalCredit, cur)}</td></tr></tfoot>
          </table>
        </div>
      ) : (
        <Card title="Modifier le brouillon">
          <ActionForm action={updateEntry.bind(null, id)} submit="Enregistrer les modifications">
            <EntryEditor accounts={accounts} journals={jrn} currency={cur} entry={e} defaultDate={today()} />
          </ActionForm>
        </Card>
      )}

      {!posted && (
        <Card title="Validation" description="Valider rend l'écriture définitive : elle reçoit son numéro et ne pourra plus être modifiée ni supprimée.">
          {e.totalDebit !== e.totalCredit && <p className="mb-3 text-sm text-danger">L’écriture n’est pas équilibrée : elle ne peut pas être validée.</p>}
          {canPost && isAuthor && needsOther && <p className="mb-3 text-sm text-[#7a5a00]">Vous avez saisi cette écriture : une autre personne doit la valider (règle de double validation, applicable à tous).</p>}
          <div className="flex flex-wrap items-start gap-3">
            {canPost ? <ActionButton action={postEntry.bind(null, id)} label="Valider l'écriture" confirm="Valider cette écriture ? Elle deviendra définitive." /> : <p className="text-sm text-slate">La validation demande la permission « Valider — finance ».</p>}
            {can(perms, "finance:write") && <ActionButton action={deleteEntry.bind(null, id)} label="Supprimer le brouillon" variant="danger" confirm="Supprimer ce brouillon ?" />}
          </div>
        </Card>
      )}

      {posted && can(perms, "finance:write") && !e.reversedByEntryId && e.origin !== "reversal" && (
        <Card title="Contre-passer" description="Pour corriger une écriture validée, on en crée une inverse (brouillon), à faire valider par une autre personne. L'historique reste lisible.">
          <ActionForm action={reverseEntry.bind(null, id)} submit="Créer la contre-passation" className="space-y-4">
            <label className="block text-sm font-medium text-night">Motif (obligatoire)<input name="reason" required minLength={5} maxLength={500} className={inputCls} placeholder="Ex. erreur de compte, montant erroné…" /></label>
            <label className="block text-sm font-medium text-night">Date comptable de la contre-passation <span className="font-normal text-slate">(aujourd’hui par défaut)</span><input name="accountingDate" type="date" className={inputCls} /></label>
          </ActionForm>
        </Card>
      )}
    </div>
  );
}
