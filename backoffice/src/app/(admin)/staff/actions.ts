"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import type { StaffUser } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

function fields(form: FormData) {
  return { email: String(form.get("email") ?? "").trim(), name: String(form.get("name") ?? "").trim(), roleIds: form.getAll("roleIds").map(String) };
}

export async function createStaff(_: FormState, form: FormData): Promise<FormState> {
  const body = fields(form);
  if (!body.email || !body.name) return { error: "Renseignez l'adresse e-mail et le nom." };
  if (body.roleIds.length === 0) return { error: "Choisissez au moins un rôle." };
  const r = await adminAct<StaffUser>("POST", "/admin/staff-users", { body });
  if (!r.ok) return { error: errorMessage(r) };
  redirect(`/staff/${r.data.id}?ok=invited`);
}

export async function updateStaff(id: string, _: FormState, form: FormData): Promise<FormState> {
  const body = fields(form);
  if (!body.name) return { error: "Le nom est obligatoire." };
  if (body.roleIds.length === 0) return { error: "Choisissez au moins un rôle." };
  const r = await adminAct<StaffUser>("PUT", `/admin/staff-users/${id}`, { body });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/staff");
  redirect(`/staff/${id}?ok=updated`);
}

function simple(path: (id: string) => string, ok: string) {
  return async (id: string): Promise<FormState> => {
    const r = await adminAct("POST", path(id));
    if (!r.ok) return { error: errorMessage(r) };
    revalidatePath("/staff");
    redirect(`/staff/${id}?ok=${ok}`);
  };
}
const deact = simple((id) => `/admin/staff-users/${id}/deactivate`, "deactivated");
const react = simple((id) => `/admin/staff-users/${id}/reactivate`, "reactivated");
const resend = simple((id) => `/admin/staff-users/${id}/resend-invitation`, "resent");
const reset2fa = simple((id) => `/admin/staff-users/${id}/reset-two-factor`, "reset2fa");

export async function deactivateStaff(id: string): Promise<FormState> { return deact(id); }
export async function reactivateStaff(id: string): Promise<FormState> { return react(id); }
export async function resendInvitation(id: string): Promise<FormState> { return resend(id); }
export async function resetTwoFactor(id: string): Promise<FormState> { return reset2fa(id); }
