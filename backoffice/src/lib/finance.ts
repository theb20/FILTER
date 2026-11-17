import "server-only";
import { adminGet } from "./admin";
import type { FinAccount, FinAccountPage, FinJournal, FinSettings, FinYear } from "./types";

/** Tous les comptes (la liste est paginée à 100). */
export async function allAccounts(returnTo: string, activeOnly = false): Promise<FinAccount[]> {
  const out: FinAccount[] = [];
  for (let page = 1; page <= 50; page++) {
    const r = await adminGet<FinAccountPage>("/admin/finance/accounts", returnTo, { query: { page, pageSize: 100, active: activeOnly ? "true" : undefined } });
    out.push(...r.items);
    if (page >= r.page.totalPages) break;
  }
  return out;
}

export const journals = async (returnTo: string, activeOnly = false) =>
  (await adminGet<{ items: FinJournal[] }>("/admin/finance/journals", returnTo, { query: { active: activeOnly ? "true" : undefined } })).items;

export const fiscalYears = async (returnTo: string) => (await adminGet<{ items: FinYear[] }>("/admin/finance/fiscal-years", returnTo)).items;

export const financeSettings = (returnTo: string) => adminGet<FinSettings>("/admin/finance/settings", returnTo);

/** Exercice contenant la date (ou, à défaut, le plus récent) : sert de plage par défaut aux états. */
export function defaultRange(years: FinYear[], today: string): { from: string; to: string } | null {
  const y = years.find((x) => x.startDate <= today && today <= x.endDate) ?? years[0];
  return y ? { from: y.startDate, to: y.endDate } : null;
}
