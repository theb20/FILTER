import { useSearchParams, Link } from "react-router-dom";
import { DEMO_PRODUCTS } from "../data/demoProducts.js";
import { CART_PRODUCT_MOCKS } from "../data/cartMocks.js";
import { DEMO_REVIEWS } from "../data/productMocks.js";
import { ratingStats, relatedProducts } from "../lib/productUtils.js";
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
  const { t } = useI18n();
  const [params] = useSearchParams();
  const product = DEMO_PRODUCTS.find((p) => String(p.id) === params.get("id")) ?? DEMO_PRODUCTS[0];
  const mock = CART_PRODUCT_MOCKS[product.id];
  const name = t(`prod.${product.id}`);
  const stats = ratingStats(DEMO_REVIEWS);

  usePageMeta(t("meta.product", { name }), t("pd.meta.desc", { name, ref: product.reference, brand: product.brand }));

  const faq = [1, 2, 3, 4].map((n) => ({ q: t(`pd.faq.q${n}`), a: t(`pd.faq.a${n}`) }));
  const crumbs = [
    [t("nav.home"), "/"],
    [t("nav.products"), "/category"],
    [t(`cat.${product.category}`), `/category?cat=${product.category}`],
  ];

  return (
    <div className="mx-2.5 mt-2.5 space-y-2.5">
      <JsonLd data={{
        "@context": "https://schema.org", "@type": "Product",
        name, sku: product.reference, brand: { "@type": "Brand", name: product.brand },
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
        <Gallery name={name} />
        <div>
          <h1 className="text-[32px] font-bold leading-tight sm:text-[36px]">{name}</h1>
          <p className="mt-1 text-gray-700">{t("prod.reference")} : <strong>{product.reference}</strong> · {product.brand}</p>
          <p className="mt-2 flex flex-wrap items-center gap-2 text-sm">
            <Stars value={stats.average} />
            <a href="#reviews" className="text-link hover:underline">{t("pd.reviews.count", { n: stats.count })}</a>
            <span className="text-gray-500">({t("pd.reviews.demoShort")})</span>
          </p>
          <div className="mt-4"><BuyBox product={product} mock={mock} name={name} /></div>
          <p className="mt-4 text-sm text-amber-800">{t("prod.demoNote")}</p>
        </div>
      </section>

      <ReassuranceStrip />

      <div className="grid items-start gap-2.5 lg:grid-cols-3">
        <ProductDetails product={product} mock={mock} name={name} />
        <ReviewsPanel />
        <section aria-labelledby="pd-faq-title" className="rounded-[8px] bg-white p-5 shadow-card">
          <h2 id="pd-faq-title" className="text-[20px] font-bold">{t("pd.faq")}</h2>
          <div className="mt-2"><FaqAccordion items={faq} /></div>
        </section>
      </div>

      <RelatedProducts products={relatedProducts(product, DEMO_PRODUCTS, 6)} />
    </div>
  );
}
