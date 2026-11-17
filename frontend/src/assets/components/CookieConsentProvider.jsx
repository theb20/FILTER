// À ADAPTER : chemin d'import des composants de consentement (registre shadcn).
// `shadcn init` n'ayant pas abouti (pas de components.json), je n'ai pas pu
// déterminer le dossier réel où le composant sera installé.
import {
  CookieConsentProvider,
  CookieBanner,
  CookieBannerBackdrop,
  CookieSettings,
  CookieTrigger,
} from "@/components/ui/cookie-consent";

const config = {
  consentVersion: "1.0.0",
  privacyPolicyUrl: "/privacy",
  size: "sm",
  googleConsentMode: {
    enabled: true,
  },
  // Désactivé : aucun backend n'expose /api/consent (requête en 404 sinon).
  traceability: {
    enabled: false,
    endpoint: "/api/consent",
  },
};

export default function CookieConsentProviderWrapper({ children }) {
  return (
    <CookieConsentProvider config={config}>
      {/* Contenu de l'application */}
      {children}

      {/* Bannière et paramètres (rendus en portail) */}
      <CookieBannerBackdrop />
      <CookieBanner size="sm" className="rounded-none" />
      <CookieSettings className="rounded-none" />
      <CookieTrigger variant="text" />
    </CookieConsentProvider>
  );
}
