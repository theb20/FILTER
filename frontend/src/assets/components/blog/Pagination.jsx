import { Link } from "react-router-dom";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { buildBlogQuery } from "../../lib/blogFilters.js";
import { useI18n } from "../../i18n/i18nContext.js";

export default function Pagination({ params, page, pages }) {
  const { t } = useI18n();
  if (pages <= 1) return null;
  const to = (p) => `/blog${buildBlogQuery(params, { page: p })}`;
  const base = "flex h-9 min-w-9 items-center justify-center rounded-[4px] border px-2 text-sm font-medium";
  return (
    <nav aria-label={t("blog.pagination")} className="mt-6">
      <ul className="flex flex-wrap items-center justify-center gap-1.5">
        <li>{page > 1 ? <Link to={to(page - 1)} aria-label={t("blog.prev")} className={`${base} border-gray-300 hover:bg-gray-50`}><ChevronLeft size={16} aria-hidden="true" /></Link> : <span aria-hidden="true" className={`${base} border-gray-200 text-gray-300`}><ChevronLeft size={16} /></span>}</li>
        {Array.from({ length: pages }, (_, i) => i + 1).map((p) => (
          <li key={p}>
            <Link to={to(p)} aria-label={t("blog.page", { n: p })} aria-current={p === page ? "page" : undefined} className={`${base} ${p === page ? "border-action bg-action text-black" : "border-gray-300 hover:bg-gray-50"}`}>{p}</Link>
          </li>
        ))}
        <li>{page < pages ? <Link to={to(page + 1)} aria-label={t("blog.next")} className={`${base} border-gray-300 hover:bg-gray-50`}><ChevronRight size={16} aria-hidden="true" /></Link> : <span aria-hidden="true" className={`${base} border-gray-200 text-gray-300`}><ChevronRight size={16} /></span>}</li>
      </ul>
    </nav>
  );
}
