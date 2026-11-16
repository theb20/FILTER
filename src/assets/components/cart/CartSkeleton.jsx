import { useI18n } from "../../i18n/i18nContext.js";

// État « panier pas encore chargé » : évite un flash de panier vide.
export default function CartSkeleton() {
  const { t } = useI18n();
  return (
    <div role="status" aria-busy="true" aria-label={t("cart.loading")} className="mt-4 animate-pulse space-y-5">
      {[0, 1].map((i) => (
        <div key={i} className="flex gap-4">
          <div className="h-24 w-24 shrink-0 bg-gray-100 sm:h-28 sm:w-28" />
          <div className="flex-1 space-y-2.5">
            <div className="h-5 w-2/3 bg-gray-100" />
            <div className="h-4 w-1/3 bg-gray-100" />
            <div className="h-4 w-1/2 bg-gray-100" />
          </div>
        </div>
      ))}
    </div>
  );
}
