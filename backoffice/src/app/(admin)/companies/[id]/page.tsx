import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, Card, inputCls, LinkButton, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { COMPANY_LABEL, COMPANY_TONE, dateFr, type Company } from "@/lib/types";
import { approveCompany, rejectCompany } from "../actions";

const OK: Record<string, string> = { approved: "Entreprise approuvée : les conditions professionnelles sont ouvertes.", rejected: "Entreprise rejetée." };

export default async function CompanyPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, c] = await Promise.all([currentStaff(), adminGet<Company>(`/admin/companies/${id}`, `/companies/${id}`)]);
  const canDecide = can(me?.permissions ?? [], "companies:approve") && c.status === "pending";

  return (
    <div className="max-w-3xl space-y-5">
      <PageHeader breadcrumb={[{ label: "Entreprises", href: "/companies" }, { label: "Détail" }]} title={c.legalName} subtitle={`${c.taxId} · ${c.country}`} action={<Badge tone={COMPANY_TONE[c.status]}>{COMPANY_LABEL[c.status]}</Badge>} />
      {ok && OK[ok] && <Notice kind="ok">{OK[ok]}</Notice>}

      <Card>
        <dl className="grid gap-3 text-sm sm:grid-cols-3">
          <div><dt className="text-slate">N° fiscal</dt><dd className="font-medium text-night">{c.taxId}</dd></div>
          <div><dt className="text-slate">Vérifié automatiquement</dt><dd className="font-medium text-night">{c.taxIdVerified ? "Oui" : "Non"}</dd></div>
          <div><dt className="text-slate">Décision</dt><dd className="font-medium text-night">{dateFr(c.decidedAt)}</dd></div>
        </dl>
        {c.rejectionReason && <p className="mt-3 text-sm text-slate">Motif du rejet : {c.rejectionReason}</p>}
        {c.status === "pending" && <p className="mt-3 text-xs text-slate">En attente, le compte est traité comme un particulier : les conditions professionnelles s&apos;ouvrent après approbation.</p>}
      </Card>

      {canDecide && (
        <>
          <Card guide="company-approve">
            <h2 className="mb-4 text-lg font-semibold text-night">Approuver</h2>
            <p className="mb-3 text-sm text-slate">L&apos;adresse e-mail du titulaire doit être vérifiée. Vérifiez le numéro fiscal avant d&apos;approuver.</p>
            <ActionButton action={approveCompany.bind(null, c.id)} label="Approuver l'entreprise" confirm={`Approuver ${c.legalName} ?`} />
          </Card>
          <Card guide="company-reject">
            <h2 className="mb-4 text-lg font-semibold text-night">Rejeter</h2>
            <ActionForm action={rejectCompany.bind(null, c.id)} submit="Rejeter l'entreprise">
              <label className="block text-sm font-medium text-night">Motif (obligatoire)
                <textarea name="reason" required maxLength={500} rows={3} className={inputCls} />
              </label>
            </ActionForm>
          </Card>
        </>
      )}
      <LinkButton href="/companies" variant="ghost">← Retour à la liste</LinkButton>
    </div>
  );
}
