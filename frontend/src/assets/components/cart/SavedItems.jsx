import { Link } from "react-router-dom";
import LineArt from "../LineArt.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

// « Enregistrés pour plus tard » (version simple ; enrichie à l'étape 3).
export default function SavedItems({ lines, onMoveToCart, onRemove }) {
  const { t, formatPrice } = useI18n();
  if (!lines.length) return null;
  return (
    <section className="rounded-[4px] bg-white p-6 shadow-card" aria-labelledby="saved-title">
      <h2 id="saved-title" className="text-[24px] font-semibold">{t("cart.saved.title")} ({lines.length})</h2>
      <ul className="mt-2 divide-y divide-gray-200">
        {lines.map((l) => {
          const name = l.name;
          return (
            <li key={l.id} className="flex flex-wrap items-center gap-4 py-3">
              <div className="flex h-14 w-14 shrink-0 items-center justify-center bg-gray-50"><LineArt stroke="#9ca3af" className="h-10" /></div>
              <div className="min-w-0 flex-1">
                <Link to={`/card?id=${l.id}`} className="font-semibold text-link hover:underline">{name}</Link>
                <p className="text-sm text-gray-600">{t("list.ref")} {l.reference} · {l.onQuote ? t("price.onQuote") : formatPrice(l.price)}</p>
              </div>
              <div className="flex gap-4 text-sm">
                <button type="button" onClick={() => onMoveToCart(l.id)} className="font-medium text-link hover:underline">{t("cart.saved.move")}</button>
                <button type="button" onClick={() => onRemove(l.id)} aria-label={t("cart.remove", { name })} className="text-gray-700 hover:text-red-700 hover:underline">{t("cart.removeShort")}</button>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
