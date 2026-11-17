// Logique pure du guide (testée sans navigateur). Même contrat que celle du backoffice.

export const tourForPath = (tours, pathname) => tours.find((t) => t.match(pathname));

export const clampIndex = (i, length) => Math.min(Math.max(i, 0), Math.max(length - 1, 0));

/** Première cible `data-guide` réellement visible : le même repère peut exister en version bureau et mobile. */
export function findTarget(name, root = document) {
  for (const el of root.querySelectorAll(`[data-guide="${name}"]`)) {
    const r = el.getBoundingClientRect();
    if (r.width > 0 && r.height > 0) return el;
  }
  return null;
}

/** Ne garde que les étapes montrables : sans cible (étape centrée) ou dont la cible est visible. Une cible absente ne bloque jamais la visite. */
export const resolveSteps = (steps, isVisible) => steps.filter((s) => !s.target || isVisible(s.target));

/** Place la bulle à côté de la cible, du côté qui tient dans la fenêtre ; à défaut, la recentre en bas de l'écran. */
export function computePlacement(target, bubble, view, preferred = "auto", gap = 14, margin = 12) {
  const fits = (x, y) => x >= margin && y >= margin && x + bubble.width <= view.width - margin && y + bubble.height <= view.height - margin;
  const cx = target.x + target.width / 2 - bubble.width / 2;
  const cy = target.y + target.height / 2 - bubble.height / 2;
  const spots = {
    bottom: [cx, target.y + target.height + gap],
    top: [cx, target.y - bubble.height - gap],
    right: [target.x + target.width + gap, cy],
    left: [target.x - bubble.width - gap, cy],
  };
  const order = preferred === "auto" ? ["bottom", "top", "right", "left"] : [preferred, "bottom", "top", "right", "left"];
  for (const side of order) {
    const [x, y] = spots[side];
    const nx = Math.min(Math.max(x, margin), view.width - bubble.width - margin);
    const ny = Math.min(Math.max(y, margin), view.height - bubble.height - margin);
    const horizontal = side === "bottom" || side === "top";
    if (fits(horizontal ? nx : x, horizontal ? y : ny)) return { x: horizontal ? nx : x, y: horizontal ? y : ny, side };
  }
  return { x: Math.max(margin, (view.width - bubble.width) / 2), y: Math.max(margin, view.height - bubble.height - margin), side: "center" };
}
