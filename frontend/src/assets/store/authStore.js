import { create } from "zustand";
import { request } from "../lib/api.js";

// Session du client. `status` : "unknown" (avant la première vérification) | "anonymous" | "authenticated".
// Le jeton d'accès n'est conservé qu'en mémoire : un rechargement de page le récupère via le cookie de rafraîchissement.
export const useAuth = create((set, get) => ({
  status: "unknown",
  user: null,
  accessToken: null,

  /** Restaure la session au chargement (cookie de rafraîchissement) — silencieux si aucune session. */
  async init() {
    if (get().status !== "unknown") return;
    const r = await request("/auth/refresh", { method: "POST" });
    if (!r.ok) return set({ status: "anonymous", user: null, accessToken: null });
    await get().loadMe(r.data.accessToken);
  },

  async loadMe(token) {
    const me = await request("/me", { token });
    if (!me.ok) return set({ status: "anonymous", user: null, accessToken: null });
    set({ status: "authenticated", user: me.data, accessToken: token });
  },

  /** @returns {Promise<{ok: true} | {ok: false, error: string, retryAfter?: number}>} */
  async login(email, password) {
    const r = await request("/auth/login", { method: "POST", body: { email, password } });
    if (!r.ok) return { ok: false, error: errorKey(r), retryAfter: r.retryAfter };
    set({ status: "authenticated", user: r.data.user, accessToken: r.data.tokens.accessToken });
    return { ok: true };
  },

  /** Inscription : réponse uniforme (202), que l'adresse existe déjà ou non. */
  async register(payload) {
    const r = await request("/auth/register", { method: "POST", body: payload });
    return r.ok ? { ok: true } : { ok: false, error: errorKey(r), retryAfter: r.retryAfter };
  },

  /** Connexion Google, étape 2 : le site reçoit un code à usage unique (60 s) et l'échange contre une session. */
  async exchangeOAuth(code) {
    const r = await request("/auth/oauth/exchange", { method: "POST", body: { code } });
    if (!r.ok) return { ok: false, error: errorKey(r) };
    set({ status: "authenticated", user: r.data.user, accessToken: r.data.tokens.accessToken });
    return { ok: true };
  },

  async logout() {
    await request("/auth/logout", { method: "POST" });
    set({ status: "anonymous", user: null, accessToken: null });
  },
}));

function errorKey(r) {
  if (r.status === 429) return "auth.err.throttle";
  if (r.status === 0 || r.status >= 500) return "auth.err.unavailable";
  if (r.code === "INVALID_CREDENTIALS") return "auth.err.invalid";
  if (r.fields?.some((f) => f.code === "PASSWORD_COMPROMISED")) return "auth.err.compromised";
  if (r.status === 422 || r.status === 400) return "auth.err.rejected";
  return "auth.err.generic";
}

/** Adresse de la page qui reçoit le retour de Google (doit figurer dans la liste blanche du serveur). */
export const oauthReturnUrl = () => `${window.location.origin}/oauth/callback`;

/**
 * Connexion Google, étape 1 : on vérifie que le serveur sait la faire (identifiants Google configurés), puis on y va.
 * @returns {Promise<{ok: true} | {ok: false, error: string}>} (si ok, la page est quittée)
 */
export async function startGoogle() {
  const url = `/v1/auth/oauth/google/start?redirectUri=${encodeURIComponent(oauthReturnUrl())}`;
  let res;
  try {
    res = await fetch(url, { redirect: "manual", headers: { Accept: "application/json" } });
  } catch {
    return { ok: false, error: "auth.err.unavailable" };
  }
  // 302 : le navigateur le signale par une réponse « opaque » ; une erreur JSON (404 : non configuré, 429…) est lisible.
  if (res.type !== "opaqueredirect" && res.status !== 302) return { ok: false, error: res.status === 429 ? "auth.err.throttle" : res.status === 404 ? "auth.err.googleOff" : "auth.err.unavailable" };
  window.location.assign(url);
  return { ok: true };
}
