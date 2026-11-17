import { useI18n } from "../../i18n/i18nContext.js";

const TONE = { in: "text-green-700", low: "text-orange-700", out: "text-red-700", request: "text-gray-700", made: "text-gray-700" };

// Disponibilité + délai (affichage seul). Tant que le stock n'est pas géré, le statut courant est « sur demande ».
export default function StockBadge({ status, stock, leadTimeDays }) {
  const { t } = useI18n();
  const label = {
    out: t("cart.stock.out"),
    low: t("cart.stock.low", { n: stock }),
    in: t("cart.stock.in"),
    made: t("stock.madeToOrder"),
    request: t("cart.stock.request"),
  }[status];
  return (
    <p className="text-sm">
      <span className={`font-semibold ${TONE[status]}`}>{label}</span>
      {leadTimeDays ? <span className="text-gray-600"> · {t("stock.leadTime", { n: leadTimeDays })}</span> : null}
    </p>
  );
}
