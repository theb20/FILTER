import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { fetchCategories, fetchProductIdsBySector, fetchProducts, fetchSectors, flattenCategories, rememberWebResults, searchWeb, useAsync } from "../lib/catalog.js";
import { useAuth } from "../store/authStore.js";
import { Globe } from "lucide-react";
import WebResultCard from "../components/WebResultCard.jsx";
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
  const brandParam = params.get("brand");
  const [brands, setBrands] = useState(brandParam ? [brandParam] : []);
  const sectorParam = params.get("sector");
  const [sectors, setSectors] = useState(sectorParam ? [sectorParam] : []);
  const [inStockOnly, setInStockOnly] = useState(false);
  // Recherche web : jamais automatique (chaque appel est facturé), toujours déclenchée par le visiteur.
  const accessToken = useAuth((a) => a.accessToken);
  const [web, setWeb] = useState(null); // null : jamais demandée
  const [webState, setWebState] = useState("idle"); // idle | loading | done | error | limit
  useEffect(() => { setWeb(null); setWebState("idle"); }, [rawQ]);
  const runWebSearch = async () => {
    setWebState("loading");
    const res = await searchWeb(rawQ, accessToken);
    if (res.ok) { setWeb(res.data); setWebState("done"); rememberWebResults(rawQ, res.data.hits ?? []); return; }
    setWeb(null);
    setWebState(res.status === 429 ? "limit" : "error");
  };

  useEffect(() => {
    document.title = t("meta.products");
  }, [t]);

  // Le paramètre ?cat= de l'URL pilote la catégorie cochée
  useEffect(() => {
    setCats(catParam ? [catParam] : []);
  }, [catParam]);
  useEffect(() => {
    setBrands(brandParam ? [brandParam] : []);
  }, [brandParam]);
  useEffect(() => {
    setSectors(sectorParam ? [sectorParam] : []);
  }, [sectorParam]);

  const productsState = useAsync(fetchProducts, [lang]);
  const catsState = useAsync(async () => flattenCategories(await fetchCategories()), [lang]);
  const sectorsState = useAsync(() => fetchSectors().catch(() => []), [lang]);
  const ALL_SECTORS = sectorsState.data ?? [];
  // Produits des secteurs cochés (union), demandés à l'API par secteur.
  const sectorKey = sectors.join(",");
  const sectorIds = useAsync(async () => {
    if (!sectors.length) return null;
    const known = (await fetchSectors().catch(() => [])).filter((x) => sectors.includes(x.slug));
    const sets = await Promise.all(known.map((x) => fetchProductIdsBySector(x.id)));
    return new Set(sets.flatMap((x) => [...x]));
  }, [sectorKey, lang]);
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
    if (sectorIds.data && !sectorIds.data.has(p.id)) return false;
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

        {ALL_SECTORS.length > 0 && (
          <fieldset className="mt-4" data-guide="catalog-sectors">
            <legend className="font-semibold">{t("list.sector")}</legend>
            {ALL_SECTORS.map((x) => (
              <Check key={x.id} checked={sectors.includes(x.slug)} onChange={() => toggle(sectors, setSectors, x.slug)}>
                {x.name}
              </Check>
            ))}
          </fieldset>
        )}

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
          {(productsState.status === "loading" || (sectors.length > 0 && sectorIds.status === "loading")) && (
            <li role="status" className="rounded-[4px] bg-white p-8 text-center shadow-card">{t("data.loading")}</li>
          )}
          {productsState.status === "error" && (
            <li role="alert" className="rounded-[4px] bg-white p-8 text-center shadow-card">{t("data.error")}</li>
          )}
          {productsState.status === "ready" && sectorIds.status !== "loading" && results.length === 0 && (
            <li className="rounded-[4px] bg-white p-8 text-center shadow-card">{t("list.none")}</li>
          )}
          {results.map((p) => (
            <ProductCard key={p.id} product={p} />
          ))}
        </ul>

        {rawQ && (
          <section data-guide="web-search" className="mt-2.5 rounded-[4px] bg-white p-5 shadow-card" aria-labelledby="web-title">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 id="web-title" className="flex items-center gap-2 text-[20px] font-semibold"><Globe size={20} aria-hidden="true" /> {t("web.title")}</h2>
                <p className="mt-1 text-sm text-gray-600">{results.length === 0 ? t("web.hintEmpty") : t("web.hint")}</p>
              </div>
              {webState !== "done" && (
                <button type="button" onClick={runWebSearch} disabled={webState === "loading"}
                  className="rounded-[3px] bg-night px-5 py-2.5 font-bold text-white transition-colors hover:bg-navy disabled:opacity-60">
                  {webState === "loading" ? t("web.searching") : t("web.action")}
                </button>
              )}
            </div>

            {webState === "limit" && <p role="status" className="mt-4 rounded-[6px] bg-amber-50 px-3 py-2 text-sm text-amber-900">{t("web.limit")}</p>}
            {webState === "error" && <p role="alert" className="mt-4 rounded-[6px] bg-red-50 px-3 py-2 text-sm text-red-800">{t("web.error")}</p>}
            {webState === "done" && web && (
              <div className="mt-4">
                {web.status !== "ok" && <p role="status" className="rounded-[6px] bg-amber-50 px-3 py-2 text-sm text-amber-900">{t(web.status === "budget" ? "web.budget" : "web.unavailable")}</p>}
                {web.status === "ok" && web.hits.length === 0 && <p className="text-gray-700">{t("web.none")}</p>}
                {web.status === "ok" && web.hits.length > 0 && (
                  <>
                    <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                      {web.hits.map((h, i) => <li key={i}><WebResultCard hit={h} /></li>)}
                    </ul>
                    <p className="mt-3 text-xs text-gray-500">{t("web.source", { name: web.source })} · {t("web.remaining", { n: web.remaining })}</p>
                  </>
                )}
              </div>
            )}
          </section>
        )}
      </section>
    </div>
  );
}
