import { Link } from "react-router-dom";
import { ArrowUp, BadgeCheck, FileText, Headphones, Mail, MapPin, Phone, Search, Wallet } from "lucide-react";
import { FaFacebookF, FaLinkedinIn, FaWhatsapp, FaYoutube } from "react-icons/fa";
import GuideFooterToggle from "../guide/GuideFooterToggle.jsx";
import Logo from "./Logo.jsx";
import Todo from "./Todo.jsx";
import PaymentMethods from "./PaymentMethods.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { useSite } from "../store/siteStore.js";
import { useAuth } from "../store/authStore.js";
import { fetchCategories, useAsync } from "../lib/catalog.js";

const PERKS = [
  { Icon: Search, key: "search" },
  { Icon: FileText, key: "quote" },
  { Icon: Wallet, key: "pay" },
  { Icon: Headphones, key: "help" },
];

// Colonne du pied de page : accordéon sur mobile (details), colonne fixe sur ordinateur.
function Column({ title, children, className = "" }) {
  return (
    <details className={`group border-b border-white/10 py-3 md:border-0 md:py-0 md:[&>summary]:pointer-events-none ${className}`} open>
      <summary className="flex cursor-pointer list-none items-center justify-between text-[13px] font-bold uppercase tracking-[0.14em] text-action md:cursor-default">
        {title}
        <span className="text-xl leading-none text-white transition-transform group-open:rotate-45 md:hidden" aria-hidden="true">+</span>
      </summary>
      <div className="mt-4">{children}</div>
    </details>
  );
}

const linkCls = "text-[14.5px] text-white/75 transition-colors hover:text-white hover:underline";

function LinkList({ items }) {
  return (
    <ul className="space-y-2.5">
      {items.map(([to, label]) => (
        <li key={to + label}><Link to={to} className={linkCls}>{label}</Link></li>
      ))}
    </ul>
  );
}

export default function Footer() {
  const { t, lang } = useI18n();
  const SITE = useSite();
  const user = useAuth((a) => a.user);
  const categories = (useAsync(() => fetchCategories().catch(() => []), [lang]).data ?? []).slice(0, 6);
  const wa = SITE.phone ? `https://wa.me/${SITE.phone.replace(/\D/g, "")}` : "";
  const socials = [
    ["LinkedIn", FaLinkedinIn, SITE.linkedinUrl], ["Facebook", FaFacebookF, SITE.facebookUrl],
    ["YouTube", FaYoutube, SITE.youtubeUrl], ["WhatsApp", FaWhatsapp, wa],
  ].filter(([, , href]) => href);
  const legalIds = [[t("quote.p.rccm"), SITE.rccm], [t("quote.p.tax"), SITE.taxNumber], [t("quote.p.capital"), SITE.capital]].filter(([, v]) => v);

  return (
    <footer className="mx-2.5 my-2.5 overflow-hidden rounded-[6px] bg-night text-white">
      {/* Bandeau d'appel à l'action */}
      <div className="relative overflow-hidden border-b border-white/10 bg-gradient-to-r from-navy via-night to-navy">
        <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-5 px-6 py-8 lg:px-10">
          <div className="max-w-xl">
            <p className="text-[24px] font-bold leading-tight sm:text-[28px]">{t("footer.cta.title")}</p>
            <p className="mt-1.5 text-white/75">{t("footer.cta.text")}</p>
          </div>
          <div className="flex flex-wrap gap-3">
            <Link to="/quote" className="inline-flex items-center gap-2 rounded-[8px] bg-action px-6 py-3 font-bold text-black transition-colors hover:bg-action-dark">{t("footer.quote")} <span aria-hidden="true">→</span></Link>
            <Link to="/contact" className="inline-flex items-center gap-2 rounded-[8px] border border-white/30 px-6 py-3 font-medium transition-colors hover:bg-white/10">{t("nav.contact")}</Link>
          </div>
        </div>
      </div>

      {/* Atouts */}
      <ul className="mx-auto grid max-w-[1500px] gap-px border-b border-white/10 bg-white/10 sm:grid-cols-2 lg:grid-cols-4">
        {PERKS.map(({ Icon, key }) => (
          <li key={key} className="flex items-start gap-3.5 bg-night px-6 py-5 lg:px-8">
            <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-white/10 text-action"><Icon size={20} aria-hidden="true" /></span>
            <span><strong className="block text-[15px]">{t(`footer.perk.${key}.t`)}</strong><span className="text-[13.5px] text-white/65">{t(`footer.perk.${key}.d`)}</span></span>
          </li>
        ))}
      </ul>

      {/* Colonnes */}
      <div className="mx-auto grid max-w-[1500px] gap-x-10 gap-y-2 px-6 py-12 md:grid-cols-2 lg:grid-cols-[1.5fr_1fr_1fr_1fr_1.3fr] lg:px-10">
        <div className="md:col-span-2 lg:col-span-1">
          <Logo />
          <p className="mt-4 max-w-xs text-[14.5px] leading-relaxed text-white/70">{t("footer.about")}</p>
          {socials.length > 0 ? (
            <ul className="mt-5 flex gap-2.5" aria-label={t("footer.follow")}>
              {socials.map(([label, Icon, href]) => (
                <li key={label}>
                  <a href={href} target="_blank" rel="noopener noreferrer" aria-label={label} className="flex h-10 w-10 items-center justify-center rounded-full bg-white/10 transition-colors hover:bg-action hover:text-black"><Icon size={16} aria-hidden="true" /></a>
                </li>
              ))}
            </ul>
          ) : (
            <p className="mt-5 text-sm"><Todo>{t("todo.linkedin")}</Todo></p>
          )}
        </div>

        <Column title={t("footer.shop")}>
          <LinkList items={[["/category", t("footer.allProducts")], ...categories.map((c) => [`/category?cat=${c.slug}`, c.name]), ["/brands", t("nav.brands")]]} />
        </Column>

        <Column title={t("footer.resources")}>
          <LinkList items={[["/about", t("info.about")], ["/blog", t("nav.blog")], ["/quote", t("footer.quote")], ["/contact", t("nav.contact")], ["/basket", t("nav.cart")]]} />
        </Column>

        <Column title={t("footer.account")}>
          <LinkList items={user
            ? [["/profil", t("nav.myAccount")], ["/profil#devis", t("account.menu.drafts")], ["/basket", t("nav.cart")]]
            : [["/sign", t("nav.signin")], ["/sign", t("footer.signup")], ["/reset", t("auth.forgot")]]} />
          <h3 className="mb-3 mt-7 text-[13px] font-bold uppercase tracking-[0.14em] text-action">{t("footer.legal")}</h3>
          <LinkList items={[["/terms", t("footer.terms")], ["/privacy", t("footer.privacy")], ["/refund", t("footer.refund")]]} />
        </Column>

        <Column title={t("footer.contactUs")}>
          <address className="space-y-3.5 text-[14.5px] not-italic text-white/80">
            <p className="flex gap-3"><MapPin size={17} className="mt-0.5 shrink-0 text-action" aria-hidden="true" /><span>{SITE.address || <Todo>{t("todo.address")}</Todo>}</span></p>
            <p className="flex gap-3"><Phone size={17} className="mt-0.5 shrink-0 text-action" aria-hidden="true" />{SITE.phone ? <a href={`tel:${SITE.phone.replace(/\s/g, "")}`} className="hover:underline">{SITE.phone}</a> : <Todo>{t("todo.phone")}</Todo>}</p>
            <p className="flex gap-3"><Mail size={17} className="mt-0.5 shrink-0 text-action" aria-hidden="true" />{SITE.email ? <a href={`mailto:${SITE.email}`} className="break-all hover:underline">{SITE.email}</a> : <Todo>{t("todo.email")}</Todo>}</p>
          </address>
          {wa && <a href={wa} target="_blank" rel="noopener noreferrer" className="mt-5 inline-flex items-center gap-2 rounded-[8px] bg-[#25D366] px-4 py-2.5 text-sm font-bold text-black hover:brightness-95"><FaWhatsapp size={17} aria-hidden="true" /> {t("footer.whatsapp")}</a>}
        </Column>
      </div>

      {/* Paiements et mentions */}
      <div className="border-t border-white/10">
        <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-5 px-6 py-6 lg:px-10">
          <div className="flex flex-wrap items-center gap-4">
            <p className="flex items-center gap-2 text-sm text-white/70"><BadgeCheck size={16} className="text-action" aria-hidden="true" /> {t("footer.accept")}</p>
            <PaymentMethods />
          </div>
          {legalIds.length > 0 && (
            <p className="max-w-xl text-xs leading-relaxed text-white/50">{legalIds.map(([k, v]) => `${k} : ${v}`).join(" · ")}</p>
          )}
        </div>
      </div>

      <div className="border-t border-white/10 bg-black/20">
        <div className="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-3 px-6 py-4 text-[13px] text-white/60 lg:px-10">
          <p>{t("footer.rights", { year: new Date().getFullYear(), name: SITE.name, publisher: SITE.publisher })}</p>
          <nav aria-label={t("footer.legal")} className="flex flex-wrap items-center gap-x-5 gap-y-1">
            <Link to="/terms" className="hover:text-white hover:underline">{t("footer.terms")}</Link>
            <Link to="/privacy" className="hover:text-white hover:underline">{t("footer.privacy")}</Link>
            <Link to="/refund" className="hover:text-white hover:underline">{t("footer.refund")}</Link>
            <button type="button" onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })} className="inline-flex items-center gap-1.5 rounded-full border border-white/25 px-3 py-1.5 text-white/80 hover:bg-white/10 hover:text-white">
              <ArrowUp size={14} aria-hidden="true" /> {t("footer.backTop")}
            </button>
          </nav>
        </div>
        <div className="pb-3 text-center"><GuideFooterToggle /></div>
      </div>
    </footer>
  );
}
