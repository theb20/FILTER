import { Link } from "react-router-dom";
import { Search } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";

// Bandeau outil : bande jaune pleine largeur vers la recherche du catalogue.
export default function ToolBanner() {
  const { t } = useI18n();
  return (
    <section aria-labelledby="tool-title" className="mt-2.5 bg-action">
      <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-4 px-6 py-6">
        <div>
          <h2 id="tool-title" className="text-[26px] font-bold leading-tight text-black">{t("blog.tool.title")}</h2>
          <p className="text-black/80">{t("blog.tool.text")}</p>
        </div>
        <Link to="/category" className="inline-flex items-center gap-2 rounded-[4px] bg-night px-6 py-3 font-bold text-white transition-colors hover:bg-navy">
          <Search size={18} aria-hidden="true" /> {t("blog.tool.cta")}
        </Link>
      </div>
    </section>
  );
}
