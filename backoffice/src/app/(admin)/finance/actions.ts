"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { parseMinor } from "@/lib/money";
import type { FinAccount, FinApproval, FinEntry, FinJournal, FinPeriod, FinSettings, FinYear, ImportResult, StatementImportResult } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

const text = (f: FormData, k: string) => String(f.get(k) ?? "").trim();
const orNull = (f: FormData, k: string) => text(f, k) || null;

// ── écritures ──

type RawLine = { accountId: string; debit: string; credit: string; memo: string };

/** Les lignes arrivent en JSON (saisie) ; les montants sont convertis en ENTIERS d'unités mineures (jamais de flottant). */
function entryBody(form: FormData, currency: string): { value?: Record<string, unknown>; error?: string } {
  let raw: RawLine[];
  try {
    raw = JSON.parse(text(form, "lines")) as RawLine[];
  } catch {
    return { error: "Les lignes de l'écriture sont illisibles : rechargez la page." };
  }
  const lines: { accountId: string; debit: number; credit: number; memo: string | null }[] = [];
  for (const [i, l] of raw.entries()) {
    const d = parseMinor(l.debit ?? "", currency);
    const c = parseMinor(l.credit ?? "", currency);
    if (d === null || c === null) return { error: `Ligne ${i + 1} : montant invalide (nombre positif, sans trop de décimales pour ${currency}).` };
    if (d > BigInt(Number.MAX_SAFE_INTEGER) || c > BigInt(Number.MAX_SAFE_INTEGER)) return { error: `Ligne ${i + 1} : montant trop élevé.` };
    if (!l.accountId) return { error: `Ligne ${i + 1} : choisissez un compte.` };
    lines.push({ accountId: l.accountId, debit: Number(d), credit: Number(c), memo: l.memo?.trim() || null });
  }
  return { value: { journalId: text(form, "journalId"), accountingDate: text(form, "accountingDate"), label: text(form, "label"), reference: orNull(form, "reference"), lines } };
}

export async function createEntry(_: FormState, form: FormData): Promise<FormState> {
  const b = entryBody(form, text(form, "currency") || "XOF");
  if (b.error) return { error: b.error };
  const r = await adminAct<FinEntry>("POST", "/admin/finance/entries", { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/entries");
  redirect(`/finance/entries/${r.data.id}?ok=created`);
}

export async function updateEntry(id: string, _: FormState, form: FormData): Promise<FormState> {
  const b = entryBody(form, text(form, "currency") || "XOF");
  if (b.error) return { error: b.error };
  const r = await adminAct<FinEntry>("PUT", `/admin/finance/entries/${id}`, { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/entries");
  redirect(`/finance/entries/${id}?ok=updated`);
}

export async function postEntry(id: string): Promise<FormState> {
  const r = await adminAct<FinEntry>("POST", `/admin/finance/entries/${id}/post`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect(`/finance/entries/${id}?ok=posted`);
}

export async function deleteEntry(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/finance/entries/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect("/finance/entries?ok=deleted");
}

export async function reverseEntry(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinEntry>("POST", `/admin/finance/entries/${id}/reverse`, { body: { reason: text(form, "reason"), accountingDate: orNull(form, "accountingDate") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect(`/finance/entries/${r.data.id}?ok=reversal`);
}

// ── plan comptable ──

const accountBody = (f: FormData) => ({ code: text(f, "code"), label: text(f, "label"), kind: text(f, "kind"), active: f.get("active") === "on" });

export async function createAccount(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinAccount>("POST", "/admin/finance/accounts", { body: accountBody(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/accounts");
  redirect(`/finance/accounts/${r.data.id}?ok=created`);
}

export async function updateAccount(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinAccount>("PUT", `/admin/finance/accounts/${id}`, { body: accountBody(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/accounts");
  redirect(`/finance/accounts/${id}?ok=updated`);
}

/** Import CSV : le bouton « Simuler » ne modifie rien ; « Importer » écrit. Le rapport s'affiche dans le formulaire. */
export async function importAccounts(_: FormState, form: FormData): Promise<FormState> {
  const csv = text(form, "csv");
  if (!csv) return { error: "Collez le contenu du fichier (une ligne par compte : code ; libellé ; nature)." };
  const simulate = text(form, "mode") !== "import";
  const r = await adminAct<ImportResult>("POST", "/admin/finance/accounts/import", { body: { csv, simulate } });
  if (!r.ok) return { error: errorMessage(r) };
  const x = r.data;
  const detail = x.errors.length ? ` Erreurs : ${x.errors.slice(0, 8).map((e) => `ligne ${e.line} (${e.message})`).join(" ; ")}${x.errors.length > 8 ? " …" : ""}` : "";
  if (!x.simulated) revalidatePath("/finance/accounts");
  const head = x.simulated ? "Simulation (rien n'a été écrit)" : "Import terminé";
  return { ok: `${head} : ${x.total} ligne(s), ${x.created} compte(s) ${x.simulated ? "à créer" : "créé(s)"}, ${x.skipped} déjà existant(s) ignoré(s), ${x.errors.length} erreur(s).${detail}` };
}

// ── journaux ──

const journalBody = (f: FormData) => ({ code: text(f, "code"), label: text(f, "label"), kind: text(f, "kind"), active: f.get("active") === "on" });

export async function createJournal(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinJournal>("POST", "/admin/finance/journals", { body: journalBody(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/journals");
  redirect(`/finance/journals?ok=created`);
}

export async function updateJournal(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinJournal>("PUT", `/admin/finance/journals/${id}`, { body: journalBody(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/journals");
  redirect(`/finance/journals?ok=updated`);
}

// ── exercices et périodes ──

export async function createFiscalYear(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinYear>("POST", "/admin/finance/fiscal-years", { body: { label: text(form, "label"), startDate: text(form, "startDate"), endDate: text(form, "endDate") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/periods");
  redirect(`/finance/periods/${r.data.id}?ok=created`);
}

export async function closePeriod(yearId: string, periodId: string): Promise<FormState> {
  const r = await adminAct<FinPeriod>("POST", `/admin/finance/fiscal-periods/${periodId}/close`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect(`/finance/periods/${yearId}?ok=closed`);
}

export async function requestReopen(yearId: string, periodId: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinApproval>("POST", `/admin/finance/fiscal-periods/${periodId}/reopen-requests`, { body: { reason: text(form, "reason") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/approvals");
  redirect(`/finance/periods/${yearId}?ok=requested`);
}

export async function decideApproval(id: string, approve: boolean, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<FinApproval>("POST", `/admin/finance/approvals/${id}/${approve ? "approve" : "reject"}`, { body: { note: orNull(form, "note") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect(`/finance/approvals?ok=${approve ? "approved" : "rejected"}`);
}

// ── réglages ──

export async function putSettings(_: FormState, form: FormData): Promise<FormState> {
  const raw = text(form, "fourEyesMinAmount") || "0";
  const cur = text(form, "baseCurrency");
  const minor = parseMinor(raw, cur);
  if (minor === null || minor > BigInt(Number.MAX_SAFE_INTEGER)) return { error: "Seuil invalide : un nombre positif est attendu." };
  const r = await adminAct<FinSettings>("PUT", "/admin/finance/settings", { body: { baseCurrency: cur, fourEyesMinAmount: Number(minor) } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  return { ok: "Réglages enregistrés." };
}

// ── F2 : tiers, catégories, opérations, règlements, récurrences, pièces ──

export async function createThirdParty(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<{ id: string }>("POST", "/admin/finance/third-parties", { body: { kind: text(form, "kind"), displayName: text(form, "displayName"), externalRef: orNull(form, "externalRef") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/third-parties");
  redirect("/finance/third-parties?ok=created");
}

export async function updateThirdParty(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct("PUT", `/admin/finance/third-parties/${id}`, { body: { kind: text(form, "kind"), displayName: text(form, "displayName"), externalRef: orNull(form, "externalRef"), active: form.get("active") === "on" } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/third-parties");
  redirect("/finance/third-parties?ok=updated");
}

const catBody = (f: FormData) => ({ kind: text(f, "kind"), label: text(f, "label"), accountId: text(f, "accountId"), nature: orNull(f, "nature"), active: f.get("active") === "on" || f.get("active") === null });

export async function createCategory(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct("POST", "/admin/finance/operation-categories", { body: catBody(form) });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/categories");
  redirect("/finance/categories?ok=created");
}

export async function updateCategory(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct("PUT", `/admin/finance/operation-categories/${id}`, { body: { ...catBody(form), active: form.get("active") === "on" } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/categories");
  redirect("/finance/categories?ok=updated");
}

function amount(f: FormData, key: string, currency: string, label: string): { v?: number; error?: string } {
  const m = parseMinor(text(f, key), currency);
  if (m === null) return { error: `${label} : montant invalide (nombre positif, ${currency === "XOF" ? "sans décimale" : "décimales limitées"}).` };
  if (m > BigInt(Number.MAX_SAFE_INTEGER)) return { error: `${label} : montant trop élevé.` };
  return { v: Number(m) };
}

function operationBody(form: FormData): { value?: Record<string, unknown>; error?: string } {
  const cur = text(form, "currency") || "XOF";
  const net = amount(form, "amountNet", cur, "Montant hors taxes");
  const tax = amount(form, "taxAmount", cur, "Taxe");
  if (net.error || tax.error) return { error: net.error ?? tax.error };
  return {
    value: {
      kind: text(form, "kind"), operationDate: text(form, "operationDate"), dueDate: orNull(form, "dueDate"), thirdPartyId: orNull(form, "thirdPartyId"), categoryId: text(form, "categoryId"),
      label: text(form, "label"), reference: orNull(form, "reference"), amountNet: net.v, taxAmount: tax.v, taxAccountId: orNull(form, "taxAccountId"),
      counterpartyAccountId: text(form, "counterpartyAccountId"), settlementMode: text(form, "settlementMode"), nature: orNull(form, "nature"), duplicateConfirmed: form.get("duplicateConfirmed") === "on",
    },
  };
}

export async function createOperation(_: FormState, form: FormData): Promise<FormState> {
  const b = operationBody(form);
  if (b.error) return { error: b.error };
  const r = await adminAct<{ id: string }>("POST", "/admin/finance/operations", { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/operations");
  redirect(`/finance/operations/${r.data.id}?ok=created`);
}

export async function updateOperation(id: string, _: FormState, form: FormData): Promise<FormState> {
  const b = operationBody(form);
  if (b.error) return { error: b.error };
  const r = await adminAct("PUT", `/admin/finance/operations/${id}`, { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/operations");
  redirect(`/finance/operations/${id}?ok=updated`);
}

export async function deleteOperation(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/finance/operations/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/operations");
  redirect("/finance/operations?ok=deleted");
}

export async function generateEntry(id: string, _: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct("POST", `/admin/finance/operations/${id}/entry`, { body: { journalId: text(form, "journalId") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect(`/finance/operations/${id}?ok=entry`);
}

export async function cancelEntry(id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/finance/operations/${id}/entry`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect(`/finance/operations/${id}?ok=entry-cancelled`);
}

export async function recordPayment(id: string, currency: string, _: FormState, form: FormData): Promise<FormState> {
  const a = amount(form, "amount", currency, "Montant");
  if (a.error) return { error: a.error };
  const r = await adminAct("POST", `/admin/finance/operations/${id}/payments`, { body: { paymentDate: text(form, "paymentDate"), amount: a.v, accountId: text(form, "accountId"), journalId: text(form, "journalId") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance");
  redirect(`/finance/operations/${id}?ok=payment`);
}

const MAX_FILE = 5 * 1024 * 1024;

export async function uploadAttachment(id: string, _: FormState, form: FormData): Promise<FormState> {
  const file = form.get("file");
  if (!(file instanceof File) || file.size === 0) return { error: "Choisissez un fichier (PDF, PNG ou JPEG)." };
  if (file.size > MAX_FILE) return { error: "Fichier trop volumineux (5 Mo au maximum)." };
  const data = Buffer.from(await file.arrayBuffer()).toString("base64");
  const r = await adminAct("POST", `/admin/finance/operations/${id}/attachments`, { body: { filename: file.name, dataBase64: data } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/finance/operations/${id}`);
  redirect(`/finance/operations/${id}?ok=attached`);
}

export async function deleteAttachment(opId: string, id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/finance/attachments/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/finance/operations/${opId}`);
  redirect(`/finance/operations/${opId}?ok=detached`);
}

function recurringBody(form: FormData, cur: string): { value?: Record<string, unknown>; error?: string } {
  const net = amount(form, "amountNet", cur, "Montant hors taxes");
  const tax = amount(form, "taxAmount", cur, "Taxe");
  if (net.error || tax.error) return { error: net.error ?? tax.error };
  return {
    value: {
      kind: text(form, "kind"), categoryId: text(form, "categoryId"), thirdPartyId: orNull(form, "thirdPartyId"), label: text(form, "label"), amountNet: net.v, taxAmount: tax.v, taxAccountId: orNull(form, "taxAccountId"),
      counterpartyAccountId: text(form, "counterpartyAccountId"), journalId: text(form, "journalId"), settlementMode: text(form, "settlementMode"), nature: orNull(form, "nature"),
      dayOfMonth: Number.parseInt(text(form, "dayOfMonth") || "0", 10), startDate: text(form, "startDate"), endDate: orNull(form, "endDate"), active: form.get("active") === "on",
    },
  };
}

export async function createRecurring(_: FormState, form: FormData): Promise<FormState> {
  const b = recurringBody(form, text(form, "currency") || "XOF");
  if (b.error) return { error: b.error };
  const r = await adminAct("POST", "/admin/finance/recurring-templates", { body: { ...b.value, active: true } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/recurring");
  redirect("/finance/recurring?ok=created");
}

export async function updateRecurring(id: string, _: FormState, form: FormData): Promise<FormState> {
  const b = recurringBody(form, text(form, "currency") || "XOF");
  if (b.error) return { error: b.error };
  const r = await adminAct("PUT", `/admin/finance/recurring-templates/${id}`, { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/recurring");
  redirect("/finance/recurring?ok=updated");
}

export async function generateRecurring(_: FormState, form: FormData): Promise<FormState> {
  const r = await adminAct<{ created: number; skipped: number }>("POST", "/admin/finance/recurring-templates/generate", { body: { upTo: text(form, "upTo") } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/operations");
  return { ok: `${r.data.created} opération(s) créée(s) en brouillon, ${r.data.skipped} déjà existante(s). Aucune écriture n'a été générée : ouvrez chaque opération pour la comptabiliser.` };
}

// ── F3 : trésorerie ──

function bankBody(form: FormData, currency: string): { value?: Record<string, unknown>; error?: string } {
  const ob = parseSigned(text(form, "openingBalance"), currency);
  if (ob === null) return { error: "Solde d'ouverture : montant invalide." };
  return { value: { name: text(form, "name"), bankName: text(form, "bankName"), accountId: text(form, "accountId"), openingBalance: ob, openingDate: text(form, "openingDate"), active: form.get("active") === "on" || form.get("active") === null, number: orNull(form, "number") } };
}

/** Montant signé (le solde d'un relevé peut être négatif) en entier d'unités mineures, sans flottant. */
function parseSigned(raw: string, currency: string): number | null {
  const t = raw.trim();
  if (t === "") return 0;
  const neg = t.startsWith("-") || t.startsWith("−");
  const m = parseMinor(t.replace(/^[-−+]/, ""), currency);
  if (m === null || m > BigInt(Number.MAX_SAFE_INTEGER)) return null;
  return neg ? -Number(m) : Number(m);
}

export async function createBankAccount(_: FormState, form: FormData): Promise<FormState> {
  const b = bankBody(form, text(form, "currency") || "XOF");
  if (b.error) return { error: b.error };
  const r = await adminAct<{ id: string }>("POST", "/admin/finance/bank-accounts", { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/bank");
  redirect(`/finance/bank/${r.data.id}?ok=created`);
}

export async function updateBankAccount(id: string, _: FormState, form: FormData): Promise<FormState> {
  const b = bankBody(form, text(form, "currency") || "XOF");
  if (b.error) return { error: b.error };
  const r = await adminAct("PUT", `/admin/finance/bank-accounts/${id}`, { body: { ...b.value, active: form.get("active") === "on" } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/finance/bank");
  redirect(`/finance/bank/${id}?ok=updated`);
}

/** Import d'un relevé CSV : « Simuler » (par défaut) ne modifie rien ; « Importer » est tout ou rien. */
export async function importStatement(accountId: string, currency: string, _: FormState, form: FormData): Promise<FormState> {
  const file = form.get("file");
  let csv = text(form, "csv");
  let name = "releve.csv";
  if (file instanceof File && file.size > 0) {
    if (file.size > 800_000) return { error: "Fichier trop volumineux (800 Ko, 5000 lignes au maximum)." };
    csv = await file.text();
    name = file.name;
  }
  if (!csv.trim()) return { error: "Choisissez un fichier CSV ou collez son contenu (date ; libellé ; référence ; montant)." };
  const simulate = text(form, "mode") !== "import";
  const closingRaw = text(form, "closingBalance");
  const closing = closingRaw ? parseSigned(closingRaw, currency) : null;
  if (closingRaw && closing === null) return { error: "Solde de clôture : montant invalide." };
  const r = await adminAct<StatementImportResult>("POST", "/admin/finance/bank-statements/import", { body: { bankAccountId: accountId, filename: name, csv, closingBalance: closing, simulate } });
  if (!r.ok) {
    // l'import refusé (lignes en erreur) renvoie un message ; on y joint le détail via une simulation
    if (!simulate && r.status === 422) {
      const sim = await adminAct<StatementImportResult>("POST", "/admin/finance/bank-statements/import", { body: { bankAccountId: accountId, filename: name, csv, simulate: true } });
      if (sim.ok && sim.data.errors.length) return { error: `${errorMessage(r)} ${sim.data.errors.slice(0, 8).map((e) => `Ligne ${e.line} : ${e.message}.`).join(" ")}${sim.data.errors.length > 8 ? " …" : ""}` };
    }
    return { error: errorMessage(r) };
  }
  const x = r.data;
  const errs = x.errors.length ? ` Erreurs : ${x.errors.slice(0, 8).map((e) => `ligne ${e.line} (${e.message})`).join(" ; ")}${x.errors.length > 8 ? " …" : ""}` : "";
  if (!x.simulated) { revalidatePath(`/finance/bank/${accountId}`); revalidatePath("/finance/bank"); }
  return { ok: `${x.simulated ? "Simulation (rien n'a été écrit)" : "Relevé importé"} : ${x.total} ligne(s), ${x.inserted} nouvelle(s), ${x.skipped} déjà importée(s), ${x.errors.length} erreur(s).${errs}` };
}

export async function reconcile(accountId: string, bankLineId: string, entryLineId: string, _: FormState, form: FormData): Promise<FormState> {
  const cur = text(form, "currency") || "XOF";
  const raw = text(form, "amount");
  const m = raw ? parseMinor(raw, cur) : 0n;
  if (m === null) return { error: "Montant invalide." };
  const r = await adminAct("POST", "/admin/finance/reconciliations", { body: { bankLineId, entryLineId, amount: Number(m) } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/finance/bank/${accountId}`);
  redirect(`/finance/bank/${accountId}/reconcile?line=${bankLineId}&ok=matched`);
}

export async function unreconcile(accountId: string, bankLineId: string, id: string): Promise<FormState> {
  const r = await adminAct("DELETE", `/admin/finance/reconciliations/${id}`);
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/finance/bank/${accountId}`);
  redirect(`/finance/bank/${accountId}/reconcile?line=${bankLineId}&ok=unmatched`);
}
