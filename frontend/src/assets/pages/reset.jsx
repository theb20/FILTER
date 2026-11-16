import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useI18n } from "../i18n/i18nContext.js";

const field =
  "w-full rounded-[3px] border bg-white px-4 py-2.5 outline-none focus:ring-2 focus:ring-action";
const btn = "w-full rounded-[3px] bg-action px-5 py-3 font-bold text-black hover:bg-action-dark";

// Parcours de démonstration : aucun email n'est envoyé, aucun mot de passe modifié.
export default function Reset() {
  const { t } = useI18n();
  const [step, setStep] = useState("request"); // request | code | password | done
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [pw, setPw] = useState({ next: "", confirm: "" });
  const [err, setErr] = useState(null);

  useEffect(() => {
    document.title = t("meta.reset");
  }, [t]);

  const submitEmail = (e) => {
    e.preventDefault();
    if (!email.trim()) return setErr("auth.err.email");
    if (!/\S+@\S+\.\S+/.test(email)) return setErr("auth.err.emailInvalid");
    setErr(null);
    setStep("code");
  };

  const submitCode = (e) => {
    e.preventDefault();
    if (!/^\d{6}$/.test(code)) return setErr("reset.err.codeLen");
    setErr(null);
    setStep("password");
  };

  const submitPassword = (e) => {
    e.preventDefault();
    if (pw.next.length < 8) return setErr("reset.err.short");
    if (pw.next !== pw.confirm) return setErr("reset.err.mismatch");
    setErr(null);
    setStep("done");
  };

  const Error = () =>
    err ? (
      <p role="alert" className="text-sm text-red-600">{t(err)}</p>
    ) : null;

  return (
    <div className="mx-2.5 mt-2.5 flex justify-center py-10">
      <section className="w-full max-w-md rounded-[4px] bg-white p-8 shadow-card" aria-labelledby="reset-title">
        <h1 id="reset-title" className="text-[40px] font-bold leading-tight">
          {step === "done" ? t("reset.done") : t("reset.title")}
        </h1>
        <p className="mt-4 rounded-[3px] bg-info p-3 text-sm" role="note">{t("demo.notice")}</p>

        {step === "request" && (
          <form onSubmit={submitEmail} noValidate className="mt-5 space-y-4">
            <p>{t("reset.lead")}</p>
            <div>
              <label htmlFor="reset-email" className="mb-1 block text-sm font-medium">{t("reset.email")}</label>
              <input id="reset-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} className={`${field} border-gray-300`} autoComplete="email" />
            </div>
            <Error />
            <button type="submit" className={btn}>{t("reset.send")} <span aria-hidden="true">→</span></button>
          </form>
        )}

        {step === "code" && (
          <form onSubmit={submitCode} noValidate className="mt-5 space-y-4">
            <h2 className="text-[24px] font-semibold">{t("reset.step2")}</h2>
            <p className="text-sm text-gray-700">{t("reset.codeHelp", { email })}</p>
            <div>
              <label htmlFor="reset-code" className="mb-1 block text-sm font-medium">{t("reset.code")}</label>
              <input id="reset-code" inputMode="numeric" maxLength={6} value={code} onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))} className={`${field} border-gray-300 tracking-[0.4em]`} autoComplete="one-time-code" />
            </div>
            <Error />
            <button type="submit" className={btn}>{t("reset.verify")} <span aria-hidden="true">→</span></button>
          </form>
        )}

        {step === "password" && (
          <form onSubmit={submitPassword} noValidate className="mt-5 space-y-4">
            <h2 className="text-[24px] font-semibold">{t("reset.step3")}</h2>
            <div>
              <label htmlFor="pw-new" className="mb-1 block text-sm font-medium">{t("reset.new")}</label>
              <input id="pw-new" type="password" value={pw.next} onChange={(e) => setPw((p) => ({ ...p, next: e.target.value }))} className={`${field} border-gray-300`} autoComplete="new-password" />
            </div>
            <div>
              <label htmlFor="pw-confirm" className="mb-1 block text-sm font-medium">{t("reset.confirm")}</label>
              <input id="pw-confirm" type="password" value={pw.confirm} onChange={(e) => setPw((p) => ({ ...p, confirm: e.target.value }))} className={`${field} border-gray-300`} autoComplete="new-password" />
            </div>
            <Error />
            <button type="submit" className={btn}>{t("reset.save")} <span aria-hidden="true">→</span></button>
          </form>
        )}

        {step === "done" && (
          <div className="mt-5 space-y-4">
            <p>{t("reset.doneText")}</p>
            <Link to="/sign" className={`${btn} block text-center`}>{t("reset.backSign")}</Link>
          </div>
        )}

        {step !== "done" && (
          <p className="mt-6 text-center text-sm">
            <Link to="/sign" className="text-link hover:underline">{t("reset.backSign")}</Link>
          </p>
        )}
      </section>
    </div>
  );
}
