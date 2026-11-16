import { Link } from "react-router-dom";
import { Clock } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";
import { formatDate, postPath } from "./format.js";

const chip = "inline-block rounded-[3px] bg-action px-2 py-0.5 text-xs font-bold uppercase tracking-wide text-black";

function Meta({ article, light = false, showAuthor = true }) {
  const { t, locale } = useI18n();
  return (
    <p className={`flex flex-wrap items-center gap-x-2 text-xs ${light ? "text-white/90" : "text-gray-600"}`}>
      {showAuthor && <span>{article.author}</span>}
      {showAuthor && <span aria-hidden="true">·</span>}
      <time dateTime={article.date}>{formatDate(locale, article.date)}</time>
      <span aria-hidden="true">·</span>
      <span className="inline-flex items-center gap-1"><Clock size={12} aria-hidden="true" />{t("blog.readMin", { n: article.readMin })}</span>
    </p>
  );
}

// Lien étiré : toute la carte est cliquable, sans imbriquer de lien dans un lien.
const Title = ({ article, className }) => (
  <h3 className={className}>
    <Link to={postPath(article.slug)} className="after:absolute after:inset-0 hover:underline focus-visible:outline-none focus-visible:after:ring-2 focus-visible:after:ring-action">
      {article.title}
    </Link>
  </h3>
);

const alt = (t, a) => t("blog.imageAlt", { title: a.title });

// size : "lg" | "md" | "wide" (variant overlay)
const OVERLAY = {
  lg: { box: "min-h-[360px] lg:min-h-full", title: "text-[26px] sm:text-[30px]", excerpt: true },
  md: { box: "min-h-[240px]", title: "text-[18px]", excerpt: false },
  wide: { box: "min-h-[220px]", title: "text-[22px]", excerpt: false },
};

/**
 * Carte d'article. variant : "overlay" (texte blanc sur image assombrie), "vertical",
 * "list" (vignette à gauche) ou "compact" (image + type, pour les guides).
 */
export default function ArticleCard({ article: a, variant = "vertical", size = "md", guideLabel }) {
  const { t } = useI18n();
  const category = t(`blog.cat.${a.category}`);

  if (variant === "overlay") {
    const o = OVERLAY[size];
    return (
      <article className={`group relative flex overflow-hidden rounded-[8px] bg-night text-white ${o.box}`}>
        <img src={a.image} alt={alt(t, a)} loading="lazy" decoding="async" className="absolute inset-0 h-full w-full object-cover" />
        <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/50 to-black/15" aria-hidden="true" />
        <div className="relative mt-auto w-full p-4 sm:p-5">
          <span className={chip}>{category}</span>
          <Title article={a} className={`mt-2 font-bold leading-tight ${o.title}`} />
          {o.excerpt && <p className="mt-2 max-w-xl text-[15px] text-white/90">{a.excerpt}</p>}
          <div className="mt-2"><Meta article={a} light /></div>
        </div>
      </article>
    );
  }

  if (variant === "list") {
    return (
      <article className="relative flex gap-4 py-4">
        <img src={a.image} alt={alt(t, a)} loading="lazy" decoding="async" className="h-20 w-20 shrink-0 rounded-[6px] object-cover sm:h-24 sm:w-24" />
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-wide text-teal">{category}</p>
          <Title article={a} className="mt-0.5 text-[17px] font-semibold leading-snug" />
          <div className="mt-1"><Meta article={a} showAuthor={false} /></div>
        </div>
      </article>
    );
  }

  if (variant === "compact") {
    return (
      <article className="group relative flex h-full flex-col overflow-hidden rounded-[8px] bg-white shadow-card">
        <img src={a.image} alt={alt(t, a)} loading="lazy" decoding="async" className="h-36 w-full object-cover" />
        <div className="flex flex-1 flex-col p-4">
          <span className={`${chip} self-start`}>{guideLabel}</span>
          <Title article={a} className="mt-2 text-[17px] font-semibold leading-snug" />
          <p className="mt-auto pt-2 text-xs text-gray-600"><time dateTime={a.date}>{formatDateShort(a.date)}</time></p>
        </div>
      </article>
    );
  }

  // vertical
  return (
    <article className="group relative flex h-full flex-col overflow-hidden rounded-[8px] bg-white shadow-card">
      <img src={a.image} alt={alt(t, a)} loading="lazy" decoding="async" className="h-56 w-full object-cover sm:h-72" />
      <div className="flex flex-1 flex-col p-5">
        <span className={`${chip} self-start`}>{category}</span>
        <Title article={a} className="mt-2 text-[24px] font-bold leading-tight" />
        <p className="mt-2 text-[15px] text-gray-700">{a.excerpt}</p>
        <div className="mt-auto pt-3"><Meta article={a} /></div>
      </div>
    </article>
  );
}

// Date courte pour les cartes compactes (la langue vient du document).
function formatDateShort(iso) {
  return new Intl.DateTimeFormat(document.documentElement.lang || "fr", { day: "numeric", month: "short", year: "numeric" }).format(new Date(`${iso}T12:00:00`));
}
