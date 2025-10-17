import { useState } from "react";
import { Mail, Phone, MapPin, Clock, Send, CheckCircle, Building, User, MessageSquare, AlertCircle } from "lucide-react";

const ContactPage = () => {
    useState(()=>{
        document.title = "Contact - Filter Finder";
    })
  const [formData, setFormData] = useState({
    firstName: "",
    lastName: "",
    email: "",
    phone: "",
    subject: "",
    message: "",
    company: "",
    agreeToTerms: false,
  });
  const [errors, setErrors] = useState({});
  const [isLoading, setIsLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  const subjects = [
    "Demande d'information",
    "Support technique",
    "Partenariat commercial",
    "Réclamation",
    "Autre",
  ];

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.firstName.trim()) {
      newErrors.firstName = "Le prénom est requis";
    }
    if (!formData.lastName.trim()) {
      newErrors.lastName = "Le nom est requis";
    }
    if (!formData.email.trim()) {
      newErrors.email = "L'email est requis";
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = "Email invalide";
    }
    if (!formData.phone.trim()) {
      newErrors.phone = "Le téléphone est requis";
    } else if (!/^[\d\s+()-]{10,}$/.test(formData.phone)) {
      newErrors.phone = "Numéro de téléphone invalide";
    }
    if (!formData.subject) {
      newErrors.subject = "Veuillez sélectionner un sujet";
    }
    if (!formData.message.trim()) {
      newErrors.message = "Le message est requis";
    } else if (formData.message.trim().length < 20) {
      newErrors.message = "Le message doit contenir au moins 20 caractères";
    }
    if (!formData.agreeToTerms) {
      newErrors.agreeToTerms = "Vous devez accepter les conditions";
    }
    
    return newErrors;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    const newErrors = validateForm();
    
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }
    
    setIsLoading(true);
    
    setTimeout(() => {
      setIsLoading(false);
      setIsSuccess(true);
      setFormData({
        firstName: "",
        lastName: "",
        email: "",
        phone: "",
        subject: "",
        message: "",
        company: "",
        agreeToTerms: false,
      });
      setErrors({});
      
      setTimeout(() => {
        setIsSuccess(false);
      }, 5000);
    }, 2000);
  };

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === "checkbox" ? checked : value,
    }));
    
    if (errors[name]) {
      setErrors((prev) => {
        const updated = { ...prev };
        delete updated[name];
        return updated;
      });
    }
  };

  return (
    <div className="min-h-screen bg-slate-50">
     

      {/* Hero Section */}
      <section className="bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 text-white py-16 relative overflow-hidden">
        <div className="absolute inset-0 opacity-10">
          <div className="absolute inset-0" style={{
            backgroundImage: 'url("data:image/svg+xml,%3Csvg width="60" height="60" viewBox="0 0 60 60" xmlns="http://www.w3.org/2000/svg"%3E%3Cg fill="none" fill-rule="evenodd"%3E%3Cg fill="white" fill-opacity="0.4"%3E%3Cpath d="M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z"/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")',
          }}></div>
        </div>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
          <div className="text-center max-w-3xl mx-auto">
            <h1 className="text-4xl md:text-5xl font-bold mb-4">Contactez-nous</h1>
            <p className="text-xl text-slate-300">
              Notre équipe d'experts est à votre disposition pour répondre à toutes vos questions
            </p>
          </div>
        </div>
      </section>

      {/* Success Message */}
      {isSuccess && (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-8">
          <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-6 flex items-start gap-4">
            <div className="flex-shrink-0">
              <CheckCircle className="w-6 h-6 text-emerald-600" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-emerald-900 mb-1">Message envoyé avec succès!</h3>
              <p className="text-emerald-700">Nous avons bien reçu votre demande. Notre équipe vous répondra dans les plus brefs délais.</p>
            </div>
          </div>
        </div>
      )}

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Contact Information */}
          <div className="lg:col-span-1 space-y-6">
            {/* Contact Cards */}
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6">
              <h2 className="text-xl font-bold text-slate-900 mb-4">Informations de contact</h2>
              <div className="space-y-4">
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 w-12 h-12 bg-amber-100 rounded-lg flex items-center justify-center">
                    <Phone className="w-6 h-6 text-amber-600" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-slate-900 mb-1">Téléphone</h3>
                    <p className="text-slate-600">+225 07 05 234 324</p>
                    <p className="text-sm text-slate-500 mt-1">Lun - Ven: 9h - 18h</p>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
                    <Mail className="w-6 h-6 text-blue-600" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-slate-900 mb-1">Email</h3>
                    <p className="text-slate-600">contact@filter-finder.com</p>
                    <p className="text-sm text-slate-500 mt-1">Réponse sous 24h</p>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 w-12 h-12 bg-emerald-100 rounded-lg flex items-center justify-center">
                    <MapPin className="w-6 h-6 text-emerald-600" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-slate-900 mb-1">Adresse</h3>
                    <p className="text-slate-600">123 Avenue des Champs-Élysées</p>
                    <p className="text-slate-600">75008 Paris, France</p>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
                    <Clock className="w-6 h-6 text-purple-600" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-slate-900 mb-1">Horaires d'ouverture</h3>
                    <p className="text-slate-600">Lundi - Vendredi: 9h - 18h</p>
                    <p className="text-slate-600">Samedi: 10h - 16h</p>
                    <p className="text-slate-600">Dimanche: Fermé</p>
                  </div>
                </div>
              </div>
            </div>

            {/* FAQ Card */}
            <div className="bg-gradient-to-br from-amber-500 to-orange-600 rounded-xl shadow-sm p-6 text-white">
              <h3 className="text-xl font-bold mb-2">Questions fréquentes</h3>
              <p className="mb-4 text-white/90">Consultez notre FAQ pour des réponses rapides</p>
              <button className="w-full py-2.5 bg-white text-amber-600 font-semibold rounded-lg hover:bg-slate-50 transition-colors">
                Voir la FAQ
              </button>
            </div>

            {/* Support Card */}
            <div className="bg-slate-900 rounded-xl shadow-sm p-6 text-white">
              <h3 className="text-xl font-bold mb-2">Support technique</h3>
              <p className="mb-4 text-slate-300">Besoin d'aide urgente? Notre équipe technique est disponible 24/7</p>
              <button 
              onClick={() => window.location.href = 'mailto:contact@filter-finder.com'}
              className="w-full py-2.5 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded-lg transition-colors">
                Contacter le support
              </button>
            </div>
          </div>

          {/* Contact Form */}
          <div className="lg:col-span-2">
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-8">
              <div className="mb-6">
                <h2 className="text-2xl font-bold text-slate-900 mb-2">Envoyez-nous un message</h2>
                <p className="text-slate-600">Remplissez le formulaire ci-dessous et nous vous répondrons dans les plus brefs délais</p>
              </div>

              <form onSubmit={handleSubmit} className="space-y-6">
                {/* Name Fields */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">
                      Prénom <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <User className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                      <input
                        type="text"
                        name="firstName"
                        value={formData.firstName}
                        onChange={handleInputChange}
                        placeholder="Jean"
                        className={`w-full pl-11 pr-4 py-3 border ${
                          errors.firstName ? "border-red-500 bg-red-50" : "border-slate-300"
                        } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent transition-all`}
                      />
                    </div>
                    {errors.firstName && (
                      <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                        <AlertCircle className="w-4 h-4" /> {errors.firstName}
                      </p>
                    )}
                  </div>
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">
                      Nom <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <User className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                      <input
                        type="text"
                        name="lastName"
                        value={formData.lastName}
                        onChange={handleInputChange}
                        placeholder="Dupont"
                        className={`w-full pl-11 pr-4 py-3 border ${
                          errors.lastName ? "border-red-500 bg-red-50" : "border-slate-300"
                        } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent transition-all`}
                      />
                    </div>
                    {errors.lastName && (
                      <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                        <AlertCircle className="w-4 h-4" /> {errors.lastName}
                      </p>
                    )}
                  </div>
                </div>

                {/* Email and Phone */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-2">
                      Email <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                      <input
                        type="email"
                        name="email"
                        value={formData.email}
                        onChange={handleInputChange}
                        placeholder="jean.dupont@email.com"
                        className={`w-full pl-11 pr-4 py-3 border ${
                          errors.email ? "border-red-500 bg-red-50" : "border-slate-300"
                        } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent transition-all`}
                      />
                    </div>
                    {errors.email && (
                      <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                        <AlertCircle className="w-4 h-4" /> {errors.email}
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
                        name="phone"
                        value={formData.phone}
                        onChange={handleInputChange}
                        placeholder="+225 07 05 234 324"
                        className={`w-full pl-11 pr-4 py-3 border ${
                          errors.phone ? "border-red-500 bg-red-50" : "border-slate-300"
                        } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent transition-all`}
                      />
                    </div>
                    {errors.phone && (
                      <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                        <AlertCircle className="w-4 h-4" /> {errors.phone}
                      </p>
                    )}
                  </div>
                </div>

                {/* Company (Optional) */}
                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Entreprise <span className="text-slate-400 font-normal">(optionnel)</span>
                  </label>
                  <div className="relative">
                    <Building className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
                    <input
                      type="text"
                      name="company"
                      value={formData.company}
                      onChange={handleInputChange}
                      placeholder="Nom de votre entreprise"
                      className="w-full pl-11 pr-4 py-3 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent transition-all"
                    />
                  </div>
                </div>

                {/* Subject */}
                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Sujet <span className="text-red-500">*</span>
                  </label>
                  <select
                    name="subject"
                    value={formData.subject}
                    onChange={handleInputChange}
                    className={`w-full px-4 py-3 border ${
                      errors.subject ? "border-red-500 bg-red-50" : "border-slate-300"
                    } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent transition-all`}
                  >
                    <option value="">Sélectionnez un sujet</option>
                    {subjects.map((subject) => (
                      <option key={subject} value={subject}>
                        {subject}
                      </option>
                    ))}
                  </select>
                  {errors.subject && (
                    <p className="text-red-600 text-sm mt-1 flex items-center gap-1">
                      <AlertCircle className="w-4 h-4" /> {errors.subject}
                    </p>
                  )}
                </div>

                {/* Message */}
                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-2">
                    Message <span className="text-red-500">*</span>
                  </label>
                  <div className="relative">
                    <MessageSquare className="absolute left-3 top-3 text-slate-400 w-5 h-5" />
                    <textarea
                      name="message"
                      value={formData.message}
                      onChange={handleInputChange}
                      placeholder="Décrivez votre demande en détail..."
                      rows={6}
                      className={`w-full pl-11 pr-4 py-3 border ${
                        errors.message ? "border-red-500 bg-red-50" : "border-slate-300"
                      } rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent transition-all resize-none`}
                    ></textarea>
                  </div>
                  <div className="flex justify-between items-center mt-1">
                    {errors.message ? (
                      <p className="text-red-600 text-sm flex items-center gap-1">
                        <AlertCircle className="w-4 h-4" /> {errors.message}
                      </p>
                    ) : (
                      <p className="text-slate-500 text-sm">Minimum 20 caractères</p>
                    )}
                    <p className="text-slate-500 text-sm">{formData.message.length} / 1000</p>
                  </div>
                </div>

                {/* Terms Checkbox */}
                <div>
                  <label className="flex items-start gap-3 cursor-pointer group">
                    <input
                      type="checkbox"
                      name="agreeToTerms"
                      checked={formData.agreeToTerms}
                      onChange={handleInputChange}
                      className="mt-1 w-5 h-5 text-amber-600 border-slate-300 rounded focus:ring-2 focus:ring-amber-500"
                    />
                    <span className="text-sm text-slate-600 group-hover:text-slate-900 transition-colors">
                      J'accepte que mes données soient utilisées pour traiter ma demande conformément à la{" "}
                      <a href="#" className="text-amber-600 hover:text-amber-700 font-semibold">
                        politique de confidentialité
                      </a>
                      {" "}<span className="text-red-500">*</span>
                    </span>
                  </label>
                  {errors.agreeToTerms && (
                    <p className="text-red-600 text-sm mt-2 flex items-center gap-1 ml-8">
                      <AlertCircle className="w-4 h-4" /> {errors.agreeToTerms}
                    </p>
                  )}
                </div>

                {/* Submit Button */}
                <div className="pt-4">
                  <button
                    type="submit"
                    disabled={isLoading}
                    className="w-full py-4 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded-lg transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed shadow-sm hover:shadow-md flex items-center justify-center gap-2"
                  >
                    {isLoading ? (
                      <>
                        <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Envoi en cours...
                      </>
                    ) : (
                      <>
                        <Send className="w-5 h-5" />
                        Envoyer le message
                      </>
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      </div>

      {/* Stats Section Professionnelle */}
        <section className="relative bg-slate-950 py-20 overflow-hidden">
        {/* Effet dégradé de fond */}
        <div className="absolute inset-0 bg-gradient-to-br from-amber-500/10 via-slate-800/40 to-slate-900"></div>

        <div className="relative max-w-7xl mx-auto px-6 lg:px-8 text-center">
            {/* Titre */}
            <h2 className="text-4xl md:text-5xl font-extrabold text-white mb-4 tracking-tight">
            Pourquoi nous faire confiance ?
            </h2>
            <p className="text-slate-400 text-lg max-w-2xl mx-auto mb-16">
            Notre engagement se reflète dans des performances concrètes et mesurables.
            </p>

            {/* Grille des statistiques */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8">
            {/* Carte 1 */}
            <div className="group bg-white/5 border border-white/10 backdrop-blur-xl rounded-2xl p-8 transition-all hover:border-amber-500/40 hover:-translate-y-2 hover:shadow-amber-500/10 hover:shadow-lg">
                <div className="flex justify-center mb-4">
                <div className="bg-amber-500/10 p-3 rounded-full">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-8 w-8 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 10l-2 2m0 0l-2-2m2 2V6m8 10a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                </div>
                </div>
                <h3 className="text-4xl font-bold text-white mb-2">10K+</h3>
                <p className="text-slate-300 font-medium">Clients satisfaits</p>
            </div>

            {/* Carte 2 */}
            <div className="group bg-white/5 border border-white/10 backdrop-blur-xl rounded-2xl p-8 transition-all hover:border-emerald-500/40 hover:-translate-y-2 hover:shadow-emerald-500/10 hover:shadow-lg">
                <div className="flex justify-center mb-4">
                <div className="bg-emerald-500/10 p-3 rounded-full">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-8 w-8 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                    </svg>
                </div>
                </div>
                <h3 className="text-4xl font-bold text-white mb-2">99.9%</h3>
                <p className="text-slate-300 font-medium">Disponibilité</p>
            </div>

            {/* Carte 3 */}
            <div className="group bg-white/5 border border-white/10 backdrop-blur-xl rounded-2xl p-8 transition-all hover:border-blue-500/40 hover:-translate-y-2 hover:shadow-blue-500/10 hover:shadow-lg">
                <div className="flex justify-center mb-4">
                <div className="bg-blue-500/10 p-3 rounded-full">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-8 w-8 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m2 9H7a2 2 0 01-2-2V7a2 2 0 012-2h4l2 2h4a2 2 0 012 2v12a2 2 0 01-2 2z" />
                    </svg>
                </div>
                </div>
                <h3 className="text-4xl font-bold text-white mb-2">24/7</h3>
                <p className="text-slate-300 font-medium">Support technique</p>
            </div>

            {/* Carte 4 */}
            <div className="group bg-white/5 border border-white/10 backdrop-blur-xl rounded-2xl p-8 transition-all hover:border-purple-500/40 hover:-translate-y-2 hover:shadow-purple-500/10 hover:shadow-lg">
                <div className="flex justify-center mb-4">
                <div className="bg-purple-500/10 p-3 rounded-full">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-8 w-8 text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                </div>
                </div>
                <h3 className="text-4xl font-bold text-white mb-2">&lt;2h</h3>
                <p className="text-slate-300 font-medium">Temps de réponse</p>
            </div>
            </div>
        </div>
        </section>



      {/* CTA Section */}
      <section className="bg-slate-100 py-16">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="bg-gradient-to-br from-amber-500 to-orange-600 rounded-2xl shadow-xl p-8 md:p-12 text-center text-white relative overflow-hidden">
            <div className="absolute inset-0 opacity-10">
              <div className="absolute inset-0" style={{
                backgroundImage: 'url("data:image/svg+xml,%3Csvg width="60" height="60" viewBox="0 0 60 60" xmlns="http://www.w3.org/2000/svg"%3E%3Cg fill="none" fill-rule="evenodd"%3E%3Cg fill="white" fill-opacity="0.4"%3E%3Cpath d="M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z"/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")',
              }}></div>
            </div>
            <div className="relative z-10">
              <h2 className="text-3xl md:text-4xl font-bold mb-4">
                Prêt à commencer ?
              </h2>
              <p className="text-xl text-white/90 mb-8 max-w-2xl mx-auto">
                Rejoignez des milliers d'entreprises qui nous font confiance pour gérer leurs projets immobiliers en toute sérénité
              </p>
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <button className="px-8 py-4 bg-white text-amber-600 font-semibold rounded-lg hover:bg-slate-50 transition-all shadow-lg hover:shadow-xl">
                  Demander une démo gratuite
                </button>
                <button className="px-8 py-4 bg-transparent border-2 border-white text-white font-semibold rounded-lg hover:bg-white/10 transition-all">
                  En savoir plus
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

    </div>
  );
};

export default ContactPage;