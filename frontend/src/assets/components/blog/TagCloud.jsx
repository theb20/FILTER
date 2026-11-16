import { Link } from "react-router-dom";
import { BLOG_TAGS } from "../../data/blog.js";
import { buildBlogQuery } from "../../lib/blogFilters.js";
import { useI18n } from "../../i18n/i18nContext.js";

// « Explorer par sujet » : étiquettes cliquables (filtre ?tag= de la page).
export default function TagCloud({ params }) {
  const { t } = useI18n();
  return (
    <section aria-labelledby="tags-title" className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-5 shadow-card">
      <h2 id="tags-title" className="text-[22px] font-bold">{t("blog.tags")}</h2>
      <ul className="mt-3 flex flex-wrap gap-2">
        {BLOG_TAGS.map((tag) => {
          const active = params.tag === tag.slug;
          return (
            <li key={tag.slug}>
              <Link
                to={`/blog${buildBlogQuery(params, { tag: active ? null : tag.slug })}`}
                aria-current={active ? "true" : undefined}
                className={`inline-block rounded-full border px-4 py-1.5 text-sm font-medium transition-colors ${active ? "border-action bg-action text-black" : "border-gray-300 hover:border-black"}`}
              >
                #{tag.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
