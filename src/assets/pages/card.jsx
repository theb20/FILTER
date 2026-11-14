import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import {
  Heart,
  Share2,
  ChevronLeft,
  ChevronRight,
  Package,
  Car,
  Info,
  ShoppingCart,
  Mail,
} from "lucide-react";
import { DEMO_PRODUCTS, formatFcfa } from "../data/demoProducts.js";
import { SITE } from "../config/site.js";

// Fiche de démonstration : une seule fiche, alimentée par des données fictives.
const product = {
  ...DEMO_PRODUCTS[0],
  name: "Filtre à huile — fiche de démonstration",
  description:
    "Fiche d'exemple illustrant la structure d'une page produit Filter Finder : référence, compatibilité véhicule, disponibilité et prix en FCFA. Le contenu réel sera fourni par le catalogue.",
  specs: [
    { label: "Référence", value: DEMO_PRODUCTS[0].reference },
    { label: "Catégorie", value: DEMO_PRODUCTS[0].category },
    { label: "Type de filtre", value: "À renseigner" },
    { label: "Marque", value: "À renseigner" },
  ],
  compatibleVehicles: [
    { make: "Marque à renseigner", model: "Modèle à renseigner", years: "Années à renseigner" },
  ],
  images: ["/img/products/placeholder.svg"],
};

export default function ProductCard() {
  const [currentImage, setCurrentImage] = useState(0);
  const [isFavorite, setIsFavorite] = useState(false);
  const [activeTab, setActiveTab] = useState("description");

  useEffect(() => {
    document.title = "Fiche produit — Filter Finder";
  }, []);

  const count = product.images.length;
  const nextImage = () => setCurrentImage((p) => (p + 1) % count);
  const prevImage = () => setCurrentImage((p) => (p - 1 + count) % count);

  const tabs = [
    { id: "description", label: "Description" },
    { id: "specs", label: "Caractéristiques" },
    { id: "compat", label: "Compatibilité" },
  ];

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Galerie */}
      <div className="relative h-48 sm:h-64 md:h-80 lg:h-96 bg-gray-200">
        <img
          src={product.images[currentImage]}
          alt={product.name}
          className="w-full h-full object-contain"
        />

        {count > 1 && (
          <>
            <button
              type="button"
              aria-label="Image précédente"
              onClick={prevImage}
              className="absolute left-2 sm:left-4 top-1/2 -translate-y-1/2 bg-white/95 hover:bg-white p-2 sm:p-3 shadow-lg rounded-full"
            >
              <ChevronLeft className="w-4 h-4 sm:w-6 sm:h-6 text-gray-900" />
            </button>
            <button
              type="button"
              aria-label="Image suivante"
              onClick={nextImage}
              className="absolute right-2 sm:right-4 top-1/2 -translate-y-1/2 bg-white/95 hover:bg-white p-2 sm:p-3 shadow-lg rounded-full"
            >
              <ChevronRight className="w-4 h-4 sm:w-6 sm:h-6 text-gray-900" />
            </button>
          </>
        )}

        <div className="absolute top-3 sm:top-4 left-3 sm:left-4 flex gap-2">
          <span className="bg-amber-500 text-white px-3 py-1.5 text-xs sm:text-sm font-semibold rounded">
            {product.category}
          </span>
        </div>

        <div className="absolute top-3 sm:top-4 right-3 sm:right-4 flex gap-2">
          <button
            type="button"
            aria-label={isFavorite ? "Retirer des favoris" : "Ajouter aux favoris"}
            aria-pressed={isFavorite}
            onClick={() => setIsFavorite(!isFavorite)}
            className="bg-white/95 hover:bg-white p-2 sm:p-3 rounded-full shadow-lg"
          >
            <Heart
              className={`w-4 h-4 sm:w-5 sm:h-5 ${isFavorite ? "fill-red-500 text-red-500" : "text-gray-900"}`}
            />
          </button>
          <button
            type="button"
            aria-label="Partager"
            className="bg-white/95 hover:bg-white p-2 sm:p-3 rounded-full shadow-lg"
          >
            <Share2 className="w-4 h-4 sm:w-5 sm:h-5 text-gray-900" />
          </button>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-3 sm:px-4 lg:px-8 py-4 sm:py-6 lg:py-8">
        <p className="mb-4 flex items-center gap-2 text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded-lg px-4 py-3">
          <Info className="w-4 h-4 shrink-0" />
          Données de démonstration : cette fiche, sa référence et son prix sont fictifs.
        </p>

        <div className="grid lg:grid-cols-3 gap-4 sm:gap-6 lg:gap-8">
          {/* Colonne principale */}
          <div className="lg:col-span-2">
            <div className="bg-white rounded-lg sm:rounded-xl shadow-lg overflow-hidden">
              <div className="border-b border-gray-200 p-4 sm:p-6">
                <h1 className="text-xl sm:text-2xl lg:text-3xl font-bold text-gray-900 mb-2">
                  {product.name}
                </h1>
                <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500">
                  <span>Réf. {product.reference}</span>
                  <span className="flex items-center gap-1">
                    <Package className="w-4 h-4" /> {product.stock}
                  </span>
                </div>
              </div>

              <div className="flex overflow-x-auto border-b border-gray-200 bg-white" role="tablist">
                {tabs.map((tab) => (
                  <button
                    key={tab.id}
                    type="button"
                    role="tab"
                    aria-selected={activeTab === tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`px-4 sm:px-6 py-3 sm:py-4 text-sm sm:text-base font-medium whitespace-nowrap transition-colors ${
                      activeTab === tab.id
                        ? "text-amber-600 border-b-2 border-amber-500"
                        : "text-gray-600 hover:text-gray-900"
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>

              <div className="p-4 sm:p-6" role="tabpanel">
                {activeTab === "description" && (
                  <div>
                    <h2 className="text-xl sm:text-2xl font-bold text-gray-900 mb-3">Description</h2>
                    <p className="text-gray-700 leading-relaxed">{product.description}</p>
                  </div>
                )}

                {activeTab === "specs" && (
                  <div>
                    <h2 className="text-xl sm:text-2xl font-bold text-gray-900 mb-3">
                      Caractéristiques
                    </h2>
                    <dl className="space-y-2">
                      {product.specs.map((s) => (
                        <div key={s.label} className="flex justify-between py-2 border-b border-gray-200">
                          <dt className="text-gray-600">{s.label}</dt>
                          <dd className="font-semibold text-gray-900">{s.value}</dd>
                        </div>
                      ))}
                    </dl>
                  </div>
                )}

                {activeTab === "compat" && (
                  <div>
                    <h2 className="text-xl sm:text-2xl font-bold text-gray-900 mb-3">
                      Véhicules compatibles
                    </h2>
                    <ul className="space-y-2">
                      {product.compatibleVehicles.map((v, i) => (
                        <li key={i} className="flex items-center gap-3 bg-gray-50 p-3 rounded-lg">
                          <Car className="w-5 h-5 text-amber-600 shrink-0" aria-hidden="true" />
                          <span className="text-gray-700">
                            {v.make} — {v.model} ({v.years})
                          </span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Encart d'achat */}
          <div className="lg:col-span-1">
            <div className="sticky top-4 space-y-4">
              <div className="bg-white rounded-lg sm:rounded-xl shadow-lg p-4 sm:p-6">
                <div className="text-3xl font-bold text-amber-600">{formatFcfa(product.price)}</div>
                <div className="text-sm text-gray-500 mb-4">Prix indicatif (démo)</div>

                <div className="space-y-3">
                  <button
                    type="button"
                    disabled
                    title="Mode démonstration : cette action n'est pas encore connectée."
                    className="w-full bg-gray-300 text-gray-600 cursor-not-allowed py-3 px-4 font-semibold rounded-lg flex items-center justify-center gap-2"
                  >
                    <ShoppingCart className="w-5 h-5" />
                    Ajouter au panier
                  </button>
                  <p className="text-xs text-gray-500">
                    Mode démonstration : cette action n'est pas encore connectée.
                  </p>

                  <Link
                    to="/contact"
                    className="w-full border-2 border-gray-300 hover:border-gray-900 text-gray-900 py-3 px-4 font-semibold rounded-lg flex items-center justify-center gap-2 transition-colors"
                  >
                    <Mail className="w-5 h-5" />
                    Nous contacter
                  </Link>
                </div>

                <p className="mt-4 text-sm text-gray-600">
                  {SITE.phone
                    ? `Téléphone : ${SITE.phone}`
                    : "Coordonnées téléphoniques bientôt disponibles."}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
