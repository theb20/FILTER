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
    <section className="mx-2.5 mt-2.5 bg-news px-4 py-10 text-center" aria-labelledby="newsletter-title">
      <h2 id="newsletter-title" className="text-[28px] font-bold">{t("newsletter.title")}</h2>
      <p className="mt-1 font-bold text-white">{t("newsletter.subtitle")}</p>
      <form onSubmit={submit} className="mx-auto mt-5 flex max-w-md">
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
