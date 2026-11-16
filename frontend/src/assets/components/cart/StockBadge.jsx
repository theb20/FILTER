import { useI18n } from "../../i18n/i18nContext.js";

const TONE = { in: "text-green-700", low: "text-orange-700", out: "text-red-700" };

// Disponibilité + délai d'expédition (affichage seul).
export default function StockBadge({ status, stock, shipping }) {
  const { t } = useI18n();
  const label = status === "out" ? t("cart.stock.out") : status === "low" ? t("cart.stock.low", { n: stock }) : t("cart.stock.in");
  return (
    <p className="text-sm">
      <span className={`font-semibold ${TONE[status]}`}>{label}</span>
      <span className="text-gray-600"> · {t(`cart.ship.${shipping}`)}</span>
    </p>
  );
}
