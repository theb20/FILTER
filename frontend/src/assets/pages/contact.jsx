import { useEffect, useState } from "react";
import { Mail, MapPin, Phone } from "lucide-react";
import { useSite } from "../store/siteStore.js";
import { useI18n } from "../i18n/i18nContext.js";
import Todo from "../components/Todo.jsx";
import { sendForm } from "../lib/forms.js";

const EMPTY = { firstName: "", lastName: "", phone: "", email: "", company: "", message: "", consent: false };

const input = "w-full rounded-[14px] border border-gray-200 bg-white px-4 py-3 text-[14px] text-black outline-none transition placeholder:text-gray-400 focus:border-gray-400 focus:ring-2 focus:ring-[#dcff00]";
const lbl = "mb-1.5 block text-[13px] font-medium text-black";

// Fond : photo bg_contact sous un dégradé linéaire (bleu nuit → transparent) qui garde le texte blanc lisible.
function Backdrop() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 bg-cover bg-center"
      style={{ backgroundImage: "linear-gradient(100deg, rgba(10,22,40,0.88) 0%, rgba(14,40,80,0.62) 45%, rgba(20,60,120,0.25) 100%), url('/img/background/bg_contact.jpg')" }}
    />
  );
}

export default function Contact() {
  const { t } = useI18n();
  const SITE = useSite();
  const [data, setData] = useState(EMPTY);
  const [submitted, setSubmitted] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    document.title = t("meta.contact");
  }, [t]);

  const onChange = (e) => {
    const { name, value, type, checked } = e.target;
    setData((d) => ({ ...d, [name]: type === "checkbox" ? checked : value }));
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setSending(true);
    setError("");
    const r = await sendForm("/contact", { name: `${data.firstName} ${data.lastName}`.trim(), email: data.email, phone: data.phone, company: data.company, message: data.message });
    setSending(false);
    if (!r.ok) return setError(t(`form.err.${r.error}`));
    setData(EMPTY);
    setSubmitted(true);
  };

  return (
    <div className="mx-2.5 mt-2.5 overflow-hidden rounded-[6px] bg-white shadow-card">
      <section className="relative isolate min-h-[640px]" aria-labelledby="contact-title">
        <Backdrop />
        <div className="relative mx-auto grid max-w-[1200px] gap-10 px-5 pb-24 pt-14 sm:px-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,480px)] lg:items-start lg:pt-16">
          <div className="text-white">
            <p className="text-[13px] font-semibold uppercase tracking-wide text-[#dcff00]">( {t("nav.contact")} )</p>
            <h1 id="contact-title" className="mt-4 text-[44px] font-bold leading-[1.02] tracking-tight sm:text-[64px]">{t("contact.hero.title")}</h1>
            <p className="mt-6 max-w-md text-[17px] leading-relaxed text-white/95">{t("contact.hero.text")}</p>
          </div>

          <div data-guide="contact-form" className="rounded-[26px] bg-white/35 p-2 shadow-2xl ring-1 ring-white/50 backdrop-blur-md">
            <div className="rounded-[20px] bg-white p-5 sm:p-6">
              {submitted ? (
                <div className="py-8 text-center">
                  <p role="status" className="rounded-[14px] bg-gray-50 p-4 text-[15px]">{t("contact.sent")}</p>
                  <button type="button" onClick={() => setSubmitted(false)} className="mt-4 text-sm font-medium underline">{t("contact.back")}</button>
                </div>
              ) : (
                <form onSubmit={onSubmit} className="space-y-4">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div><label htmlFor="firstName" className={lbl}>{t("contact.firstName")}</label><input id="firstName" name="firstName" required autoComplete="given-name" placeholder={t("contact.ph.firstName")} value={data.firstName} onChange={onChange} className={input} /></div>
                    <div><label htmlFor="lastName" className={lbl}>{t("contact.lastName")}</label><input id="lastName" name="lastName" required autoComplete="family-name" placeholder={t("contact.ph.lastName")} value={data.lastName} onChange={onChange} className={input} /></div>
                    <div><label htmlFor="phone" className={lbl}>{t("contact.phone")}</label><input id="phone" name="phone" type="tel" autoComplete="tel" placeholder={t("contact.ph.phone")} value={data.phone} onChange={onChange} className={input} /></div>
                    <div><label htmlFor="email" className={lbl}>{t("contact.email")}</label><input id="email" name="email" type="email" required autoComplete="email" placeholder={t("contact.ph.email")} value={data.email} onChange={onChange} className={input} /></div>
                  </div>
                  <div><label htmlFor="company" className={lbl}>{t("contact.companyName")}</label><input id="company" name="company" autoComplete="organization" placeholder={t("contact.ph.company")} value={data.company} onChange={onChange} className={input} /></div>
                  <div><label htmlFor="message" className={lbl}>{t("contact.message")}</label><textarea id="message" name="message" rows={4} required placeholder={t("contact.ph.message")} value={data.message} onChange={onChange} className={`${input} resize-y`} /></div>
                  <label className="flex items-start gap-2.5 text-[12px] leading-snug text-gray-600">
                    <input type="checkbox" name="consent" required checked={data.consent} onChange={onChange} className="mt-0.5 h-4 w-4 accent-black" />
                    <span>{t("contact.consent")}</span>
                  </label>
                  {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
                  <button type="submit" disabled={sending} className="w-full rounded-full bg-[#dcff00] px-6 py-4 text-[15px] font-bold text-black transition hover:brightness-95 disabled:opacity-60">
                    {sending ? t("form.sending") : t("contact.send")}
                  </button>
                </form>
              )}
            </div>
          </div>
        </div>
      </section>

      <section data-guide="contact-details" className="mx-auto grid max-w-[1200px] gap-10 px-5 py-14 sm:px-10 lg:grid-cols-2" aria-labelledby="offices-title">
        <div>
          <h2 id="offices-title" className="text-[40px] font-bold leading-tight tracking-tight sm:text-[48px]">{t("contact.offices")}</h2>
          <p className="mt-4 max-w-xs text-gray-500">{t("contact.offices.text")}</p>
        </div>
        <div>
          <p className="text-[12px] font-medium uppercase tracking-wide text-gray-500">{t("contact.office")}</p>
          <p className="mt-3 flex items-start gap-3 text-[26px] font-semibold leading-snug sm:text-[32px]">
            <MapPin className="mt-2 shrink-0" size={24} aria-hidden="true" />
            <span>{SITE.address || <Todo>{t("todo.address")}</Todo>}</span>
          </p>
          <dl className="mt-6 space-y-2 text-[16px]">
            <div className="flex items-center gap-3"><dt className="sr-only">{t("contact.email")}</dt><Mail size={18} aria-hidden="true" /><dd>{SITE.email ? <a href={`mailto:${SITE.email}`} className="underline">{SITE.email}</a> : <Todo>{t("todo.email")}</Todo>}</dd></div>
            <div className="flex items-center gap-3"><dt className="sr-only">{t("contact.phone")}</dt><Phone size={18} aria-hidden="true" /><dd>{SITE.phone ? <a href={`tel:${SITE.phone.replace(/\s/g, "")}`}>{SITE.phone}</a> : <Todo>{t("todo.phone")}</Todo>}</dd></div>
          </dl>
        </div>
      </section>
    </div>
  );
}
