import Link from "next/link";
import { FileText } from "lucide-react";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { allAccounts, journals } from "@/lib/finance";
import { dateFr, fmtMoney, today } from "@/lib/money";
import { Badge, Card, inputCls, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { OperationFields } from "@/components/finance/op-forms";
import { dateFr as dateTimeFr, KIND_LABEL, OP_STATUS_LABEL, OP_STATUS_TONE, SETTLEMENT_LABEL, type FinAccount, type FinAttachment, type FinCategory, type FinJournal, type FinOperation, type FinThirdParty, type FinThirdPartyPage } from "@/lib/types";
import { cancelEntry, deleteAttachment, deleteOperation, generateEntry, recordPayment, updateOperation, uploadAttachment } from "../../actions";

const NOTICES: Record<string, string> = {
  created: "Opération enregistrée.", updated: "Opération modifiée.", entry: "Écriture générée en brouillon : une autre personne doit la valider (menu Écritures).", "entry-cancelled": "Écriture en brouillon supprimée.",
  payment: "Règlement enregistré en brouillon : une autre personne doit valider l'écriture.", attached: "Pièce jointe ajoutée.", detached: "Pièce supprimée.",
};

export default async function OperationPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const here = `/finance/operations/${id}`;
  const [{ me }, o, atts] = await Promise.all([currentStaff(), adminGet<FinOperation>(`/admin/finance/operations/${id}`, here), adminGet<{ items: FinAttachment[] }>(`/admin/finance/operations/${id}/attachments`, here)]);
  const perms = me?.permissions ?? [];
  const write = can(perms, "finance:write");
  const frozen = o.entryId !== null;
  const [accounts, jrn, cats, tps] = write ? await Promise.all([
    allAccounts(here, true), journals(here, true),
    adminGet<{ items: FinCategory[] }>("/admin/finance/operation-categories", here, { query: { kind: o.kind, active: "true" } }),
    adminGet<FinThirdPartyPage>("/admin/finance/third-parties", here, { query: { kind: o.kind === "expense" ? "supplier" : "customer", pageSize: 100 } }),
  ]) : [[] as FinAccount[], [] as FinJournal[], { items: [] as FinCategory[] }, { items: [] as FinThirdParty[], page: { page: 1, pageSize: 100, total: 0, totalPages: 0 } }];
  const cur = o.currency;
  const canPay = write && o.settlementMode === "on_account" && o.status === "recorded" && o.remaining > 0;

  return (
    <div className="max-w-4xl space-y-5">
      <PageHeader title={o.label} subtitle={`${KIND_LABEL[o.kind]} · ${dateFr(o.operationDate)}${o.reference ? ` · ${o.reference}` : ""}`} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Dépenses et revenus", href: "/finance/operations" }, { label: o.label }]}
        action={<Badge tone={OP_STATUS_TONE[o.status]}>{OP_STATUS_LABEL[o.status]}</Badge>} />
      {ok && NOTICES[ok] && <Notice kind="ok">{NOTICES[ok]}</Notice>}

      <Card title="Détail">
        <dl className="grid gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
          <div className="flex justify-between"><dt className="text-slate">Catégorie</dt><dd className="font-medium text-night">{o.categoryLabel}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">{o.kind === "expense" ? "Fournisseur" : "Client"}</dt><dd className="font-medium text-night">{o.thirdPartyName ?? "—"}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Hors taxes</dt><dd className="font-medium tabular-nums text-night">{fmtMoney(o.amountNet, cur)}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Taxe</dt><dd className="font-medium tabular-nums text-night">{fmtMoney(o.taxAmount, cur)}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Toutes taxes</dt><dd className="font-semibold tabular-nums text-night">{fmtMoney(o.amountGross, cur)}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Contrepartie</dt><dd className="font-medium text-night">{o.counterpartyAccountCode}</dd></div>
          <div className="flex justify-between"><dt className="text-slate">Règlement</dt><dd className="font-medium text-night">{SETTLEMENT_LABEL[o.settlementMode]}</dd></div>
          {o.dueDate && <div className="flex justify-between"><dt className="text-slate">Échéance</dt><dd className="font-medium text-night">{dateFr(o.dueDate)}</dd></div>}
          {o.nature && <div className="flex justify-between"><dt className="text-slate">Nature</dt><dd className="font-medium text-night">{o.nature === "fixed" ? "Fixe" : "Variable"}</dd></div>}
          <div className="flex justify-between"><dt className="text-slate">Saisie le</dt><dd className="font-medium text-night">{dateTimeFr(o.createdAt)}</dd></div>
        </dl>
        {o.duplicateConfirmed && <p className="mt-3 text-xs text-slate">Cette opération a été confirmée « pas un doublon » à la saisie.</p>}
        {o.recurringTemplateId && <p className="mt-3 text-xs text-slate">Générée par un modèle récurrent.</p>}
        {o.entryId && <p className="mt-3 text-sm text-slate">Écriture : <Link href={`/finance/entries/${o.entryId}`} className="text-link hover:underline">{o.entryNumber ?? "brouillon"}</Link></p>}
      </Card>

      {write && !frozen && (
        <Card title="Modifier l'opération">
          <ActionForm action={updateOperation.bind(null, id)} submit="Enregistrer les modifications">
            <OperationFields kind={o.kind} currency={cur} categories={cats.items} accounts={accounts} thirdParties={tps.items.filter((t) => t.active || t.id === o.thirdPartyId)} value={o} />
          </ActionForm>
        </Card>
      )}

      <Card title="Comptabilisation" description="L'écriture est créée en brouillon ; elle compte dans les états seulement une fois validée par une autre personne.">
        {!o.entryId && write && (
          <ActionForm action={generateEntry.bind(null, id)} submit="Générer l'écriture (brouillon)" className="flex flex-wrap items-end gap-3">
            <label className="block text-sm font-medium text-night">Journal<select name="journalId" required defaultValue="" className={`${inputCls} mt-1 w-64`}><option value="" disabled>— Choisir —</option>{jrn.map((j) => <option key={j.id} value={j.id}>{j.code} — {j.label}</option>)}</select></label>
          </ActionForm>
        )}
        {o.entryId && o.status === "entry_draft" && write && <ActionButton action={cancelEntry.bind(null, id)} label="Annuler l'écriture en brouillon" variant="ghost" confirm="Supprimer l'écriture en brouillon ? L'opération redeviendra modifiable." />}
        {o.status === "recorded" && <p className="text-sm text-slate">Écriture validée : elle ne se modifie plus. Pour corriger, contre-passez-la depuis l’écriture.</p>}
        {o.status === "reversed" && <p className="text-sm text-slate">L’écriture a été contre-passée : cette opération est annulée comptablement.</p>}
      </Card>

      {o.settlementMode === "on_account" && o.entryId && (
        <Card title="Règlements" description={`Réglé : ${fmtMoney(o.settledAmount, cur)} sur ${fmtMoney(o.amountGross, cur)} · reste ${fmtMoney(o.remaining, cur)}`}>
          {o.payments.length > 0 && (
            <ul className="mb-4 divide-y divide-line text-sm">
              {o.payments.map((p) => (
                <li key={p.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                  <span>{dateFr(p.paymentDate)} · {fmtMoney(p.amount, cur)} · compte {p.accountCode}</span>
                  <Link href={`/finance/entries/${p.entryId}`} className="text-link hover:underline">{p.entryNumber ?? "écriture en brouillon"}</Link>
                </li>
              ))}
            </ul>
          )}
          {canPay ? (
            <ActionForm action={recordPayment.bind(null, id, cur)} submit="Enregistrer le règlement (brouillon)" className="grid gap-3 sm:grid-cols-2">
              <label className="block text-sm font-medium text-night">Date<input name="paymentDate" type="date" required defaultValue={today()} className={inputCls} /></label>
              <label className="block text-sm font-medium text-night">Montant ({cur})<input name="amount" required inputMode="decimal" className={`${inputCls} text-right tabular-nums`} /></label>
              <label className="block text-sm font-medium text-night">{o.kind === "expense" ? "Compte payeur (banque, caisse)" : "Compte encaisseur (banque, caisse)"}<select name="accountId" required defaultValue="" className={inputCls}><option value="" disabled>— Choisir —</option>{accounts.map((a) => <option key={a.id} value={a.id}>{a.code} — {a.label}</option>)}</select></label>
              <label className="block text-sm font-medium text-night">Journal<select name="journalId" required defaultValue="" className={inputCls}><option value="" disabled>— Choisir —</option>{jrn.map((j) => <option key={j.id} value={j.id}>{j.code} — {j.label}</option>)}</select></label>
            </ActionForm>
          ) : (
            <p className="text-sm text-slate">{o.remaining === 0 ? "Entièrement réglée." : o.status !== "recorded" ? "Un règlement est possible une fois l'écriture de l'opération validée." : ""}</p>
          )}
        </Card>
      )}

      <Card title={`Pièces justificatives (${atts.items.length})`} description="PDF, PNG ou JPEG, 5 Mo au maximum. Le format est vérifié sur le contenu du fichier.">
        {atts.items.length > 0 && (
          <ul className="mb-4 divide-y divide-line text-sm">
            {atts.items.map((a) => (
              <li key={a.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                <a href={`/finance/attachment/${a.id}`} className="inline-flex items-center gap-2 text-link hover:underline"><FileText className="h-4 w-4" aria-hidden />{a.filename}</a>
                <span className="flex items-center gap-3 text-xs text-slate">{Math.max(1, Math.round(a.size / 1024))} Ko · {dateTimeFr(a.createdAt)}
                  {write && (o.status === "draft" || o.status === "entry_draft") && <ActionButton action={deleteAttachment.bind(null, id, a.id)} label="Retirer" variant="ghost" confirm="Retirer cette pièce ?" />}
                </span>
              </li>
            ))}
          </ul>
        )}
        {write && (
          <ActionForm action={uploadAttachment.bind(null, id)} submit="Joindre" className="flex flex-wrap items-end gap-3">
            <input type="file" name="file" accept="application/pdf,image/png,image/jpeg" required aria-label="Fichier" className="text-sm" />
          </ActionForm>
        )}
        {!write && atts.items.length === 0 && <p className="text-sm text-slate">Aucune pièce.</p>}
      </Card>

      {write && !frozen && (
        <Card title="Supprimer"><p className="mb-3 text-sm text-slate">Possible tant qu’aucune écriture n’existe.</p><ActionButton action={deleteOperation.bind(null, id)} label="Supprimer l'opération" variant="danger" confirm="Supprimer cette opération ?" /></Card>
      )}
    </div>
  );
}
