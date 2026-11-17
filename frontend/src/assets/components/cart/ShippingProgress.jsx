import { Truck } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";

// Barre « Plus que X pour la livraison gratuite ».
export default function ShippingProgress({ eligible, free, remaining, progress }) {
  const { t, formatPrice } = useI18n();
  if (!eligible) return <p className="text-sm text-gray-600">{t("cart.free.notIntl")}</p>;
  const pct = Math.round(progress * 100);
  return (
    <div data-guide="cart-shipping">
      <p className="flex items-center gap-2 text-sm font-medium" aria-live="polite">
        <Truck size={16} className={free ? "text-green-700" : "text-gray-600"} aria-hidden="true" />
        {free ? <span className="text-green-700">{t("cart.free.reached")}</span> : t("cart.free.remaining", { amount: formatPrice(remaining) })}
      </p>
      <div
        role="progressbar"
        aria-label={t("cart.free.label")}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        className="mt-1.5 h-2 overflow-hidden rounded-full bg-gray-200"
      >
        <div className={`h-full rounded-full transition-[width] duration-300 ${free ? "bg-green-600" : "bg-action"}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
