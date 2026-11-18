export const QUOTE_STATUS = { new: "Nouvelle", in_progress: "En cours", quoted: "Devis envoyé", closed: "Clôturée" } as const;
export const QUOTE_TONE = { new: "warn", in_progress: "info", quoted: "ok", closed: "off" } as const;
export type QuoteStatus = keyof typeof QUOTE_STATUS;
