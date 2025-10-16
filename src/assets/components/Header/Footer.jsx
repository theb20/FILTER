import { Link } from "react-router-dom";
import { FaFacebookF, FaTwitter, FaInstagram, FaLinkedinIn } from "react-icons/fa";

const Footer = () => {
  return (
    <footer className="bg-gray-100 absolute bottom-0 w-full pb-3 dark:bg-gray-900 text-gray-800 dark:text-gray-200 pt-10">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-8 md:gap-0">
          
          {/* Logo */}
          <div className="text-2xl font-extrabold text-amber-500 cursor-pointer select-none">
            <Link to="/">
              Filter<span className="text-gray-800 dark:text-white">Finder</span>
            </Link>
          </div>

          {/* Liens */}
          <div className="flex flex-col md:flex-row gap-6">
            <Link to="/" className="hover:text-amber-500 transition">Accueil</Link>
            <Link to="/features" className="hover:text-amber-500 transition">Fonctionnalités</Link>
            <Link to="/pricing" className="hover:text-amber-500 transition">Tarifs</Link>
            <Link to="/contact" className="hover:text-amber-500 transition">Contact</Link>
          </div>

          {/* Réseaux sociaux */}
          <div className="flex gap-4">

            <a href="#" className="hover:text-amber-500 transition"><FaFacebookF /></a>
            <a href="#" className="hover:text-amber-500 transition"><FaTwitter /></a>
            <a href="#" className="hover:text-amber-500 transition"><FaInstagram /></a>
            <a href="#" className="hover:text-amber-500 transition"><FaLinkedinIn /></a>
          </div>
        </div>

        {/* Copyright */}
        <div className="mt-10 border-t border-gray-300 dark:border-gray-700 pt-4 text-center text-sm">
          © {new Date().getFullYear()} FilterFinder. Tous droits réservés.
        </div>
      </div>
    </footer>
  );
};

export default Footer;
