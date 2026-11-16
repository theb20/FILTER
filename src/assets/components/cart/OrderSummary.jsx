import { Link } from "react-router-dom";
import { Download } from "lucide-react";
import PaymentMethods from "../PaymentMethods.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

const Row = ({ label, children, className = "" }) => (
  <div className={`flex justify-between gap-3 ${className}`}><dt>{label}</dt><dd className="text-right">{children}</dd></div>
);

// Colonne de droite : affichage seul. Les montants viennent de lib/cartTotals.computeTotals.
export default function OrderSummary({
  totals, vatRate, empty, blocked, note, onNote, onCheckout, onDownloadQuote, quote, quoteThresholds, children,
}) {
  const { t, formatPrice } = useI18n();
  return (
    <aside className="h-fit rounded-[4px] bg-white p-6 shadow-card lg:sticky lg:top-28" aria-label={t("cart.summary")}>
      <h2 className="text-[24px] font-semibold">{t("cart.summary")}</h2>

      <div className="mt-4 space-y-5">{children}</div>

      <dl className="mt-5 space-y-2 text-[15px]" aria-live="polite" aria-atomic="true">
        <Row label={t("cart.subtotal")} className="font-medium">{formatPrice(totals.subtotal)}</Row>
        {totals.discount > 0 && <Row label={t("cart.discount")} className="text-green-700">−{formatPrice(totals.discount)}</Row>}
        <Row label={t("cart.shipping")}>{totals.freeShipping ? <span className="font-medium text-green-700">{t("cart.free")}</span> : formatPrice(totals.shipping)}</Row>
        <Row label={t("cart.ht")} className="border-t border-gray-200 pt-2 font-medium">{formatPrice(totals.ht)}</Row>
        <Row label={t("cart.vat", { rate: Math.round(vatRate * 100) })} className="text-gray-600">{formatPrice(totals.vat)}</Row>
        <Row label={t("cart.ttc")} className="border-t border-gray-200 pt-3 text-[20px] font-bold">{formatPrice(totals.ttc)}</Row>
      </dl>
      <p className="mt-1 text-xs text-gray-500">{t("cart.vatNote")}</p>

      <div className="mt-5">
        <label htmlFor="order-note" className="block text-sm font-medium">
          {t("cart.note.label")} <span className="font-normal text-gray-500">{t("auth.optional")}</span>
        </label>
        <input
          id="order-note"
          value={note}
          onChange={(e) => onNote(e.target.value)}
          maxLength={200}
          placeholder={t("cart.note.placeholder")}
          className="mt-1 w-full rounded-[3px] border border-gray-300 px-3 py-2"
        />
      </div>

      <button
        type="button"
        onClick={onCheckout}
        disabled={empty || blocked}
        aria-describedby={blocked ? "checkout-blocked" : undefined}
        className="mt-5 w-full rounded-[3px] bg-action px-5 py-3 font-bold text-black hover:bg-action-dark disabled:cursor-not-allowed disabled:bg-gray-300 disabled:text-gray-600"
      >
        {t("cart.checkout")}
      </button>
      {blocked && <p id="checkout-blocked" role="alert" className="mt-2 text-sm text-red-700">{t("cart.blocked")}</p>}

      {/* Devis : visible uniquement à partir des seuils (nombre d'articles ou montant HT) */}
      {quote.eligible ? (
        <div className="mt-3 grid gap-2">
          <Link to="/quote" className="rounded-[3px] border border-gray-300 px-5 py-2.5 text-center font-medium hover:bg-gray-50">{t("cart.quote")}</Link>
          <button type="button" onClick={onDownloadQuote} className="inline-flex items-center justify-center gap-2 rounded-[3px] px-5 py-2 text-sm font-medium text-link hover:underline">
            <Download size={16} aria-hidden="true" /> {t("cart.quotePdf")}
          </button>
        </div>
      ) : (
        <p className="mt-3 rounded-[3px] bg-gray-50 p-3 text-sm text-gray-700" aria-live="polite">
          {t("cart.quote.locked", { items: quoteThresholds.minItems, amount: formatPrice(quoteThresholds.minAmount) })}
          {!empty && (
            <span className="mt-1 block font-medium">
              {t("cart.quote.remaining", { items: quote.remainingItems, amount: formatPrice(quote.remainingAmount) })}
            </span>
          )}
        </p>
      )}

      <div className="mt-5 border-t border-gray-200 pt-4">
        <p className="mb-2 text-sm text-gray-600">{t("footer.accept")}</p>
        <PaymentMethods />
      </div>

      <p className="mt-4 rounded-[3px] bg-info p-2 text-sm">{t("demo.notice")}</p>
    </aside>
  );
}
