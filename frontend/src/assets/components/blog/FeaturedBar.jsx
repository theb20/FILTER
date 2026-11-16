import { Link } from "react-router-dom";
import { BLOG_ARTICLES, BLOG_HEADLINE } from "../../data/blog.js";
import { useI18n } from "../../i18n/i18nContext.js";
import { postPath } from "./format.js";

// Fine bande bleu nuit au-dessus de l'en-tête : « À la une » + titre cliquable.
export default function FeaturedBar() {
  const { t } = useI18n();
  const a = BLOG_ARTICLES.find((x) => x.slug === BLOG_HEADLINE);
  if (!a) return null;
  return (
    <div className="bg-night text-white">
      <div className="mx-auto flex max-w-[1500px] items-center gap-3 px-4 py-1.5 text-sm lg:px-6">
        <span className="shrink-0 rounded-full bg-action px-3 py-0.5 text-xs font-bold uppercase tracking-wide text-black">{t("blog.featured")}</span>
        <Link to={postPath(a.slug)} className="truncate font-medium hover:text-action hover:underline">{a.title}</Link>
      </div>
    </div>
  );
}
