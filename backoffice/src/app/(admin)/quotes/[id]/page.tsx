import { redirect } from "next/navigation";
import { adminGet, adminGetOpt, can } from "@/lib/admin";
import { gateway } from "@/lib/api";
import { currentStaff } from "@/lib/staff";
import { kindBySlug } from "@/lib/content-kinds";
import type { Doc, DocPage } from "@/lib/content";
import type { Product, ProductPage } from "@/lib/types";
import type { QDraft, QLine } from "@/lib/quote-calc";
import { QUOTE_STATUS, QUOTE_TONE } from "@/lib/quotes";
import { Badge, Card, Notice, PageHeader } from "@/components/ui/ui";
import { ActionButton, ActionForm } from "@/components/ui/client";
import { ContentFields } from "@/components/content-fields";
import QuoteStudio from "@/components/quotes/quote-studio";
import { deleteQuote, updateQuote } from "../actions";

type Item = { reference: string; quantity: number; name?: string; productId?: string };
const str = (v: unknown) => (typeof v === "string" ? v : "");
/** Date du jour décalée de n jours (yyyy-mm-dd), UTC. */
function plusDays(n: number) { return new Date(Date.now() + n * 86400000).toISOString().slice(0, 10); }
const dayOf = (iso: unknown) => (typeof iso === "string" ? iso.slice(0, 10) : "");

export default async function QuoteDetailPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ ok?: string }> }) {
  const { id } = await params;
  const { ok } = await searchParams;
  const { me } = await currentStaff();
  if (!can(me?.permissions ?? [], "quotes:read")) redirect("/denied");
  const canWrite = can(me?.permissions ?? [], "quotes:write");
  const k = kindBySlug("quote-requests")!;
  const returnTo = `/quotes/${id}`;
  const [d, resp, commerce, siteRes] = await Promise.all([
    adminGet<Doc>(`/admin/quote-requests/${id}`, returnTo),
    adminGetOpt<Doc>(`/admin/quote-responses/${id}`, returnTo),
    adminGetOpt<{ vatRate?: number }>("/admin/commerce-settings", returnTo),
    gateway<Record<string, string>>("/site-settings", { method: "GET" }), // en-tête du devis : publique, aucun droit particulier requis
  ]);
  const site = siteRes.ok ? siteRes.data : {};
  const s = (str(d.status) in QUOTE_STATUS ? d.status : "new") as keyof typeof QUOTE_STATUS;
  const items = (Array.isArray(d.items) ? d.items : []) as Item[];

  // Première ouverture : on pré-chiffre avec les prix FIXES du catalogue (référence = SKU ou référence constructeur).
  let suggested = 0;
  let lines: QLine[];
  if (resp && Array.isArray(resp.lines)) {
    lines = (resp.lines as QLine[]).map((l) => ({ ...l }));
  } else {
    const found = await Promise.all(items.slice(0, 40).map(async (it) => {
      const r = await adminGetOpt<ProductPage>("/admin/products", returnTo, { query: { q: it.reference, pageSize: 10 } });
      const ref = it.reference.trim().toUpperCase();
      return r?.items.find((p: Product) => p.sku.toUpperCase() === ref || (p.manufacturerRef ?? "").toUpperCase() === ref) ?? null;
    }));
    lines = items.map((it, n) => {
      const p = found[n];
      const price = p && p.priceMode === "fixed" && p.priceAmount != null ? p.priceAmount : null;
      if (price !== null) suggested++;
      return { reference: it.reference, name: it.name || (p?.name.fr ?? ""), quantity: it.quantity, unitPrice: price, productId: it.productId ?? p?.id };
    });
  }
  const initial: QDraft = {
    version: str(resp?.version) || "1", incoterm: str(resp?.incoterm), correspondent: str(resp?.correspondent) || me?.name || "",
    other: Number(resp?.other ?? 0), acompte: Number(resp?.acompte ?? 0), annexes: Array.isArray(resp?.annexes) ? (resp.annexes as { url: string; caption?: string }[]) : [],
    number: str(resp?.number) || `DEV-${plusDays(0).replaceAll("-", "")}-${id.slice(-4).toUpperCase()}`,
    lines, discountPct: Number(resp?.discountPct ?? 0), shipping: Number(resp?.shipping ?? 0),
    vatRate: Number(resp?.vatRate ?? commerce?.vatRate ?? 0), validUntil: dayOf(resp?.validUntil) || plusDays(30),
    deliveryPlace: str(resp?.deliveryPlace) || str(d.deliveryPlace), deliveryDelay: str(resp?.deliveryDelay) || str(d.deliveryDelay), paymentTerms: str(resp?.paymentTerms) || str(d.paymentTerms),
    intro: str(resp?.intro), internalNote: str(resp?.internalNote), followUpAt: dayOf(resp?.followUpAt),
  };
  const sentAt = resp?.status === "sent" && typeof resp.sentAt === "string" ? resp.sentAt : null;

  const letter = str(site.letterhead).split("\n").map((x) => x.trim()).filter(Boolean);
  const company = {
    legalName: str(site.publisher) || str(site.name), lines: letter.length ? letter : str(site.address).split("\n").map((x) => x.trim()).filter(Boolean),
    rccm: str(site.rccm), taxNumber: str(site.taxNumber), capital: str(site.capital), logoUrl: str(site.logoUrl), stampUrl: str(site.stampUrl), phone: str(site.phone), email: str(site.email),
  };
  const email = str(d.email);
  const others = email ? await adminGet<DocPage>("/admin/quote-requests", returnTo, { query: { q: email, pageSize: 20 } }) : null;
  const history = (others?.items ?? []).filter((o) => o.id !== id && str(o.email).toLowerCase() === email.toLowerCase())
    .map((o) => ({ id: o.id, subject: str(o.subject), status: QUOTE_STATUS[(str(o.status) in QUOTE_STATUS ? o.status : "new") as keyof typeof QUOTE_STATUS], createdAt: o.createdAt }));
  const editable = k.fields.filter((f) => f.name === "status" || f.name === "note");

  return (
    <div className="max-w-7xl space-y-5">
      <PageHeader breadcrumb={[{ label: "Devis", href: "/quotes" }, { label: "Studio" }]} title={str(d.subject) || `Demande de ${str(d.name)}`}
        subtitle={`Reçue le ${new Date(d.createdAt).toLocaleString("fr-FR")}${str(d.reference) ? ` · Réf. client ${str(d.reference)}` : ""} · ${items.length} article${items.length > 1 ? "s" : ""} demandé${items.length > 1 ? "s" : ""}`}
        action={<Badge tone={QUOTE_TONE[s]}>{QUOTE_STATUS[s]}</Badge>} />
      {ok === "updated" && <Notice kind="ok">Modifications enregistrées.</Notice>}
      <QuoteStudio requestId={id} company={company} canWrite={canWrite} initial={initial} sentAt={sentAt} history={history} suggested={suggested}
        customer={{ name: str(d.name), email, company: str(d.company), phone: str(d.phone), attention: str(d.attention), jobTitle: str(d.jobTitle), address: str(d.address), subject: str(d.subject), reference: str(d.reference), message: str(d.message), createdAt: d.createdAt }} />
      <div className="grid gap-5 md:grid-cols-2">
        <Card title="Suivi de la demande" guide="quote-process" description="Le statut est celui que le client voit. Il passe tout seul à « En cours » au premier brouillon et à « Devis envoyé » à l'envoi.">
          {canWrite ? <ActionForm action={updateQuote.bind(null, id)} submit="Enregistrer"><ContentFields fields={editable} value={d} refs={{}} /></ActionForm> : <p className="text-sm text-slate">Lecture seule.</p>}
        </Card>
        {canWrite && (
          <Card guide="danger-zone">
            <h2 className="mb-3 text-lg font-semibold text-night">Supprimer</h2>
            <p className="mb-3 text-sm text-slate">Supprime la demande du client. Le devis chiffré associé reste enregistré mais n&apos;est plus accessible.</p>
            <ActionButton action={deleteQuote.bind(null, id)} label="Supprimer" variant="danger" confirm="Supprimer définitivement cette demande ?" />
          </Card>
        )}
      </div>
    </div>
  );
}
