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
