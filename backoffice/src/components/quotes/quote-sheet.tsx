"use client";
/* eslint-disable react/no-unescaped-entities, @next/next/no-img-element -- document imprimable : textes français, images téléversées */

import { useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
// Le devis tel qu'il est remis au client : en-tête de société, destinataire, tableau, totaux, cachet, annexes.
// Composant purement présentationnel (aucun état) : le même rendu sert à l'aperçu du studio et à l'impression PDF.
export type SheetLine = { reference: string; name?: string; quantity: number; unitPrice?: number | null; discountPct?: number | null; lineTotal?: number | null; leadTime?: string; note?: string; optional?: boolean; image?: string };
export type SheetData = {
  number: string; version?: string; date: string; validUntil?: string; subject?: string; reference?: string;
  company: { legalName: string; lines: string[]; rccm?: string; taxNumber?: string; capital?: string; logoUrl?: string; stampUrl?: string; phone?: string; email?: string };
  correspondent?: string;
  customer: { company?: string; name: string; address?: string; attention?: string; jobTitle?: string; email?: string; phone?: string };
  deliveryDelay?: string; incoterm?: string; deliveryPlace?: string; paymentTerms?: string; intro?: string;
  lines: SheetLine[];
  subtotal: number; discountPct?: number; discountAmount: number; shipping: number; other: number; vatRate: number; vatAmount: number; total: number; acompte: number; remaining: number;
  annexes: { url: string; caption?: string }[];
};

const nf = new Intl.NumberFormat("fr-FR");
const money = (n: number) => nf.format(n);
const dmy = (iso?: string) => (iso ? new Date(iso).toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit", year: "numeric" }) : "");
const daysBetween = (a: string, b?: string) => (b ? Math.max(0, Math.round((new Date(b).getTime() - new Date(a).getTime()) / 86400000)) : 0);

// ── Impression ──────────────────────────────────────────────────────────────
// Le document est sorti du flux de l'application (portail en fin de <body>) et c'est le SEUL élément imprimé :
// la mise en page de l'écran (menu, colonnes, onglets) ne déforme plus la feuille et ne laisse plus de pages vides.
// `print-color-adjust: exact` force les aplats de couleur à sortir même quand « Graphiques d'arrière-plan » est décoché.
export const SHEET_PRINT_CSS = `
.qs{font-family:Arial,Helvetica,sans-serif;color:#111;font-size:12px;line-height:1.35;background:#fff}
.qs *{box-sizing:border-box}
.qs .band{display:flex}.qs .band i{flex:1;height:0;border-top:9px solid transparent}
.qs table{border-collapse:collapse;width:100%}
.qs thead{display:table-header-group}.qs tr{break-inside:avoid}
.qs .annex{break-before:page;page-break-before:always;text-align:center}
.qs-print-root{display:none}
@media print{
  @page{size:A4;margin:8mm}
  html,body{background:#fff!important;margin:0!important;padding:0!important}
  body>*:not(.qs-print-root){display:none!important}
  .qs-print-root{display:block!important}
  .qs,.qs *{-webkit-print-color-adjust:exact!important;print-color-adjust:exact!important}
  .qs{width:100%!important;max-width:none!important;margin:0!important;border:0!important;border-radius:0!important;box-shadow:none!important;font-size:10.5px!important}
  .qs .sheet-body{padding:14px 10px 18px!important}
  .qs .noprint{display:none!important}
  /* Sans « Graphiques d'arrière-plan », aucun aplat n'est imprimé : bordures de secours et texte foncé pour rester lisible. */
  .qs .panel{border:1px solid #c9c9c9}
  .qs thead tr{border-top:1px solid #000}
  .qs .total-due{background:transparent!important;color:#1f3864!important;border:1.5px solid #1f3864}
}`;

const noop = () => () => {};
const yes = () => true;
const no = () => false;

/** Copie du document placée en fin de <body> : c'est elle qui s'imprime (l'écran garde la sienne, masquée à l'impression). */
export function QuotePrintCopy({ children }: { children: React.ReactNode }) {
  // Rendu côté navigateur uniquement (document.body n'existe pas au rendu serveur).
  const mounted = useSyncExternalStore(noop, yes, no);
  if (!mounted) return null;
  return createPortal(<div className="qs-print-root" aria-hidden>{children}</div>, document.body);
}

export default function QuoteSheet({ d }: { d: SheetData }) {
  const days = daysBetween(d.date, d.validUntil);
  const c = d.company;
  const show = (n: number) => n > 0;
  return (
    <article className="qs mx-auto w-full max-w-[860px] overflow-hidden rounded-md border border-line bg-white shadow-card" aria-label={`Devis ${d.number}`}>
      <style>{SHEET_PRINT_CSS}</style>
      <div className="band" aria-hidden><i style={{ borderTopColor: "#8a3f10" }} /><i style={{ borderTopColor: "#c8601a" }} /><i style={{ borderTopColor: "#f4b183" }} /></div>

      <div className="sheet-body px-7 pb-6 pt-4">
        {/* En-tête : société à gauche, bloc DEVIS à droite */}
        <header className="grid grid-cols-[1fr_1fr] gap-4">
          <div className="flex gap-4">
            {c.logoUrl ? <img src={c.logoUrl} alt="" className="h-[92px] w-[92px] shrink-0 object-contain" /> : <div className="h-[92px] w-[92px] shrink-0 rounded bg-[#f2f2f2]" aria-hidden />}
            <div className="min-w-0 text-[11px] leading-[1.5]">
              <p className="text-[13px] font-bold">{c.legalName}</p>
              {c.lines.map((l, i) => <p key={i}>{l}</p>)}
              {c.rccm && <p>RCCM : {c.rccm}</p>}
              {c.taxNumber && <p>Compte contribuable {c.taxNumber}</p>}
              {c.capital && <p>Capital {c.capital}</p>}
            </div>
          </div>
          <div className="panel rounded-sm bg-[#f2f2f2] px-4 py-3 text-[12px]">
            <p className="text-center text-[22px] font-light tracking-[0.35em]">DEVIS</p>
            <dl className="mt-2 space-y-0.5">
              <div className="flex justify-between"><dt className="text-[#555]">Notre référence :</dt><dd className="font-semibold">D {d.number}</dd></div>
              <div className="flex justify-between"><dt className="text-[#555]">Version / Avenant :</dt><dd>{d.version || "1"}</dd></div>
              <div className="flex justify-between"><dt className="text-[#555]">Date :</dt><dd>{dmy(d.date)}</dd></div>
              {d.reference && <div className="flex justify-between"><dt className="text-[#555]">Votre référence :</dt><dd>{d.reference}</dd></div>}
            </dl>
            {d.subject && <p className="mt-2 text-center text-[13px] font-semibold uppercase">{d.subject}</p>}
          </div>
        </header>

        <div className="mt-3 grid grid-cols-[1fr_1fr] gap-4">
          <div className="panel rounded-sm border-t-2 border-black bg-[#f2f2f2] px-3 py-2 text-[12px]">
            <p><span className="text-[#555]">Correspondant :</span> <strong>{d.correspondent || c.legalName}</strong></p>
            {c.email && <p><span className="text-[#555]">E-Mail :</span> <span className="text-[#0563c1] underline">{c.email}</span></p>}
            {c.phone && <p><span className="text-[#555]">Téléphone :</span> {c.phone}</p>}
          </div>
          <div />
        </div>

        {/* Destinataire et conditions */}
        <section className="mt-4 grid grid-cols-[1.1fr_0.9fr] gap-6">
          <div>
            <p className="border-b-[3px] border-[#d9d9d9] pb-0.5 text-[10px] font-bold tracking-wide text-[#2f5597]">DESTINATAIRE</p>
            <p className="mt-1 text-[13px] font-bold">{d.customer.company || d.customer.name}</p>
            {d.customer.address && <p className="whitespace-pre-line">{d.customer.address}</p>}
            {(d.customer.company || d.customer.attention) && <p className="mt-0.5"><em className="font-bold">A l'attention de :</em> <strong>{d.customer.attention || d.customer.name}</strong></p>}
            {d.customer.jobTitle && <p>{d.customer.jobTitle}</p>}
            <p>{[d.customer.email, d.customer.phone && `Tel: ${d.customer.phone}`].filter(Boolean).join("     ")}</p>
          </div>
          <dl className="space-y-1 pt-5">
            <div className="flex gap-2"><dt className="w-[120px] shrink-0 text-[#444]">Délai de livraison :</dt><dd>{d.deliveryDelay}</dd></div>
            <div className="flex gap-2"><dt className="w-[120px] shrink-0 text-[#444]">Incoterm :</dt><dd>{d.incoterm}</dd></div>
            <div className="flex gap-2"><dt className="w-[120px] shrink-0 text-[#444]">Lieu de livraison :</dt><dd>{d.deliveryPlace}</dd></div>
            <div className="flex gap-2 pt-2"><dt className="w-[120px] shrink-0 text-[#444]">Échéance Paiement</dt><dd>{d.paymentTerms}</dd></div>
          </dl>
        </section>

        <p className="mt-4 border-y border-black py-1.5">
          {d.intro ? <span className="whitespace-pre-line">{d.intro}</span> : "Prière trouver ci-dessous le détail de notre devis"}
          {d.validUntil && <> <strong>Offre valable {days > 0 ? `${days} jours` : "jusqu'au " + dmy(d.validUntil)}{days > 0 ? ` (jusqu'au ${dmy(d.validUntil)})` : ""}.</strong></>}
        </p>

        {/* Détail */}
        <table className="mt-0 text-[11.5px]">
          <thead>
            <tr className="border-b border-black bg-[#f2f2f2] text-left font-bold">
              <th className="px-1.5 py-1">DESCRIPTION</th><th className="w-[70px] px-1.5 py-1 text-center">Quantité</th><th className="w-[96px] px-1.5 py-1 text-right">Prix<br />unitaire</th><th className="w-[104px] px-1.5 py-1 text-right">Prix total</th>
            </tr>
          </thead>
          <tbody>
            {d.lines.map((l, i) => (
              <tr key={i} className={`border-b border-[#bfbfbf] align-top ${i % 2 ? "bg-[#fafafa]" : ""} ${l.optional ? "italic text-[#555]" : ""}`}>
                <td className="px-1.5 py-1">
                  <div className="flex gap-2">
                    {l.image && <img src={l.image} alt="" className="h-9 w-9 shrink-0 rounded-sm border border-[#d9d9d9] object-contain" />}
                    <div className="min-w-0"><strong>{l.reference}</strong> {l.name}
                      {l.leadTime && <span className="block text-[10.5px] text-[#555]">Délai : {l.leadTime}</span>}
                      {l.note && <span className="block text-[10.5px] text-[#555]">{l.note}</span>}
                      {l.optional && <span className="block text-[10.5px] font-semibold text-[#2f5597]">Option — hors total</span>}</div>
                  </div>
                </td>
                <td className="px-1.5 py-1 text-center tabular-nums">{l.quantity}</td>
                <td className="px-1.5 py-1 text-right tabular-nums">{l.unitPrice == null ? "—" : money(l.unitPrice)}{l.discountPct ? <span className="block text-[10px] text-[#777]">−{l.discountPct} %</span> : null}</td>
                <td className="px-1.5 py-1 text-right font-medium tabular-nums">{l.optional || l.lineTotal == null ? "—" : money(l.lineTotal)}</td>
              </tr>
            ))}
          </tbody>
        </table>

        {/* Pied : signature à gauche, totaux à droite */}
        <div className="mt-3 grid grid-cols-[1fr_300px] gap-6" style={{ breakInside: "avoid" }}>
          <div className="pt-6">
            <p className="text-center text-[12px] font-bold">Service commercial {c.legalName}{d.correspondent ? ` / ${d.correspondent}` : ""}</p>
            {c.stampUrl && <img src={c.stampUrl} alt="Cachet et signature" className="mx-auto mt-2 max-h-[110px] object-contain" />}
          </div>
          <dl className="text-[12px]">
            {[
              ["SOUS-TOTAL", `${money(d.subtotal)} CFA`, true],
              [`REMISE${d.discountPct ? ` ${d.discountPct} %` : ""}`, `${money(d.discountAmount)} CFA`, show(d.discountAmount)],
              ["EXPEDITION", `${money(d.shipping)} CFA`, true],
              ["AUTRES", `${money(d.other)} CFA`, true],
            ].map(([k, v, on]) => on ? <div key={k as string} className="flex justify-between border-b border-[#bfbfbf] py-0.5"><dt className="font-bold">{k}</dt><dd className="tabular-nums">{v}</dd></div> : null)}
            <div className="flex justify-between border-b border-[#bfbfbf] bg-[#f2f2f2] py-0.5 font-bold"><dt>TOTAL HT</dt><dd className="tabular-nums">{money(d.subtotal - d.discountAmount + d.shipping + d.other)} CFA</dd></div>
            <div className="flex justify-between border-b border-[#bfbfbf] py-0.5"><dt className="font-bold">TVA {d.vatRate ? `${d.vatRate} %` : ""}</dt><dd className="tabular-nums">{money(d.vatAmount)} CFA</dd></div>
            <div className="flex justify-between border-b border-[#bfbfbf] bg-[#f2f2f2] py-0.5 font-bold"><dt>TOTAL TTC</dt><dd className="tabular-nums">{money(d.total)} CFA</dd></div>
            <div className="flex justify-between border-b border-[#bfbfbf] py-0.5"><dt className="font-bold">ACOMPTE</dt><dd className="tabular-nums">{money(d.acompte)}</dd></div>
            <div className="total-due mt-1 flex justify-between rounded-sm bg-[#1f3864] px-2 py-1.5 text-[13px] font-bold text-white"><dt>RESTE A PAYER</dt><dd className="tabular-nums">{money(d.remaining)} CFA</dd></div>
          </dl>
        </div>
      </div>
      <div className="band" aria-hidden><i style={{ borderTopColor: "#8faadc" }} /><i style={{ borderTopColor: "#2f5597" }} /><i style={{ borderTopColor: "#1f3864" }} /></div>

      {/* Annexes : fiches techniques et photos, chacune sur sa page à l'impression */}
      {d.annexes.map((a, i) => (
        <section key={i} className="annex border-t border-dashed border-line p-6">
          <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-[#555]">Annexe {i + 1} / {d.annexes.length}{a.caption ? ` — ${a.caption}` : ""}</p>
          <img src={a.url} alt={a.caption || `Annexe ${i + 1}`} className="mx-auto max-h-[1000px] max-w-full object-contain" />
        </section>
      ))}
    </article>
  );
}
