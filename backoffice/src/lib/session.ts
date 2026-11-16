import "server-only";
import { cookies } from "next/headers";

export const COOKIE = { access: "ff_bo_access", refresh: "ff_bo_refresh", challenge: "ff_bo_challenge", enroll: "ff_bo_enroll" } as const;
const secure = process.env.NODE_ENV === "production";

export type StaffTokens = { accessToken: string; refreshToken: string; expiresIn: number };

const base = { httpOnly: true, secure, sameSite: "strict" as const, path: "/" };

export async function setSession(t: StaffTokens) {
  const c = await cookies();
  c.set(COOKIE.access, t.accessToken, { ...base, maxAge: t.expiresIn });
  c.set(COOKIE.refresh, t.refreshToken, { ...base, maxAge: 8 * 3600 }); // 8 h : réglage staffRefreshHours d'identity
  c.delete(COOKIE.challenge);
  c.delete({ name: COOKIE.enroll, path: "/login" });
}

export async function clearSession() {
  const c = await cookies();
  for (const n of [COOKIE.access, COOKIE.refresh, COOKIE.challenge]) c.delete(n);
  c.delete({ name: COOKIE.enroll, path: "/login" });
}

/** Jeton de défi 2FA : à usage unique, 5 minutes (durée de vie fixée par identity). */
export async function setChallenge(token: string) {
  (await cookies()).set(COOKIE.challenge, token, { ...base, maxAge: 300 });
}
export async function getChallenge() {
  return (await cookies()).get(COOKIE.challenge)?.value;
}

export async function getAccess() {
  return (await cookies()).get(COOKIE.access)?.value;
}
export async function getRefresh() {
  return (await cookies()).get(COOKIE.refresh)?.value;
}
