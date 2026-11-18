"use client";

import { createContext, lazy, Suspense, useCallback, useContext, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { usePathname, useRouter } from "next/navigation";
import { TOURS } from "./content";
import { resolveSteps, tourForPath, hasPermission } from "./logic";
import { createPrefsStore, DEFAULT_PREFS } from "./prefs-store";
import type { GuidePrefs, GuideStep, GuideTour } from "./types";

// GSAP n'est chargé qu'à l'ouverture d'une visite.
const GuideOverlay = lazy(() => import("./guide-overlay"));

type Active = { tour: GuideTour; steps: GuideStep[]; index: number; path: string };

type GuideApi = {
  prefs: GuidePrefs;
  /** Visites accessibles au membre (selon ses permissions). */
  tours: GuideTour[];
  /** Visite de la page courante, si elle existe. */
  current: GuideTour | undefined;
  active: Active | null;
  /** Anime-t-on ? Faux si désactivé par le membre ou demandé par le système. */
  animate: boolean;
  setEnabled: (v: boolean) => void;
  setAnimations: (v: "auto" | "off") => void;
  start: (tourId: string) => void;
  next: () => void;
  prev: () => void;
  stop: () => void;
  resetSeen: () => void;
};

const Ctx = createContext<GuideApi | null>(null);
export const useGuide = () => {
  const c = useContext(Ctx);
  if (!c) throw new Error("useGuide doit être utilisé dans <GuideProvider>");
  return c;
};

const isVisible = (target: string) => {
  const el = document.querySelector<HTMLElement>(`[data-guide="${target}"]`);
  if (!el) return false;
  const r = el.getBoundingClientRect();
  return r.width > 0 && r.height > 0;
};

const reducedMotion = () => typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

/**
 * `initialPrefs` : préférences du serveur (lues à l'affichage de la page) ; `save` : action serveur qui les enregistre.
 * Sans l'un ou l'autre (serveur injoignable), le guide fonctionne sur le cache du navigateur.
 */
export function GuideProvider({ userId, permissions, initialPrefs, save, children }: { userId: string; permissions: string[]; initialPrefs?: GuidePrefs | null; save?: (p: GuidePrefs) => Promise<boolean>; children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const store = useMemo(() => createPrefsStore(`ff-guide:${userId}`, undefined, save ? { push: save } : undefined), [userId, save]);
  const prefs = useSyncExternalStore(store.subscribe, store.get, () => DEFAULT_PREFS);
  const [activeRaw, setActive] = useState<Active | null>(null);
  // Une visite n'est valable que sur la page où elle a démarré : naviguer la ferme, sans effet de bord.
  const active = activeRaw && activeRaw.path === pathname ? activeRaw : null;
  const [pending, setPending] = useState<string | null>(null); // visite à lancer après navigation
  const lastFocus = useRef<HTMLElement | null>(null);
  const hydrated = useRef<typeof store | null>(null);

  // Le serveur fait foi : on rapatrie ses préférences une seule fois par membre (un rafraîchissement ultérieur ne doit pas écraser un choix récent).
  useEffect(() => {
    if (!initialPrefs || hydrated.current === store) return;
    hydrated.current = store;
    store.hydrate(initialPrefs);
  }, [store, initialPrefs]);

  const tours = useMemo(() => TOURS.filter((t) => hasPermission(permissions, t.permission)), [permissions]);
  const current = useMemo(() => tourForPath(TOURS, pathname, permissions), [pathname, permissions]);
  const animate = prefs.animations === "auto" && !reducedMotion();

  // Les effets de bord (écriture des préférences) restent hors des fonctions de mise à jour d'état.
  const stop = useCallback(() => {
    if (active) {
      const seen = store.get().seen;
      if (!seen.includes(active.tour.id)) store.set({ seen: [...seen, active.tour.id] });
    }
    setActive(null);
    lastFocus.current?.focus?.();
  }, [store, active]);

  const begin = useCallback((tour: GuideTour) => {
    const steps = resolveSteps(tour.steps, isVisible);
    if (steps.length === 0) return;
    lastFocus.current = document.activeElement as HTMLElement | null;
    setActive({ tour, steps, index: 0, path: pathname });
  }, [pathname]);

  const start = useCallback((tourId: string) => {
    const tour = tours.find((t) => t.id === tourId);
    if (!tour) return;
    if (tour.match(pathname)) begin(tour);
    else { setPending(tourId); router.push(tour.path); }
  }, [tours, pathname, router, begin]);

  const next = useCallback(() => setActive((a) => (a && a.index < a.steps.length - 1 ? { ...a, index: a.index + 1 } : a)), []);
  const prev = useCallback(() => setActive((a) => (a && a.index > 0 ? { ...a, index: a.index - 1 } : a)), []);

  // Lancement différé : après navigation demandée depuis les réglages, ou automatiquement à la première visite d'une page.
  // Les préférences sont relues dans le délai (pas dans la fermeture) : elles ne sont fiables qu'après l'hydratation.
  useEffect(() => {
    if (active) return;
    const id = window.setTimeout(() => {
      if (pending) {
        const t = tours.find((x) => x.id === pending && x.match(pathname));
        if (t) { setPending(null); begin(t); }
        return;
      }
      const p = store.get();
      if (p.enabled && current?.auto && !p.seen.includes(current.id)) begin(current);
    }, 700);
    return () => window.clearTimeout(id);
  }, [pathname, current, pending, active, tours, store, begin]);

  const api: GuideApi = {
    prefs, tours, current, active, animate,
    setEnabled: (v) => { store.set({ enabled: v }); if (!v) setActive(null); },
    setAnimations: (v) => store.set({ animations: v }),
    start, next, prev, stop,
    resetSeen: () => store.set({ seen: [] }),
  };

  return (
    <Ctx.Provider value={api}>
      {children}
      {active && <Suspense fallback={null}><GuideOverlay /></Suspense>}
    </Ctx.Provider>
  );
}
