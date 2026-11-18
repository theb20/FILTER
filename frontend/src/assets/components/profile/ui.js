// Styles partagés de l'espace client (profil) : cartes, champs, boutons.
export const card = "rounded-[10px] bg-white p-5 shadow-card md:p-6";
export const cardTitle = "flex items-center gap-2 text-[19px] font-semibold text-night";
export const label = "block text-sm font-medium text-gray-700";
export const input = "mt-1.5 w-full rounded-[8px] border border-gray-300 bg-white px-3 py-2.5 text-[15px] text-night outline-none transition focus:border-link focus:ring-2 focus:ring-link/20 disabled:bg-gray-50";
export const btnPrimary = "inline-flex items-center justify-center gap-2 rounded-[8px] bg-action px-4 py-2.5 text-[15px] font-bold text-black transition hover:bg-action-dark disabled:cursor-not-allowed disabled:opacity-60";
export const btnDark = "inline-flex items-center justify-center gap-2 rounded-[8px] bg-night px-4 py-2.5 text-[15px] font-medium text-white transition hover:opacity-90 disabled:opacity-60";
export const btnGhost = "inline-flex items-center justify-center gap-2 rounded-[8px] border border-gray-300 bg-white px-4 py-2.5 text-[15px] font-medium text-night transition hover:bg-gray-50 disabled:opacity-60";
export const btnDanger = "inline-flex items-center justify-center gap-2 rounded-[8px] border border-red-200 bg-white px-3.5 py-2 text-sm font-medium text-red-700 transition hover:bg-red-50 disabled:opacity-60";
export const badge = "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-semibold";

/** Date lisible dans la langue du site. @param {string} lang @param {string | null | undefined} iso */
export const fmtDate = (lang, iso, opts = { dateStyle: "long" }) => (iso ? new Intl.DateTimeFormat(lang, opts).format(new Date(iso)) : "");

/** « il y a 5 min », « hier »… @param {string} lang @param {string} iso */
export function fmtRelative(lang, iso) {
  const diff = (new Date(iso).getTime() - Date.now()) / 1000;
  const rtf = new Intl.RelativeTimeFormat(lang, { numeric: "auto" });
  const abs = Math.abs(diff);
  if (abs < 60) return rtf.format(Math.round(diff), "second");
  if (abs < 3600) return rtf.format(Math.round(diff / 60), "minute");
  if (abs < 86400) return rtf.format(Math.round(diff / 3600), "hour");
  if (abs < 30 * 86400) return rtf.format(Math.round(diff / 86400), "day");
  return fmtDate(lang, iso);
}
