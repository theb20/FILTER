import { Link } from "react-router-dom";
import { AlertTriangle, Bookmark, Trash2 } from "lucide-react";
import LineArt from "../LineArt.jsx";
import QtyStepper from "./QtyStepper.jsx";
import StockBadge from "./StockBadge.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

const ghostBtn = "inline-flex items-center gap-1.5 text-sm text-gray-700 hover:text-black hover:underline";

// Ligne du panier : affichage uniquement. `line` vient de lib/cartLines.buildLine,
// tous les calculs (remises, totaux) sont déjà faits.
export default function CartItem({ line, onQuantity, onRemove, onSaveForLater }) {
  const { t, formatPrice } = useI18n();
  const name = line.name;
  const compat = line.compat.join(", ");

  return (
    <li className="py-5">
      <div className="flex gap-4">
        <Link to={`/card?id=${line.id}`} aria-label={name} className="flex h-24 w-24 shrink-0 items-center justify-center bg-gray-50 sm:h-28 sm:w-28">
          {line.image ? <img src={line.image} alt="" className="h-full w-full object-contain" /> : <LineArt stroke="#9ca3af" className="h-20" />}
        </Link>

        <div className="min-w-0 flex-1">
          <Link to={`/card?id=${line.id}`} className="text-[19px] font-semibold leading-snug text-link hover:underline">{name}</Link>
          <p className="text-sm text-gray-600">{line.brand} · {t("list.ref")} {line.reference}</p>

          <ul className="mt-1.5 flex flex-wrap gap-x-4 gap-y-0.5 text-sm text-gray-700">
            {line.specs.map((s) => (
              <li key={s.label}><span className="text-gray-500">{s.label} :</span> {s.value}</li>
            ))}
          </ul>
          {compat && <p className="mt-1 text-sm text-gray-600">{t("cart.compat", { list: compat })}</p>}
          <div className="mt-1.5"><StockBadge status={line.status} stock={line.stock} leadTimeDays={line.leadTimeDays} /></div>
        </div>
      </div>

      {line.status === "out" && (
        <p role="alert" className="mt-3 flex items-start gap-2 rounded-[3px] bg-red-50 p-2.5 text-sm text-red-800">
          <AlertTriangle size={16} className="mt-0.5 shrink-0" aria-hidden="true" /> {t("cart.alert.out")}
        </p>
      )}
      {line.priceChanged && (
        <p role="status" className="mt-3 flex items-start gap-2 rounded-[3px] bg-info p-2.5 text-sm">
          <AlertTriangle size={16} className="mt-0.5 shrink-0" aria-hidden="true" />
          {t("cart.alert.price", { old: formatPrice(line.priceAtAdd), new: formatPrice(line.price) })}
        </p>
      )}

      <div className="mt-4 flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
        <div className="flex flex-wrap items-start gap-x-5 gap-y-2">
          <QtyStepper value={line.quantity} max={Math.max(1, line.stock)} name={name} onChange={onQuantity} />
          <div className="flex gap-4 pt-1.5">
            <button type="button" onClick={onSaveForLater} className={ghostBtn}>
              <Bookmark size={15} aria-hidden="true" /> {t("cart.saveLater")}
            </button>
            <button type="button" onClick={onRemove} aria-label={t("cart.remove", { name })} className={`${ghostBtn} hover:text-red-700`}>
              <Trash2 size={15} aria-hidden="true" /> {t("cart.removeShort")}
            </button>
          </div>
        </div>

        {line.onQuote ? (
          <div className="text-right"><p className="text-[20px] font-bold">{t("price.onQuote")}</p><p className="text-sm text-gray-600">{t("cart.quoteOnly")}</p></div>
        ) : (
        <div className="text-right">
          <p className="text-sm text-gray-600">
            {t("cart.unitPrice")} :{" "}
            {line.percent > 0 && <s className="mr-1 text-gray-400">{formatPrice(line.price)}</s>}
            <span className="font-medium text-black">{formatPrice(line.unit)}</span>
          </p>
          <p className="text-[20px] font-bold">{formatPrice(line.total)}</p>
          {line.percent > 0 ? (
            <p className="text-sm font-medium text-green-700">
              {t("cart.discount.applied", { percent: line.percent })} · {t("cart.discount.saved", { amount: formatPrice(line.savings) })}
            </p>
          ) : (
            line.nextTier && (
              <p className="text-sm text-gray-600">{t("cart.discount.hint", { percent: line.nextTier.percent, qty: line.nextTier.minQty })}</p>
            )
          )}
        </div>
        )}
      </div>
    </li>
  );
}
