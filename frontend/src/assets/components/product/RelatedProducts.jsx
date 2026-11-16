import ProductCard from "../ProductCard.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

// Produits associés : carrousel qui défile au doigt / à la molette (scroll-snap), sans bibliothèque.
export default function RelatedProducts({ products }) {
  const { t } = useI18n();
  if (!products.length) return null;
  return (
    <section aria-labelledby="pd-related-title">
      <h2 id="pd-related-title" className="text-[22px] font-bold">{t("pd.related")}</h2>
      <ul className="mt-3 flex snap-x snap-mandatory gap-2.5 overflow-x-auto pb-2" tabIndex={0} aria-label={t("pd.related")}>
        {products.map((p) => (
          <li key={p.id} className="w-[260px] shrink-0 snap-start sm:w-[280px]"><ProductCard product={p} variant="tile" /></li>
        ))}
      </ul>
    </section>
  );
}
