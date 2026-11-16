"use server";

import { redirect } from "next/navigation";
import { cookies } from "next/headers";
import { gateway } from "@/lib/api";
import { COOKIE, clearSession, getChallenge, getRefresh, setChallenge, setSession, type StaffTokens } from "@/lib/session";

export type FormState = { error?: string; email?: string } | undefined;

type LoginResult = { status: "authenticated" | "two_factor_required" | "two_factor_enrollment_required"; challengeToken: string | null };
type Enrollment = { secret: string; otpauthUri: string; recoveryCodes: string[]; challengeToken: string };

function fail(r: { status: number; code: string; retryAfter?: number }, fallback = "Une erreur est survenue."): FormState {
  if (r.status === 429) return { error: `Trop de tentatives. Réessayez dans ${r.retryAfter ?? 60} s.` };
  if (r.status === 503 || r.code === "UNREACHABLE") return { error: "Service indisponible. Réessayez dans un instant." };
  return { error: fallback };
}

export async function login(_: FormState, form: FormData): Promise<FormState> {
  const email = String(form.get("email") ?? "").trim();
  const password = String(form.get("password") ?? "");
  if (!email || !password) return { error: "Renseignez votre adresse e-mail et votre mot de passe.", email };
  const r = await gateway<LoginResult>("/admin/auth/login", { body: { email, password } });
  if (!r.ok) return { ...(r.status === 401 ? { error: "Identifiants invalides." } : fail(r)), email };
  if (!r.data.challengeToken) return { error: "Réponse inattendue du serveur.", email };
  await setChallenge(r.data.challengeToken);
  redirect(r.data.status === "two_factor_enrollment_required" ? "/login/activation" : "/login/verification");
}

export async function verify(_: FormState, form: FormData): Promise<FormState> {
  const token = await getChallenge();
  if (!token) redirect("/login");
  const code = String(form.get("code") ?? "").trim();
  if (!code) return { error: "Saisissez le code." };
  const r = await gateway<StaffTokens>("/admin/auth/2fa/verify", { token, body: { code } });
  if (r.ok) {
    await setSession(r.data);
    redirect("/dashboard");
  }
  if (r.status === 401) {
    await clearSession();
    redirect("/login?expired=1");
  }
  return r.status === 422 || r.status === 400 ? { error: "Code incorrect." } : fail(r);
}

/** Démarre l'inscription du 2FA : CONSOMME le jeton `enroll`. Le secret et les codes de secours ne sont montrés qu'une fois. */
export async function startEnroll() {
  const token = await getChallenge();
  if (!token) redirect("/login");
  const r = await gateway<Enrollment>("/admin/auth/2fa/enroll", { token });
  if (!r.ok) {
    await clearSession();
    redirect("/login?expired=1");
  }
  const c = await cookies();
  c.set(COOKIE.enroll, JSON.stringify({ secret: r.data.secret, uri: r.data.otpauthUri, codes: r.data.recoveryCodes }), { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "strict", path: "/login", maxAge: 300 });
  await setChallenge(r.data.challengeToken);
  redirect("/login/activation");
}

export async function confirmEnroll(_: FormState, form: FormData): Promise<FormState> {
  const token = await getChallenge();
  if (!token) redirect("/login");
  const code = String(form.get("code") ?? "").trim();
  if (!code) return { error: "Saisissez le code affiché par votre application." };
  const r = await gateway<StaffTokens>("/admin/auth/2fa/enroll/confirm", { token, body: { code } });
  if (r.ok) {
    await setSession(r.data);
    redirect("/dashboard");
  }
  if (r.status === 401) {
    await clearSession();
    redirect("/login?expired=1");
  }
  return r.status === 422 || r.status === 400 ? { error: "Code incorrect. Vérifiez l'heure de votre téléphone." } : fail(r);
}

export async function logout() {
  const refresh = await getRefresh();
  if (refresh) await gateway("/admin/auth/logout", { body: { refreshToken: refresh } });
  await clearSession();
  redirect("/login");
}
