"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import type { Brand } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

function body(form: FormData) {
  return { slug: String(form.get("slug") ?? "").trim(), name: String(form.get("name") ?? "").trim(), logoUrl: String(form.get("logoUrl") ?? "").trim() || null };
}

export async function createBrand(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<Brand>("POST", "/admin/brands", { body: body(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/brands");
  redirect(`/brands/${r.data.id}?ok=created`);
}

export async function updateBrand(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<Brand>("PUT", `/admin/brands/${id}`, { body: body(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/brands");
  redirect(`/brands/${id}?ok=updated`);
}

export async function deleteBrand(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/brands/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/brands");
  redirect("/brands?ok=deleted");
}
