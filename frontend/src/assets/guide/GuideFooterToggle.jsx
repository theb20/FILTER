import { useGuide } from "./guideContext.js";
import { useI18n } from "../i18n/i18nContext.js";

/** Interrupteur discret en pied de page : permet à un visiteur sans compte de réactiver le guide après l'avoir désactivé. */
export default function GuideFooterToggle() {
  const { t } = useI18n();
  const { prefs, setEnabled } = useGuide();
  return (
    <button type="button" role="switch" aria-checked={prefs.enabled} onClick={() => setEnabled(!prefs.enabled)} className="mt-1 text-xs text-white/70 underline-offset-2 hover:text-white hover:underline">
      {prefs.enabled ? t("guide.footer.on") : t("guide.footer.off")}
    </button>
  );
}
