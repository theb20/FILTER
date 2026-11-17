"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import type { Role } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

function body(form: FormData) {
  return { name: String(form.get("name") ?? "").trim(), description: String(form.get("description") ?? "").trim() || null, permissions: form.getAll("permissions").map(String) };
}

export async function createRole(_: FormState, form: FormData): Promise<FormState> {
  const b = body(form);
  if (!b.name) return { error: "Le nom est obligatoire." };
  const r = await adminAct<Role>("POST", "/admin/roles", { body: b });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/roles");
  redirect(`/roles/${r.data.id}?ok=created`);
}

export async function updateRole(id: string, _: FormState, form: FormData): Promise<FormState> {
  const b = body(form);
  if (!b.name) return { error: "Le nom est obligatoire." };
  const r = await adminAct<Role>("PUT", `/admin/roles/${id}`, { body: b });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/roles");
  redirect(`/roles/${id}?ok=updated`);
}

export async function deleteRole(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/roles/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/roles");
  redirect("/roles?ok=deleted");
}
