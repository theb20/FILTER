import { CircleHelp } from "lucide-react";
import { useGuide } from "./guideContext.js";
import { useI18n } from "../i18n/i18nContext.js";

/** « Découvrir cette section » : visible seulement si le guide est activé et que la page a une visite. */
export default function GuideButton({ className = "" }) {
  const { t } = useI18n();
  const { prefs, current, active, start } = useGuide();
  if (!prefs.enabled || !current) return null;
  return (
    <button
      type="button"
      data-guide="guide-help"
      onClick={() => start(current.id)}
      disabled={Boolean(active)}
      aria-label={t("guide.ui.discover")}
      title={t("guide.ui.discover")}
      className={`flex items-center gap-2 text-sm text-white transition-colors hover:text-action disabled:opacity-50 ${className}`}
    >
      <span className="relative">
        <CircleHelp size={20} aria-hidden="true" />
        {!prefs.seen.includes(current.id) && <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-action ring-2 ring-night" aria-hidden="true" />}
      </span>
      <span className="hidden xl:inline">{t("guide.ui.discover")}</span>
    </button>
  );
}
