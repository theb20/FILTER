import { lazy, Suspense, useState } from "react";
import LineArt from "../LineArt.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

const FilterScene = lazy(() => import("../../animation/FilterScene.jsx"));
const VIEWS = [0, 1, 2, 3]; // 0 = vue 3D, 1..3 = photos à venir

// Galerie : visionneuse principale + vignettes (les photos sont des emplacements de démonstration).
export default function Gallery({ name }) {
  const { t } = useI18n();
  const [view, setView] = useState(0);
  return (
    <div>
      <div className="flex aspect-square items-center justify-center overflow-hidden rounded-[8px] bg-gray-50" title={t("prod.photo")}>
        {view === 0 ? (
          <Suspense fallback={<LineArt stroke="#9ca3af" className="h-4/5" />}>
            <FilterScene className="h-full w-full" drag label={name} particles={360} />
          </Suspense>
        ) : (
          <img src="/img/products/placeholder.svg" alt={t("pd.gallery.photoAlt", { n: view })} className="h-full w-full object-contain p-6" />
        )}
      </div>
      <ul className="mt-3 grid grid-cols-4 gap-2" aria-label={t("pd.gallery.label")}>
        {VIEWS.map((v) => (
          <li key={v}>
            <button
              type="button"
              onClick={() => setView(v)}
              aria-pressed={view === v}
              aria-label={v === 0 ? t("pd.gallery.view3d") : t("pd.gallery.photoAlt", { n: v })}
              className={`flex aspect-square w-full items-center justify-center rounded-[6px] border-2 bg-gray-50 transition-colors ${view === v ? "border-action" : "border-transparent hover:border-gray-300"}`}
            >
              {v === 0 ? <span className="text-xs font-bold text-gray-600">3D</span> : <img src="/img/products/placeholder.svg" alt="" loading="lazy" className="h-3/4 w-3/4 object-contain opacity-60" />}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
