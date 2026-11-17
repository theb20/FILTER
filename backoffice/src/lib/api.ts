import "server-only";

// Appels au gateway depuis le SERVEUR du backoffice uniquement : aucun jeton ne transite par du JavaScript navigateur.
export const API_URL = process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type FieldError = { code: string; message: string; field?: string };
export type ApiResult<T> = { ok: true; status: number; data: T } | { ok: false; status: number; code: string; message?: string; fields?: FieldError[]; retryAfter?: number };

export async function gateway<T>(path: string, opts: { method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE"; body?: unknown; token?: string; query?: Record<string, string | number | undefined> } = {}): Promise<ApiResult<T>> {
  let res: Response;
  try {
    const qs = opts.query ? Object.entries(opts.query).filter(([, v]) => v !== undefined && v !== "").map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`).join("&") : "";
    res = await fetch(`${API_URL}/v1${path}${qs ? `?${qs}` : ""}`, {
      method: opts.method ?? "POST",
      headers: { "Content-Type": "application/json", ...(opts.token ? { Authorization: `Bearer ${opts.token}` } : {}) },
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
      cache: "no-store",
    });
  } catch {
    return { ok: false, status: 503, code: "UNREACHABLE" };
  }
  if (res.status === 204) return { ok: true, status: 204, data: undefined as T };
  const json = await res.json().catch(() => null);
  if (res.ok) return { ok: true, status: res.status, data: json as T };
  const retry = Number(res.headers.get("Retry-After"));
  return { ok: false, status: res.status, code: json?.code ?? "ERROR", message: typeof json?.detail === "string" ? json.detail : undefined, fields: json?.errors, retryAfter: retry > 0 ? retry : undefined };
}
