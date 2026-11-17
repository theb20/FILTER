import { Link } from "react-router-dom";
import { ArrowLeft, ArrowRight } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";
import { postPath } from "./format.js";

// Navigation article précédent / suivant.
export default function PostNav({ older, newer }) {
  const { t } = useI18n();
  if (!older && !newer) return null;
  const box = "block h-full rounded-[8px] border border-gray-200 p-4 transition-colors hover:border-black";
  return (
    <nav aria-label={t("blog.postNav")} className="grid gap-3 sm:grid-cols-2">
      {older ? (
        <Link to={postPath(older.slug)} className={box}>
          <span className="flex items-center gap-1.5 text-sm text-gray-600"><ArrowLeft size={14} aria-hidden="true" /> {t("blog.prevPost")}</span>
          <span className="mt-1 block font-bold leading-snug">{older.title}</span>
        </Link>
      ) : <span />}
      {newer ? (
        <Link to={postPath(newer.slug)} className={`${box} sm:text-right`}>
          <span className="flex items-center gap-1.5 text-sm text-gray-600 sm:justify-end">{t("blog.nextPost")} <ArrowRight size={14} aria-hidden="true" /></span>
          <span className="mt-1 block font-bold leading-snug">{newer.title}</span>
        </Link>
      ) : <span />}
    </nav>
  );
}
