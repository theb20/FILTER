import ProductCard from "../ProductCard.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

// Filtres cités : cartes produit (composant partagé) reliées au panier.
export default function CitedProducts({ products, titleKey = "blog.cited", id = "cited-title" }) {
  const { t } = useI18n();
  if (!products.length) return null;
  return (
    <section aria-labelledby={id} className="mx-2.5 mt-2.5">
      <h2 id={id} className="text-[22px] font-bold">{t(titleKey)}</h2>
      <ul className="mt-3 grid gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
        {products.map((p) => <li key={p.id}><ProductCard product={p} variant="tile" /></li>)}
      </ul>
    </section>
  );
}
