"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { formToBody } from "@/lib/content";
import { kindBySlug, settingsByKey } from "@/lib/content-kinds";
import type { FormState } from "@/components/ui/client";

export async function createItem(slug: string, _: FormState, form: FormData): Promise<FormState> {
  const k = kindBySlug(slug);
  if (!k) return { error: "Collection inconnue." };
  const r = await adminAct<{ id: string }>("POST", `/admin/${slug}`, { body: formToBody(k.fields, form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/content/${slug}`);
  redirect(`/content/${slug}/${r.data.id}?ok=created`);
}

export async function updateItem(slug: string, id: string, _: FormState, form: FormData): Promise<FormState> {
  const k = kindBySlug(slug);
  if (!k) return { error: "Collection inconnue." };
  const r = await adminAct("PUT", `/admin/${slug}/${id}`, { body: formToBody(k.fields, form, k.updateFields) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/content/${slug}`);
  redirect(`/content/${slug}/${id}?ok=updated`);
}

export async function deleteItem(slug: string, id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/${slug}/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/content/${slug}`);
  redirect(`/content/${slug}?ok=deleted`);
}

export async function saveSettings(key: string, _: FormState, form: FormData): Promise<FormState> {
  const s = settingsByKey(key);
  if (!s) return { error: "Réglage inconnu." };
  const r = await adminAct("PUT", `/admin/${s.slug}`, { body: formToBody(s.fields, form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/content/settings/${key}`);
  return { ok: "Réglages enregistrés." };
}

/** Publier ou dépublier : exige la permission « content:publish » (contrôlée par le gateway). */
export async function setPublished(slug: string, id: string, publish: boolean): Promise<FormState> {
  const r = await adminAct("POST", `/admin/${slug}/${id}/${publish ? "publish" : "unpublish"}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/content/${slug}`);
  redirect(`/content/${slug}/${id}?ok=${publish ? "published" : "unpublished"}`);
}
