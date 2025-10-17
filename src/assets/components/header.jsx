import React, { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
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

const Header = ({openPopup}) => {
    const location = useLocation();
    const navigate = useNavigate();

    const [menuOpen, setMenuOpen] = useState(false);
    const [showSearch, setShowSearch] = useState(false);

    const isSignPage = location.pathname === "/sign";
    const isResetPage = location.pathname === "/reset";
    const isHomePage = location.pathname === "/";

    // Fonction pour fermer tous les menus
    const closeAllMenus = () => {
      setMenuOpen(false);
      setShowSearch(false);
    };

    // Fonction pour gérer la navigation et fermer les menus
    const handleNavigation = (path) => {
      navigate(path);
      closeAllMenus();
    };

    // Fonction pour ouvrir le popup et fermer les menus
    const handleOpenPopup = (type) => {
      openPopup(type);
      closeAllMenus();
    };

    
  return (
    <>
     {!isSignPage && !isResetPage && (
        <header className="w-full sticky top-0 left-0 z-50 shadow-md bg-white dark:bg-gray-900 dark:text-white transition-all duration-300">
        {/* ===== Barre supérieure ===== */}
        <div className="hidden md:flex justify-between items-center px-6 py-2 bg-gray-100 dark:bg-gray-800 text-sm text-gray-600 dark:text-gray-300">
            <div className="flex items-center space-x-4">
            <span className="flex items-center gap-2">
                <FaPhoneAlt className="text-amber-500" /> +33 1 23 45 67 89
            </span>
            <span className="flex items-center gap-2">
                <FaGlobe className="text-amber-500" /> FR | EUR €
            </span>
            </div>
            <div className="flex items-center gap-4">
            <a onClick={() => openPopup('quote')} className="dark:text-gray-300 hover:text-amber-500 cursor-pointer">Demander un devis</a>
            <a href="#" className="dark:text-gray-300 hover:text-amber-500">Suivre ma commande</a>
            </div>
        </div>

        {/* ===== Barre principale ===== */}
        <div className="flex items-center justify-between px-4 md:px-8 py-3">
            <button
                className="md:hidden text-2xl cursor-pointer hover:text-amber-500 border-0 transition-transform hover:scale-110"
                onClick={() => {
                  setMenuOpen(!menuOpen);
                  if (showSearch) setShowSearch(false);
                }}
            >
                {menuOpen ? <FaTimes /> : <FaBars />}
            </button>
            
            {/* Logo */}
            <h1 
              onClick={() => handleNavigation("/")} 
              className="font-extrabold w-full lg:w-auto text-center lg:text-left text-3xl md:text-6xl text-amber-500 cursor-pointer select-none"
            >
              Filter<span className="text-gray-800 dark:text-white">Finder</span>
            </h1>
            {/* Barre de recherche (desktop) */}
            {!isHomePage && (
            
            <div className="hidden md:flex items-center bg-gray-100 dark:bg-gray-800 px-3 py-2 rounded-full w-1/2 focus-within:ring-2 focus-within:ring-amber-400">
            <FaSearch className="text-gray-400 mr-2" />
            <input
                type="text"
                placeholder="Rechercher un produit, une catégorie..."
                className="bg-transparent outline-none w-full py-3 text-sm text-gray-700 dark:text-gray-200"
            />
            </div>
            )}

            <div className="flex items-center md:space-x-4 space-x-3">
                <FaHeart className="text-xl hidden md:flex cursor-pointer hover:text-amber-500 transition-transform hover:scale-110" />
                
                {/* Icône recherche mobile */}
                <button
                    className="md:hidden text-xl cursor-pointer hover:text-amber-500 transition-transform hover:scale-110"
                    onClick={() => {
                      setShowSearch(!showSearch);
                      if (menuOpen) setMenuOpen(false);
                    }}
                >
                    <FaSearch />
                </button>
                
                <FaShoppingCart 
                  onClick={() => handleNavigation("/basket")} 
                  className="text-xl cursor-pointer hover:text-amber-500 transition-transform hover:scale-110" 
                />
                
                <FaUser 
                  onClick={() => handleNavigation("/sign")} 
                  className="text-xl hidden md:flex cursor-pointer hover:text-amber-500 transition-transform hover:scale-110" 
                />
            </div>
        </div>

        {/* Barre de navigation principale */}
        <div className="hidden md:flex justify-center bg-gray-50 dark:bg-gray-800 py-3 border-t border-gray-200 dark:border-gray-700">
            <nav className="flex space-x-8 font-medium text-gray-700 dark:text-gray-200">
            <a href="/category" className="hover:text-amber-500">Nouveautés</a>
            <a href="#" className="hover:text-amber-500">Promotions</a>
            <a href="#" className="hover:text-amber-500">Filtres à air</a>
            <a href="#" className="hover:text-amber-500">Filtres à huile</a>
            <a href="#" className="hover:text-amber-500">Accessoires</a>
            <a href="/contact" className="hover:text-amber-500">Contact</a>
            </nav>
        </div>

        {/* Menu mobile */}
        <div
            className={`md:hidden absolute w-full bg-white/80 dark:bg-gray-900/80 backdrop-blur-lg border-t border-gray-200 dark:border-gray-700 shadow-lg transition-all duration-300 z-60 ${
            menuOpen ? "max-h-96 opacity-100" : "max-h-0 opacity-0 overflow-hidden"
            }`}
        >
            <nav className="flex flex-col items-center py-4 space-y-3 text-gray-700 dark:text-gray-200 font-medium">
            <a 
              onClick={() => handleNavigation("/")}
              className="hover:text-amber-500 transition-colors cursor-pointer"
            >
              Accueil
            </a>
            <a href="/category" className="hover:text-amber-500 transition-colors">Produits</a>
            <a href="#" className="hover:text-amber-500 transition-colors">Catégories</a>
            <a href="#" className="hover:text-amber-500 transition-colors">Promotions</a>
            <a 
              onClick={() => handleOpenPopup('quote')} 
              className="hover:text-amber-500 transition-colors cursor-pointer"
            >
              Devis
            </a>
            <a 
              onClick={() => handleNavigation("/sign")}
              className="hover:text-amber-500 transition-colors cursor-pointer"
            >
              Se connecter
            </a>
            <a 
              onClick={() => handleNavigation("/contact")}
              className="hover:text-amber-500 transition-colors cursor-pointer"
            >
              Contact
            </a>
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
            <div className="p-4 flex items-center justify-center">
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
                type="text"
                placeholder="Rechercher un produit..."
                className="bg-transparent py-3 outline-none w-full text-sm text-gray-700 dark:text-gray-200 placeholder:text-gray-400"
                autoFocus={showSearch}
                />
                <button
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
            </div>
        </div>
        </header>
     )}
    </>
  );
};

export default Header;