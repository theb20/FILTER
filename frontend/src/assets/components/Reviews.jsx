import { Link } from "react-router-dom";
import { Quote, Star } from "lucide-react";
import Todo from "./Todo.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { withPlaceholders } from "../config/placeholderReviews.js";
import { fetchSiteReviews } from "../lib/content.js";
import { useAsync } from "../lib/catalog.js";
import { useSite } from "../store/siteStore.js";

function Stars({ value }) {
  return (
    <span className="flex gap-0.5 text-action" aria-hidden="true">
      {[1, 2, 3, 4, 5].map((n) => (
        <Star key={n} size={18} className={n <= value ? "fill-current" : "text-gray-300"} />
      ))}
    </span>
  );
}

function ReviewCard({ review }) {
  const { t } = useI18n();
  return (
    <figure className="relative flex h-full min-h-[240px] flex-col rounded-[4px] border border-gray-200 bg-white p-6 shadow-card">
      <Quote size={28} className="absolute right-5 top-5 text-link/15" aria-hidden="true" />
      {review ? (
        <>
          <Stars value={review.rating} />
          <span className="sr-only">{t("home.reviews.rated", { n: review.rating })}</span>
          <blockquote className="mt-4 flex-1 text-[17px] leading-relaxed text-slate">“{review.text}”</blockquote>
          <figcaption className="mt-6 flex items-center gap-3 border-t border-gray-100 pt-4">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-navy font-bold text-white" aria-hidden="true">
              {review.author.charAt(0)}
            </span>
            <span className="text-sm">
              <strong className="block">{review.author}</strong>
              <span className="text-gray-600">{review.company ?? ""}</span>
            </span>
          </figcaption>
        </>
      ) : (
        <>
          <Stars value={0} />
          <blockquote className="mt-4 flex-1 text-[17px] leading-relaxed">
            <Todo>{t("todo.reviewText")}</Todo>
          </blockquote>
          <figcaption className="mt-6 flex items-center gap-3 border-t border-gray-100 pt-4">
            <span className="h-10 w-10 shrink-0 rounded-full border border-dashed border-gray-300" aria-hidden="true" />
            <span className="text-sm">
              <strong className="block"><Todo>{t("todo.customerName")}</Todo></strong>
              <Todo>{t("todo.roleCompany")}</Todo>
            </span>
          </figcaption>
        </>
      )}
    </figure>
  );
}

export default function Reviews() {
  const { t, lang } = useI18n();
  const SITE = useSite();
  const data = useAsync(fetchSiteReviews, [lang]).data;
  const REVIEWS = data?.items ?? [];
  const items = withPlaceholders(REVIEWS, lang); // 3 cartes : les avis fictifs cèdent la place aux vrais, un par un
  // Note et nombre : ceux des avis publiés dès qu'il y en a assez ; sinon ceux des cartes affichées (le tableau n'est jamais vide).
  const enough = (data?.count ?? 0) >= items.length;
  const count = enough ? data.count : items.length;
  const average = enough ? data.average : Math.round((items.reduce((sum, r) => sum + r.rating, 0) / items.length) * 10) / 10;

  return (
    <section className="mx-2.5 rounded-[4px] bg-white px-4 py-12 shadow-card md:px-10" aria-labelledby="reviews-title">
      <div className="grid gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,2.2fr)] lg:items-start">
        <div>
          <p className="font-medium text-link">{t("home.reviews.eyebrow")}</p>
          <h2 id="reviews-title" className="mt-1 text-[37px] font-bold leading-tight">{t("home.reviews.title")}</h2>
          <p className="mt-3 max-w-md">{t("home.reviews.text", { name: SITE.name })}</p>

          <div className="mt-6 flex items-center gap-4">
            {average ? (
              <span className="text-[48px] font-bold leading-none">{average.toLocaleString(lang, { minimumFractionDigits: 1 })}</span>
            ) : (
              <span className="text-sm"><Todo>{t("todo.ratingValue")}</Todo></span>
            )}
            <div>
              <Stars value={average ? Math.round(average) : 0} />
              <p className="mt-1 text-sm text-gray-600">
                {count ? t("home.reviews.count", { n: count }) : <Todo>{t("todo.reviewCount")}</Todo>}
              </p>
            </div>
          </div>

          {SITE.reviewsUrl ? (
            <a
              href={SITE.reviewsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-6 inline-flex items-center gap-2 rounded-[3px] bg-action px-5 py-2.5 font-bold text-black transition-colors hover:bg-action-dark"
            >
              {t("home.reviews.button")} <span aria-hidden="true">→</span>
            </a>
          ) : (
            <Link to="/contact" className="mt-6 inline-flex items-center gap-2 rounded-[3px] bg-action px-5 py-2.5 font-bold text-black transition-colors hover:bg-action-dark">
              {t("home.reviews.give")} <span aria-hidden="true">→</span>
            </Link>
          )}
        </div>

        <ul className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {items.map((review, i) => (
            <li key={review.id ?? i}>
              <ReviewCard review={review} />
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
