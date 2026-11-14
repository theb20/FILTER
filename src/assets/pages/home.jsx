import { useState, useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import {
  Search,
  Car,
  Wrench,
  Truck,
  ShieldCheck,
  Clock,
  Heart,
  ArrowRight,
  Phone,
  Info,
} from "lucide-react";
import SearchModule from "../components/Search";
import { DEMO_PRODUCTS, formatFcfa } from "../data/demoProducts.js";

const WORDS = [
  "pour votre véhicule",
  "à huile, à air, à carburant",
  "compatibles avec votre modèle",
  "livrés en Côte d'Ivoire",
];

const FEATURES = [
  {
    icon: Search,
    title: "Recherche par référence",
    description:
      "Retrouvez un filtre à partir de sa référence constructeur ou de votre véhicule.",
  },
  {
    icon: Car,
    title: "Compatibilité véhicule",
    description:
      "Vérifiez qu'un filtre correspond à votre marque, modèle et motorisation.",
  },
  {
    icon: Wrench,
    title: "Devis professionnel",
    description:
      "Garagistes et flottes : demandez un devis pour vos besoins en volume, en FCFA.",
  },
  {
    icon: Truck,
    title: "Livraison en Côte d'Ivoire",
    description: "Une offre pensée pour le marché ivoirien (modalités à confirmer).",
  },
];

export default function FilterFinderHome() {
  const videoRef = useRef(null);
  const [currentWordIndex, setCurrentWordIndex] = useState(0);
  const [displayedText, setDisplayedText] = useState("");
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    document.title = "Filter Finder — Filtres automobiles en Côte d'Ivoire";
  }, []);

  useEffect(() => {
    const currentWord = WORDS[currentWordIndex];
    let pause;
    const timeout = setTimeout(
      () => {
        if (!isDeleting) {
          if (displayedText.length < currentWord.length) {
            setDisplayedText(currentWord.substring(0, displayedText.length + 1));
          } else {
            pause = setTimeout(() => setIsDeleting(true), 2500);
          }
        } else if (displayedText.length > 0) {
          setDisplayedText(currentWord.substring(0, displayedText.length - 1));
        } else {
          setIsDeleting(false);
          setCurrentWordIndex((prev) => (prev + 1) % WORDS.length);
        }
      },
      isDeleting ? 40 : 120,
    );

    return () => {
      clearTimeout(timeout);
      clearTimeout(pause);
    };
  }, [displayedText, isDeleting, currentWordIndex]);

  useEffect(() => {
    const video = videoRef.current;
    if (video && video.paused) {
      video.play().catch(() => {
        /* lecture automatique refusée par le navigateur : le poster reste affiché */
      });
    }
  }, []);

  return (
    <div className="min-h-screen bg-white">
      {/* Hero */}
      <section className="relative py-32 text-white">
        <div className="absolute inset-0 z-0 overflow-hidden">
          <video
            ref={videoRef}
            autoPlay
            loop
            muted
            playsInline
            preload="metadata"
            poster="/img/background/hero-poster.webp"
            aria-hidden="true"
            className="absolute top-1/2 left-1/2 min-w-full min-h-full w-auto h-auto -translate-x-1/2 -translate-y-1/2 object-cover"
          >
            <source src="/videos/hero.mp4" type="video/mp4" />
          </video>
          <div className="absolute inset-0 bg-gradient-to-br from-gray-900/90 via-gray-800/80 to-transparent"></div>
        </div>

        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-center mb-8">
            <div className="inline-flex items-center gap-6 px-6 py-3 rounded-full bg-white border border-gray-200 shadow-sm">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-gray-600" />
                <span className="text-xs font-medium text-gray-600">Qualité</span>
              </div>
              <div className="w-px h-4 bg-gray-300"></div>
              <div className="flex items-center gap-2">
                <Car className="w-4 h-4 text-gray-600" />
                <span className="text-xs font-medium text-gray-600">Filtres auto</span>
              </div>
              <div className="w-px h-4 bg-gray-300"></div>
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-gray-600" />
                <span className="text-xs font-medium text-gray-600">Côte d'Ivoire</span>
              </div>
            </div>
          </div>

          <div className="max-w-4xl mx-auto text-center mb-16">
            <h1 className="text-4xl min-h-[130px] lg:min-h-[160px] sm:text-5xl lg:text-6xl font-light text-gray-100 mb-6 leading-tight tracking-tight">
              Trouvez le bon filtre{" "}
              <br className="hidden sm:block" />
              <span className="relative inline-block">
                <span className="font-normal text-amber-500">{displayedText}</span>
                <span
                  aria-hidden="true"
                  className="inline-block w-0.5 h-12 sm:h-14 lg:h-16 bg-gray-400 ml-1 align-middle animate-cursor"
                ></span>
              </span>
            </h1>

            <p className="text-lg sm:text-xl text-gray-100 max-w-3xl mx-auto font-light leading-relaxed mb-8">
              Filter Finder vous aide à identifier et commander les filtres
              automobiles adaptés à votre véhicule, avec des prix en FCFA.
            </p>
          </div>

          <div className="max-w-5xl relative z-100 rounded-24 mx-auto">
            <SearchModule />
          </div>
        </div>
      </section>

      {/* Sélection (données de démonstration) */}
      <section className="py-12 sm:py-16 lg:py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-8 sm:mb-12 gap-4">
            <div>
              <h2 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-gray-900 mb-2">
                Filtres à la une
              </h2>
              <p className="text-gray-600">Aperçu de la sélection du catalogue</p>
            </div>
            <Link
              to="/category"
              className="inline-flex items-center gap-2 px-6 py-2.5 border-2 border-gray-300 hover:border-amber-500 rounded-lg font-semibold text-gray-700 hover:text-amber-600 transition-all"
            >
              Voir tout le catalogue
              <ArrowRight className="w-5 h-5" />
            </Link>
          </div>

          <p className="mb-6 flex items-center gap-2 text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded-lg px-4 py-3">
            <Info className="w-4 h-4 shrink-0" />
            Données de démonstration : produits, références et prix sont fictifs.
          </p>

          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {DEMO_PRODUCTS.map((product) => (
              <Link
                key={product.id}
                to="/card"
                className="bg-white rounded-xl shadow-lg overflow-hidden hover:shadow-2xl transition-shadow group"
              >
                <div className="relative h-48 sm:h-56 overflow-hidden">
                  <img
                    src={product.image}
                    alt={product.name}
                    loading="lazy"
                    className="w-full h-full object-cover group-hover:scale-110 transition-transform duration-500"
                  />
                  {product.isNew && (
                    <span className="absolute top-4 left-4 bg-green-500 text-white px-3 py-1 rounded-full text-xs font-bold">
                      Nouveau
                    </span>
                  )}
                  <span className="absolute top-4 right-4 bg-white/90 p-2 rounded-full">
                    <Heart className="w-5 h-5 text-gray-700" aria-hidden="true" />
                  </span>
                </div>
                <div className="p-4 sm:p-6">
                  <div className="flex items-center justify-between mb-3">
                    <span className="text-2xl font-bold text-amber-600">
                      {formatFcfa(product.price)}
                    </span>
                    <span className="text-sm text-gray-500">{product.category}</span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 mb-2 line-clamp-2 group-hover:text-amber-600 transition-colors">
                    {product.name}
                  </h3>
                  <p className="text-sm text-gray-600">Réf. {product.reference}</p>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Pourquoi Filter Finder */}
      <section className="py-12 sm:py-16 lg:py-20 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12 sm:mb-16">
            <h2 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-gray-900 mb-4">
              Pourquoi choisir Filter Finder ?
            </h2>
            <p className="text-lg text-gray-600 max-w-2xl mx-auto">
              Une plateforme dédiée aux filtres automobiles, pour particuliers et professionnels.
            </p>
          </div>

          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6 sm:gap-8">
            {FEATURES.map((feature) => {
              const Icon = feature.icon;
              return (
                <div
                  key={feature.title}
                  className="bg-white p-6 sm:p-8 rounded-xl shadow-lg hover:shadow-2xl transition-shadow"
                >
                  <div className="inline-flex items-center justify-center w-14 h-14 bg-amber-100 rounded-xl mb-4">
                    <Icon className="w-7 h-7 text-amber-600" aria-hidden="true" />
                  </div>
                  <h3 className="text-lg sm:text-xl font-bold text-gray-900 mb-3">
                    {feature.title}
                  </h3>
                  <p className="text-gray-600 leading-relaxed">{feature.description}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-16 sm:py-20 lg:py-24 bg-gradient-to-br from-amber-500 to-amber-700 text-white">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-6">
            Un filtre à trouver ?
          </h2>
          <p className="text-lg sm:text-xl text-amber-50 mb-8 sm:mb-10">
            Parcourez le catalogue ou contactez l'équipe Filter Finder.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link
              to="/category"
              className="px-8 py-4 bg-white text-amber-700 font-bold rounded-lg hover:bg-amber-50 transition-colors flex items-center justify-center gap-2 shadow-xl"
            >
              Voir le catalogue
              <ArrowRight className="w-5 h-5" />
            </Link>
            <Link
              to="/contact"
              className="px-8 py-4 border-2 border-white text-white font-bold rounded-lg hover:bg-white/10 transition-colors flex items-center justify-center gap-2"
            >
              <Phone className="w-5 h-5" />
              Nous contacter
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
