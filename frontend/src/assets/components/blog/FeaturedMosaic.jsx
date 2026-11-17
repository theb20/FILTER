import ArticleCard from "./ArticleCard.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

// Mosaïque à la une : 1 grande carte + 2 petites au-dessus d'1 carte large (1 colonne sur mobile).
export default function FeaturedMosaic({ articles }) {
  const { t } = useI18n();
  const [big, s1, s2, wide] = articles;
  if (!big) return null;
  return (
    <section aria-labelledby="mosaic-title" className="mx-2.5 mt-2.5">
      <h2 id="mosaic-title" className="sr-only">{t("blog.featured")}</h2>
      <div className="grid gap-2.5 lg:grid-cols-[1.2fr_1fr]">
        <ArticleCard article={big} variant="overlay" size="lg" />
        <div className="grid gap-2.5 sm:grid-cols-2">
          {s1 && <ArticleCard article={s1} variant="overlay" size="md" />}
          {s2 && <ArticleCard article={s2} variant="overlay" size="md" />}
          {wide && <div className="sm:col-span-2"><ArticleCard article={wide} variant="overlay" size="wide" /></div>}
        </div>
      </div>
    </section>
  );
}
