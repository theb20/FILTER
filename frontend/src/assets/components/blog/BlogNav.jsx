import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { BLOG_CATEGORIES } from "../../data/blogMocks.js";
import { buildBlogQuery } from "../../lib/blogFilters.js";
import { useI18n } from "../../i18n/i18nContext.js";

// Barre blanche : titre, catégories en onglets (soulignement jaune) et retour à la boutique.
export default function BlogNav({ params, asH1 = false }) {
  const { t } = useI18n();
  const tabs = [null, ...BLOG_CATEGORIES];
  const Title = asH1 ? "h1" : "p";
  return (
    <div className="mx-2.5 mt-2.5 rounded-[4px] bg-white shadow-card">
      <div className="flex flex-wrap items-center gap-x-6 gap-y-1 px-4 pt-3 lg:px-6">
        <Title className="text-[28px] font-bold leading-none">{t("blog.title")}</Title>
        <Link to="/" className="ml-auto inline-flex items-center gap-1.5 text-sm font-medium text-link hover:underline">
          <ArrowLeft size={16} aria-hidden="true" /> {t("blog.back")}
        </Link>
      </div>
      <nav aria-label={t("blog.nav")} className="mt-1 overflow-x-auto px-2 lg:px-4">
        <ul className="flex min-w-max">
          {tabs.map((c) => {
            const active = (params.cat ?? null) === c;
            return (
              <li key={c ?? "all"}>
                <Link
                  to={`/blog${buildBlogQuery(params, { cat: c })}`}
                  aria-current={active ? "page" : undefined}
                  className={`block whitespace-nowrap border-b-[3px] px-3 py-3 text-[15px] font-medium transition-colors ${active ? "border-action text-black" : "border-transparent text-gray-600 hover:text-black"}`}
                >
                  {t(c ? `blog.cat.${c}` : "blog.cat.all")}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
}
