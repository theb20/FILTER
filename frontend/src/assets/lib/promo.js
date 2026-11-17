// @ts-check
// Vérification d'un code promotionnel par l'API : la liste des codes n'est JAMAIS envoyée au navigateur.
import { request } from "./api.js";

/**
 * @param {string} rawCode
 * @param {number} subtotal sous-total HT en XOF
 * @returns {Promise<{ status: "valid" | "invalid" | "expired" | "minimum" | "error", promo?: { code: string, type: "percent" | "fixed", value: number, minSubtotal?: number } }>}
 */
export async function checkPromo(rawCode, subtotal) {
  const code = rawCode.trim().toUpperCase();
  const r = await request("/promo-codes/validate", { method: "POST", body: { code, subtotal: Math.max(0, Math.round(subtotal)) } });
  if (!r.ok) return { status: r.status === 429 ? "error" : "error" };
  const d = r.data;
  if (d.status === "invalid" || d.status === "expired") return { status: d.status };
  return { status: d.status, promo: { code, type: d.type, value: d.value, minSubtotal: d.minSubtotal } };
}
