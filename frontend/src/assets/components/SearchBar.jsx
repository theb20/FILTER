import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Search } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";

/**
 * Barre de recherche réutilisable : input blanc collé à un bouton carré jaune.
 *
 * Props :
 * - id            (obligatoire) id de l'input, relié au label
 * - className     classes du conteneur (largeur, marges…)
 * - placeholder   texte d'aide (défaut : texte traduit)
 * - label         libellé accessible (masqué visuellement)
 * - action        route cible de la recherche (défaut "/category")
 * - param         nom du paramètre d'URL (défaut "q")
 * - defaultValue  valeur initiale
 * - onSearch      callback(term) appelé à la place de la navigation si fourni
 */
export default function SearchBar({
  id,
  className = "",
  placeholder,
  label,
  action = "/category",
  param = "q",
  defaultValue = "",
  onSearch,
}) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [term, setTerm] = useState(defaultValue);
  const text = placeholder ?? t("search.placeholder");

  const submit = (e) => {
    e.preventDefault();
    const value = term.trim();
    if (onSearch) {
      onSearch(value);
      return;
    }
    navigate(value ? `${action}?${param}=${encodeURIComponent(value)}` : action);
  };

  return (
    <form role="search" onSubmit={submit} className={`flex ${className}`}>
      <label htmlFor={id} className="sr-only">
        {label ?? text}
      </label>
      <input
        id={id}
        type="search"
        value={term}
        onChange={(e) => setTerm(e.target.value)}
        placeholder={text}
        className="min-w-0 flex-1 rounded-l-[3px] bg-white px-4 py-2.5 text-black outline-none placeholder:text-gray-500 focus:ring-2 focus:ring-action"
      />
      <button
        type="submit"
        aria-label={t("search.button")}
        className="flex w-12 shrink-0 items-center justify-center rounded-r-[3px] bg-action text-black hover:bg-action-dark"
      >
        <Search size={20} aria-hidden="true" />
      </button>
    </form>
  );
}
