import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence  } from "framer-motion";
import { MapPin, Home, Building2, Warehouse, Users, Bed, DollarSign, TrendingUp, Star, ChevronRight, Filter, Map, Grid, List, Heart, ArrowRight, CheckCircle, Clock, Shield, Award, Phone, Mail, Menu, X, SlidersHorizontal , ChevronDown,  Search, Package, Wrench, ShieldCheck, BarChart3,  Cpu } from 'lucide-react';

import Video from "../../../public/videos/industrial_dark.mp4";
import Video1 from "../../../public/videos/chantier.mp4";

export default function FilterFinderHome() {
  const [searchType, setSearchType] = useState('buy');
    const [index, setIndex] = useState(0);
  const [category, setCategory] = useState("all");
    const [currentWordIndex, setCurrentWordIndex] = useState(0);
  const [showMobileMenu, setShowMobileMenu] = useState(false);
    const [displayedText, setDisplayedText] = useState('');
  const [viewMode, setViewMode] = useState('grid');
    const [isDeleting, setIsDeleting] = useState(false);

  const words = ['équipements industriels', 'fournitures de bureau', 'matériel professionnel', 'solutions logistiques', 'outils de production'];
// ⏱ Changement de mot toutes les 3 secondes
  useEffect(() => {
    const interval = setInterval(() => {
      setIndex((prev) => (prev + 1) % words.length);
    }, 3000);
    return () => clearInterval(interval);
  }, [words.length]);

  const categories = [
    { id: "all", label: "Tous les produits", icon: Package },
    { id: "tools", label: "Outils & Matériel", icon: Wrench },
    { id: "tech", label: "Électronique & IT", icon: Cpu },
    { id: "industrial", label: "Industrie & BTP", icon: Building2 },
  ];

  const featuredProperties = [
    {
      id: 1,
      image: "https://images.unsplash.com/photo-1600596542815-ffad4c1539a9?w=800&h=600&fit=crop",
      title: "Appartement moderne centre-ville",
      price: 485000,
      location: "Paris 8ème",
      beds: 3,
      baths: 2,
      surface: 100,
      type: "Appartement",
      featured: true,
      new: true
    },
    {
      id: 2,
      image: "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?w=800&h=600&fit=crop",
      title: "Villa contemporaine avec piscine",
      price: 1250000,
      location: "Neuilly-sur-Seine",
      beds: 5,
      baths: 3,
      surface: 250,
      type: "Maison",
      featured: true
    },
    {
      id: 3,
      image: "https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?w=800&h=600&fit=crop",
      title: "Duplex lumineux vue Seine",
      price: 680000,
      location: "Paris 15ème",
      beds: 4,
      baths: 2,
      surface: 130,
      type: "Appartement",
      featured: true
    },
    {
      id: 4,
      image: "https://images.unsplash.com/photo-1600566753190-17f0baa2a6c3?w=800&h=600&fit=crop",
      title: "Loft industriel rénové",
      price: 520000,
      location: "Paris 11ème",
      beds: 2,
      baths: 1,
      surface: 90,
      type: "Appartement",
      new: true
    },
    {
      id: 5,
      image: "https://images.unsplash.com/photo-1600047509807-ba8f99d2cdde?w=800&h=600&fit=crop",
      title: "Maison familiale avec jardin",
      price: 890000,
      location: "Versailles",
      beds: 6,
      baths: 3,
      surface: 200,
      type: "Maison"
    },
    {
      id: 6,
      image: "https://images.unsplash.com/photo-1600607687644-c7171b42498b?w=800&h=600&fit=crop",
      title: "Penthouse vue panoramique",
      price: 1850000,
      location: "Paris 16ème",
      beds: 4,
      baths: 3,
      surface: 180,
      type: "Appartement",
      featured: true
    }
  ];

  const stats = [
    { icon: Home, value: "15,000+", label: "Biens disponibles" },
    { icon: Users, value: "50,000+", label: "Clients satisfaits" },
    { icon: Building2, value: "500+", label: "Agences partenaires" },
    { icon: Award, value: "15 ans", label: "D'expérience" }
  ];

  const features = [
    {
      icon: Search,
      title: "Recherche intelligente",
      description: "Notre algorithme trouve les biens qui correspondent exactement à vos critères"
    },
    {
      icon: Map,
      title: "Carte interactive",
      description: "Visualisez tous les biens disponibles sur une carte pour trouver l'emplacement idéal"
    },
    {
      icon: Shield,
      title: "Transactions sécurisées",
      description: "Tous nos biens sont vérifiés et les transactions sont 100% sécurisées"
    },
    {
      icon: Clock,
      title: "Support 24/7",
      description: "Notre équipe d'experts est disponible pour vous accompagner à tout moment"
    }
  ];

  const testimonials = [
    {
      name: "Marie Dubois",
      role: "Acheteuse",
      photo: "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=200&h=200&fit=crop",
      rating: 5,
      text: "Excellente expérience ! J'ai trouvé l'appartement de mes rêves en moins de 2 semaines. L'équipe est très professionnelle."
    },
    {
      name: "Pierre Martin",
      role: "Vendeur",
      photo: "https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=200&h=200&fit=crop",
      rating: 5,
      text: "Service impeccable du début à la fin. Mon bien a été vendu rapidement et au meilleur prix. Je recommande vivement."
    },
    {
      name: "Sophie Laurent",
      role: "Locataire",
      photo: "https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=200&h=200&fit=crop",
      rating: 5,
      text: "La recherche était simple et intuitive. J'ai pu filtrer exactement ce que je cherchais et trouver rapidement."
    }
  ];

  const formatPrice = (price) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      maximumFractionDigits: 0
    }).format(price);
  };
  useEffect(() => {
    const currentWord = words[currentWordIndex];
    
    const timeout = setTimeout(() => {
      if (!isDeleting) {
        if (displayedText.length < currentWord.length) {
          setDisplayedText(currentWord.substring(0, displayedText.length + 1));
        } else {
          setTimeout(() => setIsDeleting(true), 2500);
        }
      } else {
        if (displayedText.length > 0) {
          setDisplayedText(currentWord.substring(0, displayedText.length - 1));
        } else {
          setIsDeleting(false);
          setCurrentWordIndex((prev) => (prev + 1) % words.length);
        }
      }
    }, isDeleting ? 40 : 120);

    return () => clearTimeout(timeout);
  }, [displayedText, isDeleting, currentWordIndex]);

  return (
    <div className="min-h-screen bg-white">
      
      {/* Hero Section avec Search */}
        <section className="relative py-32 text-white transition-colors duration-500 dark:text-gray-100 overflow-hidden">
      {/* 🎥 Vidéo ou Image de fond */}
      <div className="absolute inset-0 z-0">
        <video
          className="w-full h-full object-cover hidden dark:block"
          autoPlay
          muted
          loop
          playsInline
        >
          <source src={Video} type="video/mp4" />
        </video>

        <video
          className="w-full h-full object-cover block dark:hidden"
          autoPlay
          muted
          loop
          playsInline
        >
          <source src={Video} type="video/mp4" />
        </video>

        {/* Overlay */}
        <div className="absolute inset-0 bg-gradient-to-br from-gray-900/90 via-gray-800/70 to-transparent dark:from-black/80 dark:via-gray-900/70 dark:to-transparent"></div>
      </div>

      {/* 🌟 Contenu principal */}
     <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          
          {/* Badge certifications */}
          <div className="flex rounded-full justify-center mb-8">
            <div className="inline-flex items-center gap-6 px-6 py-3 bg-white border border-gray-200 shadow-sm">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-gray-600" />
                <span className="text-xs font-medium text-gray-600">ISO 9001</span>
              </div>
              <div className="w-px h-4 bg-gray-300"></div>
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-gray-600" />
                <span className="text-xs font-medium text-gray-600">Livraison 24/48h</span>
              </div>
              <div className="w-px h-4 bg-gray-300"></div>
              <div className="flex items-center gap-2">
                <BarChart3 className="w-4 h-4 text-gray-600" />
                <span className="text-xs font-medium text-gray-600">+15 000 références</span>
              </div>
            </div>
          </div>

          {/* Titre principal avec animation */}
          <div className="max-w-4xl mx-auto text-center mb-16">
            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-light text-gray-100 mb-6 leading-tight tracking-tight">
              Optimisez vos achats de{' '}
              <br className="hidden sm:block" />
              <span className="relative inline-block">
                <span className="font-normal text-amber-600">
                  {displayedText}
                </span>
                <span className="inline-block w-0.5 h-12 sm:h-14 lg:h-16 bg-gray-900 ml-1 align-middle animate-cursor"></span>
              </span>
            </h1>

            <p className="text-lg sm:text-xl text-gray-100 max-w-3xl mx-auto font-light leading-relaxed mb-8">
              Plateforme B2B de référence pour sourcer, comparer et commander vos équipements professionnels. Tarifs négociés, service dédié, livraison garantie.
            </p>

            {/* Trust badges */}
            <div className="flex flex-wrap justify-center gap-6 text-sm text-gray-100">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span>Paiement sécurisé</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span>Factures personnalisées</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span>Support commercial dédié</span>
              </div>
            </div>
          </div>

          {/* Module de recherche B2B */}
          <div className="max-w-5xl rounded-24 mx-auto">
            <div className="bg-white rounded-2xl overflow-hidden shadow-xl border border-gray-200">
              
              {/* Tabs */}
              <div className="border-b border-gray-200">
                <div className="flex">
                  {['products', 'suppliers', 'rfq'].map((type) => (
                    <button
                      key={type}
                      onClick={() => setSearchType(type)}
                      className={`flex-1 px-6 py-4 text-sm font-medium transition-colors ${
                        searchType === type
                          ? 'text-amber-500 border-b-2 border-amber-400 bg-gray-50'
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

              {/* Catégories */}
              <div className="p-6 border-b border-gray-200 bg-gray-50">
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  {categories.map((cat) => {
                    const Icon = cat.icon;
                    return (
                      <button
                        key={cat.id}
                        onClick={() => setCategory(cat.id)}
                        className={`flex items-center justify-center gap-2 px-4 py-3 border text-sm font-medium transition-all ${
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

              {/* Champs de recherche */}
              <div className="p-6 sm:p-8">
                <div className="grid sm:grid-cols-12 gap-4">
                  {/* Recherche produit */}
                  <div className="sm:col-span-6">
                    <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                      Produit ou référence
                    </label>
                    <div className="relative">
                      <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                      <input
                        type="text"
                        placeholder="Ex: Imprimante laser, bureau professionnel..."
                        className="w-full pl-12 pr-4 py-3 border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none bg-white"
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
                      placeholder="10"
                      min="1"
                      className="w-full px-4 py-3 border border-gray-300 text-gray-900 placeholder-gray-400 focus:border-gray-900 focus:outline-none bg-white"
                    />
                  </div>

                  {/* Budget */}
                  <div className="sm:col-span-2">
                    <label className="block text-xs font-medium text-gray-700 mb-2 uppercase tracking-wide">
                      Budget max
                    </label>
                    <div className="relative">
                      <select className="w-full px-4 py-3 border border-gray-300 text-gray-900 focus:border-gray-900 focus:outline-none bg-white appearance-none cursor-pointer">
                        <option>1 000 €</option>
                        <option>5 000 €</option>
                        <option>10 000 €</option>
                        <option>25 000 €</option>
                        <option>50 000 €</option>
                        <option>100 000 €+</option>
                      </select>
                      <ChevronDown className="absolute right-4 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>

                  {/* Bouton */}
                  <div className="sm:col-span-2 flex items-end">
                    <button className="w-full bg-gray-900 text-white py-3 px-6 hover:bg-gray-800 transition-colors flex items-center justify-center gap-2 font-medium">
                      <Search className="w-5 h-5" />
                      <span>Rechercher</span>
                    </button>
                  </div>
                </div>

                {/* Filtres avancés */}
                <div className="mt-6 pt-6 border-t border-gray-200">
                  <div className="flex flex-wrap items-center gap-4">
                    <button className="text-sm text-gray-600 hover:text-gray-900 font-medium flex items-center gap-2">
                      <span>Filtres avancés</span>
                      <ChevronDown className="w-4 h-4" />
                    </button>
                    <div className="flex flex-wrap gap-2">
                      <button className="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 text-xs font-medium text-gray-700 transition-colors">
                        Stock immédiat
                      </button>
                      <button className="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 text-xs font-medium text-gray-700 transition-colors">
                        Livraison rapide
                      </button>
                      <button className="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 text-xs font-medium text-gray-700 transition-colors">
                        Remise volume
                      </button>
                      <button className="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 text-xs font-medium text-gray-700 transition-colors">
                        Made in France
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            
          </div>
        </div>
    </section>

      {/* Stats Section */}
      <section className="py-12 sm:py-16 bg-gray-50 border-y border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-6 sm:gap-8">
            {stats.map((stat, idx) => {
              const Icon = stat.icon;
              return (
                <div key={idx} className="text-center">
                  <div className="inline-flex items-center justify-center w-12 h-12 sm:w-16 sm:h-16 bg-blue-100 rounded-full mb-3 sm:mb-4">
                    <Icon className="w-6 h-6 sm:w-8 sm:h-8 text-blue-600" />
                  </div>
                  <div className="text-2xl sm:text-3xl lg:text-4xl font-bold text-gray-900 mb-1 sm:mb-2">{stat.value}</div>
                  <div className="text-xs sm:text-sm text-gray-600">{stat.label}</div>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Featured Properties */}
      <section className="py-12 sm:py-16 lg:py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-8 sm:mb-12 gap-4">
            <div>
              <h2 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-gray-900 mb-2">
                Biens à la une
              </h2>
              <p className="text-gray-600">Découvrez notre sélection de biens d'exception</p>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex gap-2">
                <button 
                  onClick={() => setViewMode('grid')}
                  className={`p-2 rounded-lg transition-colors ${viewMode === 'grid' ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-700'}`}
                >
                  <Grid className="w-5 h-5" />
                </button>
                <button 
                  onClick={() => setViewMode('list')}
                  className={`p-2 rounded-lg transition-colors ${viewMode === 'list' ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-700'}`}
                >
                  <List className="w-5 h-5" />
                </button>
              </div>
              <button className="hidden sm:flex items-center gap-2 px-6 py-2.5 border-2 border-gray-300 hover:border-blue-600 rounded-lg font-semibold text-gray-700 hover:text-blue-600 transition-all">
                Voir tout
                <ChevronRight className="w-5 h-5" />
              </button>
            </div>
          </div>

          {/* Properties Grid */}
          <div className={`grid gap-6 ${viewMode === 'grid' ? 'sm:grid-cols-2 lg:grid-cols-3' : 'grid-cols-1'}`}>
            {featuredProperties.map((property) => (
              <div key={property.id} className="bg-white rounded-xl shadow-lg overflow-hidden hover:shadow-2xl transition-shadow group">
                <div className="relative h-48 sm:h-56 overflow-hidden">
                  <img 
                    src={property.image} 
                    alt={property.title}
                    className="w-full h-full object-cover group-hover:scale-110 transition-transform duration-500"
                  />
                  <div className="absolute top-4 left-4 flex gap-2">
                    {property.featured && (
                      <span className="bg-yellow-400 text-gray-900 px-3 py-1 rounded-full text-xs font-bold">
                        ⭐ À la une
                      </span>
                    )}
                    {property.new && (
                      <span className="bg-green-500 text-white px-3 py-1 rounded-full text-xs font-bold">
                        Nouveau
                      </span>
                    )}
                  </div>
                  <button className="absolute top-4 right-4 bg-white/90 hover:bg-white p-2 rounded-full transition-colors">
                    <Heart className="w-5 h-5 text-gray-700" />
                  </button>
                </div>
                
                <div className="p-4 sm:p-6">
                  <div className="flex items-center justify-between mb-3">
                    <span className="text-2xl sm:text-3xl font-bold text-blue-600">{formatPrice(property.price)}</span>
                    <span className="text-sm text-gray-500">{property.type}</span>
                  </div>
                  
                  <h3 className="text-lg sm:text-xl font-bold text-gray-900 mb-2 line-clamp-2 group-hover:text-blue-600 transition-colors">
                    {property.title}
                  </h3>
                  
                  <div className="flex items-center gap-2 text-gray-600 mb-4">
                    <MapPin className="w-4 h-4" />
                    <span className="text-sm">{property.location}</span>
                  </div>
                  
                  <div className="flex items-center justify-between pt-4 border-t border-gray-200">
                    <div className="flex items-center gap-1 text-gray-700">
                      <Bed className="w-4 h-4" />
                      <span className="text-sm font-medium">{property.beds}</span>
                    </div>
                    <div className="flex items-center gap-1 text-gray-700">
                      <Home className="w-4 h-4" />
                      <span className="text-sm font-medium">{property.baths}</span>
                    </div>
                    <div className="flex items-center gap-1 text-gray-700">
                      <Building2 className="w-4 h-4" />
                      <span className="text-sm font-medium">{property.surface}m²</span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="text-center mt-8 sm:hidden">
            <button className="w-full px-6 py-3 border-2 border-gray-300 hover:border-blue-600 rounded-lg font-semibold text-gray-700 hover:text-blue-600 transition-all">
              Voir tous les biens
            </button>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-12 sm:py-16 lg:py-20 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12 sm:mb-16">
            <h2 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-gray-900 mb-4">
              Pourquoi choisir Filter Finder ?
            </h2>
            <p className="text-lg text-gray-600 max-w-2xl mx-auto">
              Une plateforme complète pour faciliter votre recherche immobilière
            </p>
          </div>

          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6 sm:gap-8">
            {features.map((feature, idx) => {
              const Icon = feature.icon;
              return (
                <div key={idx} className="bg-white p-6 sm:p-8 rounded-xl shadow-lg hover:shadow-2xl transition-shadow">
                  <div className="inline-flex items-center justify-center w-14 h-14 bg-blue-100 rounded-xl mb-4">
                    <Icon className="w-7 h-7 text-blue-600" />
                  </div>
                  <h3 className="text-lg sm:text-xl font-bold text-gray-900 mb-3">{feature.title}</h3>
                  <p className="text-gray-600 leading-relaxed">{feature.description}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Testimonials */}
      <section className="py-12 sm:py-16 lg:py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12 sm:mb-16">
            <h2 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-gray-900 mb-4">
              Ce que disent nos clients
            </h2>
            <p className="text-lg text-gray-600">Des milliers de personnes nous font confiance</p>
          </div>

          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
            {testimonials.map((testimonial, idx) => (
              <div key={idx} className="bg-white p-6 sm:p-8 rounded-xl shadow-lg">
                <div className="flex items-center gap-1 mb-4">
                  {[...Array(testimonial.rating)].map((_, i) => (
                    <Star key={i} className="w-5 h-5 fill-yellow-400 text-yellow-400" />
                  ))}
                </div>
                <p className="text-gray-700 mb-6 leading-relaxed italic">"{testimonial.text}"</p>
                <div className="flex items-center gap-4">
                  <img 
                    src={testimonial.photo} 
                    alt={testimonial.name}
                    className="w-12 h-12 rounded-full object-cover"
                  />
                  <div>
                    <div className="font-bold text-gray-900">{testimonial.name}</div>
                    <div className="text-sm text-gray-600">{testimonial.role}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-16 sm:py-20 lg:py-24 bg-gradient-to-br from-blue-600 to-blue-800 text-white">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-6">
            Prêt à trouver votre bien idéal ?
          </h2>
          <p className="text-lg sm:text-xl text-blue-100 mb-8 sm:mb-10">
            Inscrivez-vous gratuitement et accédez à des milliers d'annonces exclusives
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <button className="px-8 py-4 bg-white text-blue-600 font-bold rounded-lg hover:bg-blue-50 transition-colors flex items-center justify-center gap-2 shadow-xl">
              Commencer maintenant
              <ArrowRight className="w-5 h-5" />
            </button>
            <button className="px-8 py-4 border-2 border-white text-white font-bold rounded-lg hover:bg-white/10 transition-colors flex items-center justify-center gap-2">
              <Phone className="w-5 h-5" />
              Nous contacter
            </button>
          </div>
        </div>
      </section>

     
    </div>
  );
}