import { useEffect, useState } from "react";
import { useSite } from "../store/siteStore.js";
import { useI18n } from "../i18n/i18nContext.js";
import Todo from "../components/Todo.jsx";
import { sendForm } from "../lib/forms.js";

const field =
  "w-full rounded-[3px] border border-gray-300 bg-white px-4 py-2.5 outline-none focus:ring-2 focus:ring-action";

const EMPTY = { name: "", email: "", phone: "", company: "", vat: "", message: "", file: null, consent: false };

export default function Contact() {
  const { t } = useI18n();
  const SITE = useSite();
  const [type, setType] = useState("company");
  const [data, setData] = useState(EMPTY);
  const [submitted, setSubmitted] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");
  const isCompany = type === "company";

  useEffect(() => {
    document.title = t("meta.contact");
  }, [t]);

  const onChange = (e) => {
    const { name, value, type: inputType, checked } = e.target;
    setData((d) => ({ ...d, [name]: inputType === "checkbox" ? checked : value }));
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setSending(true);
    setError("");
    const vat = isCompany && data.vat ? `${t("contact.vat")} : ${data.vat}\n\n` : "";
    const r = await sendForm("/contact", { name: data.name, email: data.email, phone: data.phone, company: isCompany ? data.company : "", message: `${vat}${data.message}` });
    setSending(false);
    if (!r.ok) return setError(t(`form.err.${r.error}`));
    setData(EMPTY);
    setSubmitted(true);
  };

  return (
    <div className="mx-2.5 mt-2.5 grid gap-2.5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <section className="rounded-[4px] bg-white p-6 shadow-card md:p-8" aria-labelledby="contact-title">
        <h1 id="contact-title" className="text-[40px] font-bold leading-tight">{t("contact.title")}</h1>

        {submitted ? (
          <div className="mt-6">
            <p role="status" className="rounded-[3px] bg-info p-3">{t("contact.sent")}</p>
            <button type="button" onClick={() => setSubmitted(false)} className="mt-4 text-link underline">
              {t("contact.back")}
            </button>
          </div>
        ) : (
          <form onSubmit={onSubmit} className="mt-6 space-y-4">
            <fieldset className="flex gap-6">
              <legend className="sr-only">{t("contact.iam")}</legend>
              {[["company", "contact.company"], ["individual", "contact.individual"]].map(([v, key]) => (
                <label key={v} className="flex cursor-pointer items-center gap-2">
                  <input type="radio" name="type" value={v} checked={type === v} onChange={() => setType(v)} className="accent-teal" />
                  {t(key)}
                </label>
              ))}
            </fieldset>

            {isCompany && (
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label htmlFor="company" className="mb-1 block text-sm font-medium">{t("contact.companyName")}</label>
                  <input id="company" name="company" required value={data.company} onChange={onChange} className={field} />
                </div>
                <div>
                  <label htmlFor="vat" className="mb-1 block text-sm font-medium">{t("contact.vat")}</label>
                  <input id="vat" name="vat" value={data.vat} onChange={onChange} className={field} />
                </div>
              </div>
            )}

            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label htmlFor="name" className="mb-1 block text-sm font-medium">{t("contact.fullName")}</label>
                <input id="name" name="name" required value={data.name} onChange={onChange} className={field} />
              </div>
              <div>
                <label htmlFor="email" className="mb-1 block text-sm font-medium">{isCompany ? t("contact.workEmail") : t("contact.email")}</label>
                <input id="email" name="email" type="email" required value={data.email} onChange={onChange} className={field} />
              </div>
            </div>

            <div>
              <label htmlFor="phone" className="mb-1 block text-sm font-medium">{isCompany ? t("contact.phone") : t("contact.phoneOpt")}</label>
              <input id="phone" name="phone" type="tel" required={isCompany} value={data.phone} onChange={onChange} className={field} />
            </div>

            <div>
              <label htmlFor="message" className="mb-1 block text-sm font-medium">{t("contact.message")}</label>
              <textarea id="message" name="message" rows={5} required value={data.message} onChange={onChange} className={`${field} resize-y`} />
            </div>

            <label className="flex items-start gap-3 text-sm">
              <input type="checkbox" name="consent" required checked={data.consent} onChange={onChange} className="mt-1 h-4 w-4 accent-teal" />
              <span>{t("contact.consent")}</span>
            </label>

            {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
            <button type="submit" disabled={sending} className="inline-flex items-center gap-2 rounded-[3px] bg-action px-6 py-2.5 font-bold text-black hover:bg-action-dark disabled:opacity-60">
              {sending ? t("form.sending") : t("contact.send")} <span aria-hidden="true">→</span>
            </button>
          </form>
        )}
      </section>

      <aside className="h-fit rounded-[4px] bg-white p-6 shadow-card" aria-label={t("contact.detailsAria")}>
        <h2 className="text-[24px] font-semibold">{t("contact.details")}</h2>
        <dl className="mt-4 space-y-3">
          <div><dt className="text-sm text-gray-600">{t("contact.email")}</dt><dd>{SITE.email || <Todo>{t("todo.email")}</Todo>}</dd></div>
          <div><dt className="text-sm text-gray-600">{t("contact.phone")}</dt><dd>{SITE.phone || <Todo>{t("todo.phone")}</Todo>}</dd></div>
          <div><dt className="text-sm text-gray-600">{t("contact.office")}</dt><dd>{SITE.address || <Todo>{t("todo.address")}</Todo>}</dd></div>
        </dl>
      </aside>
    </div>
  );
}
