import { useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ChevronRight, LayoutGrid, Menu } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";
import { fetchCategories, useAsync } from "../lib/catalog.js";

// Barre de navigation sous l'en-tête, à la manière des grandes places de marché : bouton « Toutes les catégories »
// ouvrant un méga-menu (catégories à gauche, sous-catégories à droite) + liens rapides. Les catégories viennent du backoffice.
export default function CategoryBar() {
  const { t, lang } = useI18n();
  const { pathname } = useLocation();
  const tree = useAsync(() => fetchCategories().catch(() => []), [lang]).data ?? [];
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const box = useRef(null);

  useEffect(() => { setOpen(false); }, [pathname]);
  useEffect(() => {
    if (!open) return undefined;
    const onDown = (e) => { if (!box.current?.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("mousedown", onDown); document.removeEventListener("keydown", onKey); };
  }, [open]);

  const cur = tree[active];
  const link = "whitespace-nowrap rounded-[4px] px-3 py-2 text-[14.5px] font-medium text-white/90 transition-colors hover:bg-white/10 hover:text-white";

  return (
    <div className="hidden border-b border-white/10 bg-navy text-white lg:block" data-guide="category-bar">
      <div className="mx-auto flex max-w-[1500px] items-center gap-1 px-6">
        <div ref={box} className="relative">
          <button type="button" aria-haspopup="true" aria-expanded={open} onClick={() => setOpen((v) => !v)} className={`flex items-center gap-2.5 px-4 py-3 text-[14.5px] font-bold transition-colors ${open ? "bg-action text-black" : "bg-white/10 hover:bg-white/20"}`}>
            <Menu size={18} aria-hidden="true" /> {t("nav.allCategories")}
          </button>
          {open && (
            <div className="absolute left-0 top-full z-[1100] flex w-[860px] max-w-[92vw] overflow-hidden rounded-b-[8px] bg-white text-black shadow-2xl ring-1 ring-black/10">
              {tree.length === 0 ? (
                <p className="p-6 text-sm text-gray-600">{t("nav.noCategories")}</p>
              ) : (
                <>
                  <ul className="max-h-[460px] w-[280px] shrink-0 overflow-y-auto border-r border-gray-100 bg-gray-50 py-2" role="menu">
                    {tree.map((c, i) => (
                      <li key={c.slug} role="none">
                        <Link role="menuitem" to={`/category?cat=${c.slug}`} onMouseEnter={() => setActive(i)} onFocus={() => setActive(i)}
                          className={`flex items-center justify-between gap-2 px-5 py-3 text-[15px] ${i === active ? "bg-white font-semibold text-night" : "text-gray-700 hover:bg-white"}`}>
                          <span className="truncate">{c.name}</span><ChevronRight size={15} className="shrink-0 text-gray-400" aria-hidden="true" />
                        </Link>
                      </li>
                    ))}
                  </ul>
                  <div className="min-w-0 flex-1 p-6">
                    <p className="text-[20px] font-bold">{cur?.name}</p>
                    <ul className="mt-4 grid grid-cols-2 gap-x-6 gap-y-2.5">
                      {(cur?.children ?? []).map((c) => (
                        <li key={c.slug}><Link to={`/category?cat=${c.slug}`} className="text-[14.5px] text-gray-700 hover:text-link hover:underline">{c.name}</Link></li>
                      ))}
                    </ul>
                    {(cur?.children ?? []).length === 0 && <p className="mt-3 text-sm text-gray-500">{t("nav.noSub")}</p>}
                    <Link to={`/category?cat=${cur?.slug ?? ""}`} className="mt-6 inline-flex items-center gap-1.5 rounded-[6px] bg-night px-4 py-2.5 text-sm font-semibold text-white hover:bg-navy">{t("nav.seeAll")} <ChevronRight size={15} aria-hidden="true" /></Link>
                  </div>
                </>
              )}
            </div>
          )}
        </div>

        <nav aria-label={t("nav.quick")} className="flex min-w-0 flex-1 items-center gap-0.5 overflow-x-auto [scrollbar-width:none]">
          {tree.slice(0, 5).map((c) => <Link key={c.slug} to={`/category?cat=${c.slug}`} className={link}>{c.name}</Link>)}
          <Link to="/brands" className={link}>{t("nav.brands")}</Link>
          <Link to="/blog" className={link}>{t("nav.blog")}</Link>
        </nav>
        <Link to="/quote" className="ml-auto flex shrink-0 items-center gap-2 rounded-[4px] bg-action px-4 py-2 text-[14.5px] font-bold text-black transition-colors hover:bg-action-dark"><LayoutGrid size={16} aria-hidden="true" /> {t("footer.quote")}</Link>
      </div>
    </div>
  );
}
