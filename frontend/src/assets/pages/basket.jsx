import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { buildLines } from "../lib/cartLines.js";
import { itemCount, subtotal } from "../lib/cartMath.js";
import { computeTotals, deliveryWindow, quoteEligibility, validatePromo } from "../lib/cartTotals.js";
import { FREE_SHIPPING_THRESHOLD, PROMO_CODES, QUOTE_THRESHOLDS, SHIPPING_ZONES, VAT_RATE } from "../data/commerce.js";
import { useCartHydrated, useCartStore } from "../store/cartStore.js";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import CartItem from "../components/cart/CartItem.jsx";
import CartSkeleton from "../components/cart/CartSkeleton.jsx";
import DeliveryEstimate from "../components/cart/DeliveryEstimate.jsx";
import OrderSummary from "../components/cart/OrderSummary.jsx";
import PromoCode from "../components/cart/PromoCode.jsx";
import ShippingProgress from "../components/cart/ShippingProgress.jsx";
import SavedItems from "../components/cart/SavedItems.jsx";

export default function Cart() {
  const { t, formatPrice } = useI18n();
  const notify = useNotify();
  const hydrated = useCartHydrated();
  const { items, saved, promoCode, zone: zoneId, note, setQuantity, removeItem, restoreItem, saveForLater, moveToCart, removeSaved, applyPromo, clearPromo, setZone, setNote } = useCartStore();
  const [now] = useState(() => new Date());

  useEffect(() => {
    document.title = t("meta.cart");
  }, [t]);

  const lines = buildLines(items);
  const savedLines = buildLines(saved);
  const sub = subtotal(lines);
  const blocked = lines.some((l) => l.status === "out");

  const quote = quoteEligibility(itemCount(lines), sub, QUOTE_THRESHOLDS);

  const zone = SHIPPING_ZONES.find((z) => z.id === zoneId) ?? SHIPPING_ZONES[0] ?? null; // aucune zone configurée : pas de calcul de livraison
  const promoResult = promoCode ? validatePromo(promoCode, sub, now, PROMO_CODES) : null;
  const totals = computeTotals({
    subtotal: sub,
    promo: promoResult?.status === "valid" ? promoResult.promo : null,
    zone,
    freeThreshold: FREE_SHIPPING_THRESHOLD,
    vatRate: VAT_RATE,
  });

  const remove = (id) => {
    const removed = removeItem(id);
    if (!removed) return;
    const name = t(`prod.${id}`);
    notify(t("cart.removed", { name }), "info", 8000, {
      label: t("cart.undo"),
      onClick: () => restoreItem(removed.entry, removed.index),
    });
  };

  const save = (id) => {
    saveForLater(id);
    notify(t("cart.savedMsg", { name: t(`prod.${id}`) }), "info");
  };

  return (
    <div className="mx-2.5 mt-2.5 space-y-2.5">
      <div className="grid gap-2.5 lg:grid-cols-[minmax(0,1fr)_340px]">
        <section className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="cart-title">
          <h1 id="cart-title" className="text-[40px] font-bold leading-tight">{t("cart.title")}</h1>

          {!hydrated ? (
            <CartSkeleton />
          ) : lines.length === 0 ? (
            <div className="py-10 text-center">
              <p>{t("cart.empty")}</p>
              <Link to="/category" className="mt-4 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
                {t("cart.browse")} <span aria-hidden="true">&nbsp;→</span>
              </Link>
            </div>
          ) : (
            <ul className="mt-2 divide-y divide-gray-200">
              {lines.map((line) => (
                <CartItem
                  key={line.id}
                  line={line}
                  onQuantity={(q) => setQuantity(line.id, q)}
                  onRemove={() => remove(line.id)}
                  onSaveForLater={() => save(line.id)}
                />
              ))}
            </ul>
          )}
        </section>

        <OrderSummary
          totals={totals}
          vatRate={VAT_RATE}
          empty={lines.length === 0}
          blocked={blocked}
          note={note}
          onNote={setNote}
          onCheckout={() => notify(t("demo.notice"), "demo")}
          quote={quote}
          quoteThresholds={QUOTE_THRESHOLDS}
          onDownloadQuote={() => notify(t("cart.quotePdfDemo"), "demo")}
        >
          {zone && FREE_SHIPPING_THRESHOLD !== null && <ShippingProgress eligible={zone.freeEligible} free={totals.freeShipping} remaining={totals.remainingForFree} progress={totals.progress} />}
          {zone && <DeliveryEstimate zones={SHIPPING_ZONES} zoneId={zone.id} window={deliveryWindow(now, zone)} onChange={setZone} />}
          {PROMO_CODES.length > 0 && (
            <PromoCode
              appliedCode={promoCode}
              appliedResult={promoResult}
              validate={(code) => validatePromo(code, sub, now, PROMO_CODES)}
              onApply={applyPromo}
              onClear={clearPromo}
            />
          )}
        </OrderSummary>
      </div>

      {hydrated && <SavedItems lines={savedLines} onMoveToCart={moveToCart} onRemove={removeSaved} />}
    </div>
  );
}
