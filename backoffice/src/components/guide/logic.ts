import type { GuideStep, GuideTour, Placement } from "./types";

/** Même règle que `can` du backoffice (`*` = tout), sans importer de code serveur. */
export const hasPermission = (permissions: string[], needed?: string) => !needed || permissions.includes("*") || permissions.includes(needed);

export const tourForPath = (tours: GuideTour[], pathname: string, permissions: string[]) =>
  tours.find((t) => t.match(pathname) && hasPermission(permissions, t.permission));

/** Ne garde que les étapes montrables : sans cible (étape centrée) ou dont la cible est visible. Une cible absente ne bloque jamais la visite. */
export function resolveSteps(steps: GuideStep[], isVisible: (target: string) => boolean): GuideStep[] {
  return steps.filter((s) => !s.target || isVisible(s.target));
}

export const clampIndex = (i: number, length: number) => Math.min(Math.max(i, 0), Math.max(length - 1, 0));

export type Box = { x: number; y: number; width: number; height: number };

/** Place la bulle à côté de la cible, du côté qui tient entièrement dans la fenêtre ; à défaut, la recentre dans l'écran. */
export function computePlacement(target: Box, bubble: { width: number; height: number }, view: { width: number; height: number }, preferred: Placement = "auto", gap = 14, margin = 12) {
  const fits = (x: number, y: number) => x >= margin && y >= margin && x + bubble.width <= view.width - margin && y + bubble.height <= view.height - margin;
  const cx = target.x + target.width / 2 - bubble.width / 2;
  const cy = target.y + target.height / 2 - bubble.height / 2;
  const spots: Record<Exclude<Placement, "auto">, [number, number]> = {
    bottom: [cx, target.y + target.height + gap],
    top: [cx, target.y - bubble.height - gap],
    right: [target.x + target.width + gap, cy],
    left: [target.x - bubble.width - gap, cy],
  };
  const order: Exclude<Placement, "auto">[] = preferred === "auto" ? ["bottom", "top", "right", "left"] : [preferred, "bottom", "top", "right", "left"];
  for (const side of order) {
    const [x, y] = spots[side];
    // On accepte un simple décalage horizontal/vertical pour rester dans l'écran.
    const nx = Math.min(Math.max(x, margin), view.width - bubble.width - margin);
    const ny = Math.min(Math.max(y, margin), view.height - bubble.height - margin);
    const horizontal = side === "bottom" || side === "top";
    if (fits(horizontal ? nx : x, horizontal ? y : ny)) return { x: horizontal ? nx : x, y: horizontal ? y : ny, side };
  }
  return { x: Math.max(margin, (view.width - bubble.width) / 2), y: Math.max(margin, view.height - bubble.height - margin), side: "center" as const };
}
