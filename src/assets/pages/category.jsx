import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { CATEGORIES, DEMO_PRODUCTS, formatUsd } from "../data/demoProducts.js";
import { useNotify } from "../configurations/notificationContext.js";
import { DEMO_NOTICE } from "../config/site.js";
import LineArt from "../components/LineArt.jsx";

const btnPrimary =
  "inline-flex items-center gap-2 rounded-[3px] bg-action px-4 py-2 font-bold text-black hover:bg-action-dark transition-colors";

export default function Category() {
  const notify = useNotify();
  const [params] = useSearchParams();
  const q = (params.get("q") || "").trim().toLowerCase();
  const catParam = params.get("cat");

  const [cats, setCats] = useState(catParam ? [catParam] : []);
  const [brands, setBrands] = useState([]);
  const [inStockOnly, setInStockOnly] = useState(false);

  useEffect(() => {
    document.title = "Products — FilterFinder";
  }, []);

  // Le paramètre ?cat= de l'URL pilote la catégorie cochée
  useEffect(() => {
    setCats(catParam ? [catParam] : []);
  }, [catParam]);

  const allBrands = [...new Set(DEMO_PRODUCTS.map((p) => p.brand))];
  const toggle = (list, setList, v) =>
    setList(list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);

  const results = DEMO_PRODUCTS.filter((p) => {
    if (q && ![p.name, p.reference, p.brand, ...p.crossRefs].some((v) => v.toLowerCase().includes(q))) return false;
    if (cats.length && !cats.includes(p.category)) return false;
    if (brands.length && !brands.includes(p.brand)) return false;
    if (inStockOnly && p.stock !== "In stock") return false;
    return true;
  });

  const Check = ({ checked, onChange, children }) => (
    <label className="flex cursor-pointer items-center gap-2 py-1 text-[15px]">
      <input type="checkbox" checked={checked} onChange={onChange} className="h-4 w-4 accent-teal" />
      {children}
    </label>
  );

  return (
    <div className="mx-2.5 mt-2.5 grid gap-2.5 lg:grid-cols-[260px_minmax(0,1fr)]">
      <aside className="h-fit rounded-[4px] bg-white p-5 shadow-card" aria-label="Filters">
        <h2 className="text-[24px] font-semibold">Filters</h2>

        <fieldset className="mt-4">
          <legend className="font-semibold">Category</legend>
          {CATEGORIES.map((c) => (
            <Check key={c.slug} checked={cats.includes(c.slug)} onChange={() => toggle(cats, setCats, c.slug)}>
              {c.label}
            </Check>
          ))}
        </fieldset>

        <fieldset className="mt-4">
          <legend className="font-semibold">Brand</legend>
          {allBrands.map((b) => (
            <Check key={b} checked={brands.includes(b)} onChange={() => toggle(brands, setBrands, b)}>
              {b}
            </Check>
          ))}
        </fieldset>

        <div className="mt-4">
          <Check checked={inStockOnly} onChange={() => setInStockOnly(!inStockOnly)}>
            In stock only
          </Check>
        </div>
      </aside>

      <section aria-labelledby="results-title">
        <div className="rounded-[4px] bg-white p-5 shadow-card">
          <h1 id="results-title" className="text-[40px] font-bold leading-tight">
            {q ? `Results for “${params.get("q")}”` : "Products"}
          </h1>
          <p className="mt-1 text-sm text-gray-600">
            {results.length} result{results.length === 1 ? "" : "s"} · Demo data: products, references and prices are fictional.
          </p>
        </div>

        <ul className="mt-2.5 space-y-2.5">
          {results.length === 0 && (
            <li className="rounded-[4px] bg-white p-8 text-center shadow-card">No product matches your search.</li>
          )}
          {results.map((p) => (
            <li key={p.id} className="flex flex-col gap-4 rounded-[4px] bg-white p-4 shadow-card sm:flex-row">
              <Link to={`/card?id=${p.id}`} className="flex h-36 w-full shrink-0 items-center justify-center bg-gray-50 sm:w-36" aria-label={p.name}>
                <LineArt stroke="#9ca3af" className="h-28" />
              </Link>
              <div className="min-w-0 flex-1">
                <Link to={`/card?id=${p.id}`} className="text-[24px] font-semibold leading-tight text-link hover:underline">
                  {p.name}
                </Link>
                <p className="mt-1 text-sm text-gray-700">
                  Ref. <strong>{p.reference}</strong> · {p.brand}
                </p>
                <p className={`mt-1 text-sm font-medium ${p.stock === "In stock" ? "text-green-700" : "text-orange-700"}`}>
                  {p.stock}
                </p>
              </div>
              <div className="flex shrink-0 flex-col items-start gap-3 sm:items-end">
                <p className="text-[24px] font-bold">{formatUsd(p.price)}</p>
                <button type="button" onClick={() => notify(DEMO_NOTICE, "demo")} className={btnPrimary}>
                  Add to cart <span aria-hidden="true">→</span>
                </button>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
