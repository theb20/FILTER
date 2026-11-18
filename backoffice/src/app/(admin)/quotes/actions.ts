"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { readiness, type QDraft } from "@/lib/quote-calc";
import type { FormState } from "@/components/ui/client";

const STATUSES = ["new", "in_progress", "quoted", "closed"];

/** Change le statut et la note interne de la demande ; la demande du client elle-même n'est jamais modifiée. */
export async function updateQuote(id: string, _: FormState, form: FormData): Promise<FormState> {
  const status = String(form.get("status") ?? "");
  if (!STATUSES.includes(status)) return { error: "Statut inconnu." };
  const r = await adminAct("PUT", `/admin/quote-requests/${id}`, { body: { status, note: String(form.get("note") ?? "").trim() } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/quotes");
  redirect(`/quotes/${id}?ok=updated`);
}

export async function deleteQuote(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/quote-requests/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/quotes");
  redirect("/quotes?ok=deleted");
}

const iso = (day: string) => (day ? `${day}T23:59:59Z` : undefined);
const clean = (s: string) => (s.trim() ? s.trim() : undefined);

function body(requestId: string, d: QDraft, status: "draft" | "sent", sentAt?: string) {
  return {
    requestId, status, number: clean(d.number), version: clean(d.version), incoterm: clean(d.incoterm), correspondent: clean(d.correspondent),
    discountPct: d.discountPct || undefined, shipping: d.shipping || undefined, other: d.other || undefined, acompte: d.acompte || undefined, vatRate: d.vatRate || undefined,
    annexes: d.annexes.length ? d.annexes.map((a) => ({ url: a.url, caption: clean(a.caption ?? "") })) : undefined,
    validUntil: iso(d.validUntil), deliveryPlace: clean(d.deliveryPlace), deliveryDelay: clean(d.deliveryDelay), paymentTerms: clean(d.paymentTerms),
    intro: clean(d.intro), internalNote: clean(d.internalNote), followUpAt: iso(d.followUpAt), sentAt,
    lines: d.lines.map((l) => ({
      reference: l.reference.trim(), name: clean(l.name ?? ""), quantity: l.quantity, unitPrice: l.unitPrice ?? undefined, discountPct: l.discountPct || undefined,
      leadTime: clean(l.leadTime ?? ""), image: clean(l.image ?? ""), note: clean(l.note ?? ""), internal: clean(l.internal ?? ""), optional: l.optional || undefined, custom: l.custom || undefined, productId: l.productId,
    })),
  };
}

export type StudioResult = { ok?: string; error?: string; sentAt?: string };

/** Enregistre (ou crée) la réponse chiffrée. Elle est unique par demande : on la retrouve par l'identifiant de la demande. */
async function upsert(requestId: string, payload: ReturnType<typeof body>) {
  const cur = await adminAct<{ id: string }>("GET", `/admin/quote-responses/${requestId}`);
  if (cur.ok) return adminAct("PUT", `/admin/quote-responses/${cur.data.id}`, { body: payload });
  if (cur.status === 404) return adminAct("POST", "/admin/quote-responses", { body: payload });
  return cur;
}

/** Brouillon : accepte un devis incomplet ; la demande passe « En cours » si elle était « Nouvelle ». */
export async function saveQuoteDraft(requestId: string, draft: QDraft): Promise<StudioResult> {
  const r = await upsert(requestId, body(requestId, draft, "draft"));
  if (!r.ok) return { error: errorMessage(r) };
  const req = await adminAct<{ status?: string }>("GET", `/admin/quote-requests/${requestId}`);
  if (req.ok && req.data.status === "new") await adminAct("PUT", `/admin/quote-requests/${requestId}`, { body: { status: "in_progress" } });
  revalidatePath("/quotes");
  return { ok: "Brouillon enregistré." };
}

/** Envoi : le devis est complet (revérifié ici ET par le service), il devient visible dans l'espace du client et la demande passe « Devis envoyé ». */
export async function sendQuote(requestId: string, draft: QDraft): Promise<StudioResult> {
  const missing = readiness(draft).blocking;
  if (missing.length) return { error: `Devis incomplet : ${missing.join(" ; ")}.` };
  const sentAt = new Date().toISOString();
  const r = await upsert(requestId, body(requestId, draft, "sent", sentAt));
  if (!r.ok) return { error: errorMessage(r) };
  const st = await adminAct("PUT", `/admin/quote-requests/${requestId}`, { body: { status: "quoted" } });
  if (!st.ok) return { error: `Devis enregistré, mais le statut de la demande n'a pas pu être mis à jour : ${errorMessage(st)}` };
  revalidatePath("/quotes");
  return { ok: "Devis envoyé au client.", sentAt };
}
