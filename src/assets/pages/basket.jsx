import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { Trash2, ChevronDown, ChevronUp, Info, Shield } from "lucide-react";
import { DEMO_PRODUCTS, formatFcfa } from "../data/demoProducts.js";
import { DEMO_NOTICE } from "../config/site.js";
import { useNotify } from "../configurations/notificationContext.js";

// Panier de démonstration : non relié au bouton « Ajouter au panier ».
const INITIAL_ITEMS = [
  { ...DEMO_PRODUCTS[0], quantity: 1 },
  { ...DEMO_PRODUCTS[1], quantity: 2 },
  { ...DEMO_PRODUCTS[3], quantity: 1 },
];

export default function ShoppingCart() {
  const notify = useNotify();
  const [cartItems, setCartItems] = useState(INITIAL_ITEMS);
  const [promoCode, setPromoCode] = useState("");

  useEffect(() => {
    document.title = "Panier — Filter Finder";
  }, []);

  const updateQuantity = (id, newQuantity) => {
    if (newQuantity < 1) return;
    setCartItems((items) =>
      items.map((item) => (item.id === id ? { ...item, quantity: newQuantity } : item)),
    );
  };

  const removeItem = (id) => {
    setCartItems((items) => items.filter((item) => item.id !== id));
  };

  const applyPromoCode = () => {
    // Aucun code promo réel n'existe : la saisie n'est pas encore connectée.
    notify(DEMO_NOTICE, "demo");
    setPromoCode("");
  };

  const subtotal = cartItems.reduce((sum, item) => sum + item.price * item.quantity, 0);
  const totalItems = cartItems.reduce((sum, item) => sum + item.quantity, 0);

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 py-6 lg:py-8">
        <nav aria-label="Fil d'Ariane" className="text-sm text-gray-600 mb-6">
          <Link to="/" className="hover:text-amber-600">
            Accueil
          </Link>
          <span className="mx-2">/</span>
          <span className="font-medium text-gray-900">Panier</span>
        </nav>

        <div className="mb-6">
          <h1 className="text-2xl sm:text-3xl font-bold text-gray-900 mb-2">Panier</h1>
          <p className="text-gray-600">
            {totalItems} article{totalItems > 1 ? "s" : ""}
          </p>
        </div>

        <p className="mb-6 flex items-center gap-2 text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded-lg px-4 py-3">
          <Info className="w-4 h-4 shrink-0" />
          Données de démonstration : les articles, prix et le paiement sont fictifs.
        </p>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Articles */}
          <div className="lg:col-span-2 space-y-4">
            {cartItems.length === 0 ? (
              <div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
                <p className="text-gray-700 mb-4">Votre panier est vide.</p>
                <Link
                  to="/category"
                  className="inline-block px-6 py-3 bg-black text-white rounded-full hover:bg-amber-500 transition"
                >
                  Voir le catalogue
                </Link>
              </div>
            ) : (
              <ul className="bg-white rounded-lg border border-gray-200 divide-y divide-gray-200">
                {cartItems.map((item) => (
                  <li key={item.id} className="p-4 sm:p-6">
                    <div className="flex gap-4">
                      <img
                        src={item.image}
                        alt={item.name}
                        loading="lazy"
                        className="w-24 h-24 sm:w-32 sm:h-32 object-cover rounded border border-gray-200 shrink-0"
                      />

                      <div className="flex-1 min-w-0">
                        <h2 className="font-medium text-gray-900 mb-1 text-sm sm:text-base">
                          {item.name}
                        </h2>
                        <p className="text-xs text-gray-500 mb-2">Réf. {item.reference}</p>
                        <span className="inline-block text-xs font-medium text-green-700 bg-green-50 px-2 py-1 rounded mb-3">
                          {item.stock}
                        </span>

                        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                          <p className="text-xl sm:text-2xl font-bold text-gray-900">
                            {formatFcfa(item.price)}
                          </p>

                          <div className="flex items-center border border-gray-300 rounded w-fit">
                            <button
                              type="button"
                              aria-label={`Diminuer la quantité de ${item.name}`}
                              onClick={() => updateQuantity(item.id, item.quantity - 1)}
                              className="px-3 py-2 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                              disabled={item.quantity <= 1}
                            >
                              <ChevronDown className="w-4 h-4" />
                            </button>
                            <span className="px-4 py-2 font-medium min-w-[40px] text-center">
                              {item.quantity}
                            </span>
                            <button
                              type="button"
                              aria-label={`Augmenter la quantité de ${item.name}`}
                              onClick={() => updateQuantity(item.id, item.quantity + 1)}
                              className="px-3 py-2 hover:bg-gray-50"
                            >
                              <ChevronUp className="w-4 h-4" />
                            </button>
                          </div>
                        </div>

                        <div className="mt-4 pt-4 border-t border-gray-100">
                          <button
                            type="button"
                            onClick={() => removeItem(item.id)}
                            className="text-sm text-gray-700 hover:text-red-600 flex items-center gap-1"
                          >
                            <Trash2 className="w-4 h-4" />
                            Supprimer
                          </button>
                        </div>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Récapitulatif */}
          <div className="lg:col-span-1">
            <div className="bg-white rounded-lg border border-gray-200 lg:sticky lg:top-40 p-6">
              <h2 className="text-lg font-bold text-gray-900 mb-4">Récapitulatif</h2>

              <form
                className="mb-6"
                onSubmit={(e) => {
                  e.preventDefault();
                  applyPromoCode();
                }}
              >
                <label htmlFor="promo" className="block text-sm font-medium text-gray-700 mb-2">
                  Code promo
                </label>
                <div className="flex gap-2">
                  <input
                    id="promo"
                    type="text"
                    value={promoCode}
                    onChange={(e) => setPromoCode(e.target.value.toUpperCase())}
                    placeholder="Entrez votre code"
                    className="flex-1 min-w-0 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-amber-500"
                  />
                  <button
                    type="submit"
                    className="px-4 py-2 bg-gray-900 text-white text-sm font-medium rounded hover:bg-gray-800"
                  >
                    Appliquer
                  </button>
                </div>
              </form>

              <div className="space-y-3 pb-4 border-b border-gray-200 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-600">
                    Sous-total ({totalItems} article{totalItems > 1 ? "s" : ""})
                  </span>
                  <span className="font-medium text-gray-900">{formatFcfa(subtotal)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Livraison</span>
                  <span className="text-gray-500">À calculer</span>
                </div>
              </div>

              <div className="py-4 border-b border-gray-200">
                <div className="flex justify-between items-center">
                  <span className="font-bold text-gray-900">Total</span>
                  <span className="text-2xl font-bold text-gray-900">{formatFcfa(subtotal)}</span>
                </div>
              </div>

              <button
                type="button"
                disabled
                aria-describedby="checkout-demo"
                className="w-full mt-6 bg-gray-300 text-gray-600 cursor-not-allowed font-bold py-3 px-4 rounded text-sm"
              >
                Procéder au paiement
              </button>
              <p id="checkout-demo" className="mt-3 text-xs text-amber-800 bg-amber-50 border border-amber-200 rounded p-2">
                {DEMO_NOTICE}
              </p>
            </div>

            <div className="mt-4 bg-white rounded-lg border border-gray-200 p-4 flex items-center gap-3">
              <Shield className="w-8 h-8 text-gray-400" aria-hidden="true" />
              <p className="text-xs text-gray-600">
                Les modes de paiement (mobile money, carte) seront précisés lors de la mise en service.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
