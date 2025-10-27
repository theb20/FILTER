import React, { useState } from 'react';
import { Search, ChevronDown, Building2, Package, Wrench, Cpu } from 'lucide-react';

export default function SearchModule() {
  const [searchType, setSearchType] = useState("");
  const [category, setCategory] = useState("");
  const [query, setQuery] = useState("");
  const [quantity, setQuantity] = useState("");
  const [budget, setBudget] = useState("1 000 €");
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false);
  const [activeFilters, setActiveFilters] = useState([]);
  
  // États pour le formulaire RFQ
  const [rfqProductName, setRfqProductName] = useState("");
  const [rfqDescription, setRfqDescription] = useState("");
  const [rfqQuantity, setRfqQuantity] = useState("");
  const [rfqBudget, setRfqBudget] = useState("");
  const [rfqDeadline, setRfqDeadline] = useState("");
  const [rfqSubmitted, setRfqSubmitted] = useState(false);
  
  // Simuler des résultats pour chaque type
  const allResults = {
    products: [
      { name: "Imprimante laser HP LaserJet Pro", category: "tech", price: 450 },
      { name: "Bureau professionnel en chêne", category: "all", price: 890 },
      { name: "Chaise ergonomique Herman Miller", category: "all", price: 1200 },
      { name: "Tablette graphique Wacom", category: "tech", price: 350 },
      { name: "Perceuse sans fil Makita", category: "tools", price: 180 },
      { name: "Échafaudage aluminium 6m", category: "industrial", price: 2500 },
      { name: "Ordinateur portable Dell", category: "tech", price: 1100 },
      { name: "Marteau-piqueur Bosch", category: "tools", price: 650 }
    ],
    suppliers: [
      { name: "TechPro Supplies", category: "tech", rating: 4.8 },
      { name: "Outillage Expert", category: "tools", rating: 4.6 },
      { name: "Industrie Solutions", category: "industrial", rating: 4.9 },
      { name: "Bureau & Co", category: "all", rating: 4.5 }
    ],
  };
  
  const categories = [
    { id: "all", label: "Toutes les catégories", icon: Package },
    { id: "tools", label: "Outils & Matériel", icon: Wrench },
    { id: "tech", label: "Électronique & IT", icon: Cpu },
    { id: "industrial", label: "Industrie & BTP", icon: Building2 },
  ];

  const filterOptions = ["Stock immédiat", "Livraison rapide", "Remise volume", "Made in France"];

  const toggleFilter = (filter) => {
    setActiveFilters(prev => 
      prev.includes(filter) 
        ? prev.filter(f => f !== filter)
        : [...prev, filter]
    );
  };

  const handleSearch = () => {
    // Simulation d'une recherche
    console.log("Recherche lancée:", { searchType, category, query, quantity, budget, activeFilters });
  };

  const handleRfqSubmit = () => {
    if (!rfqProductName || !rfqDescription || !rfqQuantity) {
      alert("Veuillez remplir les champs obligatoires (*)");
      return;
    }
    
    console.log("Demande de devis envoyée:", {
      productName: rfqProductName,
      description: rfqDescription,
      quantity: rfqQuantity,
      budget: rfqBudget,
      deadline: rfqDeadline
    });
    
    setRfqSubmitted(true);
    setTimeout(() => {
      setRfqSubmitted(false);
      setRfqProductName("");
      setRfqDescription("");
      setRfqQuantity("");
      setRfqBudget("");
      setRfqDeadline("");
    }, 3000);
  };

  const getFilteredResults = () => {
    if (searchType === 'rfq' || !searchType) return [];
    
    const results = allResults[searchType] || [];
    
    return results.filter(item => {
      const matchesQuery = !query || item.name.toLowerCase().includes(query.toLowerCase());
      const matchesCategory = !category || category === 'all' || item.category === category || item.category === 'all';
      return matchesQuery && matchesCategory;
    });
  };

  const filteredResults = getFilteredResults();
  
  // Vérifier si on doit afficher les résultats
  const shouldShowResults = searchType && searchType !== 'rfq' && (query || category);

  return (
    <div className="relative z-100 p-4 sm:p-8">
      <div className="max-w-5xl  mx-auto">
        <div className="bg-white rounded-2xl overflow-hidden shadow-xl border border-gray-200">
          {/* Tabs */}
          <div className="border-b border-gray-200">
            <div className="flex">
              {['products', 'suppliers', 'rfq'].map((type) => (
                <button
                  key={type}
                  onClick={() => {
                    setSearchType(type);
                    setQuery("");
                    setCategory("");
                  }}
                  className={`flex-1 px-6 py-4 text-sm font-medium transition-all duration-150 active:scale-95 ${
                    searchType === type
                      ? 'text-amber-600 border-b-2 border-amber-500 bg-gray-50'
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  {type === 'products' && 'Recherche produits'}
                  {type === 'suppliers' && 'Trouver un fournisseur'}
                  {type === 'rfq' && 'Demande de devis'}
                </button>
              ))}
            </div>
          </div>

          {/* Catégories - cachées pour RFQ et si aucun type sélectionné */}
          {searchType && searchType !== 'rfq' && (
            <div className="p-6 border-b border-gray-200 bg-gray-50">
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                {categories.map((cat) => {
                  const Icon = cat.icon;
                  return (
                    <button
                      key={cat.id}
                      onClick={() => setCategory(cat.id)}
                      className={`flex items-center justify-center gap-2 px-4 py-3 rounded-lg border text-sm font-medium transition-all duration-150 active:scale-95 ${
                        category === cat.id
                          ? 'border-gray-900 bg-white text-gray-900 shadow-sm'
                          : 'border-gray-300 bg-white text-gray-600 hover:border-gray-400'
                      }`}
                    >
                      <Icon className="w-4 h-4" />
                      <span className="hidden sm:inline">{cat.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Champs de recherche */}
          <div className="p-6 sm:p-8">
            {searchType === 'rfq' ? (
              /* Formulaire Demande de Devis */
              <div className="space-y-6">
                {rfqSubmitted && (
                  <div className="bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg">
                    ✓ Votre demande de devis a été envoyée avec succès !
                  </div>
                )}
                
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                    Nom du produit recherché *
                  </label>
                  <input
                    type="text"
                    value={rfqProductName}
                    onChange={(e) => setRfqProductName(e.target.value)}
                    placeholder="Ex: Imprimante laser couleur professionnelle"
                    className="w-full px-4 py-3 rounded-lg border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white transition-all"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                    Description détaillée du besoin *
                  </label>
                  <textarea
                    rows="6"
                    value={rfqDescription}
                    onChange={(e) => setRfqDescription(e.target.value)}
                    placeholder="Décrivez précisément le produit recherché : caractéristiques techniques, spécifications, usage prévu, contraintes particulières..."
                    className="w-full px-4 py-3 rounded-lg border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white transition-all resize-none"
                  ></textarea>
                </div>

                <div className="grid sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                      Quantité souhaitée *
                    </label>
                    <input
                      type="number"
                      value={rfqQuantity}
                      onChange={(e) => setRfqQuantity(e.target.value)}
                      placeholder="10"
                      min="1"
                      className="w-full px-4 py-3 rounded-lg border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                      Budget indicatif
                    </label>
                    <input
                      type="text"
                      value={rfqBudget}
                      onChange={(e) => setRfqBudget(e.target.value)}
                      placeholder="Ex: 5 000 € - 10 000 €"
                      className="w-full px-4 py-3 rounded-lg border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white transition-all"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                    Délai de livraison souhaité
                  </label>
                  <input
                    type="text"
                    value={rfqDeadline}
                    onChange={(e) => setRfqDeadline(e.target.value)}
                    placeholder="Ex: Sous 2 semaines, Fin de mois..."
                    className="w-full px-4 py-3 rounded-lg border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white transition-all"
                  />
                </div>

                <button 
                  onClick={handleRfqSubmit}
                  className="w-full bg-amber-500 text-white py-4 px-6 rounded-lg hover:bg-amber-600 transition-all duration-150 active:scale-95 flex items-center justify-center gap-2 font-medium shadow-lg hover:shadow-xl"
                >
                  <span>Envoyer ma demande de devis</span>
                </button>

                <p className="text-xs text-gray-500 text-center">
                  Vous recevrez des réponses de fournisseurs qualifiés sous 24-48h
                </p>
              </div>
            ) : searchType ? (
              /* Recherche normale (produits et fournisseurs) */
              <>
                <div className="grid sm:grid-cols-12 gap-4">
                  {/* Recherche produit */}
                  <div className="sm:col-span-6">
                    <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                      {searchType === 'products' ? 'Produit ou référence' : 'Nom du fournisseur'}
                    </label>
                    <div className="relative">
                      <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                      <input
                        type="text"
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        placeholder={searchType === 'products' ? "Ex: Imprimante laser, bureau professionnel..." : "Ex: TechPro, Outillage Expert..."}
                        className="w-full pl-12 pr-4 py-3 rounded-lg border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white transition-all"
                      />
                    </div>
                  </div>

                  {/* Quantité */}
                  <div className="sm:col-span-2">
                    <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                      Quantité
                    </label>
                    <input
                      type="number"
                      value={quantity}
                      onChange={(e) => setQuantity(e.target.value)}
                      placeholder="10"
                      min="1"
                      className="w-full px-4 py-3 rounded-lg border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white transition-all"
                    />
                  </div>

                  {/* Budget */}
                  <div className="sm:col-span-2">
                    <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                      Budget max
                    </label>
                    <div className="relative">
                      <select 
                        value={budget}
                        onChange={(e) => setBudget(e.target.value)}
                        className="w-full px-4 py-3 rounded-lg border border-gray-300 text-gray-900 focus:border-gray-900 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-opacity-10 bg-white appearance-none cursor-pointer transition-all"
                      >
                        <option>50 000 Fcfa</option>
                        <option>100 000 Fcfa</option>
                        <option>150 000 Fcfa</option>
                        <option>200 000 Fcfa</option>
                        <option>250 000 Fcfa</option>
                        <option>+300 000 Fcfa</option>
                      </select>
                      <ChevronDown className="absolute right-4 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>

                  {/* Bouton Rechercher */}
                  <div className="sm:col-span-2 flex items-end">
                    <button 
                      onClick={handleSearch}
                      className="w-full bg-gray-900 text-white py-3 px-6 rounded-lg hover:bg-gray-800 transition-all duration-150 active:scale-95 flex items-center justify-center gap-2 font-medium shadow-lg hover:shadow-xl"
                    >
                      <Search className="w-5 h-5" />
                      <span>Rechercher</span>
                    </button>
                  </div>
                </div>

                {/* Filtres avancés */}
                <div className="mt-6 pt-6 border-t border-gray-200">
                  <div className="flex flex-wrap items-center gap-4">
                    <button 
                      onClick={() => setShowAdvancedFilters(!showAdvancedFilters)}
                      className="text-sm text-gray-600 hover:text-gray-900 font-medium flex items-center gap-2 transition-all duration-150 active:scale-95"
                    >
                      <span>Filtres avancés</span>
                      <ChevronDown className={`w-4 h-4 transition-transform ${showAdvancedFilters ? 'rotate-180' : ''}`} />
                    </button>
                    <div className="flex flex-wrap gap-2">
                      {filterOptions.map((filter) => (
                        <button
                          key={filter}
                          onClick={() => toggleFilter(filter)}
                          className={`px-3 py-1.5 rounded-full text-xs font-medium transition-all duration-150 active:scale-95 ${
                            activeFilters.includes(filter)
                              ? 'bg-amber-500 text-white'
                              : 'bg-gray-100 hover:bg-gray-200 text-gray-700'
                          }`}
                        >
                          {filter}
                        </button>
                      ))}
                    </div>
                  </div>
                  
                  {showAdvancedFilters && (
                    <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                      <p className="text-sm text-gray-600">Filtres avancés appliqués: {activeFilters.length > 0 ? activeFilters.join(', ') : 'Aucun'}</p>
                    </div>
                  )}
                </div>
              </>
            ) : (
              /* Message d'invitation si aucun onglet sélectionné */
              <div className="text-center py-12">
                <Package className="w-16 h-16 text-gray-300 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-gray-900 mb-2">
                  Commencez votre recherche
                </h3>
                <p className="text-gray-500">
                  Sélectionnez un onglet ci-dessus pour rechercher des produits, trouver un fournisseur ou faire une demande de devis
                </p>
              </div>
            )}
          </div>
        </div>

        {/* Résultats dynamiques - affichés seulement si recherche active */}
        {shouldShowResults && (
          <div className="mt-6 bg-white absolute top-50 left-0 right-0 transform translate-y-60 rounded-2xl overflow-hidden shadow-xl border border-gray-200 p-6">
            <h4 className="text-sm font-semibold mb-3 text-gray-900">
              {filteredResults.length} Résultat{filteredResults.length > 1 ? 's' : ''} : {searchType === "products" ? "Produits" : "Fournisseurs"}
            </h4>
            {filteredResults.length > 0 ? (
              <ul className="space-y-2">
                {filteredResults.map((item, idx) => (
                  <li key={idx} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer">
                    <div className="flex items-center gap-3">
                      <div className="w-2 h-2 bg-amber-500 rounded-full"></div>
                      <span className="text-gray-900 font-medium">{item.name}</span>
                    </div>
                    <div className="text-sm text-gray-600">
                      {searchType === 'products' ? `${item.price} €` : `★ ${item.rating}`}
                    </div>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-gray-500 italic text-center py-8">
                Aucun résultat trouvé pour votre recherche
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}