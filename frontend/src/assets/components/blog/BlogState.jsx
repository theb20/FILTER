import { Link } from "react-router-dom";
import { AlertTriangle, SearchX } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";

// États vide / erreur / introuvable. kind : "empty" | "error" | "notFound".
export default function BlogState({ kind, onRetry, resetTo = "/blog" }) {
  const { t } = useI18n();
  const Icon = kind === "error" ? AlertTriangle : SearchX;
  return (
    <div role={kind === "error" ? "alert" : "status"} className="rounded-[8px] bg-white p-8 text-center shadow-card">
      <Icon size={32} className="mx-auto text-gray-400" aria-hidden="true" />
      <h3 className="mt-3 text-[20px] font-semibold">{t(`blog.${kind}.title`)}</h3>
      <p className="mt-1 text-gray-700">{t(`blog.${kind}.text`)}</p>
      {kind === "error" ? (
        <button type="button" onClick={onRetry} className="mt-4 rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">{t("blog.error.retry")}</button>
      ) : (
        <Link to={resetTo} className="mt-4 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">{t("blog.empty.reset")}</Link>
      )}
    </div>
  );
}
