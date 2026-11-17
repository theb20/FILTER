import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { fetchCategories, fetchProducts, flattenCategories, useAsync } from "../lib/catalog.js";
import { useI18n } from "../i18n/i18nContext.js";
import ProductCard from "../components/ProductCard.jsx";

function Check({ checked, onChange, children }) {
  return (
    <label className="flex cursor-pointer items-center gap-2 py-1 text-[15px]">
      <input type="checkbox" checked={checked} onChange={onChange} className="h-4 w-4 accent-teal" />
      {children}
    </label>
  );
}

export default function Category() {
  const { t, lang } = useI18n();
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

  const productsState = useAsync(fetchProducts, [lang]);
  const catsState = useAsync(async () => flattenCategories(await fetchCategories()), [lang]);
  const PRODUCTS = productsState.data ?? [];
  const CATEGORIES = catsState.data ?? [];
  const allBrands = [...new Set(PRODUCTS.map((p) => p.brand).filter(Boolean))].sort();
  const toggle = (list, setList, v) =>
    setList(list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);

  const results = PRODUCTS.filter((p) => {
    // La recherche porte sur le nom traduit, la référence, la marque et les équivalences
    if (q && ![p.name, p.reference, p.brand].some((v) => v.toLowerCase().includes(q))) return false;
    if (cats.length && !cats.some((c) => p.categoryPath.includes(c))) return false;
    if (brands.length && !brands.includes(p.brand)) return false;
    if (inStockOnly && !p.inStock) return false;
    return true;
  });

  return (
    <div className="mx-2.5 mt-2.5 grid gap-2.5 lg:grid-cols-[260px_minmax(0,1fr)]">
      <aside data-guide="catalog-filters" className="h-fit rounded-[4px] bg-white p-5 shadow-card" aria-label={t("list.filters")}>
        <h2 className="text-[24px] font-semibold">{t("list.filters")}</h2>

        <fieldset className="mt-4">
          <legend className="font-semibold">{t("list.category")}</legend>
          {CATEGORIES.map((c) => (
            <div key={c.slug} style={{ paddingLeft: `${c.depth * 16}px` }}>
              <Check checked={cats.includes(c.slug)} onChange={() => toggle(cats, setCats, c.slug)}>
                {c.name}
              </Check>
            </div>
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
            {t(results.length === 1 ? "list.count.one" : "list.count.other", { n: results.length })}
          </p>
        </div>

        <ul data-guide="catalog-list" className="mt-2.5 space-y-2.5">
          {productsState.status === "loading" && (
            <li role="status" className="rounded-[4px] bg-white p-8 text-center shadow-card">{t("data.loading")}</li>
          )}
          {productsState.status === "error" && (
            <li role="alert" className="rounded-[4px] bg-white p-8 text-center shadow-card">{t("data.error")}</li>
          )}
          {productsState.status === "ready" && results.length === 0 && (
            <li className="rounded-[4px] bg-white p-8 text-center shadow-card">{t("list.none")}</li>
          )}
          {results.map((p) => (
            <ProductCard key={p.id} product={p} />
          ))}
        </ul>
      </section>
    </div>
  );
}
