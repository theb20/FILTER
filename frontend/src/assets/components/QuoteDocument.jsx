import { createPortal } from "react-dom";
import { useI18n } from "../i18n/i18nContext.js";
import { useSite } from "../store/siteStore.js";

// Le devis reçu, présenté comme le document remis par l'entreprise : en-tête de société, destinataire, détail, totaux, cachet, annexes.
// Imprimable en A4 (le bouton « Imprimer / PDF » de la page appelante) : en-têtes de tableau répétés, annexes sur leurs pages.
const nf = (locale) => new Intl.NumberFormat(locale);

function Sheet({ quote, customer, request }) {
  const { t, locale } = useI18n();
  const site = useSite();
  const n = nf(locale);
  const money = (v) => n.format(v ?? 0);
  const dmy = (iso) => (iso ? new Date(iso).toLocaleDateString(locale, { day: "2-digit", month: "2-digit", year: "numeric" }) : "");
  const days = quote.validUntil && quote.sentAt ? Math.max(0, Math.round((new Date(quote.validUntil) - new Date(quote.sentAt)) / 86400000)) : 0;
  const expired = quote.validUntil && new Date(quote.validUntil).getTime() < Date.now();
  const letter = (site.letterhead || site.address || "").split("\n").map((x) => x.trim()).filter(Boolean);
  const ht = (quote.subtotal ?? 0) - (quote.discountAmount ?? 0) + (quote.shipping ?? 0) + (quote.other ?? 0);
  const rows = [
    [t("profile.quote.subtotal"), `${money(quote.subtotal)} CFA`, true],
    [`${t("profile.quote.discount")}${quote.discountPct ? ` ${quote.discountPct} %` : ""}`, `${money(quote.discountAmount)} CFA`, quote.discountAmount > 0],
    [t("profile.quote.shipping"), `${money(quote.shipping)} CFA`, true],
    [t("profile.quote.other"), `${money(quote.other)} CFA`, true],
  ];
  return (
    <article className="qd mx-auto w-full max-w-[860px] overflow-hidden rounded-md border border-gray-200 bg-white text-[12px] leading-[1.35] text-black shadow-sm" aria-label={`${t("profile.quote.title")} ${quote.number || ""}`}>
      <div className="band flex" aria-hidden="true"><i style={{ borderTopColor: "#8a3f10" }} /><i style={{ borderTopColor: "#c8601a" }} /><i style={{ borderTopColor: "#f4b183" }} /></div>
      <div className="sheet-body px-4 pb-6 pt-4 sm:px-7">
        <header className="grid gap-4 sm:grid-cols-2">
          <div className="flex gap-4">
            {site.logoUrl ? <img src={site.logoUrl} alt="" className="h-[92px] w-[92px] shrink-0 object-contain" /> : <div className="h-[92px] w-[92px] shrink-0 rounded bg-[#f2f2f2]" aria-hidden="true" />}
            <div className="min-w-0 text-[11px] leading-[1.5]">
              <p className="text-[13px] font-bold">{site.publisher || site.name}</p>
              {letter.map((l, i) => <p key={i}>{l}</p>)}
              {site.rccm && <p>RCCM : {site.rccm}</p>}
              {site.taxNumber && <p>{t("profile.quote.taxNumber")} {site.taxNumber}</p>}
              {site.capital && <p>{t("profile.quote.capital")} {site.capital}</p>}
            </div>
          </div>
          <div className="panel rounded-sm bg-[#f2f2f2] px-4 py-3">
            <p className="text-center text-[22px] font-light tracking-[0.35em]">{t("profile.quote.doc")}</p>
            <dl className="mt-2 space-y-0.5">
              <div className="flex justify-between"><dt className="text-[#555]">{t("profile.quote.ourRef")}</dt><dd className="font-semibold">D {quote.number}</dd></div>
              <div className="flex justify-between"><dt className="text-[#555]">{t("profile.quote.version")}</dt><dd>{quote.version || "1"}</dd></div>
              <div className="flex justify-between"><dt className="text-[#555]">{t("profile.quote.date")}</dt><dd>{dmy(quote.sentAt)}</dd></div>
              {request?.reference && <div className="flex justify-between"><dt className="text-[#555]">{t("profile.quote.yourRef")}</dt><dd>{request.reference}</dd></div>}
            </dl>
            {request?.subject && <p className="mt-2 text-center text-[13px] font-semibold uppercase">{request.subject}</p>}
          </div>
        </header>

        <div className="panel mt-3 rounded-sm border-t-2 border-black bg-[#f2f2f2] px-3 py-2 sm:w-1/2">
          <p><span className="text-[#555]">{t("profile.quote.contact")}</span> <strong>{quote.correspondent || site.publisher || site.name}</strong></p>
          {site.email && <p><span className="text-[#555]">E-Mail :</span> <span className="text-[#0563c1] underline">{site.email}</span></p>}
          {site.phone && <p><span className="text-[#555]">{t("profile.quote.phone")}</span> {site.phone}</p>}
        </div>

        <section className="mt-4 grid gap-5 sm:grid-cols-[1.1fr_0.9fr]">
          <div>
            <p className="border-b-[3px] border-[#d9d9d9] pb-0.5 text-[10px] font-bold tracking-wide text-[#2f5597]">{t("profile.quote.for").toUpperCase()}</p>
            <p className="mt-1 text-[13px] font-bold">{customer}</p>
          </div>
          <dl className="space-y-1 sm:pt-5">
            <div className="flex gap-2"><dt className="w-[120px] shrink-0 text-[#444]">{t("profile.quote.delay")}</dt><dd>{quote.deliveryDelay}</dd></div>
            <div className="flex gap-2"><dt className="w-[120px] shrink-0 text-[#444]">Incoterm :</dt><dd>{quote.incoterm}</dd></div>
            <div className="flex gap-2"><dt className="w-[120px] shrink-0 text-[#444]">{t("profile.quote.place")}</dt><dd>{quote.deliveryPlace}</dd></div>
            <div className="flex gap-2 pt-2"><dt className="w-[120px] shrink-0 text-[#444]">{t("profile.quote.payment")}</dt><dd>{quote.paymentTerms}</dd></div>
          </dl>
        </section>

        <p className="mt-4 border-y border-black py-1.5">
          {quote.intro ? <span className="whitespace-pre-line">{quote.intro}</span> : t("profile.quote.pleaseFind")}
          {quote.validUntil && <> <strong className={expired ? "text-red-700" : ""}>{expired ? `${t("profile.quote.expired")} ${dmy(quote.validUntil)}.` : `${t("profile.quote.validFor")} ${days > 0 ? `${days} ${t("profile.quote.days")} ` : ""}(${t("profile.quote.validUntil")} ${dmy(quote.validUntil)}).`}</strong></>}
        </p>

        <div className="overflow-x-auto">
          <table className="min-w-[520px] text-[11.5px]">
            <thead><tr className="border-b border-black bg-[#f2f2f2] text-left font-bold">
              <th className="px-1.5 py-1">DESCRIPTION</th><th className="w-[70px] px-1.5 py-1 text-center">{t("profile.quote.qty")}</th><th className="w-[96px] px-1.5 py-1 text-right">{t("profile.quote.unit")}</th><th className="w-[104px] px-1.5 py-1 text-right">{t("profile.quote.lineTotal")}</th>
            </tr></thead>
            <tbody>
              {quote.lines.map((l, i) => (
                <tr key={i} className={`border-b border-[#bfbfbf] align-top ${i % 2 ? "bg-[#fafafa]" : ""} ${l.optional ? "italic text-[#555]" : ""}`}>
                  <td className="px-1.5 py-1"><div className="flex gap-2">
                    {l.image && <img src={l.image} alt="" loading="lazy" className="h-9 w-9 shrink-0 rounded-sm border border-[#d9d9d9] object-contain" />}
                    <div className="min-w-0"><strong>{l.reference}</strong> {l.name}
                      {l.leadTime && <span className="block text-[10.5px] text-[#555]">{t("profile.quote.leadTime")} {l.leadTime}</span>}
                      {l.note && <span className="block text-[10.5px] text-[#555]">{l.note}</span>}
                      {l.optional && <span className="block text-[10.5px] font-semibold text-[#2f5597]">{t("profile.quote.option")}</span>}</div>
                  </div></td>
                  <td className="px-1.5 py-1 text-center tabular-nums">{l.quantity}</td>
                  <td className="px-1.5 py-1 text-right tabular-nums">{l.unitPrice == null ? "—" : money(l.unitPrice)}{l.discountPct ? <span className="block text-[10px] text-[#777]">−{l.discountPct} %</span> : null}</td>
                  <td className="px-1.5 py-1 text-right font-medium tabular-nums">{l.optional || l.lineTotal == null ? "—" : money(l.lineTotal)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="mt-3 grid gap-6 sm:grid-cols-[1fr_300px]" style={{ breakInside: "avoid" }}>
          <div className="pt-6">
            <p className="text-center text-[12px] font-bold">{t("profile.quote.commercial")} {site.publisher || site.name}{quote.correspondent ? ` / ${quote.correspondent}` : ""}</p>
            {site.stampUrl && <img src={site.stampUrl} alt="" className="mx-auto mt-2 max-h-[110px] object-contain" />}
          </div>
          <dl>
            {rows.map(([k, v, on]) => on ? <div key={k} className="flex justify-between border-b border-[#bfbfbf] py-0.5"><dt className="font-bold uppercase">{k}</dt><dd className="tabular-nums">{v}</dd></div> : null)}
            <div className="flex justify-between border-b border-[#bfbfbf] bg-[#f2f2f2] py-0.5 font-bold"><dt>{t("profile.quote.totalHt")}</dt><dd className="tabular-nums">{money(ht)} CFA</dd></div>
            <div className="flex justify-between border-b border-[#bfbfbf] py-0.5"><dt className="font-bold uppercase">{t("profile.quote.vat")} {quote.vatRate ? `${quote.vatRate} %` : ""}</dt><dd className="tabular-nums">{money(quote.vatAmount)} CFA</dd></div>
            <div className="flex justify-between border-b border-[#bfbfbf] bg-[#f2f2f2] py-0.5 font-bold"><dt className="uppercase">{t("profile.quote.total")}</dt><dd className="tabular-nums">{money(quote.total)} CFA</dd></div>
            <div className="flex justify-between border-b border-[#bfbfbf] py-0.5"><dt className="font-bold">{t("profile.quote.deposit")}</dt><dd className="tabular-nums">{money(quote.acompte)}</dd></div>
            <div className="total-due mt-1 flex justify-between rounded-sm bg-[#1f3864] px-2 py-1.5 text-[13px] font-bold text-white"><dt>{t("profile.quote.remaining")}</dt><dd className="tabular-nums">{money(quote.remaining ?? quote.total)} CFA</dd></div>
          </dl>
        </div>
      </div>
      <div className="band flex" aria-hidden="true"><i style={{ borderTopColor: "#8faadc" }} /><i style={{ borderTopColor: "#2f5597" }} /><i style={{ borderTopColor: "#1f3864" }} /></div>

      {(quote.annexes ?? []).map((a, i) => (
        <section key={i} className="annex border-t border-dashed border-gray-300 p-4 text-center sm:p-6">
          <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-[#555]">{t("profile.quote.annex")} {i + 1} / {quote.annexes.length}{a.caption ? ` — ${a.caption}` : ""}</p>
          <img src={a.url} alt={a.caption || ""} loading="lazy" className="mx-auto max-h-[1000px] max-w-full object-contain" />
        </section>
      ))}
    </article>
  );
}

// Le document est affiché à l'écran ET recopié en fin de <body> : c'est cette copie qui s'imprime, hors de la mise en
// page du site (en-tête, colonnes), donc jamais déformée ni suivie de pages vides.
export default function QuoteDocument(props) {
  const sheet = <Sheet {...props} />;
  return (
    <>
      {sheet}
      {typeof document !== "undefined" && createPortal(<div className="qs-print-root" aria-hidden="true">{sheet}</div>, document.body)}
    </>
  );
}
