import React, { useState } from "react";
import {
  Shield,
  Cookie,
  Lock,
  Eye,
  FileText,
  CheckCircle,
  Mail,
  Calendar,
  Database,
  Globe,
  AlertTriangle,
  Users,
  Settings,
  Bell,
  Trash2,
  Download,
  Edit,
  MapPin,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Info,
} from "lucide-react";

const PolicySection = ({
  id,
  icon,
  title,
  badge,
  badgeColor,
  expanded,
  onToggle,
  children,
}) => {
  const colorClasses = {
    blue: "bg-blue-50 text-blue-700 border-blue-200",
    green: "bg-green-50 text-green-700 border-green-200",
    red: "bg-red-50 text-red-700 border-red-200",
    purple: "bg-purple-50 text-purple-700 border-purple-200",
    indigo: "bg-indigo-50 text-indigo-700 border-indigo-200",
  };

  return (
    <div className="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-all duration-300">
      <button
        onClick={onToggle}
        className="w-full flex justify-between items-center p-6 hover:bg-gray-50 transition-colors group"
      >
        <div className="flex items-center gap-4 flex-1 text-left">
          <div className="p-3 bg-gradient-to-br from-amber-500 to-orange-600 rounded-lg text-white shadow-sm group-hover:shadow-md transition-shadow">
            {icon}
          </div>
          <div className="flex-1">
            <h3 className="font-semibold text-gray-900 text-lg mb-1.5">{title}</h3>
            <span
              className={`inline-flex items-center gap-1 text-xs font-medium px-2.5 py-1 rounded-md border ${
                colorClasses[badgeColor] || "bg-gray-50 text-gray-700 border-gray-200"
              }`}
            >
              {badge}
            </span>
          </div>
        </div>
        <div className={`transition-transform duration-300 ${expanded ? 'rotate-180' : ''}`}>
          <ChevronDown className="w-5 h-5 text-gray-400 group-hover:text-gray-600" />
        </div>
      </button>
      {expanded && (
        <div className="px-6 pb-6 pt-2 border-t border-gray-100 animate-fadeIn">
          {children}
        </div>
      )}
    </div>
  );
};

const InfoCard = ({ icon, title, description }) => (
  <div className="bg-gradient-to-br from-gray-50 to-white border border-gray-200 rounded-lg p-5 hover:shadow-md transition-shadow">
    <div className="flex items-start gap-3">
      <div className="p-2 bg-amber-100 rounded-lg text-amber-700">
        {icon}
      </div>
      <div>
        <h4 className="font-semibold text-gray-900 mb-1">{title}</h4>
        <p className="text-sm text-gray-600 leading-relaxed">{description}</p>
      </div>
    </div>
  </div>
);

export default function PrivacyPolicyPage() {
  const [expandedSection, setExpandedSection] = useState(null);
  const lastUpdated = "17 octobre 2025";

  const toggleSection = (id) => {
    setExpandedSection(expandedSection === id ? null : id);
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-50 to-white">
      {/* Header Sticky */}
      <header className="bg-white border-b border-gray-200 sticky lg:top-42 top-15  z-10 backdrop-blur-sm bg-white/95 shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="p-2.5 bg-gradient-to-br from-amber-500 to-orange-600 rounded-xl shadow-lg">
                <Shield className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-xl sm:text-2xl font-bold text-gray-900">
                  Politique de Confidentialité
                </h1>
                <p className="text-xs sm:text-sm text-gray-500 flex items-center gap-2 mt-0.5">
                  <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                  Filter Finder
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 self-start sm:self-auto">
              <Calendar className="w-4 h-4 text-gray-400" />
              <span className="text-xs text-gray-500">Mis à jour le</span>
              <span className="px-3 py-1.5 bg-amber-50 text-amber-700 text-xs font-semibold rounded-lg border border-amber-200">
                {lastUpdated}
              </span>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative bg-gradient-to-br from-amber-500 via-orange-500 to-orange-600 text-white overflow-hidden">
        <div className="absolute inset-0 bg-grid-white/10"></div>
        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16 sm:py-20">
          <div className="max-w-3xl">
            <div className="inline-flex items-center gap-2 px-4 py-2 bg-white/20 backdrop-blur-sm rounded-full mb-6 border border-white/30">
              <CheckCircle className="w-4 h-4" />
              <span className="text-sm font-medium">Conforme RGPD & ePrivacy</span>
            </div>
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-6 leading-tight">
              Votre vie privée est<br />notre priorité absolue
            </h2>
            <p className="text-base sm:text-lg text-orange-50 leading-relaxed mb-8">
              Nous nous engageons à protéger vos données personnelles avec les plus hauts 
              standards de sécurité, de transparence et de conformité réglementaire.
            </p>
            <div className="flex flex-wrap gap-4">
              <button className="px-6 py-3 bg-white text-orange-600 rounded-lg font-semibold hover:bg-orange-50 transition-colors shadow-lg">
                Contactez-nous
              </button>
              <button className="px-6 py-3 bg-white/10 backdrop-blur-sm text-white rounded-lg font-semibold hover:bg-white/20 transition-colors border border-white/30">
                Télécharger en PDF
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Quick Overview */}
      <section className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="bg-blue-50 border border-blue-200 rounded-xl p-6 mb-12">
          <div className="flex items-start gap-3">
            <Info className="w-5 h-5 text-blue-600 mt-0.5 flex-shrink-0" />
            <div>
              <h3 className="font-semibold text-blue-900 mb-2">Résumé en bref</h3>
              <p className="text-sm text-blue-800 leading-relaxed">
                Cette politique explique comment Filter Finder collecte, utilise et protège vos données personnelles. 
                Nous respectons strictement le RGPD et ne partageons jamais vos informations sans votre consentement explicite.
              </p>
            </div>
          </div>
        </div>

        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6 mb-16">
          <InfoCard
            icon={<Lock className="w-5 h-5" />}
            title="Sécurité maximale"
            description="Chiffrement SSL/TLS et conformité aux normes ISO 27001"
          />
          <InfoCard
            icon={<Shield className="w-5 h-5" />}
            title="RGPD compliant"
            description="Respect total de vos droits et de la réglementation européenne"
          />
          <InfoCard
            icon={<Users className="w-5 h-5" />}
            title="Transparence totale"
            description="Vous savez exactement quelles données nous collectons et pourquoi"
          />
        </div>
      </section>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pb-16">
        <div className="space-y-4">
          <PolicySection
            id={1}
            icon={<Database className="w-5 h-5" />}
            title="1. Données que nous collectons"
            badge="Essentiel"
            badgeColor="blue"
            expanded={expandedSection === 1}
            onToggle={() => toggleSection(1)}
          >
            <div className="space-y-6">
              <div>
                <h4 className="font-semibold text-gray-900 mb-3">Données d'identification</h4>
                <ul className="space-y-2 text-gray-700">
                  <li className="flex items-start gap-2">
                    <ChevronRight className="w-4 h-4 text-amber-500 mt-1 flex-shrink-0" />
                    <span>Nom, prénom et adresse email</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <ChevronRight className="w-4 h-4 text-amber-500 mt-1 flex-shrink-0" />
                    <span>Informations de contact (numéro de téléphone, adresse postale)</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <ChevronRight className="w-4 h-4 text-amber-500 mt-1 flex-shrink-0" />
                    <span>Identifiants de compte et mots de passe chiffrés</span>
                  </li>
                </ul>
              </div>

              <div>
                <h4 className="font-semibold text-gray-900 mb-3">Données de navigation</h4>
                <ul className="space-y-2 text-gray-700">
                  <li className="flex items-start gap-2">
                    <ChevronRight className="w-4 h-4 text-amber-500 mt-1 flex-shrink-0" />
                    <span>Adresse IP, type de navigateur et système d'exploitation</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <ChevronRight className="w-4 h-4 text-amber-500 mt-1 flex-shrink-0" />
                    <span>Pages visitées, durée des sessions et interactions</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <ChevronRight className="w-4 h-4 text-amber-500 mt-1 flex-shrink-0" />
                    <span>Cookies analytiques et préférences utilisateur</span>
                  </li>
                </ul>
              </div>

              <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
                <p className="text-sm text-amber-900">
                  <strong>Note :</strong> Nous ne collectons que les données strictement nécessaires 
                  au fonctionnement de nos services. Vous pouvez refuser certaines collectes via vos paramètres.
                </p>
              </div>
            </div>
          </PolicySection>

          <PolicySection
            id={2}
            icon={<Eye className="w-5 h-5" />}
            title="2. Comment nous utilisons vos données"
            badge="Important"
            badgeColor="green"
            expanded={expandedSection === 2}
            onToggle={() => toggleSection(2)}
          >
            <div className="space-y-6">
              <p className="text-gray-700 leading-relaxed">
                Vos données personnelles sont utilisées exclusivement pour les finalités suivantes :
              </p>

              <div className="grid sm:grid-cols-2 gap-4">
                <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                  <h5 className="font-semibold text-green-900 mb-2 flex items-center gap-2">
                    <CheckCircle className="w-4 h-4" />
                    Services essentiels
                  </h5>
                  <ul className="text-sm text-green-800 space-y-1">
                    <li>• Création et gestion de compte</li>
                    <li>• Traitement des commandes</li>
                    <li>• Service client et support</li>
                    <li>• Communications transactionnelles</li>
                  </ul>
                </div>

                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                  <h5 className="font-semibold text-blue-900 mb-2 flex items-center gap-2">
                    <Settings className="w-4 h-4" />
                    Amélioration continue
                  </h5>
                  <ul className="text-sm text-blue-800 space-y-1">
                    <li>• Analyse d'usage et performances</li>
                    <li>• Personnalisation de l'expérience</li>
                    <li>• Prévention de la fraude</li>
                    <li>• Développement de nouvelles fonctionnalités</li>
                  </ul>
                </div>
              </div>

              <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
                <p className="text-sm text-gray-700">
                  <strong>Marketing :</strong> Nous ne vous envoyons des communications commerciales 
                  que si vous y avez explicitement consenti. Vous pouvez vous désabonner à tout moment.
                </p>
              </div>
            </div>
          </PolicySection>

          <PolicySection
            id={3}
            icon={<Globe className="w-5 h-5" />}
            title="3. Partage et transfert de données"
            badge="Transparence"
            badgeColor="purple"
            expanded={expandedSection === 3}
            onToggle={() => toggleSection(3)}
          >
            <div className="space-y-6">
              <p className="text-gray-700 leading-relaxed">
                Nous ne vendons <strong>jamais</strong> vos données personnelles. Le partage est limité 
                aux cas suivants :
              </p>

              <div className="space-y-4">
                <div className="border-l-4 border-purple-500 pl-4 py-2">
                  <h5 className="font-semibold text-gray-900 mb-1">Prestataires de services</h5>
                  <p className="text-sm text-gray-600">
                    Hébergement, paiement, livraison - tous soumis à des accords de confidentialité stricts
                  </p>
                </div>

                <div className="border-l-4 border-purple-500 pl-4 py-2">
                  <h5 className="font-semibold text-gray-900 mb-1">Obligations légales</h5>
                  <p className="text-sm text-gray-600">
                    Sur demande d'autorités judiciaires ou pour se conformer à la loi
                  </p>
                </div>

                <div className="border-l-4 border-purple-500 pl-4 py-2">
                  <h5 className="font-semibold text-gray-900 mb-1">Transferts internationaux</h5>
                  <p className="text-sm text-gray-600">
                    Uniquement vers des pays offrant un niveau de protection adéquat (clauses contractuelles types)
                  </p>
                </div>
              </div>
            </div>
          </PolicySection>

          <PolicySection
            id={4}
            icon={<Lock className="w-5 h-5" />}
            title="4. Sécurité de vos données"
            badge="Critique"
            badgeColor="red"
            expanded={expandedSection === 4}
            onToggle={() => toggleSection(4)}
          >
            <div className="space-y-6">
              <p className="text-gray-700 leading-relaxed">
                Nous mettons en œuvre des mesures de sécurité de niveau professionnel :
              </p>

              <div className="grid sm:grid-cols-2 gap-4">
                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                  <Lock className="w-6 h-6 text-red-600 mb-2" />
                  <h5 className="font-semibold text-red-900 mb-2">Chiffrement</h5>
                  <p className="text-sm text-red-800">
                    SSL/TLS 256 bits pour toutes les transmissions. Chiffrement AES-256 au repos.
                  </p>
                </div>

                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                  <Shield className="w-6 h-6 text-red-600 mb-2" />
                  <h5 className="font-semibold text-red-900 mb-2">Infrastructure</h5>
                  <p className="text-sm text-red-800">
                    Serveurs sécurisés, pare-feu avancés, audits de sécurité réguliers.
                  </p>
                </div>

                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                  <Eye className="w-6 h-6 text-red-600 mb-2" />
                  <h5 className="font-semibold text-red-900 mb-2">Surveillance</h5>
                  <p className="text-sm text-red-800">
                    Détection d'intrusion 24/7, logs d'accès, alertes en temps réel.
                  </p>
                </div>

                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                  <Users className="w-6 h-6 text-red-600 mb-2" />
                  <h5 className="font-semibold text-red-900 mb-2">Accès restreint</h5>
                  <p className="text-sm text-red-800">
                    Seuls les employés autorisés ont accès, avec authentification à deux facteurs.
                  </p>
                </div>
              </div>
            </div>
          </PolicySection>

          <PolicySection
            id={5}
            icon={<Shield className="w-5 h-5" />}
            title="5. Vos droits RGPD"
            badge="Vos droits"
            badgeColor="indigo"
            expanded={expandedSection === 5}
            onToggle={() => toggleSection(5)}
          >
            <div className="space-y-6">
              <p className="text-gray-700 leading-relaxed">
                Conformément au RGPD, vous disposez des droits suivants :
              </p>

              <div className="space-y-3">
                <div className="flex items-start gap-3 p-4 bg-indigo-50 border border-indigo-200 rounded-lg">
                  <Download className="w-5 h-5 text-indigo-600 mt-0.5 flex-shrink-0" />
                  <div>
                    <h5 className="font-semibold text-indigo-900 mb-1">Droit d'accès</h5>
                    <p className="text-sm text-indigo-800">
                      Obtenez une copie de toutes vos données personnelles
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3 p-4 bg-indigo-50 border border-indigo-200 rounded-lg">
                  <Edit className="w-5 h-5 text-indigo-600 mt-0.5 flex-shrink-0" />
                  <div>
                    <h5 className="font-semibold text-indigo-900 mb-1">Droit de rectification</h5>
                    <p className="text-sm text-indigo-800">
                      Corrigez ou mettez à jour vos informations à tout moment
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3 p-4 bg-indigo-50 border border-indigo-200 rounded-lg">
                  <Trash2 className="w-5 h-5 text-indigo-600 mt-0.5 flex-shrink-0" />
                  <div>
                    <h5 className="font-semibold text-indigo-900 mb-1">Droit à l'effacement</h5>
                    <p className="text-sm text-indigo-800">
                      Demandez la suppression définitive de vos données
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3 p-4 bg-indigo-50 border border-indigo-200 rounded-lg">
                  <AlertTriangle className="w-5 h-5 text-indigo-600 mt-0.5 flex-shrink-0" />
                  <div>
                    <h5 className="font-semibold text-indigo-900 mb-1">Droit d'opposition</h5>
                    <p className="text-sm text-indigo-800">
                      Refusez certains traitements, notamment marketing
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3 p-4 bg-indigo-50 border border-indigo-200 rounded-lg">
                  <FileText className="w-5 h-5 text-indigo-600 mt-0.5 flex-shrink-0" />
                  <div>
                    <h5 className="font-semibold text-indigo-900 mb-1">Droit à la portabilité</h5>
                    <p className="text-sm text-indigo-800">
                      Récupérez vos données dans un format structuré et exploitable
                    </p>
                  </div>
                </div>
              </div>

              <div className="bg-gradient-to-r from-indigo-500 to-purple-600 text-white rounded-lg p-6">
                <h5 className="font-semibold mb-3 flex items-center gap-2">
                  <Mail className="w-5 h-5" />
                  Exercez vos droits facilement
                </h5>
                <p className="text-sm text-indigo-50 mb-4">
                  Contactez notre délégué à la protection des données :
                </p>
                <a 
                  href="mailto:privacy@filterfinder.com"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-white text-indigo-600 rounded-lg font-semibold hover:bg-indigo-50 transition-colors"
                >
                  privacy@filterfinder.com
                  <ExternalLink className="w-4 h-4" />
                </a>
              </div>
            </div>
          </PolicySection>

          <PolicySection
            id={6}
            icon={<Cookie className="w-5 h-5" />}
            title="6. Cookies et technologies similaires"
            badge="Personnalisation"
            badgeColor="blue"
            expanded={expandedSection === 6}
            onToggle={() => toggleSection(6)}
          >
            <div className="space-y-6">
              <p className="text-gray-700 leading-relaxed">
                Nous utilisons des cookies pour améliorer votre expérience. Vous pouvez les gérer via vos préférences.
              </p>

              <div className="grid sm:grid-cols-3 gap-4">
                <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
                  <h5 className="font-semibold text-gray-900 mb-2">Essentiels</h5>
                  <p className="text-xs text-gray-600 mb-2">Nécessaires au fonctionnement</p>
                  <span className="text-xs px-2 py-1 bg-green-100 text-green-700 rounded-full">Toujours actifs</span>
                </div>

                <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
                  <h5 className="font-semibold text-gray-900 mb-2">Analytiques</h5>
                  <p className="text-xs text-gray-600 mb-2">Comprendre l'usage du site</p>
                  <span className="text-xs px-2 py-1 bg-blue-100 text-blue-700 rounded-full">Optionnels</span>
                </div>

                <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
                  <h5 className="font-semibold text-gray-900 mb-2">Marketing</h5>
                  <p className="text-xs text-gray-600 mb-2">Publicités pertinentes</p>
                  <span className="text-xs px-2 py-1 bg-blue-100 text-blue-700 rounded-full">Optionnels</span>
                </div>
              </div>

              <button className="w-full sm:w-auto px-6 py-3 bg-amber-500 hover:bg-amber-600 text-white rounded-lg font-semibold transition-colors">
                Gérer mes préférences cookies
              </button>
            </div>
          </PolicySection>

          <PolicySection
            id={7}
            icon={<Calendar className="w-5 h-5" />}
            title="7. Conservation des données"
            badge="Durées"
            badgeColor="purple"
            expanded={expandedSection === 7}
            onToggle={() => toggleSection(7)}
          >
            <div className="space-y-4">
              <p className="text-gray-700 leading-relaxed">
                Nous conservons vos données uniquement le temps nécessaire :
              </p>

              <div className="overflow-x-auto">
                <table className="w-full text-sm border border-gray-200 rounded-lg overflow-hidden">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left font-semibold text-gray-900">Type de données</th>
                      <th className="px-4 py-3 text-left font-semibold text-gray-900">Durée de conservation</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    <tr className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-700">Données de compte actif</td>
                      <td className="px-4 py-3 text-gray-600">Durée de vie du compte</td>
                    </tr>
                    <tr className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-700">Historique des commandes</td>
                      <td className="px-4 py-3 text-gray-600">10 ans (obligations légales)</td>
                    </tr>
                    <tr className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-700">Données de navigation</td>
                      <td className="px-4 py-3 text-gray-600">13 mois maximum</td>
                    </tr>
                    <tr className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-700">Compte inactif</td>
                      <td className="px-4 py-3 text-gray-600">3 ans puis suppression</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </PolicySection>

          <PolicySection
            id={8}
            icon={<Mail className="w-5 h-5" />}
            title="8. Contact et réclamations"
            badge="Support"
            badgeColor="green"
            expanded={expandedSection === 8}
            onToggle={() => toggleSection(8)}
          >
            <div className="space-y-6">
              <p className="text-gray-700 leading-relaxed">
                Pour toute question concernant vos données personnelles :
              </p>

              <div className="grid sm:grid-cols-2 gap-4">
                <div className="bg-green-50 border border-green-200 rounded-lg p-5">
                  <Mail className="w-6 h-6 text-green-600 mb-3" />
                  <h5 className="font-semibold text-green-900 mb-2">Email</h5>
                  <a href="mailto:privacy@filterfinder.com" className="text-sm text-green-700 hover:text-green-800 underline">
                    privacy@filterfinder.com
                  </a>
                </div>

                <div className="bg-green-50 border border-green-200 rounded-lg p-5">
                  <MapPin className="w-6 h-6 text-green-600 mb-3" />
                  <h5 className="font-semibold text-green-900 mb-2">Adresse postale</h5>
                  <p className="text-sm text-green-800">
                    Filter Finder<br />
                    Service Protection des Données<br />
                    123 Avenue de la République<br />
                    75011 Paris, France
                  </p>
                </div>
              </div>

              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-5">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="w-5 h-5 text-yellow-600 mt-0.5 flex-shrink-0" />
                  <div>
                    <h5 className="font-semibold text-yellow-900 mb-2">Droit de réclamation</h5>
                    <p className="text-sm text-yellow-800 mb-3">
                      Si vous estimez que vos droits ne sont pas respectés, vous pouvez déposer une réclamation 
                      auprès de la CNIL (Commission Nationale de l'Informatique et des Libertés).
                    </p>
                    <a 
                      href="https://www.cnil.fr" 
                      target="_blank" 
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 text-sm text-yellow-900 font-semibold hover:text-yellow-700"
                    >
                      Visiter le site de la CNIL
                      <ExternalLink className="w-4 h-4" />
                    </a>
                  </div>
                </div>
              </div>
            </div>
          </PolicySection>
        </div>

        {/* Call to Action */}
        <div className="mt-12 bg-gradient-to-br from-amber-500 to-orange-600 rounded-xl p-8 text-white text-center">
          <Shield className="w-12 h-12 mx-auto mb-4" />
          <h3 className="text-2xl font-bold mb-3">Des questions sur vos données ?</h3>
          <p className="text-amber-50 mb-6 max-w-2xl mx-auto">
            Notre équipe dédiée à la protection des données est à votre disposition pour répondre 
            à toutes vos questions et vous accompagner dans l'exercice de vos droits.
          </p>
          <div className="flex flex-wrap justify-center gap-4">
            <button className="px-8 py-3 bg-white text-orange-600 rounded-lg font-semibold hover:bg-orange-50 transition-colors shadow-lg">
              Nous contacter
            </button>
            <button className="px-8 py-3 bg-white/10 backdrop-blur-sm text-white rounded-lg font-semibold hover:bg-white/20 transition-colors border border-white/30">
              Centre d'aide
            </button>
          </div>
        </div>
      </main>

      
    </div>
  );
}