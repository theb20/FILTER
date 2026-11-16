// @ts-check
// Devises du site. Le XOF (franc CFA) est la devise de BASE : tous les prix du catalogue sont
// exprimés en XOF, les autres devises sont converties depuis le XOF pour l'affichage uniquement.

/**
 * @typedef {"XOF" | "EUR" | "USD"} CurrencyCode
 * @typedef {{ code: CurrencyCode, xofPerUnit: number, decimals: number, fictional: boolean }} Currency
 */

export const BASE_CURRENCY = /** @type {CurrencyCode} */ ("XOF");
export const DEFAULT_CURRENCY = BASE_CURRENCY;

/** @type {Record<CurrencyCode, Currency>} */
export const CURRENCIES = {
  XOF: { code: "XOF", xofPerUnit: 1, decimals: 0, fictional: false },
  // Parité fixe officielle : 1 EUR = 655,957 XOF.
  EUR: { code: "EUR", xofPerUnit: 655.957, decimals: 2, fictional: false },
  // TAUX FICTIF (démonstration) : à remplacer par un taux réel (API de change).
  USD: { code: "USD", xofPerUnit: 600, decimals: 2, fictional: true },
};

/** @param {string} code @returns {code is CurrencyCode} */
export const isCurrencyCode = (code) => Object.hasOwn(CURRENCIES, code);

/**
 * Convertit un montant en XOF vers la devise demandée (valeur non arrondie).
 * @param {number} amountXof
 * @param {CurrencyCode} code
 */
export function convertFromXof(amountXof, code) {
  return amountXof / CURRENCIES[code].xofPerUnit;
}

/**
 * Formate un montant exprimé en XOF dans la devise choisie.
 * XOF : sans décimale, suffixe « F CFA » (ex. « 77 000 F CFA »).
 * @param {number} amountXof
 * @param {CurrencyCode} code
 * @param {string} locale ex. "fr-FR"
 */
export function formatMoney(amountXof, code, locale) {
  const { decimals } = CURRENCIES[code];
  const value = convertFromXof(amountXof, code);
  if (code === "XOF") {
    const n = new Intl.NumberFormat(locale, { maximumFractionDigits: 0, useGrouping: true }).format(Math.round(value));
    return `${n} F CFA`;
  }
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency: code,
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(value);
}
