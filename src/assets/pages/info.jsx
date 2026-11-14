import { useEffect } from "react";
import Todo from "../components/Todo.jsx";
import { useI18n } from "../i18n/i18nContext.js";

// Pages d'information (contenu à fournir) : évite les liens morts du footer.
// titleKey : clé i18n du titre (info.blog, info.brands…)
export default function InfoPage({ titleKey }) {
  const { t } = useI18n();
  const title = t(titleKey);

  useEffect(() => {
    document.title = t("meta.page", { title });
  }, [t, title]);

  return (
    <section className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-8 shadow-card">
      <h1 className="text-[40px] font-bold leading-tight">{title}</h1>
      <p className="mt-4"><Todo>{t("todo.content", { title })}</Todo></p>
    </section>
  );
}
