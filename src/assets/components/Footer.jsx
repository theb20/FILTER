import { Link, useLocation } from "react-router-dom";
import { SITE } from "../config/site.js";

const Footer = () => {
  const location = useLocation();
  const isSignPage = location.pathname === "/sign";
  const isResetPage = location.pathname === "/reset";

  if (isSignPage || isResetPage) return null;

  return (
    <footer className="bg-dark text-gray-300 mt-16">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-8 mb-8">
          <div>
            <Link
              to="/"
              className="flex items-center text-2xl font-extrabold text-amber-500 select-none"
              aria-label="Filter Finder — accueil"
            >
              <div className="md:w-10 md:h-10 w-9 h-9">
                <img src="/img/logo/favicon.svg" className="w-full" alt="" />
              </div>
              <span className="text-gray-800 dark:text-white">Filter Finder</span>
            </Link>
            <p className="text-sm text-sub leading-relaxed mt-3">
              Votre spécialiste des filtres automobiles en Côte d'Ivoire.
            </p>
          </div>

          <nav aria-label="Catalogue" className="text-dark">
            <h2 className="font-semibold text-dark mb-4">Catalogue</h2>
            <ul className="space-y-2 text-sm">
              <li><Link to="/category" className="hover:text-amber-600 transition-colors">Tous les filtres</Link></li>
              <li><Link to="/category?q=huile" className="hover:text-amber-600 transition-colors">Filtres à huile</Link></li>
              <li><Link to="/category?q=air" className="hover:text-amber-600 transition-colors">Filtres à air</Link></li>
              <li><Link to="/basket" className="hover:text-amber-600 transition-colors">Panier</Link></li>
            </ul>
          </nav>

          <nav aria-label="Aide" className="text-dark">
            <h2 className="font-semibold text-dark mb-4">Support</h2>
            <ul className="space-y-2 text-sm">
              <li><Link to="/contact" className="hover:text-amber-600 transition-colors">Contact</Link></li>
              <li><Link to="/sign" className="hover:text-amber-600 transition-colors">Mon compte</Link></li>
            </ul>
          </nav>

          <nav aria-label="Légal" className="text-dark">
            <h2 className="font-semibold text-dark mb-4">Légal</h2>
            <ul className="space-y-2 text-sm">
              <li><Link to="/privacy" className="hover:text-amber-600 transition-colors">Politique de confidentialité</Link></li>
              <li><Link to="/privacy" className="hover:text-amber-600 transition-colors">Cookies</Link></li>
            </ul>
          </nav>
        </div>

        <div className="border-t border-gray-800 pt-8 text-center sm:text-left">
          <p className="text-sm text-gray-500">
            © {new Date().getFullYear()} {SITE.name} — édité par {SITE.publisher}. Tous droits réservés.
          </p>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
