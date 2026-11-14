import { useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { ArrowRight, ChevronLeft, ChevronRight } from "lucide-react";
import LineArt from "../components/LineArt.jsx";
import Todo from "../components/Todo.jsx";
import SearchBar from "../components/SearchBar.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { CATEGORIES } from "../data/demoProducts.js";
import { CLAIMS, SITE } from "../config/site.js";

const btnPrimary =
  "inline-flex items-center gap-2 rounded-[3px] bg-action px-5 py-2.5 font-bold text-black hover:bg-action-dark transition-colors";

const SECTORS = ["aerospace", "defense", "medical", "energy", "marine", "datacenters", "industry", "transport", "rd"];

function RoundArrow({ to, label, light = false }) {
  return (
    <Link
      to={to}
      aria-label={label}
      className={`flex h-10 w-10 items-center justify-center rounded-full border ${
        light ? "border-white text-white hover:bg-white/15" : "border-black text-black hover:bg-black/5"
      }`}
    >
      <ArrowRight size={18} />
    </Link>
  );
}

function PhotoCard({ title, subtitle, to }) {
  return (
    <div className="relative flex min-h-[150px] flex-col justify-between overflow-hidden rounded-[4px] bg-gradient-to-br from-navy to-night p-5 text-white md:aspect-[2.5/1]">
      <LineArt className="pointer-events-none absolute -right-6 -top-4 h-[120%] opacity-20" />
      <div className="relative">
        <h2 className="text-[24px] font-semibold leading-tight">{title}</h2>
        <p className="text-sm text-white/85">{subtitle}</p>
      </div>
      <div className="relative flex justify-end">
        <RoundArrow to={to} label={title} light />
      </div>
    </div>
  );
}

export default function Home() {
  const { t } = useI18n();
  const carousel = useRef(null);
  const heroVideo = useRef(null);

  useEffect(() => {
    document.title = t("meta.home");
  }, [t]);

  // Respecte « réduire les animations » : on garde l'image fixe
  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      heroVideo.current?.pause();
    }
  }, []);

  const scrollCarousel = (dir) => {
    carousel.current?.scrollBy({ left: dir * 340, behavior: "smooth" });
  };

  return (
    <div className="space-y-2.5 pb-0 pt-2.5">
      {/* 1. Hero */}
      <section className="relative mx-2.5 overflow-hidden rounded-[4px] bg-navy text-white" aria-labelledby="hero-title">
        <video
          ref={heroVideo}
          className="pointer-events-none absolute inset-0 h-full w-full object-cover"
          autoPlay
          loop
          muted
          playsInline
          preload="metadata"
          poster="/img/background/hero-poster.webp"
          aria-hidden="true"
          tabIndex={-1}
        >
          <source src="/videos/hero-web.mp4" type="video/mp4" />
        </video>
        <div className="absolute inset-0 bg-gradient-to-r from-navy/95 via-navy/80 to-navy/40" aria-hidden="true" />
        <div className="relative max-w-3xl px-6 py-10 md:min-h-[300px] md:px-12">
          <h1 id="hero-title" className="text-[40px] font-bold leading-tight">
            {t("home.hero.title1")} <span className="text-action">{t("home.hero.title2")}</span>
          </h1>
          <p className="mt-3 max-w-xl">
            {CLAIMS.crossReferences || <Todo>{t("todo.number")}</Todo>} {t("home.hero.crossRefs")}{" "}
            {CLAIMS.stockValue || <Todo>{t("todo.stockValue")}</Todo>} {t("home.hero.ofStock")}
          </p>
          <SearchBar id="hero-search" className="mt-6 max-w-xl" />
        </div>
      </section>

      {/* 2. Trois cartes */}
      <section className="mx-2.5 grid gap-2.5 md:grid-cols-3" aria-label={t("home.cards.aria")}>
        <PhotoCard title={t("home.cards.shipping")} subtitle={t("home.cards.shippingSub")} to="/contact" />
        <PhotoCard title={t("home.cards.account")} subtitle={t("home.cards.accountSub")} to="/sign" />
        <PhotoCard title={t("home.cards.chat")} subtitle={t("home.cards.chatSub")} to="/contact" />
      </section>

      {/* 3. Partenaires + catégories */}
      <section className="mx-2.5 rounded-[4px] bg-white p-6 shadow-card md:p-10" aria-labelledby="partners-title">
        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <h2 id="partners-title" className="text-[37px] font-bold leading-tight">{t("home.partners.title")}</h2>
          <Link to="/brands" className="font-medium text-link hover:underline">
            {t("home.partners.all")}
          </Link>
        </div>

        <ul className="mx-auto mt-8 grid max-w-5xl grid-cols-3 gap-x-6 gap-y-8 sm:grid-cols-4 lg:grid-cols-7">
          {Array.from({ length: 14 }, (_, i) => (
            <li
              key={i}
              className="flex h-14 items-center justify-center rounded-[3px] border border-dashed border-gray-300 text-xs text-gray-400"
              title={t("home.partners.logoTitle")}
            >
              {t("home.partners.logo")}
            </li>
          ))}
        </ul>

        <ul className="mt-12 grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-5">
          {CATEGORIES.map((c) => (
            <li key={c.slug}>
              <Link to={`/category?cat=${c.slug}`} className="block text-center hover:text-link">
                <div className="flex aspect-square items-center justify-center rounded-[3px] bg-white">
                  <LineArt stroke="#9ca3af" className="h-4/5" />
                </div>
                <span className="mt-2 block font-medium">{t(`cat.${c.slug}`)}</span>
              </Link>
            </li>
          ))}
        </ul>
      </section>

      {/* 4. Secteurs */}
      <section className="mx-2.5 grid gap-8 overflow-hidden py-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]" aria-labelledby="sectors-title">
        <div className="px-4 lg:px-10">
          <p className="font-medium text-link">{t("home.sectors.eyebrow")}</p>
          <h2 id="sectors-title" className="mt-1 text-[37px] font-bold leading-tight">{t("home.sectors.title")}</h2>
          <p className="mt-4">{t("home.sectors.text")}</p>
          <div className="mt-6 flex gap-3">
            <button type="button" aria-label={t("home.sectors.prev")} onClick={() => scrollCarousel(-1)} className="flex h-10 w-10 items-center justify-center rounded-full border border-black hover:bg-black/5">
              <ChevronLeft size={18} />
            </button>
            <button type="button" aria-label={t("home.sectors.next")} onClick={() => scrollCarousel(1)} className="flex h-10 w-10 items-center justify-center rounded-full border border-black hover:bg-black/5">
              <ChevronRight size={18} />
            </button>
          </div>
        </div>

        <ul ref={carousel} className="flex snap-x snap-mandatory gap-4 overflow-x-auto px-4 pb-2 [scrollbar-width:none] lg:-mr-[10vw]">
          {SECTORS.map((key) => (
            <li key={key} className="w-[280px] shrink-0 snap-start overflow-hidden rounded-[4px] bg-white shadow-card">
              <div className="flex h-40 items-center justify-center bg-gray-100" title={t("home.sectors.photo")}>
                <LineArt stroke="#cbd5e1" className="h-32" />
              </div>
              <div className="p-4">
                <h3 className="text-[24px] font-semibold leading-tight">{t(`sector.${key}`)}</h3>
                <p className="mt-1 text-sm text-gray-700">{t(`sector.${key}.text`)}</p>
              </div>
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
              <p><Todo>{t("todo.story1")}</Todo></p>
              <p><Todo>{t("todo.story2")}</Todo></p>
              <p><Todo>{t("todo.story3")}</Todo></p>
            </div>
            <Link to="/contact" className={`${btnPrimary} mt-6`}>
              {t("home.story.button")} <span aria-hidden="true">→</span>
            </Link>
          </div>
          <div className="flex min-h-[260px] items-center justify-center bg-slate/60 text-sm text-white/70 md:-mr-10">
            <Todo>{t("todo.archivePhoto")}</Todo>
          </div>
        </div>
      </section>

      {/* 6. Avis */}
      <section className="mx-2.5 px-4 py-12 md:px-10" aria-labelledby="reviews-title">
        <p className="font-medium text-link">{t("home.reviews.eyebrow")}</p>
        <h2 id="reviews-title" className="mt-1 text-[37px] font-bold leading-tight">{t("home.reviews.title")}</h2>
        <p className="mt-3 max-w-xl">{t("home.reviews.text", { name: SITE.name })}</p>
        {SITE.reviewsUrl ? (
          <a href={SITE.reviewsUrl} target="_blank" rel="noopener noreferrer" className={`${btnPrimary} mt-6`}>
            {t("home.reviews.button")} <span aria-hidden="true">→</span>
          </a>
        ) : (
          <p className="mt-6"><Todo>{t("todo.reviewsLink")}</Todo></p>
        )}
      </section>
    </div>
  );
}
