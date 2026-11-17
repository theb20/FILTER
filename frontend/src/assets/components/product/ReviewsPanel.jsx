import { useState } from "react";
import { BadgeCheck } from "lucide-react";
import Stars from "../Stars.jsx";
const REVIEWS_PREVIEW = 3; // avis affichés avant « Voir tous les avis »
import { ratingStats } from "../../lib/productUtils.js";
import { useI18n } from "../../i18n/i18nContext.js";

// Avis clients RÉELS uniquement : note moyenne, répartition, 3 avis puis « Voir tous les avis ».
export default function ReviewsPanel({ reviews = [] }) {
  const REVIEWS = reviews;
  const { t, locale } = useI18n();
  const [all, setAll] = useState(false);
  const stats = ratingStats(REVIEWS);
  const shown = all ? REVIEWS : REVIEWS.slice(0, REVIEWS_PREVIEW);
  const date = (iso) => new Intl.DateTimeFormat(locale, { dateStyle: "medium" }).format(new Date(iso));

  return (
    <section id="reviews" data-guide="product-reviews" aria-labelledby="pd-reviews-title" className="rounded-[8px] bg-white p-5 shadow-card">
      <h2 id="pd-reviews-title" className="text-[20px] font-bold">{t("pd.reviews")}</h2>

      {REVIEWS.length === 0 ? (
        <p className="mt-3 text-gray-700">{t("pd.reviews.none")}</p>
      ) : (<>
      <div className="mt-3 flex items-center gap-3">
        <p className="text-[40px] font-bold leading-none">{stats.average.toLocaleString(locale)}</p>
        <div>
          <Stars value={stats.average} />
          <p className="text-sm text-gray-600">{t("pd.reviews.based", { n: stats.count })}</p>
        </div>
      </div>

      <ul className="mt-3 space-y-1" aria-label={t("pd.reviews.distribution")}>
        {stats.distribution.map((d) => (
          <li key={d.stars} className="flex items-center gap-2 text-sm">
            <span className="w-8 shrink-0">{d.stars} ★</span>
            <span className="h-2 flex-1 overflow-hidden rounded-full bg-gray-200" aria-hidden="true"><span className="block h-full rounded-full bg-action" style={{ width: `${d.percent}%` }} /></span>
            <span className="w-5 shrink-0 text-right text-gray-600">{d.count}</span>
          </li>
        ))}
      </ul>

      <ul className="mt-4 divide-y divide-gray-200">
        {shown.map((r) => (
          <li key={r.id} className="py-3">
            <div className="flex items-center gap-2"><Stars value={r.rating} size={14} /><p className="font-bold">{r.title}</p></div>
            <p className="mt-1 text-sm text-gray-700">{r.text}</p>
            <p className="mt-1 flex flex-wrap items-center gap-x-2 text-xs text-gray-600">
              <span>{[r.author, r.company].filter(Boolean).join(", ")}</span><span aria-hidden="true">·</span><time dateTime={r.date}>{date(r.date)}</time>
              {r.verified && <span className="inline-flex items-center gap-1 text-green-700"><BadgeCheck size={12} aria-hidden="true" />{t("pd.reviews.verified")}</span>}
            </p>
          </li>
        ))}
      </ul>

      {REVIEWS.length > REVIEWS_PREVIEW && (
        <button type="button" onClick={() => setAll((v) => !v)} aria-expanded={all} className="mt-2 w-full rounded-[4px] border border-gray-300 py-2 font-medium text-link hover:bg-gray-50">
          {all ? t("pd.reviews.less") : t("pd.reviews.all")}
        </button>
      )}
      </>)}
    </section>
  );
}
