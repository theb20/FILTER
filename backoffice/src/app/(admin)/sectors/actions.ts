"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { readLocalized } from "@/lib/catalog";
import type { Sector } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

function body(form: FormData) {
  return { slug: String(form.get("slug") ?? "").trim(), name: readLocalized(form, "name"), imageUrl: String(form.get("imageUrl") ?? "").trim() || null };
}

export async function createSector(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<Sector>("POST", "/admin/sectors", { body: body(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/sectors");
  redirect(`/sectors/${r.data.id}?ok=created`);
}

export async function updateSector(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<Sector>("PUT", `/admin/sectors/${id}`, { body: body(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/sectors");
  redirect(`/sectors/${id}?ok=updated`);
}

export async function deleteSector(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/sectors/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/sectors");
  redirect("/sectors?ok=deleted");
}
