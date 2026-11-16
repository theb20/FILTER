"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { LANGS, readLocalized } from "@/lib/catalog";
import type { AttributeDef, AttributeOption } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

/** Une option par ligne : `valeur ; libellé fr ; libellé en ; libellé es` (en et es facultatifs). */
function parseOptions(raw: string): { options: AttributeOption[]; error?: string } {
  const options: AttributeOption[] = [];
  for (const [i, line] of raw.split("\n").map((l) => l.trim()).entries()) {
    if (!line) continue;
    const parts = line.split(";").map((p) => p.trim());
    const [value, ...labels] = parts;
    if (!value || !labels[0]) return { options, error: `Ligne ${i + 1} : « valeur ; libellé français » est obligatoire.` };
    const label: Record<string, string> = {};
    LANGS.forEach((l, k) => { if (labels[k]) label[l.code] = labels[k]; });
    options.push({ value, label });
  }
  return { options };
}

function body(form: FormData, withKey: boolean) {
  const type = String(form.get("type") ?? "text");
  const parsed = type === "enum" ? parseOptions(String(form.get("options") ?? "")) : { options: [] as AttributeOption[] };
  const unit = String(form.get("unit") ?? "").trim();
  return {
    error: parsed.error,
    value: {
      categoryId: String(form.get("categoryId") ?? ""),
      key: String(form.get("key") ?? "").trim(),
      label: readLocalized(form, "label"),
      type,
      unit: type === "number" && unit ? unit : null,
      filterable: form.get("filterable") === "on",
      required: form.get("required") === "on",
      options: parsed.options,
      sortOrder: Number.parseInt(String(form.get("sortOrder") ?? "0"), 10) || 0,
    },
    withKey,
  };
}

export async function createAttribute(_: FormState, form: FormData): Promise<FormState> {
  const b = body(form, true);
  if (b.error) return { error: b.error };
  const r = await adminAct<AttributeDef>("POST", "/admin/attribute-defs", { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/categories");
  redirect(`/categories/${b.value.categoryId}?ok=attr-created`);
}

export async function updateAttribute(id: string, _: FormState, form: FormData): Promise<FormState> {
  const b = body(form, false);
  if (b.error) return { error: b.error };
  const r = await adminAct<AttributeDef>("PUT", `/admin/attribute-defs/${id}`, { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/categories");
  redirect(`/categories/${b.value.categoryId}?ok=attr-updated`);
}

export async function deleteAttribute(id: string, categoryId: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/attribute-defs/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/categories");
  redirect(`/categories/${categoryId}?ok=attr-deleted`);
}
