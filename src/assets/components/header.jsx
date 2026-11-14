import { useState } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { Search, ShoppingCart, User, Mail, Menu, X, Globe } from "lucide-react";
import Logo from "./Logo.jsx";
import { SITE } from "../config/site.js";

export function SearchField({ id, className = "", dark = false }) {
  const navigate = useNavigate();
  const [term, setTerm] = useState("");

  const submit = (e) => {
    e.preventDefault();
    const q = term.trim();
    navigate(q ? `/category?q=${encodeURIComponent(q)}` : "/category");
  };

  return (
    <form role="search" onSubmit={submit} className={`flex ${className}`}>
      <label htmlFor={id} className="sr-only">
        Enter your item, model, SKU or part #
      </label>
      <input
        id={id}
        type="search"
        value={term}
        onChange={(e) => setTerm(e.target.value)}
        placeholder="Enter your item, model, SKU or part #"
        className={`min-w-0 flex-1 bg-white px-4 py-2.5 text-black placeholder:text-gray-500 outline-none focus:ring-2 focus:ring-action ${
          dark ? "rounded-l-[3px]" : "rounded-l-[3px]"
        }`}
      />
      <button
        type="submit"
        aria-label="Search"
        className="flex w-12 shrink-0 items-center justify-center rounded-r-[3px] bg-action text-black hover:bg-action-dark"
      >
        <Search size={20} />
      </button>
    </form>
  );
}

export default function Header() {
  const [menuOpen, setMenuOpen] = useState(false);
  const { pathname } = useLocation();
  const hideHeader = pathname === "/sign" || pathname === "/reset";
  if (hideHeader) return null;

  const iconLink =
    "flex items-center gap-2 text-white hover:text-action transition-colors";

  return (
    <header className="sticky top-0 z-[1000] bg-night text-white">
      <div className="mx-auto flex max-w-[1500px] items-center gap-4 px-4 py-3 lg:px-6">
        <button
          type="button"
          aria-label={menuOpen ? "Close menu" : "Open menu"}
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((v) => !v)}
          className="lg:hidden"
        >
          {menuOpen ? <X size={26} /> : <Menu size={26} />}
        </button>

        <Logo />

        <SearchField id="header-search" className="mx-4 hidden flex-1 lg:flex" />

        <nav aria-label="Account" className="ml-auto flex items-center gap-5 text-sm">
          <span className="hidden items-center gap-1.5 text-white/90 lg:flex" title="Currency">
            <Globe size={18} aria-hidden="true" /> {SITE.currency}
          </span>
          <Link to="/contact" className={`${iconLink} hidden lg:flex`}>
            <Mail size={20} aria-hidden="true" /> Contact
          </Link>
          <Link to="/sign" className={`${iconLink} hidden lg:flex`}>
            <User size={20} aria-hidden="true" /> Sign In
          </Link>
          <Link to="/basket" aria-label="Cart" className={iconLink}>
            <ShoppingCart size={22} aria-hidden="true" />
            <span className="hidden lg:inline">Cart</span>
          </Link>
        </nav>
      </div>

      {/* Recherche mobile pleine largeur */}
      <div className="px-4 pb-3 lg:hidden">
        <SearchField id="header-search-mobile" />
      </div>

      {menuOpen && (
        <nav aria-label="Menu" className="border-t border-white/10 bg-navy px-4 py-3 lg:hidden">
          <ul className="space-y-3 text-base">
            {[
              ["/category", "Products"],
              ["/brands", "Brands"],
              ["/blog", "Blog"],
              ["/contact", "Contact"],
              ["/sign", "Sign In"],
            ].map(([to, label]) => (
              <li key={to}>
                <Link to={to} onClick={() => setMenuOpen(false)} className="block py-1 hover:text-action">
                  {label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
      )}
    </header>
  );
}
