import { useI18n } from "../i18n/i18nContext.js";

// Emplacement visible pour une information à fournir (rien n'est inventé).
export default function Todo({ children, className = "" }) {
  const { t } = useI18n();
  return (
    <span className={`border-b border-dashed border-current opacity-80 ${className}`} title={t("todo.title")}>
      [{children}]
    </span>
  );
}
