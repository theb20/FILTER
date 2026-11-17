import { useCallback, useEffect, useMemo, useState } from "react";
import { I18nContext } from "./i18nContext.js";
import enBase from "./locales/en.js";
import frBase from "./locales/fr.js";
import esBase from "./locales/es.js";
import guideEn from "../guide/i18n/en.js";
import guideFr from "../guide/i18n/fr.js";
import guideEs from "../guide/i18n/es.js";

// Les textes du guide vivent à côté du guide ; ils rejoignent ici les dictionnaires du site.
const en = { ...enBase, ...guideEn };
const fr = { ...frBase, ...guideFr };
const es = { ...esBase, ...guideEs };
import { setApiLang } from "../lib/api.js";
import { CURRENCIES, DEFAULT_CURRENCY, formatMoney, isCurrencyCode } from "../config/currency.js";

export const LANGUAGES = [
  { code: "en", label: "English", locale: "en-US" },
  { code: "fr", label: "Français", locale: "fr-FR" },
  { code: "es", label: "Español", locale: "es-ES" },
];

const DICTIONARIES = { en, fr, es };
const STORAGE_KEY = "ff_lang";
const CURRENCY_KEY = "ff_currency";

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

function initialCurrency() {
  try {
    const saved = localStorage.getItem(CURRENCY_KEY);
    if (saved && isCurrencyCode(saved)) return saved;
  } catch {
    // stockage indisponible : devise par défaut (XOF)
  }
  return DEFAULT_CURRENCY;
}

export function I18nProvider({ children }) {
  const [lang, setLangState] = useState(initialLang);
  const [currency, setCurrencyState] = useState(initialCurrency);

  setApiLang(lang); // appliqué avant les effets des pages : les requêtes partent dans la langue choisie
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

  const setCurrency = useCallback((code) => {
    if (!isCurrencyCode(code)) return;
    setCurrencyState(code);
    try {
      localStorage.setItem(CURRENCY_KEY, code);
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

    // Les montants du site sont en XOF ; formatPrice les convertit dans la devise choisie.
    const formatPrice = (amountXof) => formatMoney(amountXof, currency, locale);

    return { lang, setLang, t, locale, formatPrice, currency, setCurrency, currencies: Object.keys(CURRENCIES) };
  }, [lang, setLang, currency, setCurrency]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}
