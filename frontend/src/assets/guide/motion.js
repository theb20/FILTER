import gsap from "gsap";

const PAD = 6;

/** Fait glisser le projecteur vers la cible (ou l'y place d'un coup si `animate` est faux). */
export function moveSpotlight(el, box, animate) {
  const vars = { x: box.x - PAD, y: box.y - PAD, width: box.width + PAD * 2, height: box.height + PAD * 2 };
  if (animate) gsap.to(el, { ...vars, duration: 0.5, ease: "power3.inOut", overwrite: "auto" });
  else gsap.set(el, vars);
}

export function moveBubble(el, pos, animate) {
  pos = { x: pos.x, y: pos.y }; // computePlacement renvoie aussi `side` : GSAP ne doit recevoir que x et y
  if (animate) gsap.to(el, { x: pos.x, y: pos.y, duration: 0.45, ease: "power3.out", overwrite: "auto" });
  else gsap.set(el, pos);
}

/** Entrée de la bulle à chaque étape : léger fondu + glissement, rien de plus. */
export function bubbleIn(el, animate) {
  if (!animate) return gsap.set(el, { opacity: 1 });
  return gsap.fromTo(el.querySelectorAll("[data-step-content]"), { opacity: 0, y: 8 }, { opacity: 1, y: 0, duration: 0.35, ease: "power2.out", stagger: 0.05, overwrite: "auto" });
}

/** Pulsation discrète de l'anneau autour de l'élément montré. */
export function pulseRing(el, animate) {
  if (!animate) return null;
  return gsap.fromTo(el, { scale: 1, opacity: 0.9 }, { scale: 1.06, opacity: 0, duration: 1.4, ease: "power1.out", repeat: -1 });
}

export { gsap };
