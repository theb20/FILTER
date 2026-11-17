import { useState } from "react";
import { Link } from "react-router-dom";
import { CalendarCheck, ShoppingCart } from "lucide-react";
import PaymentMethods from "../PaymentMethods.jsx";
import QtyStepper from "../cart/QtyStepper.jsx";
import StockBadge from "../cart/StockBadge.jsx";
import { useCommerce } from "../../store/siteStore.js";
import { clampQty } from "../../lib/cartMath.js";
import { MAX_QTY, lineStatus } from "../../lib/cartLines.js";
import { tierRows } from "../../lib/productUtils.js";
import { flyToCart } from "../../animation/gsapSetup.js";
import { useCartStore } from "../../store/cartStore.js";
import { useNotify } from "../../configurations/notificationContext.js";
import { useI18n } from "../../i18n/i18nContext.js";

// Bloc d'achat : prix HT, prix dégressifs, disponibilité, quantité, ajout au panier ou devis.
export default function BuyBox({ product }) {
  const { t, formatPrice } = useI18n();
  const notify = useNotify();
  const addItem = useCartStore((s) => s.addItem);
  const [qty, setQty] = useState(1);
  const VAT_RATE = useCommerce().vatRate;

  const status = lineStatus(product.availability);
  const rows = tierRows(product.price, product.volumeTiers.map((x) => ({ minQty: x.minQuantity, percent: x.percent })));

  const add = (e) => {
    flyToCart(e.currentTarget);
    addItem(product, qty);
    notify(t("cart.added", { name: product.name }), "success");
  };

  return (
    <div>
      <p className="text-[34px] font-bold leading-none">{product.onQuote ? t("price.onQuote") : formatPrice(product.price)}</p>
      {!product.onQuote && (
        <p className="mt-1 text-sm text-gray-600">{VAT_RATE !== null ? t("pd.priceNote", { rate: Math.round(VAT_RATE * 100) }) : t("pd.priceNoTax")}</p>
      )}

      {rows.length > 0 && !product.onQuote && (
        <div className="mt-4 rounded-[6px] border border-gray-200">
          <p className="border-b border-gray-200 bg-gray-50 px-3 py-1.5 text-sm font-bold">{t("pd.tiers")}</p>
          <table className="w-full text-sm">
            <tbody>
              {rows.map((r) => (
                <tr key={r.minQty} className="border-b border-gray-100 last:border-0">
                  <th scope="row" className="px-3 py-1.5 text-left font-medium">{t("pd.tier.from", { qty: r.minQty })}</th>
                  <td className="px-3 py-1.5 text-green-700">−{r.percent} %</td>
                  <td className="px-3 py-1.5 text-right font-medium">{formatPrice(r.unit)} <span className="font-normal text-gray-500">/ {t("pd.unit")}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-4"><StockBadge status={status} stock={MAX_QTY} leadTimeDays={product.leadTimeDays} /></div>

      {product.onQuote ? (
        <Link to="/quote" className="mt-4 block rounded-[4px] bg-action px-6 py-3 text-center font-bold text-black hover:bg-action-dark">{t("price.askQuote")}</Link>
      ) : (
        <>
          <div className="mt-4 flex flex-wrap items-start gap-3">
            <div>
              <p className="mb-1 text-sm font-medium">{t("prod.quantity")}</p>
              <QtyStepper value={qty} max={MAX_QTY} name={product.name} onChange={(n) => setQty(clampQty(n, MAX_QTY))} />
            </div>
            <button
              type="button"
              onClick={add}
              disabled={status === "out"}
              className="mt-6 inline-flex min-w-[200px] flex-1 items-center justify-center gap-2 rounded-[4px] bg-action px-6 py-3 font-bold text-black transition-colors hover:bg-action-dark disabled:cursor-not-allowed disabled:bg-gray-300"
            >
              <ShoppingCart size={18} aria-hidden="true" /> {t("list.addToCart")}
            </button>
          </div>
          <Link to="/quote" className="mt-3 block rounded-[4px] border border-gray-300 px-6 py-2.5 text-center font-medium hover:bg-gray-50">{t("pd.quote")}</Link>
        </>
      )}

      {(status === "request" || status === "made") && (
        <p className="mt-4 flex items-start gap-2 text-sm text-gray-700">
          <CalendarCheck size={16} className="mt-0.5 shrink-0 text-green-700" aria-hidden="true" />
          {product.leadTimeDays ? t("stock.leadTime", { n: product.leadTimeDays }) : t("cart.ship.request")}
        </p>
      )}

      <div className="mt-4 border-t border-gray-200 pt-3">
        <p className="mb-2 text-sm text-gray-600">{t("footer.accept")}</p>
        <PaymentMethods />
      </div>
    </div>
  );
}
