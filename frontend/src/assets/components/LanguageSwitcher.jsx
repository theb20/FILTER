import { useEffect, useId, useRef, useState } from "react";
import { Globe, ChevronDown, Check } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";
import { LANGUAGES } from "../i18n/I18nProvider.jsx";

// Sélecteur de langue (EN / FR / ES), utilisé dans toutes les variantes du header.
export default function LanguageSwitcher({ align = "right", className = "" }) {
  const { lang, setLang, t } = useI18n();
  const [open, setOpen] = useState(false);
  const ref = useRef(null);
  const menuId = useId();

  useEffect(() => {
    if (!open) return;
    const onDown = (e) => {
      if (ref.current && !ref.current.contains(e.target)) setOpen(false);
    };
    const onKey = (e) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={ref} className={`relative ${className}`}>
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={menuId}
        aria-label={`${t("lang.label")}: ${t(`lang.${lang}`)}`}
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 text-sm text-white hover:text-action"
      >
        <Globe size={18} aria-hidden="true" />
        <span className="font-medium uppercase">{lang}</span>
        <ChevronDown size={14} aria-hidden="true" />
      </button>

      {open && (
        <ul
          id={menuId}
          role="listbox"
          aria-label={t("lang.label")}
          className={`absolute top-full z-[1100] mt-2 min-w-[10rem] rounded-[4px] bg-white py-1 text-black shadow-lg ${
            align === "right" ? "right-0" : "left-0"
          }`}
        >
          {LANGUAGES.map(({ code, label }) => (
            <li key={code} role="option" aria-selected={code === lang}>
              <button
                type="button"
                onClick={() => {
                  setLang(code);
                  setOpen(false);
                }}
                className="flex w-full items-center justify-between gap-4 px-4 py-2 text-left hover:bg-gray-100"
              >
                {label}
                {code === lang && <Check size={16} aria-hidden="true" />}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
