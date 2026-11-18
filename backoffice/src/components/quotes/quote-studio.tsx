"use client";
/* eslint-disable react/no-unescaped-entities -- textes français avec apostrophes */

import { useMemo, useState, useTransition } from "react";
import { ArrowDown, ArrowUp, Check, CircleAlert, Copy, ImagePlus, Info, Loader2, Mail, Plus, Printer, Save, Send, Trash2, Undo2 } from "lucide-react";
import { btnCls, btnGhostCls } from "@/components/ui/ui";
import { ImagePicker, uploadImage } from "@/components/ui/image-upload";
import QuoteSheet, { QuotePrintCopy, type SheetData } from "./quote-sheet";
import { computeTotals, lineTotal, money, readiness, type QDraft, type QLine } from "@/lib/quote-calc";
import { saveQuoteDraft, sendQuote, type StudioResult } from "@/app/(admin)/quotes/actions";

export type Customer = { name: string; email: string; company?: string; phone?: string; attention?: string; jobTitle?: string; address?: string; subject?: string; reference?: string; message?: string; createdAt: string };
export type HistoryItem = { id: string; subject: string; status: string; createdAt: string };
type Props = { requestId: string; canWrite: boolean; customer: Customer; company: SheetData["company"]; initial: QDraft; sentAt: string | null; history: HistoryItem[]; suggested: number };

const cell = "w-full rounded-md border border-line bg-white px-2 py-1.5 text-sm text-night focus:border-link focus:outline-none focus:ring-2 focus:ring-link/20 disabled:bg-page disabled:text-slate";
const lab = "block text-xs font-semibold uppercase tracking-wide text-slate";
const TABS = [["lines", "Chiffrage"], ["terms", "Conditions & message"], ["media", "Images & annexes"], ["preview", "Aperçu du devis"], ["file", "Dossier client"]] as const;
type Tab = (typeof TABS)[number][0];
const num = (v: string) => (v.trim() === "" ? null : Number(v.replace(/\s/g, "").replace(",", ".")));
const day = (iso?: string | null) => (iso ? new Date(iso).toLocaleDateString("fr-FR", { day: "2-digit", month: "long", year: "numeric" }) : "—");

export default function QuoteStudio({ requestId, canWrite, customer, company, initial, sentAt: initialSent, history, suggested }: Props) {
  const [d, setD] = useState<QDraft>(initial);
  const [tab, setTab] = useState<Tab>("lines");
  const [sentAt, setSentAt] = useState<string | null>(initialSent);
  const [dirty, setDirty] = useState(false);
  const [open, setOpen] = useState<number | null>(null);
  const [info, setInfo] = useState<{ kind: "ok" | "error"; text: string } | null>(null);
  const [pending, start] = useTransition();
  const ro = !canWrite;

  const t = useMemo(() => computeTotals(d.lines, d.discountPct, d.shipping, d.vatRate, d.other, d.acompte), [d]);
  const ready = useMemo(() => readiness(d), [d]);
  const sheet: SheetData = useMemo(() => ({
    number: d.number, version: d.version, date: sentAt ?? new Date().toISOString(), validUntil: d.validUntil ? `${d.validUntil}T23:59:59Z` : undefined, subject: customer.subject, reference: customer.reference,
    company, correspondent: d.correspondent, customer: { company: customer.company, name: customer.name, address: customer.address, attention: customer.attention, jobTitle: customer.jobTitle, email: customer.email, phone: customer.phone },
    deliveryDelay: d.deliveryDelay, incoterm: d.incoterm, deliveryPlace: d.deliveryPlace, paymentTerms: d.paymentTerms, intro: d.intro,
    lines: d.lines.map((l) => ({ reference: l.reference, name: l.name, quantity: l.quantity, unitPrice: l.unitPrice, discountPct: l.discountPct, lineTotal: lineTotal(l), leadTime: l.leadTime, note: l.note, optional: l.optional, image: l.image })),
    subtotal: t.subtotal, discountPct: d.discountPct, discountAmount: t.discountAmount, shipping: d.shipping, other: d.other, vatRate: d.vatRate, vatAmount: t.vatAmount, total: t.total, acompte: d.acompte, remaining: t.remaining, annexes: d.annexes,
  }), [d, t, customer, company, sentAt]);
  const set = <K extends keyof QDraft>(k: K, v: QDraft[K]) => { setD((p) => ({ ...p, [k]: v })); setDirty(true); setInfo(null); };
  const setLine = (i: number, patch: Partial<QLine>) => set("lines", d.lines.map((l, n) => (n === i ? { ...l, ...patch } : l)));
  const addLine = () => { set("lines", [...d.lines, { reference: "", name: "", quantity: 1, custom: true }]); setOpen(null); };
  const delLine = (i: number) => set("lines", d.lines.filter((_, n) => n !== i));
  const [annexBusy, setAnnexBusy] = useState(false);
  const addAnnexes = async (files: FileList) => {
    setAnnexBusy(true);
    try {
      const added: { url: string; caption?: string }[] = [];
      for (const f of Array.from(files).slice(0, 30 - d.annexes.length)) added.push({ url: await uploadImage(f), caption: f.name.replace(/\.\w+$/, "") });
      set("annexes", [...d.annexes, ...added]);
    } catch (e) { setInfo({ kind: "error", text: e instanceof Error ? e.message : "Téléversement impossible." }); } finally { setAnnexBusy(false); }
  };
  const moveAnnex = (i: number, dir: -1 | 1) => { const a = [...d.annexes]; const j = i + dir; if (j < 0 || j >= a.length) return; [a[i], a[j]] = [a[j], a[i]]; set("annexes", a); };
  const applyAll = (pct: number) => set("lines", d.lines.map((l) => ({ ...l, discountPct: pct || null })));

  const run = (fn: () => Promise<StudioResult>, after?: (r: StudioResult) => void) => start(async () => {
    try {
      const r = await fn();
      if (r.error) setInfo({ kind: "error", text: r.error });
      else { setInfo({ kind: "ok", text: r.ok ?? "Enregistré." }); setDirty(false); after?.(r); }
    } catch { setInfo({ kind: "error", text: "Service indisponible ou session expirée. Réessayez." }); }
  });
  const invalidRef = d.lines.some((l) => !l.reference.trim());
  const save = () => invalidRef ? setInfo({ kind: "error", text: "Chaque ligne doit avoir une référence." }) : run(() => saveQuoteDraft(requestId, d));
  const send = () => {
    if (invalidRef) return setInfo({ kind: "error", text: "Chaque ligne doit avoir une référence." });
    if (!window.confirm(`${sentAt ? "Renvoyer" : "Envoyer"} le devis ${d.number} de ${money(t.total)} à ${customer.name} ? Il sera visible dans son espace client.`)) return;
    run(() => sendQuote(requestId, d), (r) => setSentAt(r.sentAt ?? new Date().toISOString()));
  };
  const unsend = () => {
    if (!window.confirm("Retirer le devis de l'espace du client et le repasser en brouillon ?")) return;
    run(() => saveQuoteDraft(requestId, d), () => setSentAt(null));
  };

  const mailto = () => {
    const subject = `Votre devis ${d.number}${customer.subject ? ` — ${customer.subject}` : ""}`;
    const lines = d.lines.filter((l) => !l.optional).map((l) => `- ${l.quantity} × ${l.reference}${l.name ? ` (${l.name})` : ""} : ${lineTotal(l) === null ? "sur demande" : money(lineTotal(l) as number)}`).join("\n");
    const text = `Bonjour ${customer.name},\n\n${d.intro}\n\n${lines}\n\nTotal : ${money(t.total)} (TVA ${d.vatRate} % comprise)\nValable jusqu'au ${day(`${d.validUntil}T00:00:00Z`)}.\n\nVous retrouvez le détail dans votre espace client, rubrique « Suivi de mes demandes de devis ».\n\nCordialement`;
    return `mailto:${customer.email}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(text)}`;
  };

  const statusBadge = sentAt
    ? <span className="rounded-full bg-ok/10 px-3 py-1 text-xs font-semibold text-[#007a52]">Envoyé le {day(sentAt)}</span>
    : <span className="rounded-full bg-action/20 px-3 py-1 text-xs font-semibold text-[#7a5a00]">Brouillon{dirty ? " · modifications non enregistrées" : ""}</span>;

  return (
    <div data-guide="quote-studio" className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_320px]">
      <div className="min-w-0 space-y-4">
        <div className="flex flex-wrap items-center gap-3">
          <nav className="flex flex-wrap gap-1 rounded-xl border border-line bg-white p-1" aria-label="Sections du studio">
            {TABS.map(([k, label]) => <button key={k} type="button" onClick={() => setTab(k)} aria-current={tab === k} className={`rounded-lg px-3.5 py-1.5 text-sm font-medium transition ${tab === k ? "bg-night text-white" : "text-slate hover:bg-page"}`}>{label}</button>)}
          </nav>
          <div className="ml-auto">{statusBadge}</div>
        </div>
        {info && <p role="status" className={`rounded-lg px-3 py-2 text-sm ${info.kind === "ok" ? "bg-ok/10 text-[#007a52]" : "bg-danger/10 text-danger"}`}>{info.text}</p>}

        {tab === "lines" && (
          <section className="rounded-xl border border-line bg-white p-4 shadow-card" aria-label="Chiffrage des lignes">
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <h2 className="text-lg font-semibold text-night">Lignes du devis</h2>
              {suggested > 0 && <span className="rounded-full bg-link/10 px-2.5 py-0.5 text-xs font-medium text-link">{suggested} prix repris du catalogue</span>}
              {!ro && <div className="ml-auto flex items-center gap-2">
                <label className="flex items-center gap-1.5 text-xs text-slate">Remise sur toutes les lignes
                  <input type="number" min={0} max={100} placeholder="%" className={`${cell} w-16`} onChange={(e) => { const v = Number(e.target.value); if (v >= 0 && v <= 100) applyAll(v); }} /></label>
                <button type="button" onClick={addLine} className={btnGhostCls}><Plus size={15} aria-hidden /> Ajouter une ligne</button>
              </div>}
            </div>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[900px] text-sm">
                <thead><tr className="border-b border-line text-left text-xs uppercase tracking-wide text-slate">
                  <th className="w-14 py-2 pr-2">Image</th><th className="py-2 pr-2">Référence</th><th className="py-2 pr-2">Désignation</th><th className="w-16 py-2 pr-2">Qté</th><th className="w-32 py-2 pr-2">Prix unit. HT</th><th className="w-20 py-2 pr-2">Remise %</th><th className="w-36 py-2 pr-2 text-right">Total ligne</th><th className="w-24 py-2" />
                </tr></thead>
                <tbody>
                  {d.lines.length === 0 && <tr><td colSpan={8} className="py-6 text-center text-slate">Aucune ligne. Ajoutez-en une.</td></tr>}
                  {d.lines.map((l, i) => {
                    const lt = lineTotal(l);
                    return (
                      <FragmentRow key={i}>
                        <tr className={`border-b border-line/60 align-top ${l.optional ? "bg-page/60" : ""}`}>
                          <td className="py-2 pr-2"><ImagePicker compact value={l.image ?? ""} disabled={ro} onChange={(u) => setLine(i, { image: u })} /></td>
                          <td className="py-2 pr-2"><input className={cell} value={l.reference} disabled={ro || !l.custom} onChange={(e) => setLine(i, { reference: e.target.value })} aria-label={`Référence ligne ${i + 1}`} /></td>
                          <td className="py-2 pr-2"><input className={cell} value={l.name ?? ""} disabled={ro} onChange={(e) => setLine(i, { name: e.target.value })} aria-label={`Désignation ligne ${i + 1}`} placeholder="Désignation" /></td>
                          <td className="py-2 pr-2"><input type="number" min={1} className={cell} value={l.quantity} disabled={ro} onChange={(e) => setLine(i, { quantity: Math.max(1, Math.round(Number(e.target.value) || 1)) })} aria-label={`Quantité ligne ${i + 1}`} /></td>
                          <td className="py-2 pr-2"><input inputMode="numeric" className={`${cell} ${l.unitPrice == null && !l.optional ? "border-action-dark" : ""}`} value={l.unitPrice ?? ""} disabled={ro} placeholder="à chiffrer" onChange={(e) => { const v = num(e.target.value); setLine(i, { unitPrice: v === null || !Number.isFinite(v) || v < 0 ? null : Math.round(v) }); }} aria-label={`Prix unitaire ligne ${i + 1}`} /></td>
                          <td className="py-2 pr-2"><input type="number" min={0} max={100} className={cell} value={l.discountPct ?? ""} disabled={ro} onChange={(e) => { const v = num(e.target.value); setLine(i, { discountPct: v === null ? null : Math.min(100, Math.max(0, v)) }); }} aria-label={`Remise ligne ${i + 1}`} /></td>
                          <td className="py-2 pr-2 text-right font-medium tabular-nums text-night">{l.optional ? <span className="text-xs font-normal text-slate">optionnelle</span> : lt === null ? <span className="text-slate">—</span> : money(lt)}</td>
                          <td className="py-2 text-right whitespace-nowrap">
                            <button type="button" onClick={() => setOpen(open === i ? null : i)} className="rounded-md px-2 py-1 text-xs text-link hover:bg-link/10" aria-expanded={open === i}>{l.note || l.internal || l.leadTime || l.optional ? "Notes ●" : "Notes"}</button>
                            {!ro && <button type="button" onClick={() => delLine(i)} className="rounded-md p-1.5 text-slate hover:bg-danger/10 hover:text-danger" aria-label={`Supprimer la ligne ${i + 1}`}><Trash2 size={15} aria-hidden /></button>}
                          </td>
                        </tr>
                        {open === i && (
                          <tr className="border-b border-line/60 bg-page/50"><td colSpan={8} className="p-3">
                            <div className="grid gap-3 md:grid-cols-3">
                              <label className={lab}>Délai de cette ligne<input className={`${cell} mt-1 normal-case`} disabled={ro} value={l.leadTime ?? ""} onChange={(e) => setLine(i, { leadTime: e.target.value })} placeholder="ex. 5 jours ouvrés, sur stock" /></label>
                              <label className={lab}>Note pour le client<textarea rows={2} className={`${cell} mt-1 normal-case`} disabled={ro} value={l.note ?? ""} onChange={(e) => setLine(i, { note: e.target.value })} placeholder="Visible sur le devis (équivalence proposée, variante…)" /></label>
                              <label className={lab}>Note interne (jamais visible du client)<textarea rows={2} className={`${cell} mt-1 normal-case`} disabled={ro} value={l.internal ?? ""} onChange={(e) => setLine(i, { internal: e.target.value })} placeholder="Coût, fournisseur, marge…" /></label>
                            </div>
                            <label className="mt-3 flex items-center gap-2 text-sm text-night"><input type="checkbox" disabled={ro} checked={!!l.optional} onChange={(e) => setLine(i, { optional: e.target.checked })} /> Ligne optionnelle : proposée au client mais hors total</label>
                          </td></tr>
                        )}
                      </FragmentRow>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {tab === "terms" && (
          <section className="space-y-4 rounded-xl border border-line bg-white p-5 shadow-card" aria-label="Conditions et message">
            <div className="grid gap-4 sm:grid-cols-3">
              <label className={lab}>Numéro du devis<input className={`${cell} mt-1 normal-case`} disabled={ro} value={d.number} onChange={(e) => set("number", e.target.value)} /></label>
              <label className={lab}>Version / Avenant<input className={`${cell} mt-1 normal-case`} disabled={ro} value={d.version} onChange={(e) => set("version", e.target.value)} placeholder="1" /></label>
              <label className={lab}>Correspondant (nom du commercial)<input className={`${cell} mt-1 normal-case`} disabled={ro} value={d.correspondent} onChange={(e) => set("correspondent", e.target.value)} /></label>
              <label className={lab}>Valable jusqu'au<input type="date" className={`${cell} mt-1`} disabled={ro} value={d.validUntil} onChange={(e) => set("validUntil", e.target.value)} /></label>
              <label className={lab}>Date de relance (interne)<input type="date" className={`${cell} mt-1`} disabled={ro} value={d.followUpAt} onChange={(e) => set("followUpAt", e.target.value)} /></label>
              <label className={lab}>Remise globale (%)<input type="number" min={0} max={100} className={`${cell} mt-1`} disabled={ro} value={d.discountPct || ""} onChange={(e) => set("discountPct", Math.min(100, Math.max(0, Number(e.target.value) || 0)))} /></label>
              <label className={lab}>Frais de livraison HT (F CFA)<input type="number" min={0} className={`${cell} mt-1`} disabled={ro} value={d.shipping || ""} onChange={(e) => set("shipping", Math.max(0, Math.round(Number(e.target.value) || 0)))} /></label>
              <label className={lab}>TVA (%)<input type="number" min={0} max={100} step="any" className={`${cell} mt-1`} disabled={ro} value={d.vatRate || ""} onChange={(e) => set("vatRate", Math.min(100, Math.max(0, Number(e.target.value) || 0)))} /></label>
              <label className={lab}>Autres frais HT (F CFA)<input type="number" min={0} className={`${cell} mt-1`} disabled={ro} value={d.other || ""} onChange={(e) => set("other", Math.max(0, Math.round(Number(e.target.value) || 0)))} /></label>
              <label className={lab}>Acompte demandé (F CFA)<input type="number" min={0} className={`${cell} mt-1`} disabled={ro} value={d.acompte || ""} onChange={(e) => set("acompte", Math.max(0, Math.round(Number(e.target.value) || 0)))} /></label>
              <label className={lab}>Incoterm<input className={`${cell} mt-1 normal-case`} disabled={ro} value={d.incoterm} onChange={(e) => set("incoterm", e.target.value)} placeholder="ex. DDP, site d'Abidjan" /></label>
              <label className={lab}>Lieu de livraison<input className={`${cell} mt-1 normal-case`} disabled={ro} value={d.deliveryPlace} onChange={(e) => set("deliveryPlace", e.target.value)} /></label>
              <label className={lab}>Délai de livraison<input className={`${cell} mt-1 normal-case`} disabled={ro} value={d.deliveryDelay} onChange={(e) => set("deliveryDelay", e.target.value)} /></label>
              <label className={lab}>Conditions de paiement<input className={`${cell} mt-1 normal-case`} disabled={ro} value={d.paymentTerms} onChange={(e) => set("paymentTerms", e.target.value)} /></label>
            </div>
            <label className={lab}>Message d'accompagnement (visible du client)<textarea rows={6} className={`${cell} mt-1 normal-case`} disabled={ro} value={d.intro} onChange={(e) => set("intro", e.target.value)} placeholder="Bonjour, suite à votre demande, veuillez trouver notre meilleure offre…" /></label>
            <label className={lab}>Note interne sur ce devis (jamais visible du client)<textarea rows={4} className={`${cell} mt-1 normal-case`} disabled={ro} value={d.internalNote} onChange={(e) => set("internalNote", e.target.value)} placeholder="Marge visée, négociation, points d'attention…" /></label>
          </section>
        )}

        {tab === "media" && (
          <section className="space-y-4 rounded-xl border border-line bg-white p-5 shadow-card" aria-label="Images et annexes">
            <div>
              <h2 className="text-lg font-semibold text-night">Annexes du devis</h2>
              <p className="mt-1 text-sm text-slate">Fiches techniques, photos, plans : chaque image est jointe à la suite du devis, sur sa propre page, et visible du client. Les images d'une ligne se règlent dans l'onglet « Chiffrage ».</p>
            </div>
            {!ro && (
              <label className={`${btnGhostCls} cursor-pointer`}>{annexBusy ? <Loader2 size={15} className="animate-spin" aria-hidden /> : <ImagePlus size={15} aria-hidden />} Ajouter des images
                <input type="file" multiple accept="image/png,image/jpeg,image/webp" hidden disabled={annexBusy} onChange={(e) => { if (e.target.files?.length) void addAnnexes(e.target.files); e.target.value = ""; }} />
              </label>
            )}
            {d.annexes.length === 0 ? <p className="text-sm text-slate">Aucune annexe.</p> : (
              <ol className="grid gap-3 sm:grid-cols-2">
                {d.annexes.map((a, i) => (
                  <li key={a.url + i} className="flex gap-3 rounded-lg border border-line p-3">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={a.url} alt="" className="h-24 w-24 shrink-0 rounded-md border border-line bg-white object-contain" />
                    <div className="min-w-0 flex-1 space-y-2">
                      <label className={lab}>Légende<input className={`${cell} mt-1 normal-case`} disabled={ro} value={a.caption ?? ""} onChange={(e) => set("annexes", d.annexes.map((x, n) => (n === i ? { ...x, caption: e.target.value } : x)))} placeholder="ex. Fiche technique GS50" /></label>
                      {!ro && <div className="flex gap-1">
                        <button type="button" onClick={() => moveAnnex(i, -1)} disabled={i === 0} className="rounded-md border border-line p-1.5 text-slate hover:bg-page disabled:opacity-40" aria-label="Monter"><ArrowUp size={14} aria-hidden /></button>
                        <button type="button" onClick={() => moveAnnex(i, 1)} disabled={i === d.annexes.length - 1} className="rounded-md border border-line p-1.5 text-slate hover:bg-page disabled:opacity-40" aria-label="Descendre"><ArrowDown size={14} aria-hidden /></button>
                        <button type="button" onClick={() => set("annexes", d.annexes.filter((_, n) => n !== i))} className="rounded-md border border-line p-1.5 text-slate hover:bg-danger/10 hover:text-danger" aria-label="Retirer l'annexe"><Trash2 size={14} aria-hidden /></button>
                      </div>}
                    </div>
                  </li>
                ))}
              </ol>
            )}
          </section>
        )}

        {tab === "preview" && (
          <section aria-label="Aperçu du devis tel que le client le recevra">
            <div className="mb-3 flex items-center gap-2"><p className="text-sm text-slate">Le devis tel que le client le recevra (notes internes absentes). « Imprimer / PDF » produit un document A4 avec les annexes.</p><button type="button" onClick={() => window.print()} className={`${btnGhostCls} ml-auto shrink-0`}><Printer size={15} aria-hidden /> Imprimer / PDF</button></div>
            <QuoteSheet d={sheet} />
            {/* copie imprimée : hors de la mise en page de l'écran, donc jamais déformée ni suivie de pages vides */}
            <QuotePrintCopy><QuoteSheet d={sheet} /></QuotePrintCopy>
          </section>
        )}

        {tab === "file" && (
          <section className="grid gap-4 md:grid-cols-2" aria-label="Dossier client">
            <div className="rounded-xl border border-line bg-white p-5 shadow-card"><h2 className="mb-3 text-lg font-semibold text-night">Demandeur</h2>
              <dl className="space-y-2 text-sm">{([["Nom", customer.name], ["E-mail", customer.email], ["Téléphone", customer.phone], ["Entreprise", customer.company || "Particulier"], ["Fonction", customer.jobTitle], ["Adresse", customer.address], ["Demande reçue le", day(customer.createdAt)]] as const).map(([k, v]) => v ? <div key={k}><dt className={lab}>{k}</dt><dd className="whitespace-pre-line text-night">{v}</dd></div> : null)}</dl>
              {customer.message && <div className="mt-4 rounded-lg bg-page p-3 text-sm text-night"><p className={lab}>Remarques du client</p><p className="mt-1 whitespace-pre-line">{customer.message}</p></div>}
            </div>
            <div className="rounded-xl border border-line bg-white p-5 shadow-card"><h2 className="mb-3 text-lg font-semibold text-night">Autres demandes de ce client</h2>
              {history.length === 0 ? <p className="text-sm text-slate">Aucune autre demande avec cette adresse e-mail.</p> : <ul className="divide-y divide-line text-sm">{history.map((h) => <li key={h.id} className="flex items-center justify-between gap-3 py-2"><a href={`/quotes/${h.id}`} className="font-medium text-link hover:underline">{h.subject || "Demande sans objet"}</a><span className="text-xs text-slate">{day(h.createdAt)} · {h.status}</span></li>)}</ul>}
            </div>
          </section>
        )}
      </div>

      <aside className="space-y-4 lg:sticky lg:top-4 lg:self-start" aria-label="Récapitulatif et envoi">
        <div className="rounded-xl border border-line bg-white p-5 shadow-card">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate">Totaux</h2>
          <dl className="mt-3 space-y-1.5 text-sm">
            <div className="flex justify-between"><dt className="text-slate">Sous-total HT</dt><dd className="tabular-nums">{money(t.subtotal)}</dd></div>
            {t.discountAmount > 0 && <div className="flex justify-between"><dt className="text-slate">Remise {d.discountPct} %</dt><dd className="tabular-nums">− {money(t.discountAmount)}</dd></div>}
            {d.shipping > 0 && <div className="flex justify-between"><dt className="text-slate">Livraison</dt><dd className="tabular-nums">{money(d.shipping)}</dd></div>}
            {d.other > 0 && <div className="flex justify-between"><dt className="text-slate">Autres frais</dt><dd className="tabular-nums">{money(d.other)}</dd></div>}
            <div className="flex justify-between"><dt className="text-slate">TVA {d.vatRate} %</dt><dd className="tabular-nums">{money(t.vatAmount)}</dd></div>
            <div className="flex justify-between border-t border-line pt-2 text-lg font-bold text-night"><dt>Total TTC</dt><dd className="tabular-nums">{money(t.total)}</dd></div>
            {d.acompte > 0 && <div className="flex justify-between text-slate"><dt>Acompte</dt><dd className="tabular-nums">− {money(d.acompte)}</dd></div>}
            {d.acompte > 0 && <div className="flex justify-between font-semibold text-night"><dt>Reste à payer</dt><dd className="tabular-nums">{money(t.remaining)}</dd></div>}
          </dl>
        </div>
        <div className="rounded-xl border border-line bg-white p-5 shadow-card">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate">Avant l'envoi</h2>
          {ready.blocking.length === 0
            ? <p className="mt-3 flex items-center gap-2 text-sm font-medium text-[#007a52]"><Check size={16} aria-hidden /> Le devis est prêt à partir.</p>
            : <ul className="mt-3 space-y-1.5 text-sm text-night">{ready.blocking.map((x) => <li key={x} className="flex gap-2"><CircleAlert size={16} className="mt-0.5 shrink-0 text-danger" aria-hidden /> {x}</li>)}</ul>}
          {/* Conseils : ils n'empêchent jamais l'envoi. */}
          {ready.advice.length > 0 && (
            <ul className="mt-3 space-y-1.5 border-t border-line pt-3 text-xs text-slate">
              {ready.advice.map((x) => <li key={x} className="flex gap-2"><Info size={14} className="mt-0.5 shrink-0" aria-hidden /> {x}</li>)}
            </ul>
          )}
          {!ro && (
            <div className="mt-4 space-y-2">
              {!sentAt && <button type="button" onClick={save} disabled={pending || !dirty} className={`${btnGhostCls} w-full disabled:opacity-50`}><Save size={16} aria-hidden /> {pending ? "Enregistrement…" : "Enregistrer le brouillon"}</button>}
              <button type="button" onClick={send} disabled={pending || ready.blocking.length > 0} className={`${btnCls} w-full disabled:cursor-not-allowed disabled:opacity-50`}><Send size={16} aria-hidden /> {sentAt ? "Mettre à jour et renvoyer" : "Envoyer au client"}</button>
              {sentAt && <button type="button" onClick={unsend} disabled={pending} className={`${btnGhostCls} w-full`}><Undo2 size={16} aria-hidden /> Retirer l'envoi</button>}
            </div>
          )}
          {ro && <p className="mt-3 text-sm text-slate">Lecture seule : il faut le droit « Devis — écriture » pour chiffrer.</p>}
        </div>
        {sentAt && (
          <div className="rounded-xl border border-ok/30 bg-ok/5 p-5 text-sm">
            <p className="font-semibold text-[#007a52]">Devis visible dans l'espace client</p>
            <p className="mt-1 text-slate">{customer.name} le retrouve dans « Suivi de mes demandes de devis ». Aucun service d'e-mail n'est branché : prévenez-le par votre messagerie.</p>
            <div className="mt-3 flex flex-wrap gap-2"><a href={mailto()} className={btnGhostCls}><Mail size={15} aria-hidden /> Préparer l'e-mail</a>
              <button type="button" className={btnGhostCls} onClick={() => navigator.clipboard?.writeText(`${d.number} — ${money(t.total)}`)}><Copy size={15} aria-hidden /> Copier le total</button></div>
          </div>
        )}
      </aside>
    </div>
  );
}

function FragmentRow({ children }: { children: React.ReactNode }) { return <>{children}</>; }
