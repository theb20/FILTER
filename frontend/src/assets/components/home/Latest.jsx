import { useRef } from "react";
import { Link } from "react-router-dom";
import { ChevronLeft, ChevronRight } from "lucide-react";
import LineArt from "../LineArt.jsx";
import { useI18n } from "../../i18n/i18nContext.js";
import { fetchLatest, useAsync } from "../../lib/catalog.js";

// « Nouveautés » : derniers produits publiés, en bandeau défilant. Masqué tant que le catalogue est vide.
export default function Latest() {
  const { t, lang, formatPrice } = useI18n();
  const list = useAsync(() => fetchLatest().catch(() => []), [lang]).data ?? [];
  const rail = useRef(null);
  if (list.length === 0) return null;
  const scroll = (dir) => rail.current?.scrollBy({ left: dir * Math.max(280, rail.current.clientWidth * 0.8), behavior: "smooth" });
  const arrow = "absolute top-1/2 z-10 hidden h-11 w-11 -translate-y-1/2 items-center justify-center rounded-[4px] border border-gray-300 bg-white shadow-sm hover:border-gray-500 md:flex";

  return (
    <section className="mx-2.5 rounded-[6px] bg-white px-4 py-10 shadow-card sm:px-8" aria-labelledby="latest-title">
      <h2 id="latest-title" className="text-center text-[28px] font-bold uppercase tracking-tight text-night sm:text-[32px]">{t("latest.title")}</h2>
      <div className="relative mx-auto mt-8 max-w-[1400px]">
        <button type="button" className={`${arrow} -left-3`} onClick={() => scroll(-1)} aria-label={t("hero.prev")}><ChevronLeft size={20} aria-hidden="true" /></button>
        <ul ref={rail} className="flex snap-x snap-mandatory gap-4 overflow-x-auto scroll-smooth px-1 pb-3 [scrollbar-width:none]" tabIndex={0} aria-label={t("latest.title")}>
          {list.map((p) => (
            <li key={p.id} className="w-[230px] shrink-0 snap-start sm:w-[260px]">
              <Link to={`/card?id=${p.id}`} className="group flex h-full flex-col rounded-[6px] bg-gray-50 p-4 transition-shadow hover:shadow-lg">
                <span className="flex h-44 items-center justify-center rounded-[4px] bg-white">
                  {p.image ? <img src={p.image} alt="" loading="lazy" className="max-h-full max-w-full object-contain" /> : <LineArt stroke="#9ca3af" className="h-32" />}
                </span>
                <span className="mx-auto mt-3 h-1.5 w-1.5 rounded-full bg-action" aria-hidden="true" />
                <span className="mt-2 line-clamp-2 min-h-[3rem] text-center text-[16px] font-semibold leading-snug text-night group-hover:underline">{p.name}</span>
                <span className="mt-1 text-center text-[12.5px] text-gray-500">{[p.reference, p.brand].filter(Boolean).join(" · ")}</span>
                <span className="mt-2 text-center text-[17px] font-bold">{p.onQuote ? t("price.onQuote") : formatPrice(p.price)}</span>
                <span className="mt-3 rounded-[4px] bg-night px-4 py-2.5 text-center text-sm font-semibold text-white group-hover:bg-navy">{t("latest.view")}</span>
              </Link>
            </li>
          ))}
        </ul>
        <button type="button" className={`${arrow} -right-3`} onClick={() => scroll(1)} aria-label={t("hero.next")}><ChevronRight size={20} aria-hidden="true" /></button>
      </div>
    </section>
  );
}
