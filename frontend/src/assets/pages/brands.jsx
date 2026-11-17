import { useEffect } from "react";
import { Link } from "react-router-dom";
import { fetchBrands, useAsync } from "../lib/catalog.js";
import { useI18n } from "../i18n/i18nContext.js";

// Marques du catalogue (gérées au backoffice : Catalogue > Marques).
export default function Brands() {
  const { t, lang } = useI18n();
  const { status, data } = useAsync(fetchBrands, [lang]);
  useEffect(() => {
    document.title = t("meta.page", { title: t("info.brands") });
  }, [t]);
  const brands = data ?? [];
  return (
    <section className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-8 shadow-card">
      <h1 className="text-[40px] font-bold leading-tight">{t("info.brands")}</h1>
      {status === "loading" && <p role="status" className="mt-4">{t("data.loading")}</p>}
      {status === "error" && <p role="alert" className="mt-4">{t("data.error")}</p>}
      {status === "ready" && brands.length === 0 && <p className="mt-4 text-gray-700">{t("list.none")}</p>}
      <ul className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
        {brands.map((b) => (
          <li key={b.id}>
            <Link to={`/category?brand=${encodeURIComponent(b.slug)}`} className="flex h-28 flex-col items-center justify-center gap-2 rounded-[4px] border border-gray-200 p-3 text-center hover:border-black">
              {b.logoUrl ? <img src={b.logoUrl} alt="" loading="lazy" className="max-h-12 max-w-full object-contain" /> : null}
              <span className="font-medium">{b.name}</span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
