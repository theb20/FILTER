import { Link } from "react-router-dom";
import { HelpCircle } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";

export default function HelpBubble() {
  const { t } = useI18n();
  return (
    <Link
      to="/contact"
      aria-label={t("help.label")}
      className="fixed bottom-4 right-4 z-[900] inline-flex items-center gap-2 rounded-full bg-navy px-4 py-3 text-white shadow-lg hover:bg-night"
    >
      <HelpCircle size={22} aria-hidden="true" />
      <span className="hidden text-sm font-medium sm:inline">{t("help.label")}</span>
    </Link>
  );
}
