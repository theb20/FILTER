"use server";

import { redirect } from "next/navigation";
import { adminAct, errorMessage } from "@/lib/admin";
import { clearSession } from "@/lib/session";
import type { FormState } from "@/components/ui/client";

export async function changePassword(_: FormState, form: FormData): Promise<FormState> {
  const currentPassword = String(form.get("currentPassword") ?? "");
  const newPassword = String(form.get("newPassword") ?? "");
  if (!currentPassword || !newPassword) return { error: "Renseignez les deux mots de passe." };
  if (newPassword !== String(form.get("confirm") ?? "")) return { error: "La confirmation ne correspond pas au nouveau mot de passe." };
  const r = await adminAct("POST", "/admin/me/password", { body: { currentPassword, newPassword } });
  if (!r.ok) return { error: r.status === 401 ? "Mot de passe actuel incorrect." : errorMessage(r) };
  return { ok: "Mot de passe modifié. Vos autres sessions ont été fermées." };
}

export async function logoutEverywhere(): Promise<FormState> {
  const r = await adminAct("POST", "/admin/auth/logout-all");
  if (!r.ok) return { error: errorMessage(r) };
  await clearSession();
  redirect("/login");
}
