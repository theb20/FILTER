"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import type { FormState } from "@/components/ui/client";

function done(id: string, ok: string): never {
  revalidatePath("/companies");
  revalidatePath("/dashboard");
  redirect(`/companies/${id}?ok=${ok}`);
}

export async function approveCompany(id: string): Promise<FormState> {
  const r = await adminAct("POST", `/admin/companies/${id}/approve`);
  if (!r.ok) return { error: r.code === "EMAIL_NOT_VERIFIED" ? "L'adresse e-mail du titulaire n'est pas encore vérifiée : l'entreprise ne peut pas être approuvée." : errorMessage(r) };
  done(id, "approved");
}

export async function rejectCompany(id: string, _: FormState, form: FormData): Promise<FormState> {
  const reason = String(form.get("reason") ?? "").trim();
  if (!reason) return { error: "Indiquez le motif du rejet." };
  if (reason.length > 500) return { error: "Le motif ne doit pas dépasser 500 caractères." };
  const r = await adminAct("POST", `/admin/companies/${id}/reject`, { body: { reason } });
  if (!r.ok) return { error: errorMessage(r) };
  done(id, "rejected");
}
