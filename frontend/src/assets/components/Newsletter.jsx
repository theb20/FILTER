import { useState } from "react";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";
import { sendForm } from "../lib/forms.js";

export default function Newsletter() {
  const notify = useNotify();
  const { t, lang } = useI18n();
  const [sending, setSending] = useState(false);
  const [email, setEmail] = useState("");

  const submit = async (e) => {
    e.preventDefault();
    setSending(true);
    const r = await sendForm("/newsletter/subscribe", { email, locale: lang });
    setSending(false);
    if (!r.ok) return notify(t(`form.err.${r.error}`), "error");
    notify(t("newsletter.thanks"), "success");
    setEmail("");
  };

  return (
    <section className="relative mx-2.5 mt-2.5 overflow-hidden bg-news bg-[url('/img/background/bg_newsletter.webp')] bg-cover bg-center bg-no-repeat px-4 py-10 text-center" aria-labelledby="newsletter-title">
      {/* Flou permanent (sans voile noir) */}
      <div aria-hidden="true" className="pointer-events-none absolute inset-0 backdrop-blur-[4px]" />
      <h2 id="newsletter-title" className="relative text-[28px] font-bold ">{t("newsletter.title")}</h2>
      <p className="relative mt-1 font-bold text-white">{t("newsletter.subtitle")}</p>

      <form onSubmit={submit} className="relative mx-auto mt-5 flex max-w-md">
        <label htmlFor="newsletter-email" className="sr-only">{t("newsletter.email")}</label>
        <input
          id="newsletter-email"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder={t("newsletter.email")}
          className="min-w-0 flex-1 rounded-l-[3px] bg-white px-4 py-2.5 outline-none focus:ring-2 focus:ring-slate"
        />
        <button type="submit" disabled={sending} className="rounded-r-[3px] bg-slate px-5 font-medium text-white hover:bg-night disabled:opacity-60">
          {t("newsletter.subscribe")}
        </button>
      </form>

    </section>
  );
}
