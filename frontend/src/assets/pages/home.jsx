import { useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { ChevronLeft, ChevronRight } from "lucide-react";
import LineArt from "../components/LineArt.jsx";
import Todo from "../components/Todo.jsx";
import Hero from "../components/home/Hero.jsx";
import Latest from "../components/home/Latest.jsx";
import Reviews from "../components/Reviews.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { fetchBrands, fetchCategories, fetchSectors, useAsync } from "../lib/catalog.js";
import { fetchHomeSettings, fetchPage, fetchPartners, leadParagraphs } from "../lib/content.js";
import { MarkdownInline } from "../components/Markdown.jsx";
import { useSite } from "../store/siteStore.js";

const btnPrimary =
  "inline-flex items-center gap-2 rounded-[3px] bg-action px-5 py-2.5 font-bold text-black hover:bg-action-dark transition-colors";

export default function Home() {
  const { t, lang } = useI18n();
  const CLAIMS = useSite().claims;
  const CATEGORIES = useAsync(fetchCategories, [lang]).data ?? [];
  const SECTORS = useAsync(fetchSectors, [lang]).data ?? [];
  // Partenaires saisis au backoffice ; à défaut, les logos des marques.
  const partners = useAsync(() => fetchPartners().catch(() => []), [lang]).data ?? [];
  const brandLogos = (useAsync(fetchBrands, [lang]).data ?? []).filter((b) => b.logoUrl).map((b) => ({ to: b.logoUrl, label: b.name }));
  const logoLinks = partners.length > 0 ? partners.map((p) => ({ to: p.logoUrl, label: p.name, href: p.link || "" })) : brandLogos;
  const home = useAsync(() => fetchHomeSettings().catch(() => ({})), [lang]).data ?? {};
  // Texte saisi au backoffice dans la langue courante (à défaut le français), sinon le texte par défaut du site.
  const own = (key, fallback) => home[key]?.[lang] || home[key]?.fr || fallback;
  // « Notre histoire » : page `about` publiée au backoffice (Contenu du site › Pages) ; l'accueil en montre le début.
  const story = useAsync(() => fetchPage("about").catch(() => null), [lang]).data;
  const storyLead = story ? leadParagraphs(story.body, 3) : [];
  const carousel = useRef(null);


  useEffect(() => {
    document.title = t("meta.home");
  }, [t]);

  const scrollCarousel = (dir) => {
    carousel.current?.scrollBy({ left: dir * 340, behavior: "smooth" });
  };

  return (
    <div className="space-y-2.5 pb-0 pt-2.5">
      <Hero />
      <Latest />

      {/* 3. Partenaires + catégories */}
      <section className="mx-2.5 rounded-[4px] bg-white p-6 shadow-card md:p-10" aria-labelledby="partners-title">
        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <h2 id="partners-title" className="text-[37px] font-bold leading-tight">{t("home.partners.title")}</h2>
          <Link to="/brands" className="font-medium text-link hover:underline">
            {t("home.partners.all")}
          </Link>
        </div>

        <ul className="mx-auto mt-8 grid grid-cols-3 sm:grid-cols-4 lg:grid-cols-8">
          {logoLinks.map((logo) => (
            <li
              key={logo.to}
              className="flex h-28 items-center justify-center "
              title={t("home.partners.logoTitle")}
            >
              {logo.href ? <a href={logo.href} target="_blank" rel="noopener noreferrer" className="flex h-full w-full items-center justify-center"><img src={logo.to} alt={logo.label} loading="lazy" className="max-h-full max-w-full object-contain" /></a>
                : <img src={logo.to} alt={logo.label} loading="lazy" className="max-h-full max-w-full object-contain" />}
            </li>
          ))}
        </ul>

      </section>

      {/* Achetez par catégorie */}
      {CATEGORIES.length > 0 && (
        <section data-guide="home-categories" className="mx-2.5 rounded-[6px] bg-white px-4 py-10 shadow-card sm:px-8" aria-labelledby="shop-title">
          <h2 id="shop-title" className="text-[28px] font-bold tracking-tight sm:text-[32px]">{t("home.shop.title")}</h2>
          <ul className="mx-auto mt-6 grid max-w-[1400px] grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
            {CATEGORIES.map((c) => (
              <li key={c.slug}>
                <Link to={`/category?cat=${c.slug}`} className="group relative flex h-full min-h-[190px] flex-col justify-between overflow-hidden rounded-[10px] bg-gradient-to-br from-gray-50 to-gray-100 p-4 ring-1 ring-gray-200 transition duration-200 hover:-translate-y-1 hover:shadow-xl hover:ring-action">
                  {c.imageUrl
                    ? <img src={c.imageUrl} alt="" loading="lazy" className="pointer-events-none absolute inset-0 h-full w-full object-cover opacity-90 transition-transform duration-300 group-hover:scale-105" />
                    : <LineArt stroke="#cbd2dc" className="pointer-events-none absolute -bottom-4 -right-6 h-40 opacity-80 transition-transform duration-300 group-hover:scale-110" />}
                  {c.imageUrl && <span className="pointer-events-none absolute inset-0 bg-gradient-to-t from-white via-white/70 to-transparent" aria-hidden="true" />}
                  <span className="relative text-[17px] font-bold leading-snug text-night">{c.name}</span>
                  <span className="relative mt-6">
                    {typeof c.productCount === "number" && <span className="block text-[13px] text-gray-500">{t("home.shop.items", { n: c.productCount })}</span>}
                    <span className="mt-1 inline-flex items-center gap-1 text-[14px] font-semibold text-link group-hover:underline">{t("home.shop.go")} <span aria-hidden="true">→</span></span>
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* 4. Secteurs */}
      <section data-guide="home-sectors" className="mx-2.5 grid gap-8 overflow-hidden py-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]" aria-labelledby="sectors-title">
        <div className="px-4 lg:px-10">
          <p className="font-medium text-link">{own("sectorsEyebrow", t("home.sectors.eyebrow"))}</p>
          <h2 id="sectors-title" className="mt-1 text-[37px] font-bold leading-tight">{own("sectorsTitle", t("home.sectors.title"))}</h2>
          <p className="mt-4">{own("sectorsText", t("home.sectors.text"))}</p>
          <div className="mt-6 flex gap-3">
            <button type="button" aria-label={t("home.sectors.prev")} onClick={() => scrollCarousel(-1)} className="flex h-10 w-10 items-center justify-center rounded-full border border-black hover:bg-black/5">
              <ChevronLeft size={18} />
            </button>
            <button type="button" aria-label={t("home.sectors.next")} onClick={() => scrollCarousel(1)} className="flex h-10 w-10 items-center justify-center rounded-full border border-black hover:bg-black/5">
              <ChevronRight size={18} />
            </button>
          </div>
        </div>

        <ul ref={carousel} className="flex snap-x snap-mandatory gap-4 overflow-x-auto px-4 pb-2 [scrollbar-width:none]">
          {SECTORS.map(({ id, slug, name, imageUrl }) => (
            <li key={id} className="w-[280px] shrink-0 snap-start overflow-hidden rounded-[4px] bg-white shadow-card transition duration-200 hover:-translate-y-1 hover:shadow-xl">
              <Link to={`/category?sector=${slug}`} className="block focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-night">
              <div className="h-40 bg-gray-100" title={t("home.sectors.photo")}>
                {imageUrl && <img
                  src={imageUrl}
                  alt=""
                  loading="lazy"
                  onError={(e) => { e.currentTarget.style.display = "none"; }}
                  className="h-full w-full object-cover"
                />}
              </div>
              <div className="p-4">
                <h3 className="text-[24px] font-semibold leading-tight">{name}</h3>
                <span className="mt-2 inline-flex items-center gap-1 text-[14px] font-semibold text-link">{t("home.sectors.go")} <span aria-hidden="true">→</span></span>
              </div>
              </Link>
            </li>
          ))}
        </ul>
      </section>

      {/* 5. Histoire */}
      <section className="relative mx-2.5 overflow-hidden rounded-[4px] bg-navy text-white" aria-labelledby="story-title">
        <LineArt className="pointer-events-none absolute -left-10 top-0 h-full opacity-10" />
        <div className="relative grid gap-8 md:grid-cols-2">
          <div className="px-6 py-12 md:px-12">

            <h2 id="story-title" className="text-[37px] font-bold leading-tight">
              {t("home.story.title")}{" "}
              <span className="text-action">
                {CLAIMS.foundedYear ? t("home.story.since", { year: CLAIMS.foundedYear }) : <Todo>{t("todo.sinceYear")}</Todo>}
              </span>
            </h2>

            <div className="mt-4 space-y-3">
              {storyLead.length > 0
                ? storyLead.map((p, i) => <p key={i}><MarkdownInline text={p} /></p>)
                : <p><Todo>{t("todo.story")}</Todo></p>}
            </div>

            <Link to="/about" className={`${btnPrimary} mt-6`}>
              {t("home.story.button")} <span aria-hidden="true">→</span>
            </Link>
          </div>
          <div className="relative h-[160px] bg-slate/60 md:-mr-10 md:h-auto">
            <img src={story?.imageUrl || "/img/background/bg_history.png"} className="absolute inset-0 h-full w-full object-cover" alt="" />
          </div>
        </div>
      </section>

      {/* 6. Avis */}
      <Reviews />
    </div>
  );
}
