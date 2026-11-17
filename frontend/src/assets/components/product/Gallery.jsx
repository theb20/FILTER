import { lazy, Suspense, useState } from "react";
import LineArt from "../LineArt.jsx";
import { useI18n } from "../../i18n/i18nContext.js";

const FilterScene = lazy(() => import("../../animation/FilterScene.jsx"));

// Galerie : les médias du produit (URL saisies au backoffice) ; sans média, la vue 3D de démonstration.
export default function Gallery({ name, media = [] }) {
  const { t } = useI18n();
  const [view, setView] = useState(0);
  const photos = media;
  const current = photos[view];
  return (
    <div data-guide="product-gallery">
      <div className="flex aspect-square items-center justify-center overflow-hidden rounded-[8px] bg-gray-50">
        {current ? (
          <img src={current.url} alt={current.alt || name} className="h-full w-full object-contain p-4" />
        ) : (
          <Suspense fallback={<LineArt stroke="#9ca3af" className="h-4/5" />}>
            <FilterScene className="h-full w-full" drag label={name} particles={360} />
          </Suspense>
        )}
      </div>
      {photos.length > 1 && (
        <ul className="mt-3 grid grid-cols-4 gap-2" aria-label={t("pd.gallery.label")}>
          {photos.map((m, i) => (
            <li key={m.id}>
              <button
                type="button"
                onClick={() => setView(i)}
                aria-pressed={view === i}
                aria-label={t("pd.gallery.photoAlt", { n: i + 1 })}
                className={`flex aspect-square w-full items-center justify-center rounded-[6px] border-2 bg-gray-50 transition-colors ${view === i ? "border-action" : "border-transparent hover:border-gray-300"}`}
              >
                <img src={m.url} alt="" loading="lazy" className="h-3/4 w-3/4 object-contain" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
