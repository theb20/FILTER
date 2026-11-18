import "server-only";
import { gateway } from "./api";
import { getAccess } from "./session";
import type { GuidePrefs } from "@/components/guide/types";

/** Préférences du guide du membre connecté, ou `null` si le serveur ne peut pas les donner (le navigateur garde alors son cache). */
export async function loadGuidePrefs(): Promise<GuidePrefs | null> {
  const token = await getAccess();
  if (!token) return null;
  const r = await gateway<{ guide: GuidePrefs }>("/admin/me/preferences", { method: "GET", token });
  return r.ok && r.data?.guide ? r.data.guide : null;
}
