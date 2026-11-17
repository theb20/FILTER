import { lazy, Suspense, useCallback, useEffect, useMemo, useState, useSyncExternalStore, useRef } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../store/authStore.js";
import { TOURS } from "./content/tours.js";
import { findTarget, resolveSteps, tourForPath } from "./logic.js";
import { createPrefsStore, DEFAULT_PREFS } from "./prefsStore.js";
import { GuideContext } from "./guideContext.js";

// GSAP (guide) n'est chargé qu'à l'ouverture d'une visite.
const GuideOverlay = lazy(() => import("./GuideOverlay.jsx"));

const reducedMotion = () => typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

/**
 * Guide visuel de la boutique. Les préférences sont propres au compte connecté (ou au visiteur anonyme).
 * Aucune visite ne démarre seule : dans une boutique, le visiteur garde la main (bouton « Découvrir cette section »).
 */
export default function GuideProvider({ children }) {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const userId = useAuth((a) => a.user?.id);
  const store = useMemo(() => createPrefsStore(`ff-guide:${userId ?? "guest"}`), [userId]);
  const prefs = useSyncExternalStore(store.subscribe, store.get, () => DEFAULT_PREFS);
  const [activeRaw, setActive] = useState(null);
  const [pending, setPending] = useState(null); // visite à lancer après navigation demandée depuis le profil
  const lastFocus = useRef(null);

  // Une visite n'est valable que sur la page où elle a démarré : naviguer la ferme, sans effet de bord.
  const active = activeRaw && activeRaw.path === pathname ? activeRaw : null;
  const current = useMemo(() => tourForPath(TOURS, pathname), [pathname]);
  const animate = prefs.animations === "auto" && !reducedMotion();

  const begin = useCallback((tour) => {
    const steps = resolveSteps(tour.steps, (target) => Boolean(findTarget(target)));
    if (steps.length === 0) return;
    lastFocus.current = document.activeElement;
    setActive({ tour, steps, index: 0, path: pathname });
  }, [pathname]);

  const start = useCallback((tourId) => {
    const tour = TOURS.find((t) => t.id === tourId);
    if (!tour) return;
    if (tour.match(pathname)) begin(tour);
    else { setPending(tourId); navigate(tour.path); }
  }, [pathname, navigate, begin]);

  // Les effets de bord (écriture des préférences) restent hors des fonctions de mise à jour d'état.
  const stop = useCallback(() => {
    if (active) {
      const seen = store.get().seen;
      if (!seen.includes(active.tour.id)) store.set({ seen: [...seen, active.tour.id] });
    }
    setActive(null);
    lastFocus.current?.focus?.();
  }, [store, active]);

  const next = useCallback(() => setActive((a) => (a && a.index < a.steps.length - 1 ? { ...a, index: a.index + 1 } : a)), []);
  const prev = useCallback(() => setActive((a) => (a && a.index > 0 ? { ...a, index: a.index - 1 } : a)), []);

  // Lancement après navigation demandée depuis le profil : on attend que la page soit affichée.
  useEffect(() => {
    if (!pending || active) return;
    const id = window.setTimeout(() => {
      const t = TOURS.find((x) => x.id === pending && x.match(pathname));
      if (t) { setPending(null); begin(t); }
    }, 900);
    return () => window.clearTimeout(id);
  }, [pathname, pending, active, begin]);

  const api = {
    prefs, current, active, animate, tours: TOURS,
    setEnabled: (v) => { store.set({ enabled: v }); if (!v) setActive(null); },
    setAnimations: (v) => store.set({ animations: v }),
    resetSeen: () => store.set({ seen: [] }),
    start, next, prev, stop,
  };

  return (
    <GuideContext.Provider value={api}>
      {children}
      {active && <Suspense fallback={null}><GuideOverlay /></Suspense>}
    </GuideContext.Provider>
  );
}
