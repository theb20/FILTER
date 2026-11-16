import { useId, useState } from "react";
import { CheckCircle2, XCircle, AlertTriangle } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";

// Champ code promo. `validate(code)` renvoie { status, promo? } (lib/cartTotals.validatePromo).
// `appliedCode` : code actuellement appliqué (depuis le store), ou null.
export default function PromoCode({ appliedCode, appliedResult, validate, onApply, onClear }) {
  const { t, formatPrice } = useI18n();
  const id = useId();
  const [draft, setDraft] = useState("");
  const [error, setError] = useState(null); // "invalid" | "expired" | "minimum" | "empty"
  const [minimum, setMinimum] = useState(0);

  const submit = (e) => {
    e.preventDefault();
    if (!draft.trim()) { setError("empty"); return; }
    const res = validate(draft);
    if (res.status === "valid") {
      onApply(draft);
      setDraft("");
      setError(null);
    } else {
      setError(res.status);
      setMinimum(res.promo?.minSubtotal ?? 0);
    }
  };

  const applied = appliedCode && appliedResult;
  return (
    <div>
      <form onSubmit={submit} noValidate>
        <label htmlFor={id} className="block text-sm font-medium">{t("cart.promo.label")}</label>
        <div className="mt-1 flex">
          <input
            id={id}
            value={draft}
            onChange={(e) => { setDraft(e.target.value); setError(null); }}
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            aria-invalid={Boolean(error)}
            aria-describedby={`${id}-msg`}
            placeholder={t("cart.promo.placeholder")}
            className={`min-w-0 flex-1 rounded-l-[3px] border px-3 py-2 uppercase placeholder:normal-case ${error ? "border-red-500" : "border-gray-300"}`}
          />
          <button type="submit" className="rounded-r-[3px] bg-slate px-4 font-medium text-white hover:bg-night">{t("cart.promo.apply")}</button>
        </div>
      </form>

      <div id={`${id}-msg`} aria-live="polite" className="mt-1.5 text-sm">
        {error && (
          <p className="flex items-start gap-1.5 text-red-700">
            {error === "expired" ? <AlertTriangle size={16} className="mt-0.5 shrink-0" aria-hidden="true" /> : <XCircle size={16} className="mt-0.5 shrink-0" aria-hidden="true" />}
            {error === "minimum" ? t("cart.promo.minimum", { amount: formatPrice(minimum) }) : t(`cart.promo.${error}`)}
          </p>
        )}
        {applied?.status === "valid" && (
          <p className="flex flex-wrap items-center gap-x-2 text-green-700">
            <CheckCircle2 size={16} aria-hidden="true" />
            {t("cart.promo.valid", { code: appliedCode })}
            <button type="button" onClick={onClear} className="text-gray-700 underline hover:text-black">{t("cart.promo.remove")}</button>
          </p>
        )}
        {applied && applied.status !== "valid" && (
          <p className="flex flex-wrap items-center gap-x-2 text-red-700">
            <AlertTriangle size={16} aria-hidden="true" />
            {applied.status === "minimum"
              ? t("cart.promo.minimum", { amount: formatPrice(applied.promo.minSubtotal) })
              : t(`cart.promo.${applied.status}`)}
            <button type="button" onClick={onClear} className="text-gray-700 underline hover:text-black">{t("cart.promo.remove")}</button>
          </p>
        )}
      </div>
    </div>
  );
}
