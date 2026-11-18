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
export type Category = { id: string; slug: string; parentId: string | null; sortOrder: number; name: Localized; description: Localized | null; imageUrl?: string | null; depth: number; productCount: number };
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

// ── finance : comptabilité ──
export type FinSettings = { baseCurrency: string; fourEyesMinAmount: number };
export type AccountKind = "asset" | "liability" | "equity" | "income" | "expense" | "other";
export type FinAccount = { id: string; code: string; label: string; kind: AccountKind; active: boolean };
export type FinAccountPage = { items: FinAccount[]; page: PageInfo };
export type FinJournal = { id: string; code: string; label: string; kind: "sales" | "purchases" | "bank" | "cash" | "general"; active: boolean };
export type FinPeriod = { id: string; fiscalYearId: string; number: number; startDate: string; endDate: string; status: "open" | "closed"; closedAt: string | null; closedBy: string | null };
export type FinYear = { id: string; label: string; startDate: string; endDate: string; status: "open" | "partially_closed" | "closed"; periods: FinPeriod[] };
export type FinApproval = { id: string; type: "reopen_period"; subjectId: string; subjectLabel: string; reason: string; status: "pending" | "approved" | "rejected"; requestedBy: string; requestedAt: string; decidedBy: string | null; decidedAt: string | null; decisionNote: string | null };
export type FinApprovalPage = { items: FinApproval[]; page: PageInfo };
export type FinEntryLine = { id: string; accountId: string; accountCode: string; accountLabel: string; debit: number; credit: number; memo: string | null };
export type FinEntry = {
  id: string; journalId: string; journalCode: string; fiscalPeriodId: string; accountingDate: string; status: "draft" | "posted"; entryNumber: number | null; number: string | null;
  label: string; reference: string | null; currency: string; origin: "manual" | "operation" | "reversal"; source: { type: string; id: string; kind: string } | null;
  reversesEntryId: string | null; reversedByEntryId: string | null; reversalReason: string | null; lines: FinEntryLine[]; totalDebit: number; totalCredit: number;
  createdBy: string; createdAt: string; postedBy: string | null; postedAt: string | null;
};
export type FinEntryPage = { items: FinEntry[]; page: PageInfo };
export type TrialRow = { accountId: string; code: string; label: string; kind: string; openingBalance: number; debit: number; credit: number; closingBalance: number };
export type TrialBalance = { from: string; to: string; currency: string; rows: TrialRow[]; totalDebit: number; totalCredit: number; balanced: boolean };
export type LedgerLine = { entryId: string; entryNumber: string; accountingDate: string; journalCode: string; label: string; reference: string | null; memo: string | null; debit: number; credit: number; balance: number };
export type GeneralLedger = { account: FinAccount; currency: string; openingBalance: number; closingBalance: number; items: LedgerLine[]; page: PageInfo };
export type ImportResult = { simulated: boolean; total: number; created: number; skipped: number; errors: { line: number; message: string }[] };

export const ACCOUNT_KIND_LABEL: Record<AccountKind, string> = { asset: "Actif", liability: "Passif", equity: "Capitaux propres", income: "Produit", expense: "Charge", other: "Autre" };
export const JOURNAL_KIND_LABEL: Record<FinJournal["kind"], string> = { sales: "Ventes", purchases: "Achats", bank: "Banque", cash: "Caisse", general: "Opérations diverses" };
export const YEAR_STATUS_LABEL = { open: "Ouvert", partially_closed: "Partiellement clos", closed: "Clos" } as const;

// ── finance : dépenses et revenus ──
export type FinThirdParty = { id: string; kind: "customer" | "supplier"; externalRef: string | null; displayName: string; active: boolean };
export type FinThirdPartyPage = { items: FinThirdParty[]; page: PageInfo };
export type FinCategory = { id: string; kind: "expense" | "income"; label: string; accountId: string; accountCode: string; accountLabel: string; nature: "fixed" | "variable" | null; active: boolean };
export type OpStatus = "draft" | "entry_draft" | "recorded" | "reversed";
export type FinOpPayment = { id: string; paymentDate: string; amount: number; accountId: string; accountCode: string; entryId: string; entryStatus: "draft" | "posted"; entryNumber: string | null };
export type FinOperation = {
  id: string; kind: "expense" | "income"; operationDate: string; dueDate: string | null; thirdPartyId: string | null; thirdPartyName: string | null; categoryId: string; categoryLabel: string; label: string;
  reference: string | null; amountNet: number; taxAmount: number; taxAccountId: string | null; amountGross: number; counterpartyAccountId: string; counterpartyAccountCode: string;
  settlementMode: "immediate" | "on_account"; nature: "fixed" | "variable" | null; currency: string; status: OpStatus; entryId: string | null; entryNumber: string | null; settledAmount: number; remaining: number;
  duplicateConfirmed: boolean; recurringTemplateId: string | null; attachmentCount: number; payments: FinOpPayment[]; createdBy: string; createdAt: string;
};
export type FinOperationPage = { items: FinOperation[]; page: PageInfo; totalNet: number; totalTax: number; totalGross: number; currency: string };
export type FinRecurring = {
  id: string; kind: "expense" | "income"; categoryId: string; categoryLabel: string; thirdPartyId: string | null; label: string; amountNet: number; taxAmount: number; taxAccountId: string | null;
  counterpartyAccountId: string; journalId: string; settlementMode: "immediate" | "on_account"; nature: "fixed" | "variable" | null; dayOfMonth: number; startDate: string; endDate: string | null; active: boolean; lastGeneratedMonth: string | null;
};
export type FinAttachment = { id: string; operationId: string; filename: string; contentType: string; size: number; sha256: string; createdAt: string };

export const OP_STATUS_LABEL: Record<OpStatus, string> = { draft: "Sans écriture", entry_draft: "Écriture en brouillon", recorded: "Comptabilisée", reversed: "Contre-passée" };
export const OP_STATUS_TONE: Record<OpStatus, "off" | "warn" | "ok" | "info"> = { draft: "off", entry_draft: "warn", recorded: "ok", reversed: "info" };
export const KIND_LABEL = { expense: "Dépense", income: "Revenu" } as const;
export const SETTLEMENT_LABEL = { immediate: "Réglée immédiatement", on_account: "En compte (à régler plus tard)" } as const;

// ── finance : trésorerie ──
export type FinBankAccount = {
  id: string; name: string; bankName: string; currency: string; accountId: string; accountCode: string; accountLabel: string; hasNumber: boolean; numberLast4: string | null;
  openingBalance: number; openingDate: string; active: boolean; ledgerBalance: number; statementBalance: number; lastStatementDate: string | null;
};
export type FinStatement = { id: string; bankAccountId: string; filename: string; periodStart: string; periodEnd: string; lineCount: number; insertedCount: number; skippedCount: number; closingBalance: number | null; importedBy: string; importedAt: string };
export type ReconMatch = { id: string; entryLineId: string; entryId: string; entryNumber: string; amount: number; matchedAt: string; matchedBy: string };
export type BankLine = { id: string; statementId: string; bankAccountId: string; lineDate: string; label: string; reference: string | null; amount: number; matched: number; remaining: number; status: "unmatched" | "partial" | "matched"; matches: ReconMatch[] };
export type BankLinePage = { items: BankLine[]; page: PageInfo };
export type LedgerCandidate = { entryLineId: string; entryId: string; entryNumber: string; accountingDate: string; label: string; reference: string | null; amount: number; remaining: number; score: number; exactAmount: boolean };
export type LedgerLinePage = { items: LedgerCandidate[]; page: PageInfo };
export type ReconSummary = { bankAccountId: string; currency: string; ledgerBalance: number; statementBalance: number; difference: number; unmatchedStatement: number; unmatchedLedger: number; unmatchedStatementCount: number; unmatchedLedgerCount: number; openingGap: number; unexplained: number; lastStatementDate: string | null };
export type ForecastBucket = { startDate: string; endDate: string; inflows: number; outflows: number; projectedBalance: number };
export type CashForecast = { currency: string; asOf: string; horizonDays: number; openingBalance: number; buckets: ForecastBucket[]; overdueInflows: number; overdueOutflows: number; undatedInflows: number; undatedOutflows: number; excludedDrafts: number; excludedDraftsAmount: number; bankAccounts: number; assumptions: string[] };
export type StatementImportResult = { simulated: boolean; total: number; inserted: number; skipped: number; errors: { line: number; message: string }[]; periodStart: string | null; periodEnd: string | null; sumAmount: number; statement: FinStatement | null };

export const LINE_STATUS_LABEL = { unmatched: "Non rapprochée", partial: "Partiellement", matched: "Rapprochée" } as const;
export const LINE_STATUS_TONE = { unmatched: "warn", partial: "info", matched: "ok" } as const;

// ── moteur de recherche : sources externes ──
export type SourceKind = "web_search_api" | "manufacturer_api" | "distributor_feed";
export type Source = {
  id: string; kind: SourceKind; name: string; baseUrl: string; enabled: boolean; suspended: boolean; priority: number;
  timeoutMs: number; maxRetries: number; secretRef: string; costPerCallMicros: number; currency: string;
  allowsRawStorage: boolean; termsCheckedAt: string | null; maxCacheTtlSeconds: number | null; createdAt: string;
};
export type SourcePage = { items: Source[]; page: PageInfo };
export type SourceQuota = {
  sourceId: string; currency: string; monthlyBudgetMicros: number; dailyCapMicros: number; alertThresholdPct: number; fallbackSourceId: string | null;
  monthSpentMicros: number; monthReservedMicros: number; daySpentMicros: number; dayReservedMicros: number; alert: boolean;
};
export type SourceHealth = {
  sourceId: string; status: "healthy" | "degraded" | "down" | "suspended" | "disabled"; circuitOpen: boolean; p50Ms: number; p95Ms: number;
  errorRate: number; calls24h: number; lastSuccessAt: string | null; lastErrorAt: string | null; lastErrorKind: string | null; lastSyncAt: string | null;
};
export type SourceTestResult = { ok: boolean; durationMs: number; resultCount: number; costMicros: number; currency: string; errorKind: string | null; sample: string[] };
export type SourceCostLine = { sourceId: string; sourceName: string; currency: string; calls: number; cachedCalls: number; failedCalls: number; costMicros: number; costPerSearchMicros: number };
export type SourceCostReport = { from: string; to: string; lines: SourceCostLine[]; note: string };
