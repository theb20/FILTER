import { useId } from "react";
import { useI18n } from "../../i18n/i18nContext.js";

// Choix de la zone + délai estimé. `zones` et `window` viennent de la page (calculs purs).
export default function DeliveryEstimate({ zones, zoneId, window, onChange }) {
  const { t, locale } = useI18n();
  const id = useId();
  const fmt = new Intl.DateTimeFormat(locale, { weekday: "short", day: "numeric", month: "short" });
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium">{t("cart.zone.label")}</label>
      <select
        id={id}
        value={zoneId}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 w-full rounded-[3px] border border-gray-300 bg-white px-3 py-2"
      >
        {zones.map((z) => <option key={z.id} value={z.id}>{t(`cart.zone.${z.id}`)}</option>)}
      </select>
      <p className="mt-1.5 text-sm text-gray-700" aria-live="polite">
        {t("cart.zone.eta", { from: fmt.format(window.from), to: fmt.format(window.to) })}
      </p>
    </div>
  );
}
