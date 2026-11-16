"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { readLocalized, readOptionalLocalized } from "@/lib/catalog";
import type { AttrValue, Product } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

const text = (f: FormData, k: string) => String(f.get(k) ?? "").trim();
const orNull = (f: FormData, k: string) => text(f, k) || null;

function intOrNull(f: FormData, k: string, label: string): { v: number | null; error?: string } {
  const raw = text(f, k);
  if (!raw) return { v: null };
  if (!/^\d+$/.test(raw)) return { v: null, error: `${label} : un nombre entier positif est attendu.` };
  return { v: Number(raw) };
}

/** Une ligne par média : `adresse ; texte alternatif` (le texte est facultatif). L'ordre des lignes est l'ordre d'affichage. */
function parseMedia(raw: string) {
  return raw.split("\n").map((l) => l.trim()).filter(Boolean).map((l) => {
    const [url, ...alt] = l.split(";");
    return { url: url.trim(), alt: alt.join(";").trim() || null };
  });
}

/** Une ligne par référence : `marque ; numéro ; type` avec type = oem | equivalent | supersedes. */
function parseCrossRefs(raw: string): { items: { brand: string; number: string; kind: string }[]; error?: string } {
  const items: { brand: string; number: string; kind: string }[] = [];
  for (const [i, line] of raw.split("\n").map((l) => l.trim()).entries()) {
    if (!line) continue;
    const [brand, number, kind] = line.split(";").map((p) => p.trim());
    if (!brand || !number || !["oem", "equivalent", "supersedes"].includes(kind ?? "")) return { items, error: `Références croisées, ligne ${i + 1} : « marque ; numéro ; oem | equivalent | supersedes ».` };
    items.push({ brand, number, kind });
  }
  return { items };
}

function parseAttributes(form: FormData): { items: AttrValue[]; error?: string } {
  const items: AttrValue[] = [];
  for (const [name, raw] of form.entries()) {
    if (!name.startsWith("attr_")) continue;
    const key = name.slice(5);
    const value = String(raw).trim();
    if (!value) continue;
    const type = text(form, `attrtype_${key}`);
    if (type === "number") {
      const n = Number(value.replace(",", "."));
      if (!Number.isFinite(n)) return { items, error: `« ${key} » : un nombre est attendu.` };
      items.push({ key, value: n });
    } else if (type === "boolean") items.push({ key, value: value === "true" });
    else items.push({ key, value });
  }
  return { items };
}

function body(form: FormData): { value?: Record<string, unknown>; error?: string } {
  const lead = intOrNull(form, "leadTimeDays", "Délai");
  const price = intOrNull(form, "priceAmount", "Prix");
  const minQty = intOrNull(form, "minOrderQuantity", "Quantité minimale");
  const weight = intOrNull(form, "weightGrams", "Poids");
  const len = intOrNull(form, "lengthMm", "Longueur");
  const wid = intOrNull(form, "widthMm", "Largeur");
  const hei = intOrNull(form, "heightMm", "Hauteur");
  const attrs = parseAttributes(form);
  const refs = parseCrossRefs(text(form, "crossReferences"));
  const error = [lead, price, minQty, weight, len, wid, hei].find((x) => x.error)?.error ?? attrs.error ?? refs.error;
  if (error) return { error };
  const version = intOrNull(form, "version", "Version").v;
  return {
    value: {
      sku: orNull(form, "sku"), slug: orNull(form, "slug"),
      name: readLocalized(form, "name"), description: readOptionalLocalized(form, "description"),
      brandId: orNull(form, "brandId"), categoryId: text(form, "categoryId"), manufacturerRef: orNull(form, "manufacturerRef"),
      fulfillmentMode: text(form, "fulfillmentMode") || "stock", leadTimeDays: lead.v,
      priceMode: text(form, "priceMode") || "fixed", priceAmount: text(form, "priceMode") === "on_quote" ? null : price.v,
      gtin: orNull(form, "gtin"), saleUnit: orNull(form, "saleUnit"), minOrderQuantity: minQty.v, weightGrams: weight.v, lengthMm: len.v, widthMm: wid.v, heightMm: hei.v,
      attributes: attrs.items, sectorIds: form.getAll("sectorIds").map(String),
      media: parseMedia(text(form, "media")), crossReferences: refs.items, version,
    },
  };
}

export async function createProduct(_: FormState, form: FormData): Promise<FormState> {
  const b = body(form);
  if (b.error) return { error: b.error };
  const r = await adminAct<Product>("POST", "/admin/products", { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/products");
  redirect(`/products/${r.data.id}?ok=created`);
}

export async function updateProduct(id: string, _: FormState, form: FormData): Promise<FormState> {
  const b = body(form);
  if (b.error) return { error: b.error };
  const r = await adminAct<Product>("PUT", `/admin/products/${id}`, { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/products");
  redirect(`/products/${id}?ok=updated`);
}

function lifecycle(path: string, ok: string) {
  return async (id: string): Promise<FormState> => {
    const r = await adminAct<Product>("POST", `/admin/products/${id}/${path}`);
    if (!r.ok) return { error: errorMessage(r) };
    revalidatePath("/products");
    redirect(`/products/${id}?ok=${ok}`);
  };
}

export const submitReview = async (id: string) => lifecycle("submit-review", "review")(id);
export const publishProduct = async (id: string) => lifecycle("publish", "published")(id);
export const archiveProduct = async (id: string) => lifecycle("archive", "archived")(id);
export const restoreProduct = async (id: string) => lifecycle("restore", "restored")(id);

export async function deleteProduct(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/products/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/products");
  redirect("/products?ok=deleted");
}
