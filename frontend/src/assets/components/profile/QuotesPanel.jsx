import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Bookmark, FileText, Trash2 } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";
import { useNotify } from "../../configurations/notificationContext.js";
import { useAuth } from "../../store/authStore.js";
import QuoteDocument from "../QuoteDocument.jsx";
import OrderTracker from "../tracker/OrderTracker.jsx";
import { Box, Clipboard, House, Truck } from "../tracker/Illustrations.jsx";
import { btnDark, btnPrimary, card, cardTitle, fmtDate } from "./ui.js";

/** Brouillon de demande de devis (un par compte) : undefined = chargement, null = aucun. */
export function useQuoteDraft() {
  const call = useAuth((s) => s.call);
  const [draft, setDraft] = useState(undefined);
  useEffect(() => { call("/me/quote-draft").then((r) => setDraft(r.ok ? r.data : null)); }, [call]);
  return [draft, setDraft];
}

/** Demandes de devis : undefined = chargement, "unverified" = e-mail non confirmé. */
export function useQuoteRequests() {
  const call = useAuth((s) => s.call);
  const [requests, setRequests] = useState(undefined);
  useEffect(() => { call("/me/quote-requests").then((r) => setRequests(r.ok ? (r.data?.items ?? []) : r.status === 403 ? "unverified" : [])); }, [call]);
  return requests;
}

// Onglet « Devis » : brouillon enregistré et suivi des demandes (statuts réels : new → in_progress → quoted → closed).
export default function QuotesPanel({ user, fullName }) {
  const { t, lang } = useI18n();
  const notify = useNotify();
  const call = useAuth((s) => s.call);
  const [draft, setDraft] = useQuoteDraft();
  const requests = useQuoteRequests();
  const [openQuote, setOpenQuote] = useState(null);
  const steps = [
    { key: "new", label: t("profile.track.new"), Icon: Clipboard },
    { key: "in_progress", label: t("profile.track.in_progress"), Icon: Box },
    { key: "quoted", label: t("profile.track.quoted"), Icon: Truck },
    { key: "closed", label: t("profile.track.closed"), Icon: House },
  ];

  const removeDraft = async () => {
    const r = await call("/me/quote-draft", { method: "DELETE" });
    if (r.ok) { setDraft(null); notify(t("profile.drafts.deleted"), "success"); }
  };

  return (
    <div className="space-y-4">
      <section id="devis" className={card} aria-labelledby="drafts-title">
        <h2 id="drafts-title" className={cardTitle}><Bookmark size={20} aria-hidden="true" /> {t("profile.drafts.title")}</h2>
        {draft === undefined && <p className="mt-3 text-gray-500" role="status">{t("loading")}</p>}
        {draft === null && (
          <div className="mt-3">
            <p className="text-gray-700">{t("profile.drafts.none")}</p>
            <Link to="/quote" className={`${btnPrimary} mt-4`}>{t("profile.drafts.new")}</Link>
          </div>
        )}
        {draft && (
          <div className="mt-4 flex flex-wrap items-center justify-between gap-4 rounded-[10px] border border-gray-200 p-4">
            <div className="min-w-0">
              <p className="truncate text-[17px] font-semibold">{draft.subject || draft.company || t("profile.drafts.untitled")}</p>
              <p className="mt-0.5 text-sm text-gray-600">
                {t("profile.drafts.lines", { n: draft.items?.length ?? 0 })}
                {draft.updatedAt && <> · {t("profile.drafts.saved", { date: fmtDate(lang, draft.updatedAt, { dateStyle: "long", timeStyle: "short" }) })}</>}
              </p>
            </div>
            <div className="flex gap-2.5">
              <Link to="/quote" className={btnDark}>{t("profile.drafts.resume")}</Link>
              <button type="button" onClick={removeDraft} className="inline-flex items-center gap-1.5 rounded-[8px] border border-gray-300 px-4 py-2 font-medium hover:bg-red-50 hover:text-red-700"><Trash2 size={15} aria-hidden="true" /> {t("profile.drafts.delete")}</button>
            </div>
          </div>
        )}
      </section>

      <section id="demandes" className={card} aria-labelledby="requests-title">
        <h2 id="requests-title" className={cardTitle}><FileText size={20} aria-hidden="true" /> {t("profile.track.title")}</h2>
        {requests === undefined && <p className="mt-3 text-gray-500" role="status">{t("loading")}</p>}
        {requests === "unverified" && <p className="mt-3 text-gray-700">{t("profile.track.unverified")}</p>}
        {Array.isArray(requests) && requests.length === 0 && <p className="mt-3 text-gray-700">{t("profile.track.none")}</p>}
        {Array.isArray(requests) && requests.length > 0 && (
          <ul className="mt-4 space-y-5">
            {requests.map((q) => (
              <li key={q.id}>
                <OrderTracker
                  steps={steps}
                  current={Math.max(0, steps.findIndex((x) => x.key === q.status))}
                  tone={q.status === "closed" ? "green" : "blue"}
                  title={t("profile.track.request")}
                  titleAccent={q.reference ? `#${q.reference}` : q.subject || ""}
                  meta={[[t("profile.track.sent"), fmtDate(lang, q.createdAt)], [t("profile.track.lines"), String(q.itemCount ?? 0)], ...(q.updatedAt ? [[t("profile.track.updated"), fmtDate(lang, q.updatedAt)]] : [])]}
                />
                {q.quote && (
                  <div className="mt-3 rounded-[10px] border border-green-200 bg-green-50 p-4">
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <p className="font-semibold text-green-900">{t("profile.quote.received", { n: q.quote.number || "" })} · {new Intl.NumberFormat(lang).format(q.quote.total)} F CFA</p>
                      <div className="flex gap-2">
                        <button type="button" onClick={() => setOpenQuote(openQuote === q.id ? null : q.id)} aria-expanded={openQuote === q.id} className="rounded-[6px] bg-night px-4 py-2 text-sm font-medium text-white hover:opacity-90">{openQuote === q.id ? t("profile.quote.hide") : t("profile.quote.view")}</button>
                        {openQuote === q.id && <button type="button" onClick={() => window.print()} className="rounded-[6px] border border-gray-300 bg-white px-4 py-2 text-sm font-medium hover:bg-gray-50">{t("profile.quote.print")}</button>}
                      </div>
                    </div>
                    {openQuote === q.id && <div className="mt-4"><QuoteDocument quote={q.quote} request={q} customer={fullName || user?.email || ""} /></div>}
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
