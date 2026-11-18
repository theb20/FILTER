"use server";

import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { allCategories } from "@/lib/catalog";
import { localized, optLocalized, parseFulfillment, parsePriceMode, text, toInt, norm, type BulkResult, type BulkRow } from "@/lib/bulk-shared";
import type { Brand, BrandPage, Product } from "@/lib/types";

async function allBrands(): Promise<Brand[]> {
  const out: Brand[] = [];
  for (let page = 1; page <= 20; page++) {
    const r = await adminAct<BrandPage>("GET", "/admin/brands", { query: { page, pageSize: 100 } });
    if (!r.ok) break;
    out.push(...r.data.items);
    if (page >= r.data.page.totalPages) break;
  }
  return out;
}

/** Enregistre en série les lignes du tableur produits : suppressions, modifications (fiche relue puis fusionnée) et créations. */
export async function saveProducts(rows: BulkRow[]): Promise<BulkResult[]> {
  const [cats, brands] = await Promise.all([allCategories("/products/bulk"), allBrands()]);
  const catBySlug = new Map(cats.map((c) => [norm(c.slug), c.id]));
  const brandByKey = new Map<string, string>(brands.flatMap((b) => [[norm(b.slug), b.id], [norm(b.name), b.id]] as [string, string][]));
  const out = new Map<string, BulkResult>();
  const fail = (r: BulkRow, error: string) => out.set(r._k, { k: r._k, ok: false, error });

  /** Valeurs de la ligne ; `error` si une cellule est invalide. Les cellules de la grille remplacent celles de la fiche. */
  const fields = (r: BulkRow) => {
    const price = toInt(r.priceAmount), lead = toInt(r.leadTimeDays), min = toInt(r.minOrderQuantity);
    const priceMode = parsePriceMode(r.priceMode), fulfillmentMode = parseFulfillment(r.fulfillmentMode);
    const categoryId = catBySlug.get(norm(r.category));
    const brandKey = norm(r.brand);
    const error =
      !text(r.nameFr) ? "Le nom en français est obligatoire."
      : !categoryId ? `Catégorie « ${text(r.category)} » introuvable (utilisez son identifiant d'URL).`
      : brandKey && !brandByKey.has(brandKey) ? `Marque « ${text(r.brand)} » introuvable.`
      : !priceMode ? "Mode de prix : « Fixe » ou « Sur devis »."
      : !fulfillmentMode ? "Mode de commande : « Stock » ou « Sur commande »."
      : [price, lead, min].some(Number.isNaN) ? "Prix, délai et quantité minimale : des nombres entiers positifs."
      : null;
    return {
      error,
      value: {
        sku: text(r.sku) || null, name: localized(r.nameFr, r.nameEn, r.nameEs), description: optLocalized(r.descFr, r.descEn, r.descEs),
        categoryId, brandId: brandKey ? brandByKey.get(brandKey) ?? null : null, manufacturerRef: text(r.manufacturerRef) || null,
        priceMode, priceAmount: priceMode === "on_quote" ? null : price, fulfillmentMode, leadTimeDays: lead, minOrderQuantity: min,
        saleUnit: text(r.saleUnit) || null, gtin: text(r.gtin) || null,
      },
    };
  };

  for (const r of rows.filter((x) => x.del && x._id)) {
    const res = await adminAct("DELETE", `/admin/products/${r._id}`);
    if (res.ok) out.set(r._k, { k: r._k, ok: true, deleted: true }); else fail(r, errorMessage(res));
  }

  for (const r of rows.filter((x) => !x.del)) {
    const f = fields(r);
    if (f.error) { fail(r, f.error); continue; }
    if (r._id) {
      const cur = await adminAct<Product>("GET", `/admin/products/${r._id}`);
      if (!cur.ok) { fail(r, errorMessage(cur)); continue; }
      const p = cur.data;
      const res = await adminAct<Product>("PUT", `/admin/products/${r._id}`, {
        body: {
          slug: p.slug, weightGrams: p.weightGrams, lengthMm: p.lengthMm, widthMm: p.widthMm, heightMm: p.heightMm, attributes: p.attributes, sectorIds: p.sectorIds,
          media: p.media.map((m) => ({ url: m.url, alt: m.alt })), crossReferences: p.crossReferences, version: p.version,
          ...f.value, sku: f.value.sku ?? p.sku,
        },
      });
      if (res.ok) out.set(r._k, { k: r._k, ok: true, id: r._id }); else fail(r, errorMessage(res));
    } else {
      const res = await adminAct<Product>("POST", "/admin/products", { body: { ...f.value, attributes: [], sectorIds: [], media: [], crossReferences: [], version: null } });
      if (res.ok) out.set(r._k, { k: r._k, ok: true, id: res.data.id }); else fail(r, errorMessage(res));
    }
  }

  revalidatePath("/products");
  return rows.map((r) => out.get(r._k) ?? { k: r._k, ok: false, error: "Ignorée." });
}
