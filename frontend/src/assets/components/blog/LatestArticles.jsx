import { Link } from "react-router-dom";
import ArticleCard from "./ArticleCard.jsx";
import BlogState from "./BlogState.jsx";
import Pagination from "./Pagination.jsx";
import { BLOG_KINDS } from "../../data/blog.js";
import { buildBlogQuery } from "../../lib/blogFilters.js";
import { useI18n } from "../../i18n/i18nContext.js";

// Colonne principale : pastilles de type, 1 grande carte + liste, « Voir tous », pagination.
export default function LatestArticles({ params, pageData }) {
  const { t } = useI18n();
  const { items, page, pages } = pageData;
  const [first, ...rest] = items;
  const filtered = Boolean(params.cat || params.kind || params.tag);
  const kinds = [null, ...BLOG_KINDS];

  return (
    <section aria-labelledby="latest-title">
      <div className="flex flex-wrap items-end justify-between gap-3 border-b border-gray-200 pb-2">
        <h2 id="latest-title" className="-mb-[9px] border-b-[3px] border-action pb-2 text-[22px] font-bold uppercase tracking-wide">{t("blog.latest")}</h2>
        <ul aria-label={t("blog.filterKind")} className="flex flex-wrap gap-1.5">
          {kinds.map((k) => {
            const active = (params.kind ?? null) === k;
            return (
              <li key={k ?? "all"}>
                <Link
                  to={`/blog${buildBlogQuery(params, { kind: k })}`}
                  aria-current={active ? "true" : undefined}
                  className={`rounded-full px-3 py-1 text-sm font-medium transition-colors ${active ? "bg-action text-black" : "bg-gray-100 text-gray-700 hover:bg-gray-200"}`}
                >
                  {t(k ? `blog.kind.${k}` : "blog.kind.all")}
                </Link>
              </li>
            );
          })}
        </ul>
      </div>

      {params.tag && (
        <p className="mt-3 text-sm text-gray-700">
          {t("blog.tagActive", { tag: params.tagLabel })}{" "}
          <Link to={`/blog${buildBlogQuery(params, { tag: null })}`} className="text-link underline">{t("blog.tagClear")}</Link>
        </p>
      )}

      {!first ? (
        <div className="mt-4"><BlogState kind="empty" /></div>
      ) : (
        <div className="mt-4 grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)]">
          <ArticleCard article={first} variant="vertical" />
          <div className="divide-y divide-gray-200">
            {rest.map((a) => <ArticleCard key={a.slug} article={a} variant="list" />)}
          </div>
        </div>
      )}

      <div className="mt-4 text-center">
        {filtered && (
          <Link to="/blog" className="inline-flex rounded-[3px] border border-gray-300 bg-white px-5 py-2.5 font-medium hover:bg-gray-50">{t("blog.viewAll")}</Link>
        )}
      </div>
      <Pagination params={params} page={page} pages={pages} />
    </section>
  );
}
