"use server";

import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import type { FormState } from "@/components/ui/client";

const num = (form: FormData, k: string) => Number(String(form.get(k) ?? "").trim());
const optNum = (form: FormData, k: string) => {
  const v = String(form.get(k) ?? "").trim();
  return v === "" ? null : Number(v);
};

export async function saveIdentitySettings(_: FormState, form: FormData): Promise<FormState> {
  const body = {
    accessTokenMinutes: num(form, "accessTokenMinutes"),
    customerRefreshDays: num(form, "customerRefreshDays"),
    staffRefreshHours: num(form, "staffRefreshHours"),
    passwordResetCodeMinutes: num(form, "passwordResetCodeMinutes"),
    passwordResetMaxAttempts: num(form, "passwordResetMaxAttempts"),
    staffChallengeMinutes: num(form, "staffChallengeMinutes"),
    invitationHours: num(form, "invitationHours"),
    emailVerificationHours: num(form, "emailVerificationHours"),
    emailChangeHours: num(form, "emailChangeHours"),
    recentLoginMinutes: num(form, "recentLoginMinutes"),
    loginAlertAfterFailures: num(form, "loginAlertAfterFailures"),
    loginThrottle: { freeAttempts: num(form, "freeAttempts"), baseDelaySeconds: num(form, "baseDelaySeconds"), maxDelaySeconds: num(form, "maxDelaySeconds") },
    retention: { personalDataDays: optNum(form, "personalDataDays"), auditLogDays: optNum(form, "auditLogDays") },
  };
  if (Object.values(body).some((v) => typeof v === "number" && Number.isNaN(v))) return { error: "Toutes les valeurs doivent être des nombres entiers." };
  const r = await adminAct("PUT", "/admin/identity-settings", { body });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/settings");
  return { ok: "Réglages enregistrés. Ils s'appliquent aux nouvelles sessions (jusqu'à 30 secondes de délai selon l'instance)." };
}
