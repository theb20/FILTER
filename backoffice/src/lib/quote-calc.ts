// Chiffrage d'un devis : MÊME formule que le service content (application/quote.go), qui recalcule de toute façon à l'enregistrement.
export type QLine = {
  reference: string; name?: string; quantity: number; unitPrice?: number | null; discountPct?: number | null;
  leadTime?: string; note?: string; internal?: string; optional?: boolean; custom?: boolean; productId?: string; image?: string;
};
export type QDraft = {
  number: string; version: string; incoterm: string; correspondent: string;
  lines: QLine[]; discountPct: number; shipping: number; other: number; acompte: number; vatRate: number; validUntil: string; // yyyy-mm-dd
  deliveryPlace: string; deliveryDelay: string; paymentTerms: string; intro: string; internalNote: string; followUpAt: string;
  annexes: { url: string; caption?: string }[];
};
export type QTotals = { subtotal: number; discountAmount: number; taxable: number; vatAmount: number; total: number; remaining: number; priced: number; missing: number };

const hasPrice = (l: QLine) => typeof l.unitPrice === "number" && Number.isFinite(l.unitPrice);

/** Total d'une ligne (remise de ligne comprise), ou null tant qu'elle n'a pas de prix. */
export function lineTotal(l: QLine): number | null {
  if (!hasPrice(l)) return null;
  return Math.round(l.quantity * (l.unitPrice as number) * (1 - (l.discountPct ?? 0) / 100));
}

export function computeTotals(lines: QLine[], discountPct: number, shipping: number, vatRate: number, other = 0, acompte = 0): QTotals {
  let subtotal = 0, priced = 0, missing = 0;
  for (const l of lines) {
    if (l.optional) continue; // proposée au client, hors total
    const t = lineTotal(l);
    if (t === null) missing++; else { subtotal += t; priced++; }
  }
  const discountAmount = Math.round((subtotal * (discountPct || 0)) / 100);
  const taxable = subtotal - discountAmount + (shipping || 0) + (other || 0);
  const vatAmount = Math.round((taxable * (vatRate || 0)) / 100);
  const total = taxable + vatAmount;
  return { subtotal, discountAmount, taxable, vatAmount, total, remaining: total - (acompte || 0), priced, missing };
}

/** Ce qui empêche l'envoi (`blocking`, mêmes règles que le service) et ce qui est seulement conseillé (`advice`). */
export type QReadiness = { blocking: string[]; advice: string[] };

export function readiness(d: QDraft): QReadiness {
  const t = computeTotals(d.lines, d.discountPct, d.shipping, d.vatRate, d.other, d.acompte);
  const blocking: string[] = [];
  const advice: string[] = [];
  if (d.lines.length === 0 || t.priced === 0) blocking.push("Chiffrer au moins une ligne");
  if (t.missing > 0) blocking.push(`${t.missing} ligne(s) sans prix (ou marquez-les « optionnelle »)`);
  if (!d.number.trim()) blocking.push("Donner un numéro au devis");
  if (!d.validUntil) blocking.push("Indiquer la date de validité");
  else if (new Date(`${d.validUntil}T23:59:59Z`).getTime() < Date.now()) blocking.push("La date de validité est dépassée");
  // Conseils : le devis part quand même sans eux.
  if (!d.intro.trim()) advice.push("Aucun message d'accompagnement : le client recevra le devis sans un mot d'introduction");
  if (!d.paymentTerms.trim()) advice.push("Conditions de paiement non précisées");
  if (!d.deliveryDelay.trim()) advice.push("Délai de livraison non précisé");
  return { blocking, advice };
}

export const money = (n: number) => `${new Intl.NumberFormat("fr-FR").format(n)} F CFA`;
