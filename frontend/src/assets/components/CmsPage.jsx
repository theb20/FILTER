import Markdown from "./Markdown.jsx";
import { useI18n } from "../i18n/i18nContext.js";

// Page légale ou éditoriale gérée au backoffice (Contenu du site > Pages).
export default function CmsPage({ page }) {
  const { locale } = useI18n();
  return (
    <article className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-6 shadow-card md:p-10">
      <h1 className="text-[40px] font-bold leading-tight">{page.title}</h1>
      <p className="mt-1 text-sm text-gray-600">{new Intl.DateTimeFormat(locale, { dateStyle: "long" }).format(new Date(page.updatedAt))}</p>
      <div className="mt-6 max-w-3xl"><Markdown source={page.body} /></div>
    </article>
  );
}
