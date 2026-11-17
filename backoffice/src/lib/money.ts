// Montants : ENTIERS en unités mineures de la devise (XOF : le franc ; EUR : le centime). Jamais de flottant.
const EXPONENT: Record<string, number> = { XOF: 0, XAF: 0, EUR: 2, USD: 2, GBP: 2, MAD: 2 };
export const exponent = (currency: string) => EXPONENT[currency] ?? 2;

/** Affiche un montant entier en unités mineures (« 125 000 F CFA », « 12,50 € »). Division par une puissance de 10 sur des chaînes, sans flottant. */
export function fmtMoney(minor: number | bigint, currency: string, opts: { sign?: boolean } = {}): string {
  const e = exponent(currency);
  const n = BigInt(minor);
  const neg = n < 0n;
  const abs = neg ? -n : n;
  const s = abs.toString().padStart(e + 1, "0");
  const whole = e === 0 ? s : s.slice(0, -e);
  const frac = e === 0 ? "" : s.slice(-e);
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, " ");
  const unit = currency === "XOF" || currency === "XAF" ? "F CFA" : currency === "EUR" ? "€" : currency;
  return `${neg ? "−" : opts.sign && n > 0n ? "+" : ""}${grouped}${frac ? `,${frac}` : ""} ${unit}`;
}

/** Saisie utilisateur → unités mineures. Accepte « 1 250,5 » ; refuse les excès de décimales. Renvoie null si invalide, 0n si vide. */
export function parseMinor(input: string, currency: string): bigint | null {
  const t = input.replace(/[\s  ]/g, "").replace(",", ".");
  if (t === "") return 0n;
  const e = exponent(currency);
  const m = /^(\d+)(?:\.(\d*))?$/.exec(t);
  if (!m) return null;
  const frac = m[2] ?? "";
  if (frac.length > e) return null;
  return BigInt(m[1] + frac.padEnd(e, "0"));
}

/** Valeur initiale d'un champ de saisie. */
export function toInput(minor: number | bigint, currency: string): string {
  const n = BigInt(minor);
  if (n === 0n) return "";
  const e = exponent(currency);
  if (e === 0) return n.toString();
  const s = n.toString().padStart(e + 1, "0");
  return `${s.slice(0, -e)},${s.slice(-e)}`;
}

/** Solde signé (débit − crédit) avec sa nature. */
export function fmtBalance(n: number, currency: string): string {
  if (n === 0) return fmtMoney(0, currency);
  return `${fmtMoney(Math.abs(n), currency)} ${n > 0 ? "D" : "C"}`;
}

export const today = () => new Date().toISOString().slice(0, 10);
export const dateFr = (iso: string) => new Intl.DateTimeFormat("fr-FR", { dateStyle: "medium" }).format(new Date(`${iso}T12:00:00Z`));
