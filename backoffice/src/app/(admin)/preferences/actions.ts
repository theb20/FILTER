"use server";

import { adminAct } from "@/lib/admin";
import type { GuidePrefs } from "@/components/guide/types";

/** Enregistre les préférences du guide du membre connecté. `true` si le serveur les a prises. Le gateway valide de toute façon les bornes. */
export async function saveGuidePrefs(prefs: GuidePrefs): Promise<boolean> {
  const p = prefs as Partial<GuidePrefs> | null;
  if (!p || typeof p.enabled !== "boolean" || (p.animations !== "auto" && p.animations !== "off") || !Array.isArray(p.seen) || p.seen.some((s) => typeof s !== "string")) return false;
  const r = await adminAct("PUT", "/admin/me/preferences", { body: { guide: { enabled: p.enabled, animations: p.animations, seen: p.seen } } });
  return r.ok;
}
