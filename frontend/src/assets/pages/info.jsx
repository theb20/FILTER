import { useEffect } from "react";
import Todo from "../components/Todo.jsx";
import CmsPage from "../components/CmsPage.jsx";
import { fetchPage } from "../lib/content.js";
import { useAsync } from "../lib/catalog.js";
import { useI18n } from "../i18n/i18nContext.js";

// Page d'information : son contenu se gère au backoffice (Contenu du site > Pages, avec l'adresse `slug`).
// Tant qu'elle n'est pas publiée, un emplacement « à compléter » s'affiche (jamais de faux texte).
export default function InfoPage({ titleKey, slug }) {
  const { t, lang } = useI18n();
  const title = t(titleKey);
  const page = useAsync(() => fetchPage(slug).catch(() => null), [slug, lang]);

  useEffect(() => {
    document.title = t("meta.page", { title: page.data?.title ?? title });
  }, [t, title, page.data]);

  if (page.status === "loading") return <div role="status" className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-8 shadow-card">{t("data.loading")}</div>;
  if (page.data) return <CmsPage page={page.data} />;
  return (
    <section className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-8 shadow-card">
      <h1 className="text-[40px] font-bold leading-tight">{title}</h1>
      <p className="mt-4"><Todo>{t("todo.content", { title })}</Todo></p>
    </section>
  );
}
