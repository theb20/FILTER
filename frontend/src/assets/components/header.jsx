import { useEffect, useRef, useState } from "react";
import { gsap, prefersReducedMotion } from "../animation/gsapSetup.js";

import { Link, useNavigate } from "react-router-dom";
import { ShoppingCart, User, Mail, Menu, X, LogOut } from "lucide-react";
import Logo from "./Logo.jsx";
import SearchBar from "./SearchBar.jsx";
import LanguageSwitcher from "./LanguageSwitcher.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { useCartCount } from "../store/cartStore.js";
import { useAuth } from "../store/authStore.js";
import GuideButton from "../guide/GuideButton.jsx";
import AccountMenu from "./AccountMenu.jsx";

export default function Header() {
  const { t, currency, setCurrency, currencies } = useI18n();
  const user = useAuth((a) => a.user);
  const authStatus = useAuth((a) => a.status); // "unknown" tant que la session n'est pas restaurée
  const logout = useAuth((a) => a.logout);
  const navigate = useNavigate();
  const count = useCartCount();
  const [menuOpen, setMenuOpen] = useState(false);

  const headerRef = useRef(null);
  const badgeRef = useRef(null);
  const cartRef = useRef(null);
  const menuRef = useRef(null);
  const prevCount = useRef(count);

  // Entrée du header : barre qui descend, puis éléments en cascade
  useEffect(() => {
    if (prefersReducedMotion()) return;
    const ctx = gsap.context(() => {
      gsap.from(headerRef.current, { yPercent: -100, duration: 0.7, ease: "power3.out" });
      gsap.from("[data-h]", { y: -16, opacity: 0, duration: 0.6, ease: "power3.out", stagger: 0.07, delay: 0.25 });
    }, headerRef);
    return () => ctx.revert();
  }, []);

  // Le panier « saute » et le badge rebondit à chaque ajout
  useEffect(() => {
    if (count > prevCount.current && !prefersReducedMotion()) {
      gsap.fromTo(cartRef.current, { rotation: -18, scale: 1.35 }, { rotation: 0, scale: 1, duration: 0.9, ease: "elastic.out(1,0.3)" });
      if (badgeRef.current) gsap.fromTo(badgeRef.current, { scale: 2.2 }, { scale: 1, duration: 0.8, ease: "elastic.out(1,0.4)" });
    }
    prevCount.current = count;
  }, [count]);

  // Menu mobile : les liens glissent en cascade
  useEffect(() => {
    if (!menuOpen || !menuRef.current || prefersReducedMotion()) return;
    gsap.fromTo(menuRef.current.querySelectorAll("li"), { x: -24, opacity: 0 }, { x: 0, opacity: 1, duration: 0.45, ease: "power3.out", stagger: 0.06 });
  }, [menuOpen]);

  const iconLink = "flex items-center gap-2 text-white hover:text-action transition-colors";


  return (
    <header ref={headerRef} className="sticky top-0 z-[1000] bg-night text-white">
      <div className="mx-auto flex max-w-[1500px] items-center gap-4 px-4 py-3 lg:px-6">
        <button
          type="button"
          aria-label={menuOpen ? t("nav.closeMenu") : t("nav.openMenu")}
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((v) => !v)}
          data-guide="menu"
          className="lg:hidden"
        >
          {menuOpen ? <X size={26} /> : <Menu size={26} />}
        </button>

        <div data-h><Logo /></div>

        <SearchBar id="header-search" guide="search" className="mx-4 hidden flex-1 lg:flex" />

        <nav aria-label={t("nav.account")} data-h className="ml-auto flex items-center gap-3 text-sm sm:gap-4 lg:gap-5">
          {/* Langue : visible sur desktop ET mobile */}
          <GuideButton />
          <LanguageSwitcher />
          <label data-guide="currency" className="hidden lg:block">
            <span className="sr-only">{t("currency.label")}</span>
            <select
              value={currency}
              onChange={(e) => setCurrency(e.target.value)}
              className="rounded-[3px]  w-[110px] bg-night px-2 py-1 font-medium text-white/90 hover:border-white"
            >
              {currencies.map((c) => <option key={c} value={c}>{t(`currency.${c}`)}</option>)}
            </select>
          </label>
          <Link to="/contact" className={`${iconLink} hidden lg:flex`}>
            <Mail size={20} aria-hidden="true" /> {t("nav.contact")}
          </Link>
          {authStatus === "unknown" ? (
            // Session en cours de restauration : réserve la place sans afficher « Connexion » à tort.
            <span aria-hidden="true" className="hidden h-8 w-[120px] animate-pulse rounded-full bg-white/10 lg:block" />
          ) : user ? (
            <AccountMenu user={user} />
          ) : (
            <Link to="/sign" data-guide="account" className={`${iconLink} hidden lg:flex`}>
              <User size={20} aria-hidden="true" /> {t("nav.signin")}
            </Link>
          )}
          <Link to="/basket" data-guide="cart" aria-label={t("nav.cartN", { n: count })} className={iconLink}>
            <span ref={cartRef} data-cart-icon className="relative">
              <ShoppingCart size={22} aria-hidden="true" />
              {count > 0 && (
                <span ref={badgeRef} aria-hidden="true" className="absolute -right-2 -top-2 min-w-[18px] rounded-full bg-action px-1 text-center text-[11px] font-bold leading-[18px] text-black">
                  {count > 99 ? "99+" : count}
                </span>
              )}
            </span>
            <span className="hidden lg:inline">{t("nav.cart")}</span>
          </Link>
        </nav>
      </div>

      {/* Recherche mobile pleine largeur */}
      <div className="px-4 pb-3 lg:hidden">
        <SearchBar id="header-search-mobile" guide="search" />
      </div>

      {menuOpen && (
        <nav ref={menuRef} aria-label={t("nav.menu")} className="border-t border-white/10 bg-navy px-4 py-3 lg:hidden">
          {user && (
            <div className="mb-3 border-b border-white/10 pb-3">
              <p className="text-[11px] uppercase tracking-wide text-white/60">{t("account.menu.signedAs")}</p>
              <p className="truncate font-semibold">{[user.firstName, user.lastName].filter(Boolean).join(" ") || user.email}</p>
            </div>
          )}
          <ul className="space-y-3 text-base">
            {[
              ["/category", "nav.products"],
              ["/brands", "nav.brands"],
              ["/blog", "nav.blog"],
              ["/contact", "nav.contact"],
              [user ? "/profil" : "/sign", user ? "nav.myAccount" : "nav.signin"],
            ].map(([to, key]) => (
              <li key={to}>
                <Link to={to} onClick={() => setMenuOpen(false)} className="block py-1 hover:text-action">
                  {t(key)}
                </Link>
              </li>
            ))}
            {user && (
              <li>
                <button
                  type="button"
                  onClick={async () => { setMenuOpen(false); await logout(); navigate("/"); }}
                  className="flex items-center gap-2 py-1 text-red-300 hover:text-red-200"
                >
                  <LogOut size={16} aria-hidden="true" /> {t("nav.signout")}
                </button>
              </li>
            )}
          </ul>
        </nav>
      )}
    </header>
  );
}
