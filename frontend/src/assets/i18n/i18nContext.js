import { createContext, useContext } from "react";

export const I18nContext = createContext(null);

// const { t, lang, setLang } = useI18n();  t("key", { var: "valeur" })
export const useI18n = () => {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n doit être utilisé à l'intérieur de I18nProvider");
  return ctx;
};
