import { useState } from "react";
import { 
  Building, User, Mail, Phone, MapPin, Calendar, Plus, Trash2, 
  Save, Send, Download, FileText, Calculator, CheckCircle, 
  AlertCircle, X, Eye, Edit2, Percent, Euro
} from "lucide-react";

const QuoteComponent = ({ closePopup }) => {
  const [showPreview, setShowPreview] = useState(false);
  const [currentStep, setCurrentStep] = useState(1);
  const [isSaving, setIsSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  const [quoteData, setQuoteData] = useState({
    clientType: "entreprise",
    clientName: "",
    clientCompany: "",
    clientEmail: "",
    clientPhone: "",
    clientAddress: "",
    clientCity: "",
    clientPostal: "",
    quoteNumber: `DEV-${Date.now().toString().slice(-6)}`,
    quoteDate: new Date().toISOString().split('T')[0],
    validityDate: new Date(Date.now() + 30*24*60*60*1000).toISOString().split('T')[0],
    projectType: "",
    items: [
      { id: 1, description: "", quantity: 1, unitPrice: 0, taxRate: 20 }
    ],
    discount: 0,
    discountType: "percent",
    notes: "",
    termsAndConditions: "Conditions générales :\n- Devis valable 30 jours\n- Paiement à 30 jours fin de mois\n- TVA non applicable, article 293 B du CGI",
  });

  const [errors, setErrors] = useState({});

  const projectTypes = [
    "Site Web Vitrine",
    "E-commerce",
    "Application Mobile",
    "Développement sur mesure",
    "Consulting IT",
    "Formation",
    "Maintenance",
    "Autre"
  ];

  const calculateItemTotal = (item) => item.quantity * item.unitPrice;
  const calculateItemTax = (item) => (calculateItemTotal(item) * item.taxRate) / 100;
  const calculateSubtotal = () => quoteData.items.reduce((sum, item) => sum + calculateItemTotal(item), 0);
  
  const calculateDiscount = () => {
    const subtotal = calculateSubtotal();
    return quoteData.discountType === "percent" 
      ? (subtotal * quoteData.discount) / 100 
      : quoteData.discount;
  };
  
  const calculateTotalTax = () => quoteData.items.reduce((sum, item) => sum + calculateItemTax(item), 0);
  const calculateTotal = () => calculateSubtotal() - calculateDiscount() + calculateTotalTax();

  const handleInputChange = (field, value) => {
    setQuoteData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  };

  const handleItemChange = (id, field, value) => {
    setQuoteData(prev => ({
      ...prev,
      items: prev.items.map(item => 
        item.id === id ? { ...item, [field]: value } : item
      )
    }));
  };

  const addItem = () => {
    const newId = Math.max(...quoteData.items.map(i => i.id), 0) + 1;
    setQuoteData(prev => ({
      ...prev,
      items: [...prev.items, { 
        id: newId, 
        description: "", 
        quantity: 1, 
        unitPrice: 0, 
        taxRate: 20 
      }]
    }));
  };

  const removeItem = (id) => {
    if (quoteData.items.length > 1) {
      setQuoteData(prev => ({
        ...prev,
        items: prev.items.filter(item => item.id !== id)
      }));
    }
  };

  const validateStep = (step) => {
    const newErrors = {};
    
    if (step === 1) {
      if (!quoteData.clientName.trim()) newErrors.clientName = "Le nom est requis";
      if (quoteData.clientType === "entreprise" && !quoteData.clientCompany.trim()) {
        newErrors.clientCompany = "Le nom de l'entreprise est requis";
      }
      if (!quoteData.clientEmail.trim()) {
        newErrors.clientEmail = "L'email est requis";
      } else if (!/\S+@\S+\.\S+/.test(quoteData.clientEmail)) {
        newErrors.clientEmail = "Email invalide";
      }
      if (!quoteData.clientPhone.trim()) newErrors.clientPhone = "Le téléphone est requis";
    }
    
    if (step === 2) {
      if (!quoteData.projectType) newErrors.projectType = "Le type de projet est requis";
      
      const hasValidItems = quoteData.items.some(item => 
        item.description.trim() && item.quantity > 0 && item.unitPrice > 0
      );
      
      if (!hasValidItems) newErrors.items = "Au moins une ligne valide est requise";
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const nextStep = () => {
    if (validateStep(currentStep)) {
      setCurrentStep(prev => Math.min(prev + 1, 3));
    }
  };

  const prevStep = () => setCurrentStep(prev => Math.max(prev - 1, 1));

  const handleSave = () => {
    setIsSaving(true);
    setTimeout(() => {
      setIsSaving(false);
      setSaveSuccess(true);
      setTimeout(() => setSaveSuccess(false), 3000);
    }, 1500);
  };

  const handleDownload = () => {
    alert("Téléchargement du devis PDF en cours...");
  };

  const handleSend = () => {
    if (validateStep(1) && validateStep(2)) {
      alert(`Devis envoyé à ${quoteData.clientEmail}`);
    }
  };

  const formatCurrency = (value) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'XOF',
      minimumFractionDigits: 0,
    }).format(value);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="w-full max-w-7xl bg-white rounded-2xl shadow-2xl max-h-[95vh] overflow-y-auto my-8">
        {/* Header avec bouton fermer */}
        <div className="sticky top-0 z-10 bg-gradient-to-r from-slate-900 to-slate-800 text-white px-6 py-4 flex items-center justify-between rounded-t-2xl shadow-lg">
          <div className="flex items-center gap-3">
            <div className="bg-amber-500 p-2 rounded-lg">
              <FileText className="w-6 h-6" />
            </div>
            <div>
              <h1 className="text-xl font-bold">Créer un Devis</h1>
              <p className="text-sm text-slate-300">Gestion professionnelle de devis</p>
            </div>
          </div>
          <button
            onClick={closePopup}
            className="w-10 h-10 flex items-center justify-center rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
            title="Fermer"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        <div className="p-6">
          {/* Progress Steps */}
          <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6 mb-6">
            <div className="flex items-center justify-between">
              {[
                { num: 1, label: "Client" },
                { num: 2, label: "Détails" },
                { num: 3, label: "Résumé" }
              ].map((step, index) => (
                <div key={step.num} className="flex items-center flex-1">
                  <div className="flex flex-col items-center flex-1">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold transition-all ${
                      currentStep >= step.num 
                        ? "bg-amber-500 text-white shadow-lg" 
                        : "bg-slate-200 text-slate-600"
                    }`}>
                      {currentStep > step.num ? <CheckCircle className="w-6 h-6" /> : step.num}
                    </div>
                    <span className={`text-sm mt-2 font-medium ${
                      currentStep >= step.num ? "text-amber-500" : "text-slate-400"
                    }`}>
                      {step.label}
                    </span>
                  </div>
                  {index < 2 && (
                    <div className={`h-1 flex-1 mx-2 rounded transition-all ${
                      currentStep > step.num ? "bg-amber-500" : "bg-slate-200"
                    }`}></div>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Step 1: Client Information */}
          {currentStep === 1 && (
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-8">
              <h2 className="text-2xl font-bold text-slate-900 mb-6 flex items-center gap-2">
                <User className="w-6 h-6 text-amber-500" />
                Informations Client
              </h2>
              
              <div className="mb-6">
                <label className="block text-sm font-semibold text-slate-700 mb-3">
                  Type de client
                </label>
                <div className="flex gap-4">
                  <label className="flex items-center gap-2 cursor-pointer bg-slate-50 px-4 py-3 rounded-lg hover:bg-slate-100 transition-colors">
                    <input
                      type="radio"
                      name="clientType"
                      value="entreprise"
                      checked={quoteData.clientType === "entreprise"}
                      onChange={(e) => handleInputChange("clientType", e.target.value)}
                      className="w-4 h-4 text-amber-500"
                    />
                    <Building className="w-5 h-5 text-slate-600" />
                    <span className="text-slate-700 font-medium">Entreprise</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer bg-slate-50 px-4 py-3 rounded-lg hover:bg-slate-100 transition-colors">
                    <input
                      type="radio"
                      name="clientType"
                      value="particulier"
                      checked={quoteData.clientType === "particulier"}
                      onChange={(e) => handleInputChange("clientType", e.target.value)}
                      className="w-4 h-4 text-amber-500"
                    />
                    <User className="w-5 h-5 text-slate-600" />
                    <span className="text-slate-700 font-medium">Particulier</span>
                  </label>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {quoteData.clientType === "entreprise" && (
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">
                      Nom de l'entreprise <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <Building className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                      <input
                        type="text"
                        value={quoteData.clientCompany}
                        onChange={(e) => handleInputChange("clientCompany", e.target.value)}
                        placeholder="ACME Corporation"
                        className={`w-full pl-11 pr-4 py-3 border ${
                          errors.clientCompany ? "border-red-500 bg-red-50" : "border-slate-300"
                        } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500`}
                      />
                    </div>
                    {errors.clientCompany && (
                      <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                        <AlertCircle className="w-4 h-4" /> {errors.clientCompany}
                      </p>
                    )}
                  </div>
                )}

                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Nom du contact <span className="text-red-500">*</span>
                  </label>
                  <div className="relative">
                    <User className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                    <input
                      type="text"
                      value={quoteData.clientName}
                      onChange={(e) => handleInputChange("clientName", e.target.value)}
                      placeholder="Jean Dupont"
                      className={`w-full pl-11 pr-4 py-3 border ${
                        errors.clientName ? "border-red-500 bg-red-50" : "border-slate-300"
                      } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500`}
                    />
                  </div>
                  {errors.clientName && (
                    <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                      <AlertCircle className="w-4 h-4" /> {errors.clientName}
                    </p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Email <span className="text-red-500">*</span>
                  </label>
                  <div className="relative">
                    <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                    <input
                      type="email"
                      value={quoteData.clientEmail}
                      onChange={(e) => handleInputChange("clientEmail", e.target.value)}
                      placeholder="contact@example.com"
                      className={`w-full pl-11 pr-4 py-3 border ${
                        errors.clientEmail ? "border-red-500 bg-red-50" : "border-slate-300"
                      } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500`}
                    />
                  </div>
                  {errors.clientEmail && (
                    <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                      <AlertCircle className="w-4 h-4" /> {errors.clientEmail}
                    </p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Téléphone <span className="text-red-500">*</span>
                  </label>
                  <div className="relative">
                    <Phone className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                    <input
                      type="tel"
                      value={quoteData.clientPhone}
                      onChange={(e) => handleInputChange("clientPhone", e.target.value)}
                      placeholder="+33 6 12 34 56 78"
                      className={`w-full pl-11 pr-4 py-3 border ${
                        errors.clientPhone ? "border-red-500 bg-red-50" : "border-slate-300"
                      } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500`}
                    />
                  </div>
                  {errors.clientPhone && (
                    <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                      <AlertCircle className="w-4 h-4" /> {errors.clientPhone}
                    </p>
                  )}
                </div>

                <div className="md:col-span-2">
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Adresse
                  </label>
                  <div className="relative">
                    <MapPin className="absolute left-3 top-3 text-slate-400 w-5 h-5" />
                    <input
                      type="text"
                      value={quoteData.clientAddress}
                      onChange={(e) => handleInputChange("clientAddress", e.target.value)}
                      placeholder="123 Rue de la République"
                      className="w-full pl-11 pr-4 py-3 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Ville
                  </label>
                  <input
                    type="text"
                    value={quoteData.clientCity}
                    onChange={(e) => handleInputChange("clientCity", e.target.value)}
                    placeholder="Paris"
                    className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Code postal
                  </label>
                  <input
                    type="text"
                    value={quoteData.clientPostal}
                    onChange={(e) => handleInputChange("clientPostal", e.target.value)}
                    placeholder="75001"
                    className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-4 mt-8 pt-6 border-t border-slate-200">
                <button
                  onClick={nextStep}
                  className="px-6 py-3 bg-amber-500 hover:bg-amber-600 text-white font-semibold rounded-lg transition-colors shadow-md hover:shadow-lg"
                >
                  Suivant →
                </button>
              </div>
            </div>
          )}

          {/* Step 2: Quote Details */}
          {currentStep === 2 && (
            <div className="space-y-6">
              <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
                <h2 className="text-2xl font-bold text-slate-900 mb-6 flex items-center gap-2">
                  <Calculator className="w-6 h-6 text-amber-500" />
                  Informations du Devis
                </h2>

                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">N° Devis</label>
                    <input
                      type="text"
                      value={quoteData.quoteNumber}
                      readOnly
                      className="w-full px-3 py-2 border border-slate-300 rounded-lg bg-slate-50 font-mono text-sm"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">Date du devis</label>
                    <input
                      type="date"
                      value={quoteData.quoteDate}
                      onChange={(e) => handleInputChange("quoteDate", e.target.value)}
                      className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">Valable jusqu'au</label>
                    <input
                      type="date"
                      value={quoteData.validityDate}
                      onChange={(e) => handleInputChange("validityDate", e.target.value)}
                      className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">
                      Type de projet <span className="text-red-500">*</span>
                    </label>
                    <select
                      value={quoteData.projectType}
                      onChange={(e) => handleInputChange("projectType", e.target.value)}
                      className={`w-full px-3 py-2 border ${
                        errors.projectType ? "border-red-500 bg-red-50" : "border-slate-300"
                      } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500`}
                    >
                      <option value="">Sélectionner</option>
                      {projectTypes.map((type) => (
                        <option key={type} value={type}>{type}</option>
                      ))}
                    </select>
                    {errors.projectType && (
                      <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                        <AlertCircle className="w-4 h-4" /> {errors.projectType}
                      </p>
                    )}
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
                <div className="flex items-center justify-between mb-6">
                  <h2 className="text-xl font-bold text-slate-900">Lignes du Devis</h2>
                  <button
                    onClick={addItem}
                    className="flex items-center gap-2 px-4 py-2 bg-amber-500 hover:bg-amber-600 text-white rounded-lg transition-colors shadow-md"
                  >
                    <Plus className="w-5 h-5" />
                    Ajouter
                  </button>
                </div>

                {errors.items && (
                  <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-600">
                    <AlertCircle className="w-5 h-5" />
                    <span>{errors.items}</span>
                  </div>
                )}

                <div className="space-y-4">
                  {quoteData.items.map((item, index) => (
                    <div key={item.id} className="border border-slate-200 rounded-lg p-4 bg-slate-50">
                      <div className="flex items-start justify-between mb-4">
                        <span className="text-sm font-semibold text-slate-700">Ligne {index + 1}</span>
                        {quoteData.items.length > 1 && (
                          <button
                            onClick={() => removeItem(item.id)}
                            className="text-red-600 hover:text-red-700 transition-colors"
                          >
                            <Trash2 className="w-5 h-5" />
                          </button>
                        )}
                      </div>
                      
                      <div className="grid grid-cols-1 md:grid-cols-12 gap-4">
                        <div className="md:col-span-5">
                          <label className="block text-sm font-medium text-slate-700 mb-2">
                            Description
                          </label>
                          <textarea
                            value={item.description}
                            onChange={(e) => handleItemChange(item.id, "description", e.target.value)}
                            placeholder="Décrivez le produit..."
                            rows={2}
                            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 resize-none"
                          />
                        </div>

                        <div className="md:col-span-2">
                          <label className="block text-sm font-medium text-slate-700 mb-2">
                            Quantité
                          </label>
                          <input
                            type="number"
                            min="1"
                            value={item.quantity}
                            onChange={(e) => handleItemChange(item.id, "quantity", parseFloat(e.target.value) || 0)}
                            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                          />
                        </div>

                        <div className="md:col-span-2">
                          <label className="block text-sm font-medium text-slate-700 mb-2">
                            Prix unitaire (FCFA)
                          </label>
                          <input
                            type="number"
                            min="0"
                            step="0.01"
                            value={item.unitPrice}
                            onChange={(e) => handleItemChange(item.id, "unitPrice", parseFloat(e.target.value) || 0)}
                            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                          />
                        </div>

                        <div className="md:col-span-2">
                          <label className="block text-sm font-medium text-slate-700 mb-2">
                            TVA (%)
                          </label>
                          <select
                            value={item.taxRate}
                            onChange={(e) => handleItemChange(item.id, "taxRate", parseFloat(e.target.value))}
                            className="w-full px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                          >
                            <option value="0">0%</option>
                            <option value="5.5">5.5%</option>
                            <option value="10">10%</option>
                            <option value="20">20%</option>
                          </select>
                        </div>

                        <div className="md:col-span-1">
                          <label className="block text-sm font-medium text-slate-700 mb-2">
                            Total HT
                          </label>
                          <div className="px-2 py-2 bg-slate-100 border border-slate-300 rounded-lg font-semibold text-slate-900 text-sm">
                            {formatCurrency(calculateItemTotal(item))}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 mb-4">Remise</h3>
                    <div className="space-y-4">
                      <div className="flex gap-4">
                        <label className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="radio"
                            name="discountType"
                            value="percent"
                            checked={quoteData.discountType === "percent"}
                            onChange={(e) => handleInputChange("discountType", e.target.value)}
                            className="w-4 h-4 text-amber-500"
                          />
                          <Percent className="w-5 h-5 text-slate-600" />
                          <span className="text-slate-700">Pourcentage</span>
                        </label>
                        <label className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="radio"
                            name="discountType"
                            value="fixed"
                            checked={quoteData.discountType === "fixed"}
                            onChange={(e) => handleInputChange("discountType", e.target.value)}
                            className="w-4 h-4 text-amber-500"
                          />
                          <Euro className="w-5 h-5 text-slate-600" />
                          <span className="text-slate-700">Montant fixe</span>
                        </label>
                      </div>
                      <div className="relative">
                        <input
                          type="number"
                          min="0"
                          step="0.01"
                          value={quoteData.discount}
                          onChange={(e) => handleInputChange("discount", parseFloat(e.target.value) || 0)}
                          placeholder="0"
                          className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
                        />
                        <span className="absolute right-4 top-1/2 transform -translate-y-1/2 text-slate-500">
                          {quoteData.discountType === "percent" ? "%" : "FCFA"}
                        </span>
                      </div>
                      <div className="p-4 bg-amber-50 border border-amber-200 rounded-lg">
                        <div className="flex justify-between items-center">
                          <span className="text-slate-700">Montant de la remise :</span>
                          <span className="font-bold text-amber-600">{formatCurrency(calculateDiscount())}</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h3 className="text-lg font-bold text-slate-900 mb-4">Totaux</h3>
                    <div className="space-y-3">
                      <div className="flex justify-between items-center py-2">
                        <span className="text-slate-700">Sous-total HT :</span>
                        <span className="font-semibold text-slate-900">{formatCurrency(calculateSubtotal())}</span>
                      </div>
                      {quoteData.discount > 0 && (
                        <div className="flex justify-between items-center py-2 text-emerald-600">
                          <span>Remise :</span>
                          <span className="font-semibold">- {formatCurrency(calculateDiscount())}</span>
                        </div>
                      )}
                      <div className="flex justify-between items-center py-2 border-t border-slate-200">
                        <span className="text-slate-700">Total HT :</span>
                        <span className="font-semibold text-slate-900">
                          {formatCurrency(calculateSubtotal() - calculateDiscount())}
                        </span>
                      </div>
                      <div className="flex justify-between items-center py-2">
                        <span className="text-slate-700">TVA :</span>
                        <span className="font-semibold text-slate-900">{formatCurrency(calculateTotalTax())}</span>
                      </div>
                      <div className="flex justify-between items-center py-3 border-t-2 border-slate-300">
                        <span className="text-lg font-bold text-slate-900">Total TTC :</span>
                        <span className="text-2xl font-bold text-amber-600">{formatCurrency(calculateTotal())}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="flex justify-between">
                <button
                  onClick={prevStep}
                  className="px-6 py-3 bg-slate-200 hover:bg-slate-300 text-slate-700 font-semibold rounded-lg transition-colors"
                >
                  ← Précédent
                </button>
                <button
                  onClick={nextStep}
                  className="px-6 py-3 bg-amber-500 hover:bg-amber-600 text-white font-semibold rounded-lg transition-colors shadow-md hover:shadow-lg"
                >
                  Suivant →
                </button>
              </div>
            </div>
          )}

          {/* Step 3: Summary */}
          {currentStep === 3 && (
            <div className="space-y-6">
              <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-8">
                <h2 className="text-2xl font-bold text-slate-900 mb-6 flex items-center gap-2">
                  <FileText className="w-6 h-6 text-amber-500" />
                  Notes et Conditions
                </h2>
                
                <div className="space-y-6">
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">
                      Notes personnalisées
                    </label>
                    <textarea
                      value={quoteData.notes}
                      onChange={(e) => handleInputChange("notes", e.target.value)}
                      placeholder="Ajoutez des notes spécifiques pour ce devis..."
                      rows={4}
                      className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 resize-none"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">
                      Conditions générales
                    </label>
                    <textarea
                      value={quoteData.termsAndConditions}
                      onChange={(e) => handleInputChange("termsAndConditions", e.target.value)}
                      rows={6}
                      className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 resize-none"
                    />
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-8">
                <h2 className="text-2xl font-bold text-slate-900 mb-6">Récapitulatif</h2>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mb-8">
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 mb-4">Informations client</h3>
                    <div className="space-y-2 text-sm">
                      {quoteData.clientType === "entreprise" && quoteData.clientCompany && (
                        <div className="flex items-start gap-2">
                          <Building className="w-4 h-4 text-slate-600 mt-0.5" />
                          <span className="text-slate-700">{quoteData.clientCompany}</span>
                        </div>
                      )}
                      <div className="flex items-start gap-2">
                        <User className="w-4 h-4 text-slate-600 mt-0.5" />
                        <span className="text-slate-700">{quoteData.clientName}</span>
                      </div>
                      <div className="flex items-start gap-2">
                        <Mail className="w-4 h-4 text-slate-600 mt-0.5" />
                        <span className="text-slate-700">{quoteData.clientEmail}</span>
                      </div>
                      <div className="flex items-start gap-2">
                        <Phone className="w-4 h-4 text-slate-600 mt-0.5" />
                        <span className="text-slate-700">{quoteData.clientPhone}</span>
                      </div>
                      {quoteData.clientAddress && (
                        <div className="flex items-start gap-2">
                          <MapPin className="w-4 h-4 text-slate-600 mt-0.5" />
                          <span className="text-slate-700">
                            {quoteData.clientAddress}
                            {quoteData.clientCity && `, ${quoteData.clientCity}`}
                            {quoteData.clientPostal && ` ${quoteData.clientPostal}`}
                          </span>
                        </div>
                      )}
                    </div>
                  </div>

                  <div>
                    <h3 className="text-lg font-bold text-slate-900 mb-4">Détails du devis</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-slate-600">N° Devis :</span>
                        <span className="font-mono font-semibold text-slate-900">{quoteData.quoteNumber}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-slate-600">Date :</span>
                        <span className="text-slate-900">{new Date(quoteData.quoteDate).toLocaleDateString('fr-FR')}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-slate-600">Valable jusqu'au :</span>
                        <span className="text-slate-900">{new Date(quoteData.validityDate).toLocaleDateString('fr-FR')}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-slate-600">Type de projet :</span>
                        <span className="text-slate-900">{quoteData.projectType}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <div>
                  <h3 className="text-lg font-bold text-slate-900 mb-4">Lignes du devis</h3>
                  <div className="border border-slate-200 rounded-lg overflow-hidden">
                    <table className="w-full">
                      <thead className="bg-slate-50">
                        <tr>
                          <th className="px-4 py-3 text-left text-sm font-semibold text-slate-700">Description</th>
                          <th className="px-4 py-3 text-center text-sm font-semibold text-slate-700">Qté</th>
                          <th className="px-4 py-3 text-right text-sm font-semibold text-slate-700">P.U. HT</th>
                          <th className="px-4 py-3 text-center text-sm font-semibold text-slate-700">TVA</th>
                          <th className="px-4 py-3 text-right text-sm font-semibold text-slate-700">Total HT</th>
                        </tr>
                      </thead>
                      <tbody>
                        {quoteData.items.map((item, index) => (
                          <tr key={item.id} className={index % 2 === 0 ? "bg-white" : "bg-slate-50"}>
                            <td className="px-4 py-3 text-sm text-slate-700">{item.description || "-"}</td>
                            <td className="px-4 py-3 text-sm text-center text-slate-700">{item.quantity}</td>
                            <td className="px-4 py-3 text-sm text-right text-slate-700">{formatCurrency(item.unitPrice)}</td>
                            <td className="px-4 py-3 text-sm text-center text-slate-700">{item.taxRate}%</td>
                            <td className="px-4 py-3 text-sm text-right font-semibold text-slate-900">
                              {formatCurrency(calculateItemTotal(item))}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>

                <div className="mt-8 flex justify-end">
                  <div className="w-full md:w-1/2 space-y-2">
                    <div className="flex justify-between py-2">
                      <span className="text-slate-700">Sous-total HT :</span>
                      <span className="font-semibold text-slate-900">{formatCurrency(calculateSubtotal())}</span>
                    </div>
                    {quoteData.discount > 0 && (
                      <div className="flex justify-between py-2 text-emerald-600">
                        <span>Remise ({quoteData.discountType === "percent" ? `${quoteData.discount}%` : "fixe"}) :</span>
                        <span className="font-semibold">- {formatCurrency(calculateDiscount())}</span>
                      </div>
                    )}
                    <div className="flex justify-between py-2 border-t border-slate-200">
                      <span className="text-slate-700">Total HT :</span>
                      <span className="font-semibold text-slate-900">
                        {formatCurrency(calculateSubtotal() - calculateDiscount())}
                      </span>
                    </div>
                    <div className="flex justify-between py-2">
                      <span className="text-slate-700">TVA :</span>
                      <span className="font-semibold text-slate-900">{formatCurrency(calculateTotalTax())}</span>
                    </div>
                    <div className="flex justify-between py-3 border-t-2 border-slate-300">
                      <span className="text-xl font-bold text-slate-900">Total TTC :</span>
                      <span className="text-2xl font-bold text-amber-600">{formatCurrency(calculateTotal())}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-8">
                <h3 className="text-lg font-bold text-slate-900 mb-6">Actions</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <button
                    onClick={handleSave}
                    disabled={isSaving}
                    className="flex items-center justify-center gap-2 px-6 py-4 bg-slate-600 hover:bg-slate-700 text-white font-semibold rounded-lg transition-colors disabled:opacity-50 shadow-md hover:shadow-lg"
                  >
                    <Save className="w-5 h-5" />
                    Sauvegarder
                  </button>
                  <button
                    onClick={() => setShowPreview(true)}
                    className="flex items-center justify-center gap-2 px-6 py-4 bg-purple-600 hover:bg-purple-700 text-white font-semibold rounded-lg transition-colors shadow-md hover:shadow-lg"
                  >
                    <Eye className="w-5 h-5" />
                    Aperçu
                  </button>
                  <button
                    onClick={handleDownload}
                    className="flex items-center justify-center gap-2 px-6 py-4 bg-blue-600 hover:bg-blue-700 text-white font-semibold rounded-lg transition-colors shadow-md hover:shadow-lg"
                  >
                    <Download className="w-5 h-5" />
                    Télécharger
                  </button>
                  <button
                    onClick={handleSend}
                    className="flex items-center justify-center gap-2 px-6 py-4 bg-emerald-600 hover:bg-emerald-700 text-white font-semibold rounded-lg transition-colors shadow-md hover:shadow-lg"
                  >
                    <Send className="w-5 h-5" />
                    Envoyer
                  </button>
                </div>
                
                {saveSuccess && (
                  <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-lg flex items-center gap-2 text-green-700">
                    <CheckCircle className="w-5 h-5" />
                    <span>Devis sauvegardé avec succès !</span>
                  </div>
                )}
              </div>

              <div className="flex justify-between">
                <button
                  onClick={prevStep}
                  className="px-6 py-3 bg-slate-200 hover:bg-slate-300 text-slate-700 font-semibold rounded-lg transition-colors"
                >
                  ← Précédent
                </button>
                <button
                  onClick={closePopup}
                  className="px-6 py-3 bg-red-600 hover:bg-red-700 text-white font-semibold rounded-lg transition-colors shadow-md hover:shadow-lg"
                >
                  Terminer
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Preview Modal */}
      {showPreview && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-auto">
            <div className="sticky top-0 bg-gradient-to-r from-slate-900 to-slate-800 text-white px-8 py-6 flex items-center justify-between rounded-t-2xl z-10">
              <h2 className="text-2xl font-bold">Aperçu du Devis</h2>
              <button
                onClick={() => setShowPreview(false)}
                className="w-10 h-10 flex items-center justify-center rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
              >
                <X className="w-6 h-6" />
              </button>
            </div>

            <div className="p-8 md:p-12">
              <div className="flex justify-between items-start mb-12">
                <div>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="w-12 h-12 bg-amber-500 rounded-lg flex items-center justify-center">
                      <Building className="w-7 h-7 text-white" />
                    </div>
                    <div>
                      <h1 className="text-2xl font-bold text-slate-900">Votre Entreprise</h1>
                      <p className="text-slate-600">Plateforme Professionnelle</p>
                    </div>
                  </div>
                  <div className="text-sm text-slate-600 space-y-1">
                    <p>123 Avenue des Champs-Élysées</p>
                    <p>75008 Paris, France</p>
                    <p>contact@entreprise.com</p>
                    <p>+33 1 23 45 67 89</p>
                  </div>
                </div>
                <div className="text-right">
                  <h2 className="text-3xl font-bold text-slate-900 mb-4">DEVIS</h2>
                  <div className="text-sm space-y-1">
                    <p className="text-slate-600">N° <span className="font-mono font-semibold text-slate-900">{quoteData.quoteNumber}</span></p>
                    <p className="text-slate-600">Date : {new Date(quoteData.quoteDate).toLocaleDateString('fr-FR')}</p>
                    <p className="text-slate-600">Valable jusqu'au : {new Date(quoteData.validityDate).toLocaleDateString('fr-FR')}</p>
                  </div>
                </div>
              </div>

              <div className="mb-8 p-6 bg-slate-50 rounded-lg">
                <h3 className="text-sm font-bold text-slate-700 mb-3">CLIENT</h3>
                <div className="space-y-1 text-sm">
                  {quoteData.clientCompany && <p className="font-semibold text-slate-900">{quoteData.clientCompany}</p>}
                  <p className="text-slate-700">{quoteData.clientName}</p>
                  {quoteData.clientAddress && <p className="text-slate-700">{quoteData.clientAddress}</p>}
                  {(quoteData.clientCity || quoteData.clientPostal) && (
                    <p className="text-slate-700">{quoteData.clientPostal} {quoteData.clientCity}</p>
                  )}
                  <p className="text-slate-700">{quoteData.clientEmail}</p>
                  <p className="text-slate-700">{quoteData.clientPhone}</p>
                </div>
              </div>

              {quoteData.projectType && (
                <div className="mb-8">
                  <p className="text-sm text-slate-600">
                    Type de projet : <span className="font-semibold text-slate-900">{quoteData.projectType}</span>
                  </p>
                </div>
              )}

              <div className="mb-8 border border-slate-200 rounded-lg overflow-hidden">
                <table className="w-full">
                  <thead className="bg-slate-900 text-white">
                    <tr>
                      <th className="px-4 py-3 text-left text-sm font-semibold">Description</th>
                      <th className="px-4 py-3 text-center text-sm font-semibold">Qté</th>
                      <th className="px-4 py-3 text-right text-sm font-semibold">P.U. HT</th>
                      <th className="px-4 py-3 text-center text-sm font-semibold">TVA</th>
                      <th className="px-4 py-3 text-right text-sm font-semibold">Total HT</th>
                    </tr>
                  </thead>
                  <tbody>
                    {quoteData.items.map((item, index) => (
                      <tr key={item.id} className={index % 2 === 0 ? "bg-white" : "bg-slate-50"}>
                        <td className="px-4 py-3 text-sm text-slate-700">{item.description || "-"}</td>
                        <td className="px-4 py-3 text-sm text-center text-slate-700">{item.quantity}</td>
                        <td className="px-4 py-3 text-sm text-right text-slate-700">{formatCurrency(item.unitPrice)}</td>
                        <td className="px-4 py-3 text-sm text-center text-slate-700">{item.taxRate}%</td>
                        <td className="px-4 py-3 text-sm text-right font-semibold text-slate-900">
                          {formatCurrency(calculateItemTotal(item))}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="flex justify-end mb-8">
                <div className="w-full md:w-1/2 space-y-2">
                  <div className="flex justify-between py-2 text-sm">
                    <span className="text-slate-600">Sous-total HT :</span>
                    <span className="font-semibold text-slate-900">{formatCurrency(calculateSubtotal())}</span>
                  </div>
                  {quoteData.discount > 0 && (
                    <div className="flex justify-between py-2 text-sm text-emerald-600">
                      <span>Remise :</span>
                      <span className="font-semibold">- {formatCurrency(calculateDiscount())}</span>
                    </div>
                  )}
                  <div className="flex justify-between py-2 text-sm border-t border-slate-200">
                    <span className="text-slate-600">Total HT :</span>
                    <span className="font-semibold text-slate-900">
                      {formatCurrency(calculateSubtotal() - calculateDiscount())}
                    </span>
                  </div>
                  <div className="flex justify-between py-2 text-sm">
                    <span className="text-slate-600">TVA :</span>
                    <span className="font-semibold text-slate-900">{formatCurrency(calculateTotalTax())}</span>
                  </div>
                  <div className="flex justify-between py-3 border-t-2 border-slate-900">
                    <span className="text-lg font-bold text-slate-900">Total TTC :</span>
                    <span className="text-2xl font-bold text-amber-600">{formatCurrency(calculateTotal())}</span>
                  </div>
                </div>
              </div>

              {quoteData.notes && (
                <div className="mb-8">
                  <h3 className="text-sm font-bold text-slate-900 mb-2">NOTES</h3>
                  <p className="text-sm text-slate-700 whitespace-pre-wrap">{quoteData.notes}</p>
                </div>
              )}

              <div className="pt-8 border-t border-slate-200">
                <h3 className="text-sm font-bold text-slate-900 mb-2">CONDITIONS GÉNÉRALES</h3>
                <p className="text-xs text-slate-600 whitespace-pre-wrap">{quoteData.termsAndConditions}</p>
              </div>
            </div>

            <div className="sticky bottom-0 bg-white border-t border-slate-200 px-8 py-6 flex justify-end gap-4 rounded-b-2xl">
              <button
                onClick={() => setShowPreview(false)}
                className="px-6 py-3 bg-slate-200 hover:bg-slate-300 text-slate-700 font-semibold rounded-lg transition-colors"
              >
                Fermer
              </button>
              <button
                onClick={handleDownload}
                className="flex items-center gap-2 px-6 py-3 bg-amber-500 hover:bg-amber-600 text-white font-semibold rounded-lg transition-colors shadow-md hover:shadow-lg"
              >
                <Download className="w-5 h-5" />
                Télécharger PDF
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default QuoteComponent;