import { useState } from "react";
import { useNotify } from "../configurations/notificationContext.js";
import { useI18n } from "../i18n/i18nContext.js";

export default function Newsletter() {
  const notify = useNotify();
  const { t } = useI18n();
  const [email, setEmail] = useState("");

  const submit = (e) => {
    e.preventDefault();
    notify(t("demo.notice"), "demo"); // aucune inscription réelle
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
        <button type="submit" className="rounded-r-[3px] bg-slate px-5 font-medium text-white hover:bg-night">
          {t("newsletter.subscribe")}
        </button>
      </form>

    </section>
  );
}
