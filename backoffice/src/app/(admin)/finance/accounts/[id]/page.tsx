import Link from "next/link";
import { adminGet, can } from "@/lib/admin";
import { currentStaff } from "@/lib/staff";
import { Card, Notice, PageHeader } from "@/components/ui/ui";
import { ActionForm } from "@/components/ui/client";
import { AccountFields } from "@/components/finance/forms";
import { ACCOUNT_KIND_LABEL, type FinAccount } from "@/lib/types";
import { updateAccount } from "../../actions";

export default async function AccountPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const [{ me }, a] = await Promise.all([currentStaff(), adminGet<FinAccount>(`/admin/finance/accounts/${id}`, `/finance/accounts/${id}`)]);
  const admin = can(me?.permissions ?? [], "finance:admin");
  return (
    <div className="max-w-2xl space-y-5">
      <PageHeader title={`${a.code} — ${a.label}`} breadcrumb={[{ label: "Finance", href: "/finance" }, { label: "Plan comptable", href: "/finance/accounts" }, { label: a.code }]} />
      {ok === "created" && <Notice kind="ok">Compte créé.</Notice>}
      {ok === "updated" && <Notice kind="ok">Modifications enregistrées.</Notice>}
      <Card>
        {admin ? <ActionForm action={updateAccount.bind(null, id)} submit="Enregistrer"><AccountFields value={a} /></ActionForm> : <p className="text-sm text-slate">{ACCOUNT_KIND_LABEL[a.kind]} · {a.active ? "actif" : "inactif"}</p>}
      </Card>
      <p className="text-sm"><Link href={`/finance/reports/ledger?accountId=${a.id}`} className="text-link hover:underline">Voir le grand livre de ce compte →</Link></p>
    </div>
  );
}
