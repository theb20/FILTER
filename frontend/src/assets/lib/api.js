// Client HTTP du site vers le gateway. En développement, Vite relaie /v1 vers le gateway (même origine : pas de CORS) ;
// en production le site et l'API partagent le même domaine parent (cookie de session SameSite=Strict).
// Le jeton d'accès reste EN MÉMOIRE (jamais dans localStorage) ; le jeton de rafraîchissement vit dans un cookie HttpOnly.
const BASE = import.meta.env.VITE_API_URL ?? "";

/** @returns {Promise<{ok: boolean, status: number, data: any, code?: string, retryAfter?: number, fields?: Array<{code: string, field?: string}>}>} */
export async function request(path, { method = "GET", body, token } = {}) {
  let res;
  try {
    res = await fetch(`${BASE}/v1${path}`, {
      method,
      credentials: "include",
      headers: {
        Accept: "application/json",
        "X-Requested-With": "ff-web", // exigé par le gateway sur les routes qui lisent le cookie (défense CSRF)
        ...(body === undefined ? {} : { "Content-Type": "application/json" }),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    return { ok: false, status: 0, data: null, code: "UNREACHABLE" };
  }
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (res.ok) return { ok: true, status: res.status, data };
  const retry = Number(res.headers.get("Retry-After"));
  return { ok: false, status: res.status, data, code: data?.code ?? "ERROR", retryAfter: retry > 0 ? retry : undefined, fields: data?.errors ?? [] };
}
