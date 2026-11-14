import React, { useState, useEffect, useRef } from 'react';
import { Link } from 'react-router-dom';

const STORAGE_KEY = "ff_cookie_consent";
const COOKIE_NAME = "ff_cookie_consent";

export default function CookiePopup() {
  const [isVisible, setIsVisible] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [preferences, setPreferences] = useState({
    necessary: true,
    analytics: true,
    marketing: false,
    functional: false
  });
  const [isAnimating, setIsAnimating] = useState(false);

  const mainActionRef = useRef(null);
  const previouslyFocused = useRef(null);

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        if (parsed?.consented === true || parsed?.decisionDate) {
          return;
        }
      } catch {
      // stockage ou cookie indisponible : on continue sans persister
    }
    }

    previouslyFocused.current = document.activeElement;
    setIsVisible(true);
    requestAnimationFrame(() => setIsAnimating(true));
  }, []);

  useEffect(() => {
    if (isAnimating && mainActionRef.current) {
      mainActionRef.current.focus();
    }
  }, [isAnimating]);

  useEffect(() => {
    const onKey = (e) => {
      if (e.key === "Escape" && isVisible) {
        handleCloseDismiss();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [isVisible]);

  const saveConsent = (prefs, extra = {}) => {
    const payload = {
      preferences: prefs,
      consented: !!(prefs.analytics || prefs.marketing || prefs.functional),
      decisionDate: new Date().toISOString(),
      ...extra
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));

    try {
      const expires = new Date();
      expires.setFullYear(expires.getFullYear() + 1);
      document.cookie = `${COOKIE_NAME}=${encodeURIComponent(JSON.stringify(payload))}; expires=${expires.toUTCString()}; path=/; samesite=lax`;
    } catch {
      // stockage ou cookie indisponible : on continue sans persister
    }
  };

  const handleAcceptAll = () => {
    const prefs = { necessary: true, analytics: true, marketing: true, functional: true };
    setPreferences(prefs);
    saveConsent(prefs, { acceptedAll: true });
    closePopup();
  };

  const handleAcceptSelected = () => {
    const prefs = { ...preferences, necessary: true };
    setPreferences(prefs);
    saveConsent(prefs, { acceptedSelected: true });
    closePopup();
  };

  const handleRejectAll = () => {
    const prefs = { necessary: true, analytics: false, marketing: false, functional: false };
    setPreferences(prefs);
    saveConsent(prefs, { rejectedAll: true });
    closePopup();
  };

  const handleCloseDismiss = () => {
    const prefs = { necessary: true, analytics: false, marketing: false, functional: false };
    saveConsent(prefs, { dismissed: true });
    closePopup();
  };

  const closePopup = () => {
    setIsAnimating(false);
    setTimeout(() => {
      setIsVisible(false);
      if (previouslyFocused.current && previouslyFocused.current.focus) {
        previouslyFocused.current.focus();
      }
    }, 300);
  };

  const togglePreference = (key) => {
    if (key === 'necessary') return;
    setPreferences(prev => ({ ...prev, [key]: !prev[key] }));
  };

  if (!isVisible) return null;

  return (
    <>
    {/* Flou de la page derrière la bannière */}
    <div
      aria-hidden="true"
      className={`fixed inset-0 z-[1099] bg-black/20 backdrop-blur-md transition-opacity duration-300 ${
        isAnimating ? 'opacity-100' : 'opacity-0'
      }`}
    />
    <div
      role="dialog"
      aria-label="Cookies"
      className={`fixed bottom-4 left-4 right-4 sm:right-auto sm:w-[22rem] z-[1100] bg-white border border-gray-200 shadow-xl p-5 transition-all duration-300 ${
        isAnimating ? 'translate-y-0 opacity-100' : 'translate-y-4 opacity-0'
      }`}
    >
      {!showSettings ? (
        <>
          <p className="text-sm text-slate-600 leading-6 mb-4">
            Ce site utilise des cookies pour fonctionner et, avec votre accord, mesurer l'audience.{' '}
            <Link to="/privacy" className="underline text-slate-900 hover:text-amber-600">
              En savoir plus
            </Link>
          </p>

          <div className="flex items-center gap-2">
            <button
              type="button"
              ref={mainActionRef}
              onClick={handleAcceptAll}
              className="px-4 py-2 bg-slate-900 hover:bg-amber-500 text-white text-sm font-medium transition-colors"
            >
              Accepter
            </button>
            <button
              type="button"
              onClick={handleRejectAll}
              className="px-4 py-2 border border-gray-300 hover:border-slate-900 text-slate-700 text-sm font-medium transition-colors"
            >
              Refuser
            </button>
            <button
              type="button"
              onClick={() => setShowSettings(true)}
              className="ml-auto text-sm text-slate-500 hover:text-slate-900 underline"
            >
              Choisir
            </button>
          </div>
        </>
      ) : (
        <>
          <div className="space-y-2 mb-4">
            <CookieOption
              title="Nécessaires"
              description="Fonctionnement du site"
              checked={preferences.necessary}
              disabled={true}
              onChange={() => {}}
            />
            <CookieOption
              title="Fonctionnels"
              description="Préférences"
              checked={preferences.functional}
              onChange={() => togglePreference('functional')}
            />
            <CookieOption
              title="Analytiques"
              description="Mesure d'audience"
              checked={preferences.analytics}
              onChange={() => togglePreference('analytics')}
            />
            <CookieOption
              title="Marketing"
              description="Publicités personnalisées"
              checked={preferences.marketing}
              onChange={() => togglePreference('marketing')}
            />
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              ref={mainActionRef}
              onClick={handleAcceptSelected}
              className="px-4 py-2 bg-slate-900 hover:bg-amber-500 text-white text-sm font-medium transition-colors"
            >
              Enregistrer
            </button>
            <button
              type="button"
              onClick={() => setShowSettings(false)}
              className="text-sm text-slate-500 hover:text-slate-900 underline"
            >
              Retour
            </button>
          </div>
        </>
      )}
    </div>
    </>
  );
}

function CookieOption({ title, description, checked, disabled, onChange }) {
  return (
    <label
      className={`flex items-center justify-between gap-3 text-sm ${
        disabled ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'
      }`}
    >
      <span>
        <span className="font-medium text-slate-900">{title}</span>
        <span className="block text-xs text-slate-500">{description}</span>
      </span>
      <input
        type="checkbox"
        checked={checked}
        onChange={onChange}
        disabled={disabled}
        className="h-4 w-4 accent-slate-900"
      />
    </label>
  );
}
