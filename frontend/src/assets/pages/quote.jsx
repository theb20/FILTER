import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft, Bookmark, Plus, Send, Trash2 } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";
import { useSite } from "../store/siteStore.js";
import { useNotify } from "../configurations/notificationContext.js";
import { sendForm } from "../lib/forms.js";
import { useCartStore } from "../store/cartStore.js";
import { useProductsById } from "../lib/catalog.js";

const DRAFT_KEY = "ff_quote_draft_v1"; // brouillon gardé sur cet appareil (simple confort : jamais envoyé)
const EMPTY = { contact: { company: "", name: "", email: "", phone: "" }, rows: [{ ref: "", qty: 1 }], notes: "" };

const field = "w-full rounded-[8px] border border-gray-200 bg-gray-50/60 px-3.5 py-2.5 text-[15px] outline-none transition focus:border-gray-400 focus:bg-white focus:ring-2 focus:ring-action";
const label = "mb-1.5 block text-[13px] font-medium text-gray-700";
const Req = () => <span className="text-red-500"> *</span>;

function loadDraft() {
  try {
    const d = JSON.parse(localStorage.getItem(DRAFT_KEY) ?? "null");
    if (d && d.contact && Array.isArray(d.rows) && d.rows.length) return d;
  } catch {
    // brouillon illisible ou stockage indisponible : on repart de zéro
  }
  return null;
}

// Demande de devis : formulaire à gauche, aperçu du document en direct à droite. Envoyée à l'API ; les lignes du panier
// préremplissent la liste des références ; les prix sont communiqués dans le devis (aucun prix n'est demandé ici).
export default function Quote() {
  const { t, locale } = useI18n();
  const SITE = useSite();
  const notify = useNotify();
  const [initial] = useState(() => loadDraft() ?? EMPTY);
  const [contact, setContact] = useState(initial.contact);
  const [rows, setRows] = useState(initial.rows);
  const [notes, setNotes] = useState(initial.notes);
  const [submitted, setSubmitted] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");
  const [now] = useState(() => new Date());
  const cart = useCartStore((c) => c.items);
  const { products, ready } = useProductsById(cart.map((i) => i.id));
  const [prefilled, setPrefilled] = useState(Boolean(loadDraft()));

  useEffect(() => {
    document.title = t("meta.quote");
  }, [t]);

  // Les articles du panier deviennent les lignes de la demande (une fois, si aucun brouillon n'existe).
  useEffect(() => {
    if (prefilled || !ready || cart.length === 0) return;
    const lines = cart.filter((i) => products[i.id]).map((i) => ({ ref: products[i.id].reference, qty: i.quantity, productId: i.id }));
    if (lines.length) setRows(lines);
    setPrefilled(true);
  }, [ready, cart, products, prefilled]);

  const setRow = (i, patch) => setRows((r) => r.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));
  const date = new Intl.DateTimeFormat(locale, { dateStyle: "long" }).format(now);
  const lines = rows.filter((r) => r.ref.trim());

  const saveDraft = () => {
    try {
      localStorage.setItem(DRAFT_KEY, JSON.stringify({ contact, rows, notes }));
      notify(t("quote.draftSaved"), "success");
    } catch {
      notify(t("form.err.unavailable"), "error");
    }
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setSending(true);
    setError("");
    const r = await sendForm("/quote-requests", {
      name: contact.name, email: contact.email, company: contact.company, phone: contact.phone, message: notes,
      items: lines.map((x) => ({ reference: x.ref.trim(), quantity: x.qty, ...(x.productId ? { productId: x.productId } : {}) })),
    });
    setSending(false);
    if (!r.ok) return setError(t(`form.err.${r.error}`));
    try { localStorage.removeItem(DRAFT_KEY); } catch { /* sans conséquence */ }
    setSubmitted(true);
  };

  if (submitted) {
    return (
      <div className="mx-2.5 mt-2.5 rounded-[12px] bg-white p-10 text-center shadow-card">
        <h1 className="text-[32px] font-bold">{t("quote.title")}</h1>
        <p role="status" className="mx-auto mt-4 max-w-md rounded-[8px] bg-info p-3">{t("quote.sent")}</p>
        <Link to="/category" className="mt-6 inline-flex rounded-[8px] bg-night px-5 py-2.5 font-medium text-white hover:opacity-90">{t("cart.browse")} <span aria-hidden="true">&nbsp;→</span></Link>
      </div>
    );
  }

  return (
    <div className="mx-2.5 mt-2.5 rounded-[16px] bg-gray-100 p-3 sm:p-5">
      <Link to="/" className="inline-flex items-center gap-1.5 text-sm text-gray-600 hover:text-black"><ArrowLeft size={14} aria-hidden="true" /> {t("quote.back")}</Link>
      <div className="mt-3 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-[34px] font-bold leading-tight tracking-tight sm:text-[44px]">{t("quote.title")}</h1>
          <p className="mt-1 max-w-xl text-gray-600">{t("quote.lead")}</p>
        </div>
        <div className="flex gap-2.5">
          <button type="button" onClick={saveDraft} className="inline-flex items-center gap-2 rounded-[10px] border border-gray-200 bg-white px-5 py-3 font-medium hover:bg-gray-50">
            <Bookmark size={16} aria-hidden="true" /> {t("quote.draft")}
          </button>
          <button type="submit" form="quote-form" disabled={sending} className="inline-flex items-center gap-2 rounded-[10px] bg-night px-5 py-3 font-medium text-white hover:opacity-90 disabled:opacity-60">
            <Send size={16} aria-hidden="true" /> {sending ? t("form.sending") : t("quote.send")}
          </button>
        </div>
      </div>

      <div className="mt-5 grid gap-3 lg:grid-cols-2">
        {/* ── Formulaire ── */}
        <form id="quote-form" onSubmit={onSubmit} className="rounded-[12px] bg-white shadow-card" aria-labelledby="quote-details">
          <h2 id="quote-details" className="border-b border-gray-100 px-5 py-4 text-[15px] font-semibold">{t("quote.details")}</h2>
          <div className="space-y-5 p-5">
            <div className="grid gap-4 sm:grid-cols-2">
              {[["company", "contact.companyName", "text", true], ["name", "contact.fullName", "text", true], ["email", "contact.workEmail", "email", true], ["phone", "contact.phone", "tel", false]].map(([key, lbl, type, req]) => (
                <div key={key}>
                  <label htmlFor={`q-${key}`} className={label}>{t(lbl)}{req && <Req />}</label>
                  <input id={`q-${key}`} type={type} required={req} value={contact[key]} onChange={(e) => setContact((c) => ({ ...c, [key]: e.target.value }))} className={field} />
                </div>
              ))}
            </div>

            <fieldset>
              <legend className={label}>{t("quote.items")}<Req /></legend>
              <div className="overflow-hidden rounded-[10px] border border-gray-100">
                <div className="grid grid-cols-[1fr_88px_36px] gap-2 bg-gray-50 px-3 py-2 text-xs text-gray-500">
                  <span>{t("quote.reference")}</span><span>{t("quote.qtyShort")}</span><span />
                </div>
                <ul className="divide-y divide-gray-100">
                  {rows.map((row, i) => (
                    <li key={i} className="grid grid-cols-[1fr_88px_36px] items-center gap-2 px-3 py-2">
                      <input aria-label={`${t("quote.reference")} ${i + 1}`} required={i === 0} value={row.ref} onChange={(e) => setRow(i, { ref: e.target.value, productId: undefined })} className={field} />
                      <input aria-label={`${t("quote.qty")} ${i + 1}`} type="number" min="1" value={row.qty} onChange={(e) => setRow(i, { qty: Math.max(1, Number(e.target.value) || 1) })} className={field} />
                      {rows.length > 1 ? (
                        <button type="button" aria-label={t("quote.removeRow", { n: i + 1 })} onClick={() => setRows((r) => r.filter((_, idx) => idx !== i))} className="flex h-9 w-9 items-center justify-center rounded-full text-gray-400 hover:bg-red-50 hover:text-red-600"><Trash2 size={16} /></button>
                      ) : <span />}
                    </li>
                  ))}
                </ul>
              </div>
              <button type="button" onClick={() => setRows((r) => [...r, { ref: "", qty: 1 }])} className="mt-3 inline-flex items-center gap-1.5 text-sm font-medium text-gray-700 hover:text-black">
                <Plus size={14} aria-hidden="true" /> {t("quote.addRow")}
              </button>
            </fieldset>

            <div>
              <label htmlFor="q-notes" className={label}>{t("quote.notes")}</label>
              <textarea id="q-notes" rows={5} value={notes} onChange={(e) => setNotes(e.target.value)} className={`${field} resize-y`} />
            </div>
            {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
          </div>
        </form>

        {/* ── Aperçu du document ── */}
        <aside className="rounded-[12px] bg-white shadow-card" aria-label={t("quote.preview")}>
          <h2 className="border-b border-gray-100 px-5 py-4 text-[15px] font-semibold">{t("quote.preview")}</h2>
          <div className="p-4 sm:p-5">
            <div className="rounded-[10px] bg-gray-100 p-4 sm:p-6">
              <article className="rounded-[8px] bg-white p-5 text-[13px] shadow-sm sm:p-7">
                <header className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-2">
                    <img src="/img/logo/favicon.svg" alt="" className="h-9 w-9" />
                    <span className="text-[15px] font-bold leading-tight">{SITE.name}</span>
                  </div>
                  <p className="text-right text-[18px] font-bold leading-tight">{t("quote.docTitle")}</p>
                </header>

                <dl className="mt-6 grid grid-cols-2 gap-4 text-gray-700">
                  <div><dt className="text-[11px] text-gray-500">{t("quote.date")}</dt><dd className="font-medium text-black">{date}</dd></div>
                  <div><dt className="text-[11px] text-gray-500">{t("contact.email")}</dt><dd className="break-all font-medium text-black">{contact.email || "—"}</dd></div>
                </dl>

                <div className="mt-5 grid gap-4 sm:grid-cols-2">
                  <section>
                    <h3 className="text-[11px] text-gray-500">{t("quote.from")}</h3>
                    <p className="mt-0.5 font-semibold text-black">{contact.company || "—"}</p>
                    <p className="text-gray-600">{contact.name}</p>
                    {contact.phone && <p className="text-gray-600">+225 {contact.phone}</p>}
                  </section>
                  <section>
                    <h3 className="text-[11px] text-gray-500">{t("quote.to")}</h3>
                    <p className="mt-0.5 font-semibold text-black">{SITE.publisher || SITE.name}</p>
                    {SITE.address && <p className="text-gray-600">{SITE.address}</p>}
                    {SITE.email && <p className="text-gray-600">{SITE.email}</p>}
                  </section>
                </div>

                <table className="mt-6 w-full text-left">
                  <thead className="bg-gray-50 text-[11px] text-gray-500">
                    <tr><th className="px-2.5 py-2 font-medium">{t("quote.reference")}</th><th className="w-16 px-2.5 py-2 text-right font-medium">{t("quote.qtyShort")}</th></tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {lines.length === 0 && <tr><td colSpan={2} className="px-2.5 py-4 text-center text-gray-400">{t("quote.noItems")}</td></tr>}
                    {lines.map((l, i) => (
                      <tr key={i}><td className="break-all px-2.5 py-2">{l.ref}</td><td className="px-2.5 py-2 text-right">{l.qty}</td></tr>
                    ))}
                  </tbody>
                </table>

                <section className="mt-5">
                  <h3 className="text-[11px] text-gray-500">{t("quote.notes")}</h3>
                  <p className="mt-0.5 whitespace-pre-line break-words text-gray-700">{notes.trim() || <span className="text-gray-400">{t("quote.notesEmpty")}</span>}</p>
                </section>
                <p className="mt-5 border-t border-gray-100 pt-3 text-[11px] text-gray-500">{t("quote.dateHint")}</p>
              </article>
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}
