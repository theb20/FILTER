import "server-only";
import { notFound, redirect } from "next/navigation";
import { gateway, type ApiResult } from "./api";
import { clearSession, getAccess, getRefresh, setSession, type StaffTokens } from "./session";

type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
type Opts = { body?: unknown; query?: Record<string, string | number | undefined> };

/** Lecture depuis un composant serveur. 401 → renouvellement de session puis retour sur `returnTo` ; 403/404 → pages dédiées. */
export async function adminGet<T>(path: string, returnTo: string, opts: Opts = {}): Promise<T> {
  const token = await getAccess();
  const r = token ? await gateway<T>(path, { method: "GET", token, query: opts.query }) : null;
  if (r?.ok) return r.data;
  if (!r || r.status === 401) redirect((await getRefresh()) ? `/session/refresh?next=${encodeURIComponent(returnTo)}` : "/login");
  if (r.status === 403) redirect("/denied");
  if (r.status === 404) notFound();
  throw new Error(`Gateway ${path} : ${r.status} ${r.code}`);
}

/** Lecture facultative : null si l'élément n'existe pas (404) ou si le droit manque (403), sans redirection (l'appelant a une alternative). */
export async function adminGetOpt<T>(path: string, returnTo: string, opts: Opts = {}): Promise<T | null> {
  const token = await getAccess();
  const r = token ? await gateway<T>(path, { method: "GET", token, query: opts.query }) : null;
  if (r?.ok) return r.data;
  if (!r || r.status === 401) redirect((await getRefresh()) ? `/session/refresh?next=${encodeURIComponent(returnTo)}` : "/login");
  if (r.status === 403 || r.status === 404) return null;
  throw new Error(`Gateway ${path} : ${r.status} ${r.code}`);
}

/** Écriture depuis une action serveur : renouvelle la session si besoin (les cookies sont modifiables ici) puis réessaie une fois. */
export async function adminAct<T = unknown>(method: Method, path: string, opts: Opts = {}): Promise<ApiResult<T>> {
  let token = await getAccess();
  let r = token ? await gateway<T>(path, { method, token, body: opts.body, query: opts.query }) : null;
  if (!r || (!r.ok && r.status === 401)) {
    const refresh = await getRefresh();
    const t = refresh ? await gateway<StaffTokens>("/admin/auth/refresh", { method: "POST", body: { refreshToken: refresh } }) : null;
    if (!t || !t.ok) {
      await clearSession();
      redirect("/login?expired=1");
    }
    await setSession(t.data);
    token = t.data.accessToken;
    r = await gateway<T>(path, { method, token, body: opts.body, query: opts.query });
  }
  return r;
}

/** Message affichable pour une réponse en erreur (le gateway renvoie déjà un texte en français). */
export function errorMessage(r: Extract<ApiResult<unknown>, { ok: false }>): string {
  if (r.status === 429) return `Trop de tentatives. Réessayez dans ${r.retryAfter ?? 60} s.`;
  // La double validation (auteur ≠ validateur) renvoie 403 avec un message explicite : on l'affiche tel quel.
  if (r.status === 403 && r.code === "SELF_ACTION_FORBIDDEN") return r.message ?? "Une autre personne doit effectuer cette action.";
  if (r.status === 403) return "Vous n'avez pas la permission d'effectuer cette action.";
  if (r.status === 503 || r.code === "UNREACHABLE") return "Service indisponible. Réessayez dans un instant.";
  const first = r.fields?.[0];
  // Validation : le détail par champ est plus utile que l'intitulé général (« attribut invalide »).
  if ((r.status === 422 || r.status === 400) && r.fields?.length) return r.fields.map((f) => f.message).filter(Boolean).join(" ");
  if (r.status === 422 || r.status === 409) return r.message ?? first?.message ?? "Action refusée.";
  return r.message ?? "Une erreur est survenue.";
}

export const can = (permissions: string[], needed: string) => permissions.includes("*") || permissions.includes(needed);
