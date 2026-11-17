import { lazy, Suspense, useCallback, useEffect, useLayoutEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { useGuide } from "./guideContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import { computePlacement, findTarget } from "./logic.js";
import { bubbleIn, gsap, moveBubble, moveSpotlight, pulseRing } from "./motion.js";

// three.js n'est chargé que lorsqu'une étape affiche une scène 3D.
const SceneCanvas = lazy(() => import("./SceneCanvas.jsx"));

const FOCUSABLE = 'button:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])';

export default function GuideOverlay() {
  const { t } = useI18n();
  const { active, animate, next, prev, stop, setEnabled } = useGuide();
  const spot = useRef(null);
  const ring = useRef(null);
  const bubble = useRef(null);
  const first = useRef(true);

  const step = active?.steps[active.index];
  const total = active?.steps.length ?? 0;
  const index = active?.index ?? 0;
  const isLast = index === total - 1;

  /** Place projecteur et bulle ; `smooth` = animé (changement d'étape), sinon recalage immédiat (scroll, redimensionnement). */
  const layout = useCallback((smooth) => {
    const b = bubble.current, s = spot.current;
    if (!b || !s || !step) return;
    const el = step.target ? findTarget(step.target) : null;
    const vw = window.innerWidth, vh = window.innerHeight;
    const phone = vw < 640;
    const width = phone ? vw - 16 : Math.min(380, vw - 24);
    gsap.set(b, { width });
    const bh = b.getBoundingClientRect().height;
    const doAnimate = smooth && animate && !first.current;

    if (el) {
      const r = el.getBoundingClientRect();
      gsap.set(s, { autoAlpha: 1 });
      moveSpotlight(s, r, doAnimate);
      const pos = phone ? { x: 8, y: vh - bh - 8 } : computePlacement(r, { width, height: bh }, { width: vw, height: vh }, step.placement);
      moveBubble(b, pos, doAnimate);
    } else {
      gsap.set(s, { autoAlpha: 0 });
      moveBubble(b, { x: (vw - width) / 2, y: phone ? vh - bh - 8 : Math.max(12, (vh - bh) / 2) }, false);
    }
  }, [step, animate]);

  // Changement d'étape : on amène la cible à l'écran, puis on place et on anime.
  useLayoutEffect(() => {
    if (!step) return;
    const el = step.target ? findTarget(step.target) : null;
    const r = el?.getBoundingClientRect();
    // L'en-tête de la boutique est collant : on garde de la marge en haut.
    if (el && r && (r.top < 140 || r.bottom > window.innerHeight - 80)) el.scrollIntoView({ block: "center", behavior: animate ? "smooth" : "auto" });
    layout(true);
    const settle = window.setTimeout(() => layout(false), animate ? 450 : 50); // après le défilement
    const ctx = gsap.context(() => {
      if (bubble.current) bubbleIn(bubble.current, animate);
      if (ring.current) pulseRing(ring.current, animate);
    });
    first.current = false;
    return () => { window.clearTimeout(settle); ctx.revert(); };
  }, [step, animate, layout]);

  // La scène 3D se charge après l'affichage de la bulle : on recale si sa hauteur change.
  useEffect(() => {
    const b = bubble.current;
    if (!b) return;
    const ro = new ResizeObserver(() => layout(false));
    ro.observe(b);
    return () => ro.disconnect();
  }, [layout]);

  // Suit le défilement et le redimensionnement (sans animation, une fois par image).
  useEffect(() => {
    let raf = 0;
    const on = () => { cancelAnimationFrame(raf); raf = requestAnimationFrame(() => layout(false)); };
    window.addEventListener("scroll", on, true);
    window.addEventListener("resize", on);
    return () => { cancelAnimationFrame(raf); window.removeEventListener("scroll", on, true); window.removeEventListener("resize", on); };
  }, [layout]);

  // Clavier : ← → Échap, et focus piégé dans la bulle.
  useEffect(() => {
    const onKey = (e) => {
      const typing = e.target?.closest?.("input, textarea, select");
      if (e.key === "Escape") { e.preventDefault(); stop(); }
      else if (e.key === "ArrowRight" && !typing) { e.preventDefault(); if (isLast) stop(); else next(); }
      else if (e.key === "ArrowLeft" && !typing) { e.preventDefault(); prev(); }
      else if (e.key === "Tab" && bubble.current) {
        const items = [...bubble.current.querySelectorAll(FOCUSABLE)];
        if (!items.length) return;
        const a = document.activeElement;
        if (!bubble.current.contains(a)) { e.preventDefault(); items[0].focus(); }
        else if (e.shiftKey && (a === items[0] || a === bubble.current)) { e.preventDefault(); items[items.length - 1].focus(); }
        else if (!e.shiftKey && a === items[items.length - 1]) { e.preventDefault(); items[0].focus(); }
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [next, prev, stop, isLast]);

  // Le focus passe à la bulle à chaque étape (lecteurs d'écran : le titre est annoncé).
  useEffect(() => { bubble.current?.focus({ preventScroll: true }); }, [step]);

  if (!active || !step) return null;

  return createPortal(
    <div className="pointer-events-none fixed inset-0 z-[4500]" data-guide-overlay>
      {/* Étape centrée : voile léger sans découpe. Le fond reste cliquable. */}
      {!step.target && <div className="absolute inset-0 bg-night/50" aria-hidden="true" />}
      {/* Projecteur : la découpe est une ombre géante autour de la cible. */}
      <div ref={spot} aria-hidden="true" className="absolute left-0 top-0 rounded-[6px] opacity-0 shadow-[0_0_0_9999px_rgba(19,25,33,0.6)]">
        <span className="absolute inset-0 rounded-[6px] border-2 border-action" />
        <span ref={ring} className="absolute inset-0 rounded-[6px] border-2 border-action" />
      </div>

      <section
        ref={bubble}
        role="dialog"
        aria-modal="false"
        aria-labelledby="guide-title"
        aria-describedby="guide-text"
        tabIndex={-1}
        style={{ outline: "none" }}
        className="pointer-events-auto absolute left-0 top-0 rounded-[8px] bg-white p-4 text-black shadow-2xl sm:p-5"
      >
        <div className="mb-3 flex items-center gap-3" data-step-content>
          <div role="progressbar" aria-valuemin={1} aria-valuemax={total} aria-valuenow={index + 1} aria-label={t("guide.ui.stepOf", { n: index + 1, total })} className="h-1.5 flex-1 overflow-hidden rounded-full bg-gray-200">
            <div className="h-full rounded-full bg-action transition-[width] duration-300" style={{ width: `${((index + 1) / total) * 100}%` }} />
          </div>
          <span className="text-xs tabular-nums text-gray-600">{index + 1} / {total}</span>
          <button type="button" onClick={stop} aria-label={t("guide.ui.quit")} className="-mr-1 rounded-[3px] p-1 text-gray-600 transition hover:bg-gray-100 hover:text-black"><X size={16} /></button>
        </div>
        {step.scene && (
          <Suspense fallback={<div className="mb-3 h-[13.5rem] rounded-[6px] bg-gray-100 [@media(max-height:560px)]:hidden" aria-hidden="true" />}>
            <SceneCanvas id={step.scene} animate={animate} />
          </Suspense>
        )}
        <div aria-live="polite">
          <h2 id="guide-title" data-step-content className="text-base font-semibold">{t(step.titleKey)}</h2>
          <p id="guide-text" data-step-content className="mt-1.5 text-sm leading-relaxed text-gray-700">{t(step.textKey)}</p>
        </div>
        <div className="mt-4 flex items-center justify-between gap-2" data-step-content>
          <button type="button" onClick={() => setEnabled(false)} className="text-xs text-gray-600 underline-offset-2 hover:text-black hover:underline">{t("guide.ui.disable")}</button>
          <div className="flex gap-2">
            <button type="button" onClick={prev} disabled={index === 0} className="rounded-[3px] border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium transition hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-40">{t("guide.ui.prev")}</button>
            <button type="button" onClick={isLast ? stop : next} className="rounded-[3px] bg-action px-3.5 py-1.5 text-sm font-bold transition hover:bg-action-dark">{isLast ? t("guide.ui.finish") : t("guide.ui.next")}</button>
          </div>
        </div>
      </section>
    </div>,
    document.body,
  );
}
