import { useCallback, useEffect, useMemo, useState } from "react";
import { I18nContext } from "./i18nContext.js";
import en from "./locales/en.js";
import fr from "./locales/fr.js";
import es from "./locales/es.js";

export const LANGUAGES = [
  { code: "en", label: "English", locale: "en-US" },
  { code: "fr", label: "Français", locale: "fr-FR" },
  { code: "es", label: "Español", locale: "es-ES" },
];

const DICTIONARIES = { en, fr, es };
const STORAGE_KEY = "ff_lang";

function initialLang() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && DICTIONARIES[saved]) return saved;
  } catch {
    // stockage indisponible : on passe à la langue du navigateur
  }
  const nav = (navigator.language || "en").slice(0, 2);
  return DICTIONARIES[nav] ? nav : "en";
}

export function I18nProvider({ children }) {
  const [lang, setLangState] = useState(initialLang);

  useEffect(() => {
    document.documentElement.lang = lang;
  }, [lang]);

  const setLang = useCallback((code) => {
    if (!DICTIONARIES[code]) return;
    setLangState(code);
    try {
      localStorage.setItem(STORAGE_KEY, code);
    } catch {
      // préférence non mémorisée, sans conséquence
    }
  }, []);

  const value = useMemo(() => {
    const dict = DICTIONARIES[lang];
    const locale = LANGUAGES.find((l) => l.code === lang).locale;

    const t = (key, vars) => {
      let text = dict[key] ?? en[key] ?? key;
      if (vars) {
        text = text.replace(/\{(\w+)\}/g, (_, name) => (name in vars ? String(vars[name]) : `{${name}}`));
      }
      return text;
    };

    const formatPrice = (n) =>
      new Intl.NumberFormat(locale, { style: "currency", currency: "USD" }).format(n);

    return { lang, setLang, t, locale, formatPrice };
  }, [lang, setLang]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}
