export type StaffUser = { id: string; email: string; name: string; roleIds: string[]; status: "invited" | "active" | "deactivated"; twoFactorEnabled: boolean; lastLoginAt: string | null };
export type StaffPage = { items: StaffUser[]; page: { page: number; pageSize: number; total: number; totalPages: number } };
export type Role = { id: string; name: string; description: string | null; permissions: string[]; system: boolean };
export type Permission = { code: string; description: string; group: string };
export type AuditEntry = { id: string; at: string; actorType: "staff" | "customer" | "system"; actorId: string | null; action: string; entity: string; entityId: string | null; before: Record<string, unknown> | null; after: Record<string, unknown> | null; ip: string | null; traceId: string | null };
export type AuditPage = { items: AuditEntry[]; nextCursor: string | null };

export const STATUS_LABEL = { invited: "Invité", active: "Actif", deactivated: "Désactivé" } as const;
export const STATUS_TONE = { invited: "warn", active: "ok", deactivated: "off" } as const;
export const dateFr = (iso: string | null) => (iso ? new Intl.DateTimeFormat("fr-FR", { dateStyle: "medium", timeStyle: "short" }).format(new Date(iso)) : "—");

export type CompanyStatus = "pending" | "approved" | "rejected";
export type Company = { id: string; legalName: string; taxId: string; country: string; status: CompanyStatus; taxIdVerified: boolean; rejectionReason: string | null; decidedAt: string | null };
export type CompanyPage = { items: Company[]; page: { page: number; pageSize: number; total: number; totalPages: number } };
export type Customer = { id: string; email: string; kind: "individual" | "company"; firstName: string | null; lastName: string | null; phone: string | null; status: "active" | "suspended"; company: Company | null; orderCount: number; lifetimeValue: unknown; createdAt: string };
export type CustomerPage = { items: Customer[]; page: { page: number; pageSize: number; total: number; totalPages: number } };

export const COMPANY_LABEL = { pending: "En attente", approved: "Approuvée", rejected: "Rejetée" } as const;
export const COMPANY_TONE = { pending: "warn", approved: "ok", rejected: "off" } as const;
export const customerName = (c: Pick<Customer, "firstName" | "lastName" | "email">) => [c.firstName, c.lastName].filter(Boolean).join(" ") || c.email;

// ── catalogue (taxonomie) ──
export type Localized = Record<string, string>;
export type PageInfo = { page: number; pageSize: number; total: number; totalPages: number };
export type Category = { id: string; slug: string; parentId: string | null; sortOrder: number; name: Localized; description: Localized | null; depth: number; productCount: number };
export type CategoryPage = { items: Category[]; page: PageInfo };
export type AttributeOption = { value: string; label: Localized };
export type AttributeType = "text" | "number" | "boolean" | "enum";
export type AttributeDef = { id: string; categoryId: string; key: string; label: Localized; type: AttributeType; unit: string | null; filterable: boolean; required: boolean; options: AttributeOption[]; sortOrder: number };
export type AttributeDefPage = { items: AttributeDef[]; page: PageInfo };
export type Brand = { id: string; slug: string; name: string; logoUrl: string | null };
export type BrandPage = { items: Brand[]; page: PageInfo };
export type Sector = { id: string; slug: string; name: Localized; imageUrl: string | null };
export type SectorPage = { items: Sector[]; page: PageInfo };

export const ATTRIBUTE_TYPE_LABEL: Record<AttributeType, string> = { text: "Texte libre", number: "Nombre", boolean: "Oui / non", enum: "Liste de choix" };

// ── produits ──
export type ProductStatus = "draft" | "review" | "active" | "archived";
export type AttrValue = { key: string; value: string | number | boolean };
export type Product = {
  id: string; sku: string; slug: string; name: Localized; description: Localized | null; brandId: string | null; categoryId: string; manufacturerRef: string | null;
  status: ProductStatus; fulfillmentMode: "stock" | "made_to_order"; leadTimeDays: number | null; priceMode: "fixed" | "on_quote"; priceAmount: number | null; currency: string;
  gtin: string | null; saleUnit: string | null; minOrderQuantity: number | null; weightGrams: number | null; lengthMm: number | null; widthMm: number | null; heightMm: number | null;
  attributes: AttrValue[]; sectorIds: string[]; media: { id: string; url: string; alt: string | null; sortOrder: number }[]; crossReferences: { brand: string; number: string; kind: "oem" | "equivalent" | "supersedes" }[];
  origins?: Record<string, string>; version: number; publishedAt: string | null; createdAt: string; updatedAt: string;
};
export type ProductPage = { items: Product[]; page: PageInfo };
export type ProductVersion = { version: number; at: string; actorId: string | null; action: string; changes: Record<string, { before: unknown; after: unknown; origin: string }> };

export const PRODUCT_STATUS_LABEL: Record<ProductStatus, string> = { draft: "Brouillon", review: "À valider", active: "Publié", archived: "Archivé" };
export const PRODUCT_STATUS_TONE: Record<ProductStatus, "off" | "warn" | "ok" | "info"> = { draft: "off", review: "warn", active: "ok", archived: "info" };
export const ACTION_LABEL: Record<string, string> = { create: "Création", update: "Modification", submit_review: "Soumis à validation", publish: "Publication", archive: "Archivage", restore: "Restauration" };
export const FIELD_LABEL: Record<string, string> = {
  sku: "SKU", slug: "Adresse (slug)", name: "Nom", description: "Description", brandId: "Marque", categoryId: "Catégorie", manufacturerRef: "Référence constructeur", status: "Statut",
  fulfillmentMode: "Mode de commande", leadTimeDays: "Délai (jours)", priceMode: "Mode de prix", priceAmount: "Prix HT", gtin: "GTIN/EAN", saleUnit: "Unité de vente", minOrderQuantity: "Quantité minimale",
  weightGrams: "Poids (g)", lengthMm: "Longueur (mm)", widthMm: "Largeur (mm)", heightMm: "Hauteur (mm)", attributes: "Attributs", sectorIds: "Secteurs", media: "Médias", crossReferences: "Références croisées",
};
/** Montant en francs CFA (XOF n'a pas de décimale), hors taxes. */
export const xof = (n: number | null) => (n === null ? "—" : `${new Intl.NumberFormat("fr-FR").format(n)} F CFA HT`);
