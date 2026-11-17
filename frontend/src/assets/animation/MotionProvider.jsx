import { useEffect, useRef } from "react";
import { useLocation } from "react-router-dom";
import { gsap, ScrollTrigger, prefersReducedMotion, hasFinePointer } from "./gsapSetup.js";

// Éléments révélés au scroll (fondu + montée, en cascade).
const REVEAL_SEL = [
  "main section", "main article", "main h1", "main h2", "main form", "main table",
  'main li[class*="shadow"]', 'main li[class*="rounded"]', "footer", "[data-reveal]",
].join(",");
const MAGNET_SEL = 'a[class*="bg-action"], button[class*="bg-action"], [data-magnet]';
const INTERACTIVE_SEL = "a, button, input, select, textarea, label, summary, [role='button']";

// Couche d'animation globale : réagit à chaque scroll, survol, clic et changement de page.
export default function MotionProvider({ children }) {
  const { pathname } = useLocation();
  const bar = useRef(null);
  const first = useRef(true);

  // 1. Barre de progression du scroll
  useEffect(() => {
    if (prefersReducedMotion()) return;
    const ctx = gsap.context(() => {
      gsap.set(bar.current, { scaleX: 0, transformOrigin: "0 50%" });
      gsap.to(bar.current, {
        scaleX: 1, ease: "none",
        scrollTrigger: { start: 0, end: "max", scrub: 0.3 },
      });
    });
    return () => ctx.revert();
  }, []);

  // 2. Révélation au scroll de tout élément qui apparaît dans le DOM (pages lazy comprises)
  useEffect(() => {
    if (prefersReducedMotion()) return;
    const seen = new WeakSet();
    let refreshTimer;

    const reveal = (root) => {
      const found = [...root.querySelectorAll(REVEAL_SEL), ...(root.matches?.(REVEAL_SEL) ? [root] : [])]
        .filter((el) => !seen.has(el) && !el.closest("header, [data-no-anim]"));
      if (!found.length) return;
      found.forEach((el) => seen.add(el));
      gsap.set(found, { opacity: 0, y: 28 });
      ScrollTrigger.batch(found, {
        start: "top 92%",
        once: true,
        onEnter: (batch) =>
          gsap.to(batch, {
            opacity: 1, y: 0, duration: 0.8, ease: "power3.out", stagger: 0.09,
            overwrite: "auto", clearProps: "opacity",
          }),
      });
      clearTimeout(refreshTimer);
      refreshTimer = setTimeout(() => ScrollTrigger.refresh(), 300);
    };

    reveal(document.body);
    const mo = new MutationObserver((records) => {
      for (const r of records) r.addedNodes.forEach((n) => n.nodeType === 1 && reveal(n));
    });
    mo.observe(document.body, { childList: true, subtree: true });
    return () => { mo.disconnect(); clearTimeout(refreshTimer); };
  }, []);

  // 3. Pointeur : clic (onde + pression), boutons magnétiques
  useEffect(() => {
    if (prefersReducedMotion()) return;
    const fine = hasFinePointer();
    let magnet = null;
    let pressed = null;

    const resetMagnet = (el) =>
      el && gsap.to(el, { x: 0, y: 0, duration: 0.7, ease: "elastic.out(1,0.4)", overwrite: "auto" });

    const onMove = (e) => {
      if (!fine || e.pointerType !== "mouse") return;

      const mg = e.target.closest?.(MAGNET_SEL);
      if (mg !== magnet) { resetMagnet(magnet); magnet = mg; }
      if (magnet) {
        const r = magnet.getBoundingClientRect();
        gsap.to(magnet, {
          x: (e.clientX - (r.left + r.width / 2)) * 0.28, y: (e.clientY - (r.top + r.height / 2)) * 0.4,
          duration: 0.3, ease: "power2.out", overwrite: "auto",
        });
      }
    };

    const onDown = (e) => {
      // onde de clic
      const w = document.createElement("div");
      w.setAttribute("aria-hidden", "true");
      Object.assign(w.style, {
        position: "fixed", left: e.clientX - 20 + "px", top: e.clientY - 20 + "px", width: "40px", height: "40px",
        borderRadius: "9999px", border: "2px solid #F7BE14", pointerEvents: "none", zIndex: "4000",
      });
      document.body.appendChild(w);
      gsap.fromTo(w, { scale: 0.2, opacity: 0.9 }, { scale: 2.6, opacity: 0, duration: 0.7, ease: "power2.out", onComplete: () => w.remove() });
      // pression des boutons / liens
      pressed = e.target.closest?.("a, button");
      if (pressed) gsap.to(pressed, { scale: 0.94, duration: 0.12, ease: "power2.out", overwrite: "auto" });
    };
    const onUp = () => {
      if (pressed) gsap.to(pressed, { scale: 1, duration: 0.6, ease: "elastic.out(1,0.4)", overwrite: "auto" });
      pressed = null;
    };
    const onLeaveDoc = () => {
      resetMagnet(magnet); magnet = null;
    };

    document.addEventListener("pointermove", onMove, { passive: true });
    document.addEventListener("pointerdown", onDown, { passive: true });
    document.addEventListener("pointerup", onUp, { passive: true });
    document.addEventListener("pointercancel", onUp, { passive: true });
    document.documentElement.addEventListener("pointerleave", onLeaveDoc);
    return () => {
      document.removeEventListener("pointermove", onMove);
      document.removeEventListener("pointerdown", onDown);
      document.removeEventListener("pointerup", onUp);
      document.removeEventListener("pointercancel", onUp);
      document.documentElement.removeEventListener("pointerleave", onLeaveDoc);
    };
  }, []);

  // 4. Changement de page : retour en haut, sans transition
  useEffect(() => {
    if (first.current) { first.current = false; return; }
    window.scrollTo(0, 0);
    ScrollTrigger.refresh();
  }, [pathname]);

  return (
    <>
      {children}
      <div ref={bar} aria-hidden="true" className="pointer-events-none fixed left-0 top-0 z-[3000] h-[3px] w-full bg-action" style={{ transform: "scaleX(0)" }} />
    </>
  );
}
