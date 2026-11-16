import { Link } from "react-router-dom";
import { useI18n } from "../../i18n/i18nContext.js";
import { postPath } from "./format.js";

// « Sujets du moment » : 4 cartes compactes, vignette ronde cerclée de jaune.
export default function TopicsRow({ articles }) {
  const { t } = useI18n();
  return (
    <section aria-labelledby="topics-title" className="mx-2.5 mt-2.5">
      <h2 id="topics-title" className="sr-only">{t("blog.topics")}</h2>
      <ul className="grid gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
        {articles.map((a) => (
          <li key={a.slug} className="relative flex items-center gap-3 rounded-[8px] bg-white p-3 shadow-card transition-shadow hover:shadow-md">
            <img src={a.image} alt="" loading="lazy" decoding="async" className="h-14 w-14 shrink-0 rounded-full object-cover ring-2 ring-action ring-offset-2" />
            <div className="min-w-0">
              <p className="text-xs font-bold uppercase tracking-wide text-teal">{t(`blog.cat.${a.category}`)}</p>
              <Link to={postPath(a.slug)} className="line-clamp-2 text-[15px] font-semibold leading-snug after:absolute after:inset-0 hover:underline">{a.title}</Link>
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}
