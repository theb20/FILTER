import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ChevronDown, ChevronLeft, ChevronRight, Search } from "lucide-react";
import { gsap, ScrollTrigger, prefersReducedMotion } from "../../animation/gsapSetup.js";
import OrderTracker from "../tracker/OrderTracker.jsx";
import { Box, Clipboard, House, Truck } from "../tracker/Illustrations.jsx";
import LineArt from "../LineArt.jsx";
import { useI18n } from "../../i18n/i18nContext.js";
import { useSite } from "../../store/siteStore.js";
import { useAuth } from "../../store/authStore.js";
import { fetchHeroCards } from "../../lib/content.js";
import { fetchBrands, fetchCategories, fetchStats, flattenCategories, useAsync } from "../../lib/catalog.js";

// Chiffre qui défile jusqu'à sa valeur quand le bandeau entre à l'écran (GSAP) ; le texte final reste lisible sans animation.
function CountUp({ value }) {
  const ref = useRef(null);
  useEffect(() => {
    const el = ref.current;
    const m = /^(\D*)([\d\s.,\u202f\u00a0]+)(\D*)$/.exec(String(value));
    const target = m ? Number(m[2].replace(/\D/g, "")) : NaN;
    if (!el || !m || !Number.isFinite(target) || prefersReducedMotion() || /^\s*(19|20)\d\d\s*$/.test(m[2])) return undefined;
    const fmt = new Intl.NumberFormat();
    const o = { n: 0 };
    const tw = gsap.to(o, { n: target, duration: 1.6, ease: "power2.out", paused: true, onUpdate: () => { el.textContent = `${m[1]}${fmt.format(Math.round(o.n))}${m[3]}`; }, onComplete: () => { el.textContent = String(value); } });
    el.textContent = `${m[1]}0${m[3]}`;
    const st = ScrollTrigger.create({ trigger: el, start: "top 90%", once: true, onEnter: () => tw.play() });
    return () => { st.kill(); tw.kill(); el.textContent = String(value); };
  }, [value]);
  return <span ref={ref}>{value}</span>;
}

const nf = (locale, n) => new Intl.NumberFormat(locale).format(n);

// ── Sélecteur en trois étapes, en une seule ligne (catégorie → marque → référence) ──
function Finder() {
  const { t, lang } = useI18n();
  const navigate = useNavigate();
  const cats = flattenCategories(useAsync(() => fetchCategories().catch(() => []), [lang]).data ?? []);
  const brands = useAsync(() => fetchBrands().catch(() => []), [lang]).data ?? [];
  const [cat, setCat] = useState("");
  const [brand, setBrand] = useState("");
  const [q, setQ] = useState("");

  const go = (e) => {
    e.preventDefault();
    const sp = new URLSearchParams();
    if (cat) sp.set("cat", cat);
    if (brand) sp.set("brand", brand);
    if (q.trim()) sp.set("q", q.trim());
    navigate(`/category${sp.toString() ? `?${sp}` : ""}`);
  };
  const box = "relative flex min-w-0 items-center rounded-full border bg-white transition-colors focus-within:border-night";
  const num = (n, on) => <span className={`ml-2.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[12px] font-bold ${on ? "bg-night text-white" : "bg-gray-100 text-gray-400"}`} aria-hidden="true">{n}</span>;
  const sel = "h-[50px] w-full min-w-0 cursor-pointer appearance-none bg-transparent pl-3 pr-9 text-[15px] text-gray-800 outline-none";

  return (
    <form onSubmit={go} data-guide="hero-search" className="mt-10" aria-labelledby="finder-title">
      <h2 id="finder-title" className="text-[26px] font-bold leading-tight tracking-tight text-night">{t("finder.title")}</h2>
      <p className="mt-0.5 text-[15px] text-gray-600">{t("finder.lead")}</p>
      <div className="mt-4 grid gap-3 md:grid-cols-[1fr_1fr_1fr_auto]">
        <label className={`${box} ${cat ? "border-night" : "border-gray-300"}`}>{num(1, Boolean(cat))}<span className="sr-only">{t("finder.cat")}</span>
          <select value={cat} onChange={(e) => setCat(e.target.value)} className={sel}><option value="">{t("finder.cat")}</option>{cats.map((c) => <option key={c.slug} value={c.slug}>{`${"— ".repeat(c.depth)}${c.name}`}</option>)}</select>
          <ChevronDown size={18} className="pointer-events-none absolute right-4 text-gray-500" aria-hidden="true" /></label>
        <label className={`${box} ${brand ? "border-night" : "border-gray-300"}`}>{num(2, Boolean(brand))}<span className="sr-only">{t("finder.brand")}</span>
          <select value={brand} onChange={(e) => setBrand(e.target.value)} className={sel}><option value="">{t("finder.brand")}</option>{brands.map((b) => <option key={b.id} value={b.name}>{b.name}</option>)}</select>
          <ChevronDown size={18} className="pointer-events-none absolute right-4 text-gray-500" aria-hidden="true" /></label>
        <label className={`${box} ${q ? "border-night" : "border-gray-300"}`}>{num(3, Boolean(q))}<span className="sr-only">{t("finder.ref")}</span>
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder={t("finder.ref")} className={`${sel} pr-4`} autoComplete="off" /></label>
        <button type="submit" className="flex h-[52px] items-center justify-center gap-2 rounded-full bg-night px-9 text-[15.5px] font-bold text-white transition-colors hover:bg-navy"><Search size={18} aria-hidden="true" /> {t("finder.search")}</button>
      </div>
      <p className="mt-3 text-[13.5px] font-semibold uppercase tracking-wide"><Link to="/quote" className="text-link hover:underline">{t("finder.notFound")}</Link></p>
    </form>
  );
}

// ── Rangée de grandes cartes : image en haut, panneau coloré en bas (titre, texte, « COMMENCER ▸ ») ──
function PromoRow() {
  const { t } = useI18n();
  const SITE = useSite();
  const user = useAuth((a) => a.user);
  const rail = useRef(null);
  const [edge, setEdge] = useState({ start: true, end: false });
  const wa = SITE.phone ? `https://wa.me/${SITE.phone.replace(/\D/g, "")}` : "";

  const { lang } = useI18n();
  const managed = useAsync(() => fetchHeroCards().catch(() => []), [lang]).data ?? [];
  // Cartes saisies au backoffice ; à défaut (aucune publiée), les cartes d'accès rapide par défaut.
  const defaults = [
    { key: "find", to: "/category", img: "/img/background/bg_card_1.png", bg: "#3d44e0", fg: "#fff" },
    { key: "quote", to: "/quote", img: "/img/background/bg_card_4.png", bg: "#b23a31", fg: "#fff" },
    { key: "refs", to: "/category", img: "/img/background/bg_card_6.png", bg: "#e6d6a0", fg: "#111" },
    { key: user ? "account" : "signin", to: user ? "/profil" : "/sign", img: "/img/background/bg_sign.webp", bg: "#1f6b45", fg: "#fff" },
    { key: "team", to: wa || "/contact", external: Boolean(wa), img: "/img/background/bg_card_9.png", bg: "#1b2433", fg: "#fff" },
  ].map((c) => ({ ...c, title: t(`hero.sc.${c.key}.t`), text: t(`hero.sc.${c.key}.d`), cta: t("hero.sc.go") }));
  const cards = managed.length > 0
    ? managed.map((c) => ({ key: c.id, to: c.link || "/", external: /^https?:\/\//.test(c.link || ""), img: c.imageUrl, bg: c.bgColor || "#1b2433", fg: c.textColor || "#fff", title: c.title, text: c.text || "", cta: c.cta || t("hero.sc.go") }))
    : defaults;

  const measure = () => {
    const el = rail.current;
    if (el) setEdge({ start: el.scrollLeft < 8, end: el.scrollLeft + el.clientWidth >= el.scrollWidth - 8 });
  };
  useEffect(measure, []);
  const scroll = (dir) => rail.current?.scrollBy({ left: dir * (rail.current.clientWidth * 0.5), behavior: "smooth" });
  const arrow = "absolute top-[34%] z-10 hidden h-14 w-14 -translate-y-1/2 items-center justify-center rounded-full bg-night/85 text-white shadow-xl backdrop-blur transition hover:scale-105 hover:bg-night md:flex";

  return (
    <div className="relative" data-guide="home-promos">
      {!edge.start && <button type="button" onClick={() => scroll(-1)} aria-label={t("hero.prev")} className={`${arrow} left-3`}><ChevronLeft size={26} aria-hidden="true" /></button>}
      <ul ref={rail} onScroll={measure} className="flex snap-x snap-mandatory gap-3 overflow-x-auto scroll-smooth pb-1 [scrollbar-width:none]" aria-label={t("hero.carousel")} tabIndex={0}>
        {cards.map((c) => {
          const body = (
            <>
              <span className="relative block aspect-[1/1.02] overflow-hidden bg-gray-200">
                <img src={c.img} alt="" loading="eager" className="h-full w-full object-cover transition-transform duration-700 group-hover:scale-105" />
                <span className="absolute inset-0 bg-gradient-to-t from-black/25 to-transparent" aria-hidden="true" />
              </span>
              <span className="block px-6 pb-7 pt-6" style={{ background: c.bg, color: c.fg }}>
                <span className="block text-[26px] font-extrabold leading-tight tracking-tight">{c.title}</span>
                <span className="mt-2 block min-h-[3rem] text-[15.5px] leading-snug opacity-90">{c.text}</span>
                <span className="mt-5 inline-flex items-center gap-1.5 text-[15px] font-extrabold uppercase tracking-wide">{c.cta} <span aria-hidden="true">▸</span></span>
              </span>
            </>
          );
          const cls = "group block overflow-hidden rounded-[10px] shadow-md transition duration-300 hover:-translate-y-1 hover:shadow-2xl focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-night";
          return (
            <li key={c.key} className="w-[82%] shrink-0 snap-start sm:w-[46%] lg:w-[31.4%] xl:w-[32%]">
              {c.external ? <a href={c.to} target="_blank" rel="noopener noreferrer" className={cls}>{body}</a> : <Link to={c.to} className={cls}>{body}</Link>}
            </li>
          );
        })}
      </ul>
      {!edge.end && <button type="button" onClick={() => scroll(1)} aria-label={t("hero.next")} className={`${arrow} right-3`}><ChevronRight size={26} aria-hidden="true" /></button>}
    </div>
  );
}

// ── Bande de rubriques rondes (catégories du backoffice) ──
function Rubrics({ tree }) {
  const { t } = useI18n();
  if (tree.length === 0) return null;
  return (
    <nav aria-label={t("nav.quick")} className="mt-6">
      <ul className="flex gap-5 overflow-x-auto px-1 pb-2 [scrollbar-width:none] sm:gap-7">
        {tree.map((c) => (
          <li key={c.slug} className="shrink-0">
            <Link to={`/category?cat=${c.slug}`} className="group flex w-[92px] flex-col items-center gap-2.5 text-center sm:w-[104px]">
              <span className="flex h-[76px] w-[76px] items-center justify-center rounded-full bg-gray-100 ring-2 ring-transparent transition duration-200 group-hover:-translate-y-1 group-hover:bg-white group-hover:shadow-lg group-hover:ring-action sm:h-[88px] sm:w-[88px]">{c.imageUrl ? <img src={c.imageUrl} alt="" loading="lazy" className="h-full w-full rounded-full object-cover" /> : <LineArt stroke="#6b7280" className="h-12 sm:h-14" />}</span>
              <span className="line-clamp-2 text-[13px] font-semibold leading-tight text-night group-hover:underline">{c.name}</span>
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}

// ── Réassurance : une ligne aérée, pictogrammes dans des pastilles jaunes, filets fins entre les blocs ──
function Trust() {
  const { t } = useI18n();
  // Parcours « comment ça marche » : les 4 atouts du site sous forme d'étapes qui s'animent à l'arrivée à l'écran (GSAP + colis Three.js).
  const steps = [
    { key: "search", Icon: Clipboard, label: t("footer.perk.search.t"), desc: t("footer.perk.search.d") },
    { key: "quote", Icon: Box, label: t("footer.perk.quote.t"), desc: t("footer.perk.quote.d") },
    { key: "pay", Icon: Truck, label: t("footer.perk.pay.t"), desc: t("footer.perk.pay.d") },
    { key: "help", Icon: House, label: t("footer.perk.help.t"), desc: t("footer.perk.help.d") },
  ];
  return (
    <div className="mt-10" data-guide="home-trust">
      <OrderTracker steps={steps} current={steps.length - 1} described onView tone="blue" title={t("hero.journey.title")} titleAccent={t("hero.journey.accent")} />
    </div>
  );
}

// Chiffres clés en cartes d'inégale hauteur (escalier) ; la dernière, bleue, porte un graphique à barres qui se lève puis ondule (GSAP).
const BARS = [0.2, 0.45, 0.7, 0.35, 0.9, 0.5, 0.3, 0.62, 0.8, 0.4, 0.95, 0.55, 0.28, 0.72, 0.48, 0.85, 0.6, 0.33, 0.78, 0.42, 0.66, 0.25, 0.9, 0.52];
function Figures({ figures }) {
  const root = useRef(null);
  useEffect(() => {
    const el = root.current;
    if (!el || figures.length === 0) return undefined;
    if (prefersReducedMotion()) return undefined;
    const ctx = gsap.context(() => {
      gsap.set("[data-fig]", { y: 60, opacity: 0 });
      gsap.set("[data-bar]", { scaleY: 0, transformOrigin: "50% 100%" });
      gsap.timeline({ scrollTrigger: { trigger: el, start: "top 85%", once: true }, defaults: { ease: "power3.out" } })
        .to("[data-fig]", { y: 0, opacity: 1, duration: 0.8, stagger: 0.15 })
        .to("[data-bar]", { scaleY: 1, duration: 0.7, stagger: 0.03, ease: "back.out(1.6)" }, "-=0.4")
        .add(() => {
          gsap.to("[data-bar]", { scaleY: () => 0.55 + Math.random() * 0.45, duration: 0.9, ease: "sine.inOut", repeat: -1, yoyo: true, stagger: { each: 0.05, from: "random" } });
        });
    }, el);
    return () => ctx.revert();
  }, [figures.length]);
  if (figures.length === 0) return null;
  const last = figures.length - 1;
  return (
    <div ref={root} className="mt-8 grid grid-cols-2 items-end gap-3 sm:gap-4 md:flex" data-guide="home-figures">
      {figures.map(([value, label], i) => {
        const hot = i === last && figures.length > 1;
        return (
          <div key={label} data-fig style={{ minHeight: `${150 + i * 56 + (i === last && figures.length > 1 ? 110 : 0)}px` }}
            className={`relative flex flex-1 flex-col overflow-hidden rounded-[22px] p-6 sm:p-8 ${hot ? "col-span-2 bg-[#3b6cf0] text-white md:order-none" : "bg-[#f1f1f1] text-[#4a4a4a]"}`}>
            <p className="text-[48px] font-normal leading-none tracking-tight sm:text-[64px]"><CountUp value={value} /></p>
            <p className={`${hot ? "mt-4" : "mt-auto pt-6"} max-w-[14ch] text-[14px] leading-snug ${hot ? "text-white/85" : "text-[#4a4a4a]/80"}`}>{label}</p>
            {hot && (
              <div className="pointer-events-none absolute inset-x-6 bottom-0 flex h-[42%] items-end justify-between gap-[3px]" aria-hidden="true">
                {BARS.map((h, k) => <span key={k} data-bar className="w-[3px] shrink-0 rounded-t-full bg-white" style={{ height: `${h * 100}%` }} />)}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

export default function Hero() {
  const { t, lang, locale } = useI18n();
  const SITE = useSite();
  const tree = useAsync(() => fetchCategories().catch(() => []), [lang]).data ?? [];
  const stats = useAsync(() => fetchStats().catch(() => ({ products: 0, brands: 0 })), []).data ?? { products: 0, brands: 0 };
  const figures = [
    [stats.products, t("hero.stat.products")],
    [stats.brands, t("hero.stat.brands")],
    [flattenCategories(tree).length, t("hero.stat.categories")],
  ].filter(([n]) => n > 0).map(([n, label]) => [nf(locale, n), label]);
  if (SITE.claims.crossReferences) figures.push([SITE.claims.crossReferences, t("hero.stat.crossRefs")]);
  if (SITE.claims.foundedYear) figures.push([SITE.claims.foundedYear, t("hero.stat.since")]);

  return (
    <section data-guide="home-hero" className="mx-2.5 rounded-[10px] bg-white p-4 shadow-card sm:p-6 lg:p-7" aria-labelledby="hero-title">
      <h1 id="hero-title" className="sr-only">{t("hero.title1")} {t("hero.title2")}</h1>
      <div className="mx-auto max-w-[1500px]">
        <PromoRow />
        <Finder />
        <Rubrics tree={tree} />
        <Trust />
        <Figures figures={figures} />
      </div>
    </section>
  );
}
