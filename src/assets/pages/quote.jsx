import { useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { useI18n } from "../i18n/i18nContext.js";
import { SITE } from "../config/site.js";
import Todo from "../components/Todo.jsx";

const field =
  "w-full rounded-[3px] border border-gray-300 bg-white px-4 py-2.5 outline-none focus:ring-2 focus:ring-action";

// Demande de devis (RFQ) : formulaire de démonstration, rien n'est transmis.
export default function Quote() {
  const { t } = useI18n();
  const [contact, setContact] = useState({ company: "", name: "", email: "", phone: "" });
  const [rows, setRows] = useState([{ ref: "", qty: 1 }]);
  const [notes, setNotes] = useState("");
  const [submitted, setSubmitted] = useState(false);

  useEffect(() => {
    document.title = t("meta.quote");
  }, [t]);

  const setRow = (i, patch) => setRows((r) => r.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));

  const onSubmit = (e) => {
    e.preventDefault();
    setSubmitted(true);
  };

  return (
    <div className="mx-2.5 mt-2.5 grid gap-2.5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <section className="rounded-[4px] bg-white p-6 shadow-card md:p-8" aria-labelledby="quote-title">
        <h1 id="quote-title" className="text-[40px] font-bold leading-tight">{t("quote.title")}</h1>
        <p className="mt-2">{t("quote.lead")}</p>

        {submitted ? (
          <div className="mt-6">
            <p className="rounded-[3px] bg-info p-3">{t("demo.notice")} {t("quote.notSent")}</p>
            <button type="button" onClick={() => setSubmitted(false)} className="mt-4 text-link underline">{t("contact.back")}</button>
          </div>
        ) : (
          <form onSubmit={onSubmit} className="mt-6 space-y-6">
            <p className="rounded-[3px] bg-info p-3 text-sm">{t("demo.notice")}</p>

            <fieldset className="grid gap-4 sm:grid-cols-2">
              <legend className="mb-2 text-[24px] font-semibold">{t("quote.contact")}</legend>
              {[
                ["company", "contact.companyName", "text"],
                ["name", "contact.fullName", "text"],
                ["email", "contact.workEmail", "email"],
                ["phone", "contact.phone", "tel"],
              ].map(([key, label, type]) => (
                <div key={key}>
                  <label htmlFor={`q-${key}`} className="mb-1 block text-sm font-medium">{t(label)}</label>
                  <input id={`q-${key}`} type={type} required={key !== "phone"} value={contact[key]} onChange={(e) => setContact((c) => ({ ...c, [key]: e.target.value }))} className={field} />
                </div>
              ))}
            </fieldset>

            <fieldset>
              <legend className="mb-2 text-[24px] font-semibold">{t("quote.items")}</legend>
              <ul className="space-y-3">
                {rows.map((row, i) => (
                  <li key={i} className="flex flex-wrap items-end gap-3">
                    <div className="min-w-[12rem] flex-1">
                      <label htmlFor={`q-ref-${i}`} className="mb-1 block text-sm font-medium">{t("quote.reference")}</label>
                      <input id={`q-ref-${i}`} required value={row.ref} onChange={(e) => setRow(i, { ref: e.target.value })} className={field} />
                    </div>
                    <div className="w-24">
                      <label htmlFor={`q-qty-${i}`} className="mb-1 block text-sm font-medium">{t("quote.qty")}</label>
                      <input id={`q-qty-${i}`} type="number" min="1" value={row.qty} onChange={(e) => setRow(i, { qty: Math.max(1, Number(e.target.value) || 1) })} className={field} />
                    </div>
                    {rows.length > 1 && (
                      <button type="button" aria-label={t("quote.removeRow", { n: i + 1 })} onClick={() => setRows((r) => r.filter((_, idx) => idx !== i))} className="pb-3 text-gray-500 hover:text-red-600">
                        <Trash2 size={18} />
                      </button>
                    )}
                  </li>
                ))}
              </ul>
              <button type="button" onClick={() => setRows((r) => [...r, { ref: "", qty: 1 }])} className="mt-3 inline-flex items-center gap-2 text-link hover:underline">
                <Plus size={16} aria-hidden="true" /> {t("quote.addRow")}
              </button>
            </fieldset>

            <div>
              <label htmlFor="q-notes" className="mb-1 block text-sm font-medium">{t("quote.notes")}</label>
              <textarea id="q-notes" rows={4} value={notes} onChange={(e) => setNotes(e.target.value)} className={`${field} resize-y`} />
            </div>

            <button type="submit" className="inline-flex items-center gap-2 rounded-[3px] bg-action px-6 py-2.5 font-bold text-black hover:bg-action-dark">
              {t("quote.submit")} <span aria-hidden="true">→</span>
            </button>
          </form>
        )}
      </section>

      <aside className="h-fit rounded-[4px] bg-white p-6 shadow-card" aria-label={t("contact.detailsAria")}>
        <h2 className="text-[24px] font-semibold">{t("contact.details")}</h2>
        <dl className="mt-4 space-y-3">
          <div><dt className="text-sm text-gray-600">{t("contact.email")}</dt><dd>{SITE.email || <Todo>{t("todo.email")}</Todo>}</dd></div>
          <div><dt className="text-sm text-gray-600">{t("contact.phone")}</dt><dd>{SITE.phone || <Todo>{t("todo.phone")}</Todo>}</dd></div>
        </dl>
      </aside>
    </div>
  );
}
