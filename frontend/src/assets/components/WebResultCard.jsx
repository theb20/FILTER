import { Link } from "react-router-dom";
import { ExternalLink, Globe } from "lucide-react";
import { rememberWebHit, webHitId } from "../lib/catalog.js";
import LineArt from "./LineArt.jsx";
import { useI18n } from "../i18n/i18nContext.js";

// Résultat d'une recherche web présenté comme une carte du site. AUCUN PRIX n'est affiché : une source web n'en donne
// pas de fiable, et le prix d'une pièce dépend ici du devis. La carte mène donc à une demande de devis, avec la
// désignation déjà reprise, ou à la page d'origine.
export default function WebResultCard({ hit }) {
  const { t } = useI18n();
  const quoteLink = `/quote?ref=${encodeURIComponent(hit.title.slice(0, 120))}`;
  // La fiche s'ouvre SUR LE SITE : le résultat est confié à l'onglet, la page le relit par son identifiant.
  const detail = `/web/${webHitId(hit.url)}`;
  const open = () => rememberWebHit(hit);
  return (
    <article className="group flex h-full flex-col overflow-hidden rounded-[10px] bg-white shadow-card ring-1 ring-gray-100 transition duration-200 hover:-translate-y-1 hover:shadow-xl hover:ring-action">
      <Link to={detail} state={{ hit }} onClick={open} className="flex h-36 items-center justify-center overflow-hidden bg-gray-50" aria-label={hit.title}>
        {hit.image
          ? <img src={hit.image} alt="" loading="lazy" referrerPolicy="no-referrer" onError={(e) => { e.currentTarget.style.display = "none"; }} className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105" />
          : <LineArt stroke="#9ca3af" className="h-20" />}
      </Link>
      <div className="flex flex-1 flex-col p-4">
        {hit.site && (
          <p className="flex items-center gap-1.5 text-[12px] font-semibold uppercase tracking-wide text-gray-500">
            <Globe size={13} aria-hidden="true" /> <span className="truncate">{hit.site}</span>
          </p>
        )}
        <h3 className="mt-1.5 line-clamp-2 text-[16px] font-semibold leading-snug">
          <Link to={detail} state={{ hit }} onClick={open} className="text-night hover:text-link hover:underline">{hit.title}</Link>
        </h3>
        {hit.description && <p className="mt-2 line-clamp-3 text-[13.5px] leading-snug text-gray-600">{hit.description}</p>}
        <div className="mt-4 flex flex-wrap items-center gap-2 pt-1">
          <Link to={detail} state={{ hit }} onClick={open} className="inline-flex items-center gap-1.5 rounded-[3px] bg-action px-4 py-2 text-[14px] font-bold text-black transition-colors hover:bg-action-dark">
            {t("web.card.see")} <span aria-hidden="true">→</span>
          </Link>
          <Link to={quoteLink} className="inline-flex items-center gap-1.5 rounded-[3px] border border-gray-300 px-3 py-2 text-[13.5px] font-medium text-gray-700 hover:bg-gray-50">
            {t("web.card.quote")}
          </Link>
        </div>
      </div>
    </article>
  );
}
