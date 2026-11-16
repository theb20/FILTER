import { Star } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";

// Étoiles de notation (affichage seul, accessibles via aria-label).
export default function Stars({ value, size = 16 }) {
  const { t } = useI18n();
  return (
    <span role="img" aria-label={t("pd.stars", { n: value })} className="inline-flex gap-0.5">
      {[1, 2, 3, 4, 5].map((i) => (
        <Star key={i} size={size} aria-hidden="true" className={i <= Math.round(value) ? "fill-action text-action" : "text-gray-300"} />
      ))}
    </span>
  );
}
