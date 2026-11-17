import { useSearchParams, Link } from "react-router-dom";
import { ratingStats } from "../lib/productUtils.js";
import { fetchProduct, fetchRelated, useAsync } from "../lib/catalog.js";
import { fetchFaq, fetchProductReviews } from "../lib/content.js";
import { usePageMeta } from "../lib/usePageMeta.js";
import { useI18n } from "../i18n/i18nContext.js";
import JsonLd from "../components/JsonLd.jsx";
import Stars from "../components/Stars.jsx";
import BuyBox from "../components/product/BuyBox.jsx";
import Gallery from "../components/product/Gallery.jsx";
import ProductDetails from "../components/product/ProductDetails.jsx";
import RelatedProducts from "../components/product/RelatedProducts.jsx";
import ReviewsPanel from "../components/product/ReviewsPanel.jsx";
import FaqAccordion from "../components/sections/FaqAccordion.jsx";
import ReassuranceStrip from "../components/sections/ReassuranceStrip.jsx";

// Fiche produit détaillée : galerie, prix et achat, réassurance, détails / avis / FAQ, produits associés.
export default function ProductDetail() {
  const { t, lang } = useI18n();
  const [params] = useSearchParams();
  const id = params.get("id") ?? "";
  const { status, data: product } = useAsync(() => (id ? fetchProduct(id) : Promise.reject(Object.assign(new Error("none"), { status: 404 }))), [id, lang]);
  const related = useAsync(() => (id ? fetchRelated(id).catch(() => []) : Promise.resolve([])), [id, lang]);
  const reviewsState = useAsync(() => (id ? fetchProductReviews(id).catch(() => ({ items: [], count: 0, average: 0 })) : Promise.resolve({ items: [], count: 0, average: 0 })), [id, lang]);
  const REVIEWS = reviewsState.data?.items ?? [];
  const faqState = useAsync(() => fetchFaq().catch(() => []), [lang]);
  const stats = ratingStats(REVIEWS);
  const name = product?.name ?? "";

  usePageMeta(product ? t("meta.product", { name }) : t("meta.notFound"), product ? t("pd.meta.desc", { name, ref: product.reference, brand: product.brand }) : "");

  if (status === "loading") {
    return <div className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-10 text-center shadow-card" role="status">{t("data.loading")}</div>;
  }
  if (!product) {
    return (
      <div className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-10 text-center shadow-card">
        <h1 className="text-[32px] font-bold">{t("prod.notFound")}</h1>
        <p className="mt-2">{t("prod.notFound.text")}</p>
        <Link to="/category" className="mt-6 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
          {t("cart.browse")} <span aria-hidden="true">&nbsp;→</span>
        </Link>
      </div>
    );
  }

  const faq = (faqState.data ?? []).map((f) => ({ q: f.question, a: f.answer }));
  const crumbs = [
    [t("nav.home"), "/"],
    [t("nav.products"), "/category"],
    [product.categoryName, `/category?cat=${product.category}`],
  ];

  return (
    <div className="mx-2.5 mt-2.5 space-y-2.5">
      <JsonLd data={{
        "@context": "https://schema.org", "@type": "Product",
        name, sku: product.reference, ...(product.brand ? { brand: { "@type": "Brand", name: product.brand } } : {}),
      }} />

      <nav aria-label={t("prod.breadcrumb")} className="px-1 text-sm text-gray-600">
        <ol className="flex flex-wrap items-center gap-x-1.5">
          {crumbs.map(([label, to]) => (
            <li key={to} className="flex items-center gap-1.5"><Link to={to} className="text-link hover:underline">{label}</Link><span aria-hidden="true">/</span></li>
          ))}
          <li aria-current="page" className="text-gray-800">{name}</li>
        </ol>
      </nav>

      <section className="grid gap-6 rounded-[8px] bg-white p-5 shadow-card md:grid-cols-2 lg:grid-cols-[1fr_1fr] lg:p-6">
        <Gallery name={name} media={product.media} />
        <div>
          <h1 className="text-[32px] font-bold leading-tight sm:text-[36px]">{name}</h1>
          <p className="mt-1 text-gray-700">{t("prod.reference")} : <strong>{product.reference}</strong>{product.brand ? ` · ${product.brand}` : ""}</p>
          {stats.count > 0 && (
            <p className="mt-2 flex flex-wrap items-center gap-2 text-sm">
              <Stars value={stats.average} />
              <a href="#reviews" className="text-link hover:underline">{t("pd.reviews.count", { n: stats.count })}</a>
            </p>
          )}
          <div className="mt-4"><BuyBox product={product} /></div>
        </div>
      </section>

      <ReassuranceStrip />

      <div className="grid items-start gap-2.5 lg:grid-cols-3">
        <ProductDetails product={product} />
        <ReviewsPanel reviews={REVIEWS} />
        {faq.length > 0 && (
          <section aria-labelledby="pd-faq-title" className="rounded-[8px] bg-white p-5 shadow-card">
            <h2 id="pd-faq-title" className="text-[20px] font-bold">{t("pd.faq")}</h2>
            <div className="mt-2"><FaqAccordion items={faq} /></div>
          </section>
        )}
      </div>

      <RelatedProducts products={related.data ?? []} />
    </div>
  );
}
