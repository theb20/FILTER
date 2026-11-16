import { Link } from "react-router-dom";
import ArticleCard from "./ArticleCard.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

// Guides d'entretien : 4 cartes (type « Pas à pas » ou « Check-list »).
export default function GuidesRow({ articles }) {
  const { t } = useI18n();
  return (
    <section aria-labelledby="guides-title" className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-5 shadow-card">
      <div className="flex flex-wrap items-end justify-between gap-2 border-b border-gray-200 pb-2">
        <h2 id="guides-title" className="-mb-[9px] border-b-[3px] border-action pb-2 text-[22px] font-bold uppercase tracking-wide">{t("blog.guides")}</h2>
        <Link to="/blog?cat=guides" className="text-sm font-medium text-link hover:underline">{t("blog.guides.all")} <span aria-hidden="true">→</span></Link>
      </div>
      <ul className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {articles.map((a) => (
          <li key={a.slug}><ArticleCard article={a} variant="compact" guideLabel={t(`blog.guide.${a.guide ?? "steps"}`)} /></li>
        ))}
      </ul>
    </section>
  );
}
