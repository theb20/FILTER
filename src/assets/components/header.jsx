import React, { useState } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { SITE, DEMO_NOTICE } from "../config/site.js";
import { useNotify } from "../configurations/notificationContext.js";
import {
  FaSearch,
  FaShoppingCart,
  FaUser,
  FaHeart,
  FaBars,
  FaTimes,
  FaGlobe,
  FaPhoneAlt,
} from "react-icons/fa";

const Header = () => {
    const location = useLocation();
    const navigate = useNavigate();
    const notify = useNotify();
    const [searchTerm, setSearchTerm] = useState("");

    const [menuOpen, setMenuOpen] = useState(false);
    const [showSearch, setShowSearch] = useState(false);

    const isSignPage = location.pathname === "/sign";
    const isResetPage = location.pathname === "/reset";
    const isHomePage = location.pathname === "/";
    const isContactPage = location.pathname === "/contact";

    // Fonction pour fermer tous les menus
    const closeAllMenus = () => {
      setMenuOpen(false);
      setShowSearch(false);
    };

    // Recherche : envoie vers /category?q=<terme>
    const handleSearch = (e) => {
      e.preventDefault();
      const term = searchTerm.trim();
      navigate(term ? `/category?q=${encodeURIComponent(term)}` : "/category");
      closeAllMenus();
    };

  return (
    <>
     {!isSignPage && !isResetPage && !isContactPage && (
        <header className="w-full sticky top-0 left-0 z-[1000] shadow-md bg-white dark:bg-gray-900 dark:text-white transition-all duration-300">
        {/* ===== Barre supérieure ===== */}
        <div className="hidden md:flex justify-between items-center px-6 py-2 bg-gray-100 dark:bg-gray-800 text-sm text-gray-600 dark:text-gray-300">
            <div className="flex items-center space-x-4">
            <span className="flex items-center gap-2">
                <FaPhoneAlt className="text-amber-500" /> {SITE.phone || "Téléphone bientôt disponible"}
            </span>
            <span className="flex items-center gap-2">
                <FaGlobe className="text-amber-500" /> Côte d'Ivoire | Fcfa
            </span>
            </div>
            <div className="flex items-center gap-4">
            <Link to="/quote" className="dark:text-gray-300 hover:text-amber-500">Demander un devis</Link>
            <Link to="/profil" className="dark:text-gray-300 hover:text-amber-500">Mes commandes</Link>
            </div>
        </div>

        {/* ===== Barre principale ===== */}
        <div className="flex items-center justify-between px-4 md:px-8 py-3">
            <button
                type="button"
                aria-label={menuOpen ? "Fermer le menu" : "Ouvrir le menu"}
                aria-expanded={menuOpen}
                className="md:hidden text-2xl cursor-pointer hover:text-amber-500 border-0 transition-transform hover:scale-110"
                onClick={() => {
                  setMenuOpen(!menuOpen);
                  if (showSearch) setShowSearch(false);
                }}
            >
                {menuOpen ? <FaTimes /> : <FaBars />}
            </button>
            
            {/* Logo (lien, pas un titre : un seul h1 par page) */}
            <Link
              to="/"
              onClick={closeAllMenus}
              aria-label="Filter Finder — accueil"
              className="font-extrabold w-full flex items-center justify-center lg:w-auto text-center lg:text-left text-2xl md:text-3xl lg:text-4xl text-amber-500 select-none"
            >
              <div className="md:w-12 md:h-12 w-10 h-10">
                <img src="/img/logo/favicon.svg" className="w-full" alt="" />
              </div>
              <span className="text-gray-800 dark:text-white">Filter Finder</span>
            </Link>
            {/* Barre de recherche (desktop) */}
            {!isHomePage && (
            
            <form
              role="search"
              onSubmit={handleSearch}
              className="hidden md:flex items-center bg-gray-100 dark:bg-gray-800 px-3 py-2 rounded-full w-1/2 focus-within:ring-2 focus-within:ring-amber-400"
            >
            <FaSearch className="text-gray-400 mr-2" aria-hidden="true" />
            <input
                type="search"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                aria-label="Rechercher un filtre, une référence ou une catégorie"
                placeholder="Rechercher un filtre, une référence..."
                className="bg-transparent outline-none w-full py-3 text-sm text-gray-700 dark:text-gray-200"
            />
            </form>
            )}

            {/* Barre de navigation principale */}
            {isHomePage && (
            <div className="hidden md:flex justify-center  dark:bg-gray-800 py-3 ">
                <nav className="flex space-x-8 font-medium text-gray-700 dark:text-gray-200">
                <Link to="/category" className="hover:text-amber-500">Catalogue</Link>
                <Link to="/category?q=air" className="hover:text-amber-500">Filtres à air</Link>
                <Link to="/category?q=huile" className="hover:text-amber-500">Filtres à huile</Link>
                <Link to="/category?q=carburant" className="hover:text-amber-500">Filtres à carburant</Link>
                <Link to="/contact" className="hover:text-amber-500">Contact</Link>
                </nav>
            </div>
            )}

            <div className="flex items-center md:space-x-4 space-x-3">
                <button
                  type="button"
                  aria-label="Favoris"
                  onClick={() => notify(DEMO_NOTICE, "demo")}
                  className="text-xl hidden md:flex cursor-pointer hover:text-amber-500 transition-transform hover:scale-110"
                >
                  <FaHeart aria-hidden="true" />
                </button>

                {/* Icône recherche mobile */}
                <button
                    type="button"
                    aria-label="Rechercher"
                    aria-expanded={showSearch}
                    className="md:hidden text-xl cursor-pointer hover:text-amber-500 transition-transform hover:scale-110"
                    onClick={() => {
                      setShowSearch(!showSearch);
                      if (menuOpen) setMenuOpen(false);
                    }}
                >
                    <FaSearch aria-hidden="true" />
                </button>

                <Link
                  to="/basket"
                  aria-label="Panier"
                  onClick={closeAllMenus}
                  className="text-xl cursor-pointer hover:text-amber-500 transition-transform hover:scale-110"
                >
                  <FaShoppingCart aria-hidden="true" />
                </Link>

                <Link
                  to="/sign"
                  aria-label="Mon compte"
                  onClick={closeAllMenus}
                  className="text-xl hidden md:flex cursor-pointer hover:text-amber-500 transition-transform hover:scale-110"
                >
                  <FaUser aria-hidden="true" />
                </Link>
            </div>
        </div>

        {/* Barre de navigation principale */}
        {!isHomePage && (
        <div className="hidden md:flex justify-center bg-gray-50 dark:bg-gray-800 py-3 border-t border-gray-200 dark:border-gray-700">
            <nav className="flex space-x-8 font-medium text-gray-700 dark:text-gray-200">
              <Link to="/" className="hover:text-amber-500">Accueil</Link>
              <Link to="/category?q=air" className="hover:text-amber-500">Filtres à air</Link>
              <Link to="/category?q=huile" className="hover:text-amber-500">Filtres à huile</Link>
              <Link to="/category?q=carburant" className="hover:text-amber-500">Filtres à carburant</Link>
              <Link to="/contact" className="hover:text-amber-500">Contact</Link>
            </nav>
        </div>
        )}
        {/* Menu mobile */}
        <div
            className={`md:hidden absolute w-full bg-white/80 dark:bg-gray-900/80 backdrop-blur-lg border-t border-gray-200 dark:border-gray-700 shadow-lg transition-all duration-300 z-60 ${
            menuOpen ? "max-h-96 opacity-100" : "max-h-0 opacity-0 overflow-hidden"
            }`}
        >
            <nav className="flex flex-col items-center py-4 space-y-3 text-gray-700 dark:text-gray-200 font-medium">
              {!isHomePage && (
                <Link to="/" onClick={closeAllMenus} className="hover:text-amber-500 transition-colors">Accueil</Link>
              )}
              <Link to="/category" onClick={closeAllMenus} className="hover:text-amber-500 transition-colors">Catalogue</Link>
              <Link to="/quote" onClick={closeAllMenus} className="hover:text-amber-500 transition-colors">Devis</Link>
              <Link to="/sign" onClick={closeAllMenus} className="hover:text-amber-500 transition-colors">Se connecter</Link>
              <Link to="/contact" onClick={closeAllMenus} className="hover:text-amber-500 transition-colors">Contact</Link>
            </nav>
        </div>

        {/* Barre de recherche mobile avec animations */}
        <div
            className={`md:hidden bg-white/90 dark:bg-gray-900/90 backdrop-blur-lg border-t border-gray-200 dark:border-gray-700 shadow-lg overflow-hidden transition-all duration-500 ease-in-out ${
            showSearch 
                ? "max-h-36 opacity-100 translate-y-0" 
                : "max-h-0 opacity-0 -translate-y-4"
            }`}
        >
            <form role="search" onSubmit={handleSearch} className="p-4 flex items-center justify-center">
            <div className={`bg-gray-100 dark:bg-gray-800 backdrop-blur-lg rounded-full px-4 py-2 flex items-center w-full transition-all duration-500 ${
                showSearch 
                ? "scale-100 opacity-100" 
                : "scale-95 opacity-0"
            }`}>
                <FaSearch className={`text-gray-400 mr-2 transition-all duration-500 ${
                showSearch 
                    ? "rotate-0 scale-100" 
                    : "rotate-90 scale-0"
                }`} />
                <input
                type="search"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                aria-label="Rechercher un filtre"
                placeholder="Rechercher un filtre..."
                className="bg-transparent py-3 outline-none w-full text-sm text-gray-700 dark:text-gray-200 placeholder:text-gray-400"
                autoFocus={showSearch}
                />
                <button
                type="button"
                aria-label="Fermer la recherche"
                onClick={() => setShowSearch(false)}
                className={`ml-2 text-gray-400 hover:text-amber-500 transition-all duration-300 ${
                    showSearch 
                    ? "rotate-0 opacity-100 scale-100" 
                    : "rotate-180 opacity-0 scale-0"
                }`}
                >
                <FaTimes />
                </button>
            </div>
            </form>
        </div>
        </header>
     )}
      {/* barre for contact */}
        {isContactPage && (
          <div className="absolute top-0 left-0 w-full dark:bg-gray-800 text-black dark:text-white py-10 px-24 flex items-center justify-between z-[1000]">
              <Link to="/" className="logo" aria-label="Filter Finder — accueil">
                <img src="/img/logo/favicon.svg" className="w-22" alt="" />
              </Link>
          </div>
        )}
    </>
  );
};

export default Header;