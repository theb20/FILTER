import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { CATEGORIES, DEMO_PRODUCTS } from "../data/demoProducts.js";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import LineArt from "../components/LineArt.jsx";

const btnPrimary =
  "inline-flex items-center gap-2 rounded-[3px] bg-action px-4 py-2 font-bold text-black hover:bg-action-dark transition-colors";

function Check({ checked, onChange, children }) {
  return (
    <label className="flex cursor-pointer items-center gap-2 py-1 text-[15px]">
      <input type="checkbox" checked={checked} onChange={onChange} className="h-4 w-4 accent-teal" />
      {children}
    </label>
  );
}

export default function Category() {
  const notify = useNotify();
  const { t, formatPrice } = useI18n();
  const [params] = useSearchParams();
  const rawQ = (params.get("q") || "").trim();
  const q = rawQ.toLowerCase();
  const catParam = params.get("cat");

  const [cats, setCats] = useState(catParam ? [catParam] : []);
  const [brands, setBrands] = useState([]);
  const [inStockOnly, setInStockOnly] = useState(false);

  useEffect(() => {
    document.title = t("meta.products");
  }, [t]);

  // Le paramètre ?cat= de l'URL pilote la catégorie cochée
  useEffect(() => {
    setCats(catParam ? [catParam] : []);
  }, [catParam]);

  const allBrands = [...new Set(DEMO_PRODUCTS.map((p) => p.brand))];
  const toggle = (list, setList, v) =>
    setList(list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);

  const results = DEMO_PRODUCTS.filter((p) => {
    // La recherche porte sur le nom traduit, la référence, la marque et les équivalences
    if (q && ![t(`prod.${p.id}`), p.reference, p.brand, ...p.crossRefs].some((v) => v.toLowerCase().includes(q))) return false;
    if (cats.length && !cats.includes(p.category)) return false;
    if (brands.length && !brands.includes(p.brand)) return false;
    if (inStockOnly && !p.inStock) return false;
    return true;
  });

  return (
    <div className="mx-2.5 mt-2.5 grid gap-2.5 lg:grid-cols-[260px_minmax(0,1fr)]">
      <aside className="h-fit rounded-[4px] bg-white p-5 shadow-card" aria-label={t("list.filters")}>
        <h2 className="text-[24px] font-semibold">{t("list.filters")}</h2>

        <fieldset className="mt-4">
          <legend className="font-semibold">{t("list.category")}</legend>
          {CATEGORIES.map((c) => (
            <Check key={c.slug} checked={cats.includes(c.slug)} onChange={() => toggle(cats, setCats, c.slug)}>
              {t(`cat.${c.slug}`)}
            </Check>
          ))}
        </fieldset>

        <fieldset className="mt-4">
          <legend className="font-semibold">{t("list.brand")}</legend>
          {allBrands.map((b) => (
            <Check key={b} checked={brands.includes(b)} onChange={() => toggle(brands, setBrands, b)}>
              {b}
            </Check>
          ))}
        </fieldset>

        <div className="mt-4">
          <Check checked={inStockOnly} onChange={() => setInStockOnly(!inStockOnly)}>
            {t("list.inStockOnly")}
          </Check>
        </div>
      </aside>

      <section aria-labelledby="results-title">
        <div className="rounded-[4px] bg-white p-5 shadow-card">
          <h1 id="results-title" className="text-[40px] font-bold leading-tight">
            {rawQ ? t("list.resultsFor", { q: rawQ }) : t("list.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-600">
            {t(results.length === 1 ? "list.count.one" : "list.count.other", { n: results.length })} · {t("demo.data")}
          </p>
        </div>

        <ul className="mt-2.5 space-y-2.5">
          {results.length === 0 && (
            <li className="rounded-[4px] bg-white p-8 text-center shadow-card">{t("list.none")}</li>
          )}
          {results.map((p) => (
            <li key={p.id} className="flex flex-col gap-4 rounded-[4px] bg-white p-4 shadow-card sm:flex-row">
              <Link to={`/card?id=${p.id}`} className="flex h-36 w-full shrink-0 items-center justify-center bg-gray-50 sm:w-36" aria-label={t(`prod.${p.id}`)}>
                <LineArt stroke="#9ca3af" className="h-28" />
              </Link>
              <div className="min-w-0 flex-1">
                <Link to={`/card?id=${p.id}`} className="text-[24px] font-semibold leading-tight text-link hover:underline">
                  {t(`prod.${p.id}`)}
                </Link>
                <p className="mt-1 text-sm text-gray-700">
                  {t("list.ref")} <strong>{p.reference}</strong> · {p.brand}
                </p>
                <p className={`mt-1 text-sm font-medium ${p.inStock ? "text-green-700" : "text-orange-700"}`}>
                  {p.inStock ? t("stock.in") : t("stock.back")}
                </p>
              </div>
              <div className="flex shrink-0 flex-col items-start gap-3 sm:items-end">
                <p className="text-[24px] font-bold">{formatPrice(p.price)}</p>
                <button type="button" onClick={() => notify(t("demo.notice"), "demo")} className={btnPrimary}>
                  {t("list.addToCart")} <span aria-hidden="true">→</span>
                </button>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
