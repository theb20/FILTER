import { useI18n } from "../../i18n/i18nContext.js";

// Squelette de chargement de la page d'accueil du blog.
export default function BlogSkeleton() {
  const { t } = useI18n();
  return (
    <div role="status" aria-busy="true" aria-label={t("blog.loading")} className="mx-2.5 mt-2.5 animate-pulse space-y-2.5">
      <div className="grid gap-2.5 sm:grid-cols-2 lg:grid-cols-4">{[0, 1, 2, 3].map((i) => <div key={i} className="h-20 rounded-[8px] bg-gray-200" />)}</div>
      <div className="grid gap-2.5 lg:grid-cols-[1.2fr_1fr]">
        <div className="h-[360px] rounded-[8px] bg-gray-200" />
        <div className="grid gap-2.5 sm:grid-cols-2"><div className="h-[240px] rounded-[8px] bg-gray-200" /><div className="h-[240px] rounded-[8px] bg-gray-200" /><div className="h-[220px] rounded-[8px] bg-gray-200 sm:col-span-2" /></div>
      </div>
      <div className="grid gap-2.5 lg:grid-cols-[minmax(0,1fr)_320px]"><div className="h-80 rounded-[8px] bg-gray-200" /><div className="h-80 rounded-[8px] bg-gray-200" /></div>
    </div>
  );
}
