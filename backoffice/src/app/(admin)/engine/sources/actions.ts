"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import { toMicros } from "@/lib/engine";
import type { Source, SourceTestResult } from "@/lib/types";
import type { FormState } from "@/components/ui/client";

const text = (f: FormData, k: string) => String(f.get(k) ?? "").trim();

function intOf(f: FormData, k: string, label: string, min: number): { v: number; error?: string } {
  const raw = text(f, k);
  if (!/^\d+$/.test(raw) || Number(raw) < min) return { v: 0, error: `${label} : un nombre entier ≥ ${min} est attendu.` };
  return { v: Number(raw) };
}

/** Corps commun création / modification. Les montants sont saisis dans la devise et convertis en micro-unités. */
function body(f: FormData): { value?: Record<string, unknown>; error?: string } {
  const priority = intOf(f, "priority", "Priorité", 0);
  const timeoutMs = intOf(f, "timeoutMs", "Délai maximal", 1);
  const maxRetries = intOf(f, "maxRetries", "Tentatives", 0);
  const cost = toMicros(text(f, "costPerCall"));
  const currency = text(f, "currency").toUpperCase();
  const ttlRaw = text(f, "maxCacheTtlSeconds");
  const ttl = ttlRaw === "" ? null : Number(ttlRaw);
  const error =
    [priority, timeoutMs, maxRetries].find((x) => x.error)?.error
    ?? (!text(f, "name") ? "Le nom est obligatoire." : null)
    ?? (!/^https?:\/\//.test(text(f, "baseUrl")) ? "L'adresse de la source doit commencer par https://." : null)
    ?? (!text(f, "secretRef") ? "Le nom de la variable d'environnement qui porte la clé est obligatoire." : null)
    // Un secret ne doit jamais arriver jusqu'à la base : on refuse tout ce qui n'est pas un nom de variable.
    ?? (!/^[A-Z][A-Z0-9_]{2,63}$/.test(text(f, "secretRef"))
      ? "Ce champ attend le NOM de la variable d'environnement (ex. BRAVE_SEARCH_API_KEY), jamais la clé ni la ligne « NOM=valeur »."
      : null)
    ?? (Number.isNaN(cost) ? "Coût par appel : un nombre positif est attendu (ex. 0,005)." : null)
    ?? (!/^[A-Z]{3}$/.test(currency) ? "Devise : trois lettres (ex. USD)." : null)
    ?? (ttl !== null && (!Number.isInteger(ttl) || ttl < 1) ? "Durée de cache : un nombre entier de secondes, ou vide pour aucune mise en cache." : null);
  if (error) return { error };
  return {
    value: {
      kind: text(f, "kind"), name: text(f, "name"), baseUrl: text(f, "baseUrl"), priority: priority.v, timeoutMs: timeoutMs.v,
      maxRetries: maxRetries.v, secretRef: text(f, "secretRef"), costPerCallMicros: cost, currency, maxCacheTtlSeconds: ttl,
    },
  };
}

export async function createSource(_: FormState, f: FormData): Promise<FormState> {
  const b = body(f);
  if (b.error) return { error: b.error };
  const r = await adminAct<Source>("POST", "/admin/sources", { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/engine/sources");
  redirect(`/engine/sources/${r.data.id}?ok=created`);
}

export async function updateSource(id: string, _: FormState, f: FormData): Promise<FormState> {
  const b = body(f);
  if (b.error) return { error: b.error };
  const r = await adminAct<Source>("PUT", `/admin/sources/${id}`, { body: b.value });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/engine/sources");
  redirect(`/engine/sources/${id}?ok=updated`);
}

/** Budget : plafond journalier ≤ budget mensuel (contrôlé aussi par le service). */
export async function saveQuota(id: string, _: FormState, f: FormData): Promise<FormState> {
  const monthly = toMicros(text(f, "monthlyBudget"));
  const daily = toMicros(text(f, "dailyCap"));
  const pct = Number(text(f, "alertThresholdPct"));
  if (Number.isNaN(monthly) || Number.isNaN(daily)) return { error: "Budget et plafond : des nombres positifs sont attendus (ex. 5 ou 0,333)." };
  if (daily > monthly) return { error: "Le plafond journalier doit être inférieur ou égal au budget mensuel." };
  if (!Number.isInteger(pct) || pct < 1 || pct > 100) return { error: "Seuil d'alerte : un entier entre 1 et 100." };
  const fallback = text(f, "fallbackSourceId");
  const r = await adminAct("PUT", `/admin/sources/${id}/quota`, { body: { monthlyBudgetMicros: monthly, dailyCapMicros: daily, alertThresholdPct: pct, fallbackSourceId: fallback || null } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/engine/sources/${id}`);
  redirect(`/engine/sources/${id}?ok=quota`);
}

function flip(path: string, ok: string) {
  return async (id: string): Promise<FormState> => {
    const r = await adminAct("POST", `/admin/sources/${id}/${path}`);
    if (!r.ok) return { error: errorMessage(r) };
    revalidatePath("/engine/sources");
    redirect(`/engine/sources/${id}?ok=${ok}`);
  };
}
export const enableSource = async (id: string) => flip("enable", "enabled")(id);
export const disableSource = async (id: string) => flip("disable", "disabled")(id);

/** Stockage des résultats bruts : interdit tant que les conditions d'usage de la source ne l'autorisent pas par écrit. */
export async function setRawStorage(id: string, allowed: boolean): Promise<FormState> {
  const r = await adminAct("PUT", `/admin/sources/${id}/raw-storage`, { body: { allowed } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath(`/engine/sources/${id}`);
  redirect(`/engine/sources/${id}?ok=${allowed ? "raw-on" : "raw-off"}`);
}

/** Test réel : un appel facturé à la source, avec la clé lue dans l'environnement du service.
 *  Le pays reste facultatif : toutes les sources ne couvrent pas tous les pays (Brave refuse « CI » par exemple). */
export async function testSource(id: string, query: string, country = ""): Promise<{ result?: SourceTestResult; error?: string }> {
  const q = query.trim();
  if (!q) return { error: "Saisissez une requête de test." };
  const body: Record<string, string> = { query: q, language: "fr" };
  if (/^[A-Za-z]{2}$/.test(country)) body.country = country.toUpperCase();
  const r = await adminAct<SourceTestResult>("POST", `/admin/sources/${id}/test`, { body });
  if (!r.ok) return { error: errorMessage(r) };
  return { result: r.data };
}
