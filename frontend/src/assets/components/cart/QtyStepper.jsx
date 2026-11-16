import { useState } from "react";
import { Minus, Plus } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";

const btn =
  "flex h-9 w-9 items-center justify-center text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:text-gray-300 disabled:hover:bg-transparent";

// Sélecteur de quantité : boutons −/+ et saisie directe. Affichage seul : la borne (min 1, max = stock)
// est appliquée par le parent via onChange (cartMath.clampQty).
export default function QtyStepper({ value, max, name, onChange }) {
  const { t } = useI18n();
  const [draft, setDraft] = useState(null); // texte en cours de saisie
  const atMax = value >= max;

  const commit = () => {
    if (draft !== null) {
      const n = parseInt(draft, 10);
      onChange(Number.isNaN(n) ? value : n);
      setDraft(null);
    }
  };

  return (
    <div>
      <div className="inline-flex items-center rounded-[3px] border border-gray-300 bg-white">
        <button type="button" className={btn} disabled={value <= 1} onClick={() => onChange(value - 1)} aria-label={t("cart.qty.dec", { name })}>
          <Minus size={16} aria-hidden="true" />
        </button>
        <input
          type="text"
          inputMode="numeric"
          pattern="[0-9]*"
          value={draft ?? value}
          onChange={(e) => setDraft(e.target.value.replace(/\D/g, ""))}
          onBlur={commit}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); commit(); } }}
          aria-label={t("cart.qtyFor", { name })}
          className="h-9 w-12 border-x border-gray-300 text-center outline-none focus:bg-gray-50"
        />
        <button type="button" className={btn} disabled={atMax} onClick={() => onChange(value + 1)} aria-label={t("cart.qty.inc", { name })}>
          <Plus size={16} aria-hidden="true" />
        </button>
      </div>
      {atMax && <p className="mt-1 text-xs text-gray-600">{t("cart.qty.max", { n: max })}</p>}
    </div>
  );
}
