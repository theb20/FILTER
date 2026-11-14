import { useState } from "react";
import { Info, X } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";
import { SITE } from "../config/site.js";

// Affiché seulement si un site régional est configuré (SITE.regionalSite).
export default function GeoBanner() {
  const { t } = useI18n();
  const [open, setOpen] = useState(true);
  const regional = SITE.regionalSite;
  if (!open || !regional) return null;

  return (
    <div className="bg-info" role="region" aria-label={t("geo.region")}>
      <div className="mx-auto flex max-w-[1500px] items-center gap-3 px-4 py-2 text-sm lg:px-6">
        <Info size={18} aria-hidden="true" className="shrink-0" />
        <p className="flex-1">{t("geo.message", { country: regional.country })}</p>
        <a href={regional.url} className="rounded-[3px] bg-teal px-3 py-1.5 font-medium text-white hover:opacity-90">
          {t("geo.yes")}
        </a>
        <button type="button" aria-label={t("geo.dismiss")} onClick={() => setOpen(false)}>
          <X size={18} />
        </button>
      </div>
    </div>
  );
}
