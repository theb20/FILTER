import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

// Vrai si l'utilisateur demande de réduire les animations (accessibilité).
export const prefersReducedMotion = () =>
  typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

// Vrai pour une souris / un trackpad (pas de tilt, curseur ni magnétisme sur tactile).
export const hasFinePointer = () =>
  typeof window !== "undefined" && window.matchMedia("(hover: hover) and (pointer: fine)").matches;

// Petit « vol » d'une pastille jusqu'à l'icône du panier quand on ajoute un produit.
export function flyToCart(fromEl) {
  if (prefersReducedMotion() || !fromEl) return;
  const target = document.querySelector("[data-cart-icon]");
  if (!target) return;
  const a = fromEl.getBoundingClientRect();
  const b = target.getBoundingClientRect();
  const dot = document.createElement("div");
  dot.setAttribute("aria-hidden", "true");
  Object.assign(dot.style, {
    position: "fixed", left: "0", top: "0", width: "18px", height: "18px", borderRadius: "9999px",
    background: "#F7BE14", boxShadow: "0 0 18px 4px rgba(247,190,20,.7)", zIndex: "4000", pointerEvents: "none",
  });
  document.body.appendChild(dot);
  const sx = a.left + a.width / 2 - 9, sy = a.top + a.height / 2 - 9;
  const ex = b.left + b.width / 2 - 9, ey = b.top + b.height / 2 - 9;
  gsap.set(dot, { x: sx, y: sy, scale: 0.4 });
  const tl = gsap.timeline({ onComplete: () => dot.remove() });
  tl.to(dot, { scale: 1.2, duration: 0.15, ease: "power2.out" })
    .to(dot, { x: ex, duration: 0.7, ease: "power1.in" }, 0)
    .to(dot, { y: ey, duration: 0.7, ease: "back.in(1.2)" }, 0)
    .to(dot, { scale: 0.2, opacity: 0.4, duration: 0.2 }, "-=0.2");
}

export { gsap, ScrollTrigger };
