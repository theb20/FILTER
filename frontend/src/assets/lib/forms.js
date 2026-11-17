// @ts-check
// Envoi des formulaires publics (contact, devis, lettre d'information) à l'API.
import { request } from "./api.js";

/**
 * @param {string} path
 * @param {Record<string, unknown>} body les valeurs vides sont retirées
 * @returns {Promise<{ ok: true } | { ok: false, error: "throttle" | "invalid" | "unavailable" }>}
 */
export async function sendForm(path, body) {
  const clean = Object.fromEntries(Object.entries(body).filter(([, v]) => v !== "" && v !== undefined && v !== null));
  const r = await request(path, { method: "POST", body: clean });
  if (r.ok) return { ok: true };
  if (r.status === 429) return { ok: false, error: "throttle" };
  if (r.status === 400 || r.status === 422) return { ok: false, error: "invalid" };
  return { ok: false, error: "unavailable" };
}
