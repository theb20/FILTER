"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { readLocalized, readOptionalLocalized } from "@/lib/catalog";
import type { Category } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

function body(form: FormData) {
  return { slug: String(form.get("slug") ?? "").trim(), name: readLocalized(form, "name"), description: readOptionalLocalized(form, "description"), imageUrl: String(form.get("imageUrl") ?? "").trim() || null };
}

export async function createCategory(_: FormState, form: FormData): Promise<FormState> {
  const parent = String(form.get("parentId") ?? "") || null;
  const r = await adminAct<Category>("POST", "/admin/categories", { body: { ...body(form), parentId: parent } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/categories");
  redirect(`/categories/${r.data.id}?ok=created`);
}

export async function updateCategory(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<Category>("PUT", `/admin/categories/${id}`, { body: body(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/categories");
  redirect(`/categories/${id}?ok=updated`);
}

export async function moveCategory(id: string, _: FormState, form: FormData): Promise<FormState> {
  const parent = String(form.get("parentId") ?? "") || null;
  const position = Number.parseInt(String(form.get("position") ?? "0"), 10);
  if (!Number.isInteger(position) || position < 0) return { error: "La position doit être un nombre entier supérieur ou égal à 0." };
  const r = await adminAct<Category>("POST", `/admin/categories/${id}/move`, { body: { parentId: parent, position } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/categories");
  redirect(`/categories/${id}?ok=moved`);
}

export async function deleteCategory(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/categories/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/categories");
  redirect("/categories?ok=deleted");
}
