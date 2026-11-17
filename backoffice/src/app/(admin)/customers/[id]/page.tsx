import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Badge, Card, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton } from "@/components/ui/client";
import { COMPANY_LABEL, COMPANY_TONE, customerName, dateFr, type Customer } from "@/lib/types";
import { setCustomerStatus } from "../actions";

const OK: Record<string, string> = { suspended: "Compte suspendu : ses sessions ont été fermées.", reactivated: "Compte réactivé." };

export default async function CustomerPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, c] = await Promise.all([currentStaff(), adminGet<Customer>(`/admin/customers/${id}`, `/customers/${id}`)]);
  const perms = me?.permissions ?? [];
  const canWrite = can(perms, "customers:write");

  return (
    <div className="max-w-3xl space-y-5">
      <PageHeader breadcrumb={[{ label: "Clients", href: "/customers" }, { label: "Détail" }]} title={customerName(c)} subtitle={c.email} action={<Badge tone={c.status === "active" ? "ok" : "off"}>{c.status === "active" ? "Actif" : "Suspendu"}</Badge>} />
      {ok && OK[ok] && <Notice kind="ok">{OK[ok]}</Notice>}

      <Card>
        <dl className="grid gap-3 text-sm sm:grid-cols-3">
          <div><dt className="text-slate">Type</dt><dd className="font-medium text-night">{c.kind === "company" ? "Entreprise" : "Particulier"}</dd></div>
          <div><dt className="text-slate">Inscrit le</dt><dd className="font-medium text-night">{dateFr(c.createdAt)}</dd></div>
          <div><dt className="text-slate">Commandes</dt><dd className="font-medium text-night">{c.orderCount}</dd></div>
        </dl>
      </Card>

      {c.company && (
        <Card>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-night">{c.company.legalName}</h2>
            <Badge tone={COMPANY_TONE[c.company.status]}>{COMPANY_LABEL[c.company.status]}</Badge>
          </div>
          <dl className="grid gap-3 text-sm sm:grid-cols-3">
            <div><dt className="text-slate">N° fiscal</dt><dd className="font-medium text-night">{c.company.taxId}</dd></div>
            <div><dt className="text-slate">Pays</dt><dd className="font-medium text-night">{c.company.country}</dd></div>
            <div><dt className="text-slate">Décision</dt><dd className="font-medium text-night">{dateFr(c.company.decidedAt)}</dd></div>
          </dl>
          {c.company.rejectionReason && <p className="mt-3 text-sm text-slate">Motif du rejet : {c.company.rejectionReason}</p>}
          {can(perms, "companies:approve") && c.company.status === "pending" && <p className="mt-3 text-sm"><Link href={`/companies/${c.company.id}`} className="text-link hover:underline">Examiner la demande →</Link></p>}
        </Card>
      )}

      {canWrite && (
        <Card guide="customer-actions">
          <h2 className="mb-4 text-lg font-semibold text-night">Actions</h2>
          {c.status === "active" ? (
            <ActionButton action={setCustomerStatus.bind(null, c.id, "suspended")} label="Suspendre le compte" variant="danger" confirm={`Suspendre ${customerName(c)} ? Ses sessions seront fermées immédiatement.`} />
          ) : (
            <ActionButton action={setCustomerStatus.bind(null, c.id, "active")} label="Réactiver le compte" variant="ghost" />
          )}
        </Card>
      )}
    </div>
  );
}
