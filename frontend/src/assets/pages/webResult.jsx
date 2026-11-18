import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Check, Copy, ExternalLink, FileText, Headphones, Minus, Plus, ShieldCheck, Store, Truck } from "lucide-react";
import LineArt from "../components/LineArt.jsx";
import ProductCard from "../components/ProductCard.jsx";
import WebResultCard from "../components/WebResultCard.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { useAuth } from "../store/authStore.js";
import { useSite } from "../store/siteStore.js";
import { fetchProducts, guessReference, recallWebHit, recallWebResults, searchWebImages, useAsync, webHitId } from "../lib/catalog.js";

// Mots d'au moins quatre lettres, hors mots vides : servent à rapprocher le produit trouvé de notre propre catalogue.
const STOP = new Set(["pour", "avec", "sans", "dans", "tous", "tout", "votre", "notre", "plus", "filtre", "filtres"]);
const keywords = (title) => [...new Set(String(title).toLowerCase().match(/[a-zà-ÿ0-9]{4,}/g) ?? [])].filter((w) => !STOP.has(w)).slice(0, 8);

/** Fiche produit complète d'un résultat web, bâtie sur le plan d'une grande place de marché (JoyBuy) : fil d'Ariane,
 *  galerie à vignettes verticales avec loupe, colonne d'achat, engagements, puis onglets.
 *  Une seule différence de fond, assumée : AUCUN PRIX. Il se fixe au devis, selon la quantité et le délai. */
export default function WebResult() {
  const { t, lang } = useI18n();
  const { id } = useParams();
  const { state } = useLocation();
  const navigate = useNavigate();
  const accessToken = useAuth((a) => a.accessToken);
  const SITE = useSite();
  const hit = state?.hit ?? recallWebHit(id);
  const series = recallWebResults();

  const [images, setImages] = useState([]);
  const [imgState, setImgState] = useState("idle"); // idle | loading | done | none | error | limit
  const [active, setActive] = useState(0);
  const [qty, setQty] = useState(1);
  const [copied, setCopied] = useState(false);
  const [tab, setTab] = useState("desc");
  const [zoom, setZoom] = useState(null); // { x, y } en pourcentage, pendant le survol
  const frame = useRef(null);

  const reference = useMemo(() => (hit ? guessReference(hit.title) : ""), [hit]);

  useEffect(() => {
    if (hit) document.title = `${hit.title} — ${t("web.detail.title")}`;
  }, [hit, t]);

  // Les photos sont cherchées à l'ouverture de la fiche : c'est le geste du visiteur qui déclenche l'appel payant.
  useEffect(() => {
    if (!hit) return undefined;
    let live = true;
    setImgState("loading");
    searchWebImages(hit.title, accessToken).then((r) => {
      if (!live) return;
      if (r.ok && r.data.status === "ok" && (r.data.images?.length ?? 0) > 0) { setImages(r.data.images); setImgState("done"); return; }
      setImgState(r.ok ? "none" : r.status === 429 ? "limit" : "error");
    });
    return () => { live = false; };
  }, [hit, accessToken]);

  // Équivalents possibles dans notre catalogue : rapprochement sur les mots de la désignation.
  const catalog = useAsync(() => fetchProducts().catch(() => []), [lang]).data ?? [];
  const similar = useMemo(() => {
    if (!hit) return [];
    const words = keywords(hit.title);
    if (words.length === 0) return [];
    return catalog
      .map((p) => ({ p, score: words.filter((w) => `${p.name} ${p.reference} ${p.brand}`.toLowerCase().includes(w)).length }))
      .filter((x) => x.score > 0).sort((a, b) => b.score - a.score).slice(0, 5).map((x) => x.p);
  }, [catalog, hit]);

  if (!hit) {
    return (
      <div className="mx-2.5 mt-2.5 rounded-[4px] bg-white p-8 text-center shadow-card">
        <p className="text-gray-700">{t("web.detail.lost")}</p>
        <button type="button" onClick={() => navigate(-1)} className="mt-4 inline-flex items-center gap-2 rounded-[3px] bg-night px-5 py-2.5 font-bold text-white hover:bg-navy">{t("web.detail.back")}</button>
      </div>
    );
  }

  const gallery = images.length > 0 ? images : (hit.image ? [{ url: hit.image, title: hit.title }] : []);
  const main = gallery[Math.min(active, Math.max(0, gallery.length - 1))];
  const all = series?.hits ?? [];
  const rank = all.findIndex((h) => webHitId(h.url) === id);
  const others = all.filter((h) => webHitId(h.url) !== id);
  const quoteLink = `/quote?ref=${encodeURIComponent((reference || hit.title).slice(0, 120))}&qty=${qty}`;
  const copyRef = async () => {
    try { await navigator.clipboard.writeText(reference); setCopied(true); setTimeout(() => setCopied(false), 1800); } catch { /* presse-papiers refusé */ }
  };
  const onMove = (e) => {
    const r = frame.current?.getBoundingClientRect();
    if (r) setZoom({ x: ((e.clientX - r.left) / r.width) * 100, y: ((e.clientY - r.top) / r.height) * 100 });
  };

  const promises = [
    [Store, t("web.detail.soldBy"), SITE.publisher || SITE.name],
    [FileText, t("footer.perk.quote.t"), t("footer.perk.quote.d")],
    [Truck, t("web.detail.shipping"), SITE.claims.freeShippingThreshold ? t("web.detail.freeFrom", { amount: SITE.claims.freeShippingThreshold }) : t("web.detail.shippingQuoted")],
    [ShieldCheck, t("footer.perk.pay.t"), t("footer.perk.pay.d")],
    [Headphones, t("footer.perk.help.t"), t("footer.perk.help.d")],
  ];
  const TABS = [["desc", t("web.detail.description")], ["info", t("web.detail.important")], ["specs", t("web.detail.specs")], ["others", `${t("web.detail.others")}${others.length ? ` (${others.length})` : ""}`]];
  const specs = [
    [t("web.detail.spec.site"), hit.site],
    [t("web.detail.spec.ref"), reference],
    [t("web.detail.spec.availability"), t("web.detail.onRequest")],
    [t("web.detail.spec.qty"), String(qty)],
    [t("web.detail.spec.origin"), hit.url],
  ].filter(([, v]) => v);

  const panel = "rounded-[4px] bg-white shadow-card";
  return (
    <div className="mx-2.5 mt-2.5 space-y-2.5 pb-2.5">
      {/* 1. Fil d'Ariane */}
      <nav aria-label={t("web.detail.breadcrumb")} className={`${panel} px-5 py-2.5 text-[13px]`}>
        <ol className="flex flex-wrap items-center gap-1.5 text-gray-600">
          <li><Link to="/" className="hover:text-link hover:underline">{t("nav.home")}</Link></li>
          <li aria-hidden="true">›</li>
          <li>
            <button type="button" onClick={() => navigate(-1)} className="inline-flex items-center gap-1 hover:text-link hover:underline">
              <ArrowLeft size={12} aria-hidden="true" /> {series?.query ? t("web.detail.resultsFor", { q: series.query }) : t("web.detail.back")}
            </button>
          </li>
          <li aria-hidden="true">›</li>
          <li className="max-w-[46ch] truncate font-medium text-night" aria-current="page">{hit.title}</li>
        </ol>
      </nav>

      {/* 2. Galerie + colonne d'achat */}
      <div className={`${panel} grid gap-6 p-5 lg:grid-cols-[minmax(0,560px)_minmax(0,1fr)]`}>
        <section aria-label={t("web.detail.photos")} className="flex gap-3">
          {/* vignettes verticales, à la façon des grandes places de marché */}
          {gallery.length > 1 && (
            <ul className="flex w-[68px] shrink-0 flex-col gap-2 overflow-y-auto" style={{ maxHeight: 420 }}>
              {gallery.slice(0, 12).map((im, i) => (
                <li key={i}>
                  <button type="button" onMouseEnter={() => setActive(i)} onFocus={() => setActive(i)} onClick={() => setActive(i)}
                    aria-label={`${t("web.detail.photo")} ${i + 1}`} aria-current={i === active}
                    className={`flex h-[64px] w-full items-center justify-center overflow-hidden rounded-[4px] bg-gray-50 ring-1 transition ${i === active ? "ring-2 ring-action" : "ring-gray-200 hover:ring-night/40"}`}>
                    <img src={im.url} alt="" loading="lazy" referrerPolicy="no-referrer" className="h-full w-full object-cover" />
                  </button>
                </li>
              ))}
            </ul>
          )}
          <div className="min-w-0 flex-1">
            <div ref={frame} onMouseMove={onMove} onMouseLeave={() => setZoom(null)}
              className="relative flex h-[420px] items-center justify-center overflow-hidden rounded-[6px] bg-gray-50">
              {main
                ? <img src={main.url} alt={main.title || hit.title} referrerPolicy="no-referrer"
                    style={zoom ? { transform: "scale(2)", transformOrigin: `${zoom.x}% ${zoom.y}%` } : undefined}
                    className="h-full w-full object-contain transition-transform duration-150" />
                : <LineArt stroke="#9ca3af" className="h-32" />}
              {gallery.length > 1 && <span className="absolute bottom-2.5 right-2.5 rounded-full bg-night/80 px-2.5 py-0.5 text-[11px] font-medium text-white">{active + 1} / {gallery.length}</span>}
              {main && <span className="pointer-events-none absolute bottom-2.5 left-2.5 rounded-full bg-white/90 px-2.5 py-0.5 text-[11px] text-gray-600">{t("web.detail.zoom")}</span>}
            </div>
            <p className="mt-2 text-[11.5px] text-gray-500">
              {imgState === "loading" ? t("web.detail.loadingPhotos")
                : imgState === "limit" ? t("web.limit")
                : imgState === "error" || imgState === "none" ? t("web.detail.noPhoto")
                : t("web.detail.photosFrom", { n: gallery.length })}
            </p>
          </div>
        </section>

        {/* colonne d'achat */}
        <section aria-labelledby="web-detail-title" className="min-w-0">
          {hit.site && (
            <p className="flex flex-wrap items-center gap-2 text-[12.5px]">
              <span className="font-semibold uppercase tracking-wide text-night">{hit.site}</span>
              <a href={hit.url} target="_blank" rel="noopener noreferrer nofollow" className="inline-flex items-center gap-1 text-link hover:underline">
                {t("web.detail.showStore")} <ExternalLink size={11} aria-hidden="true" />
              </a>
            </p>
          )}
          <h1 id="web-detail-title" className="mt-1.5 text-[22px] font-bold leading-snug text-night">{hit.title}</h1>

          {/* bandeau de repères, à la place des avis et des ventes */}
          <div className="mt-2.5 flex flex-wrap items-center gap-2 text-[12.5px]">
            <span className="rounded-[3px] bg-action px-2 py-0.5 font-bold text-black">{t("web.detail.webResult")}</span>
            {rank >= 0 && all.length > 0 && <span className="text-gray-600">{t("web.detail.rank", { n: rank + 1, total: all.length })}</span>}
            {reference && (
              <button type="button" onClick={copyRef} className="inline-flex items-center gap-1.5 rounded-[3px] border border-gray-300 px-2 py-0.5 font-medium text-gray-700 hover:bg-gray-50">
                {t("web.detail.spec.ref")} <strong className="text-night">{reference}</strong>
                {copied ? <Check size={12} aria-hidden="true" /> : <Copy size={12} aria-hidden="true" />}
              </button>
            )}
          </div>

          {/* bloc « prix » : chez nous, le prix vient du devis */}
          <div className="mt-3 rounded-[6px] bg-[#fff8e1] p-4">
            <p className="text-[26px] font-extrabold leading-none text-night">{t("price.onQuote")}</p>
            <p className="mt-1.5 text-[13px] text-gray-700">{t("web.detail.noPrice")}</p>
          </div>

          {/* quantité puis actions, l'une sous l'autre */}
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <span className="text-[13px] font-medium text-gray-700">{t("web.detail.qty")}</span>
            <div className="inline-flex items-center rounded-[4px] border border-gray-300">
              <button type="button" onClick={() => setQty((n) => Math.max(1, n - 1))} aria-label={t("web.detail.less")} className="px-3 py-2 text-gray-600 hover:bg-gray-50"><Minus size={14} aria-hidden="true" /></button>
              <input type="number" min={1} max={100000} value={qty} aria-label={t("web.detail.qty")}
                onChange={(e) => setQty(Math.max(1, Math.min(100000, Math.round(Number(e.target.value) || 1))))}
                className="w-16 border-x border-gray-300 py-2 text-center text-night focus:outline-none" />
              <button type="button" onClick={() => setQty((n) => Math.min(100000, n + 1))} aria-label={t("web.detail.more")} className="px-3 py-2 text-gray-600 hover:bg-gray-50"><Plus size={14} aria-hidden="true" /></button>
            </div>
          </div>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <Link to={quoteLink} className="inline-flex items-center justify-center gap-2 rounded-[4px] bg-action px-5 py-3 text-[15px] font-bold text-black transition-colors hover:bg-action-dark">
              {t("web.card.quote")} <span aria-hidden="true">→</span>
            </Link>
            <a href={hit.url} target="_blank" rel="noopener noreferrer nofollow" className="inline-flex items-center justify-center gap-2 rounded-[4px] border border-night px-5 py-3 text-[15px] font-bold text-night transition-colors hover:bg-night hover:text-white">
              {t("web.detail.origin")} <ExternalLink size={14} aria-hidden="true" />
            </a>
          </div>

          {/* engagements */}
          <ul className="mt-5 space-y-2 rounded-[6px] border border-gray-200 p-4">
            {promises.map(([Icon, label, value]) => (
              <li key={label} className="flex gap-2.5 text-[13px] leading-snug">
                <Icon size={15} className="mt-0.5 shrink-0 text-night" aria-hidden="true" />
                <span><strong className="text-night">{label}</strong> <span className="text-gray-600">— {value}</span></span>
              </li>
            ))}
          </ul>
        </section>
      </div>

      {/* 3. Onglets */}
      <div className={panel}>
        <div role="tablist" aria-label={t("web.detail.sections")} className="flex flex-wrap gap-1 border-b border-gray-200 px-3 pt-3">
          {TABS.map(([k, label]) => (
            <button key={k} type="button" role="tab" aria-selected={tab === k} onClick={() => setTab(k)}
              className={`rounded-t-[4px] px-4 py-2.5 text-[14px] font-semibold transition ${tab === k ? "bg-night text-white" : "text-gray-600 hover:bg-gray-50"}`}>
              {label}
            </button>
          ))}
        </div>

        <div role="tabpanel" className="p-6">
          {tab === "desc" && (
            <>
              {hit.description ? <p className="text-[15px] leading-relaxed text-gray-800">{hit.description}</p> : <p className="text-gray-600">{t("web.detail.noDescription")}</p>}
              {hit.snippets?.length > 0 && (
                <ul className="mt-4 space-y-2.5 text-[14.5px] leading-snug text-gray-700">
                  {hit.snippets.map((sn, i) => (
                    <li key={i} className="flex gap-2"><span aria-hidden="true" className="mt-[8px] h-1.5 w-1.5 shrink-0 rounded-full bg-action" />{sn}</li>
                  ))}
                </ul>
              )}
            </>
          )}

          {tab === "info" && (
            <div className="space-y-3 text-[14.5px] leading-relaxed text-gray-800">
              <p>{t("web.detail.notice")}</p>
              <p>{t("web.detail.checkRef")}</p>
              <p className="text-gray-600">{t("web.source", { name: series?.source || "Brave Search" })}</p>
            </div>
          )}

          {tab === "specs" && (
            <dl className="divide-y divide-gray-100 text-[14.5px]">
              {specs.map(([k, v]) => (
                <div key={k} className="grid gap-1 py-2.5 sm:grid-cols-[240px_minmax(0,1fr)]">
                  <dt className="text-gray-500">{k}</dt>
                  <dd className="break-words text-night">
                    {k === t("web.detail.spec.origin") ? <a href={v} target="_blank" rel="noopener noreferrer nofollow" className="text-link hover:underline">{v}</a> : v}
                  </dd>
                </div>
              ))}
            </dl>
          )}

          {tab === "others" && (
            others.length > 0
              ? <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">{others.map((h, i) => <li key={i}><WebResultCard hit={h} /></li>)}</ul>
              : <p className="text-gray-600">{t("web.detail.noOthers")}</p>
          )}
        </div>
      </div>

      {/* 4. Équivalents de notre catalogue */}
      {similar.length > 0 && (
        <section className={`${panel} p-6`} aria-labelledby="web-similar-title">
          <h2 id="web-similar-title" className="text-[20px] font-semibold">{t("web.detail.similar")}</h2>
          <p className="mt-1 text-sm text-gray-600">{t("web.detail.similarHint")}</p>
          <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            {similar.map((p) => <li key={p.id}><ProductCard product={p} variant="tile" /></li>)}
          </ul>
        </section>
      )}
    </div>
  );
}
