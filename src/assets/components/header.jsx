import { useState } from "react";
import { Link } from "react-router-dom";
import { ShoppingCart, User, Mail, Menu, X } from "lucide-react";
import Logo from "./Logo.jsx";
import SearchBar from "./SearchBar.jsx";
import LanguageSwitcher from "./LanguageSwitcher.jsx";
import { useI18n } from "../i18n/i18nContext.js";
import { SITE } from "../config/site.js";

export default function Header() {
  const { t } = useI18n();
  const [menuOpen, setMenuOpen] = useState(false);

  const iconLink = "flex items-center gap-2 text-white hover:text-action transition-colors";

  return (
    <header className="sticky top-0 z-[1000] bg-night text-white">
      <div className="mx-auto flex max-w-[1500px] items-center gap-4 px-4 py-3 lg:px-6">
        <button
          type="button"
          aria-label={menuOpen ? t("nav.closeMenu") : t("nav.openMenu")}
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((v) => !v)}
          className="lg:hidden"
        >
          {menuOpen ? <X size={26} /> : <Menu size={26} />}
        </button>

        <Logo />

        <SearchBar id="header-search" className="mx-4 hidden flex-1 lg:flex" />

        <nav aria-label={t("nav.account")} className="ml-auto flex items-center gap-4 text-sm lg:gap-5">
          {/* Langue : visible sur desktop ET mobile */}
          <LanguageSwitcher />
          <span className="hidden font-medium text-white/90 lg:inline" title={t("currency.label")}>
            {SITE.currency}
          </span>
          <Link to="/contact" className={`${iconLink} hidden lg:flex`}>
            <Mail size={20} aria-hidden="true" /> {t("nav.contact")}
          </Link>
          <Link to="/sign" className={`${iconLink} hidden lg:flex`}>
            <User size={20} aria-hidden="true" /> {t("nav.signin")}
          </Link>
          <Link to="/basket" aria-label={t("nav.cart")} className={iconLink}>
            <ShoppingCart size={22} aria-hidden="true" />
            <span className="hidden lg:inline">{t("nav.cart")}</span>
          </Link>
        </nav>
      </div>

      {/* Recherche mobile pleine largeur */}
      <div className="px-4 pb-3 lg:hidden">
        <SearchBar id="header-search-mobile" />
      </div>

      {menuOpen && (
        <nav aria-label={t("nav.menu")} className="border-t border-white/10 bg-navy px-4 py-3 lg:hidden">
          <ul className="space-y-3 text-base">
            {[
              ["/category", "nav.products"],
              ["/brands", "nav.brands"],
              ["/blog", "nav.blog"],
              ["/contact", "nav.contact"],
              ["/sign", "nav.signin"],
            ].map(([to, key]) => (
              <li key={to}>
                <Link to={to} onClick={() => setMenuOpen(false)} className="block py-1 hover:text-action">
                  {t(key)}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
      )}
    </header>
  );
}
