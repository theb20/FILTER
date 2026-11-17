import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { request } from "../lib/api.js";
import { useAuth } from "../store/authStore.js";
import { useI18n } from "../i18n/i18nContext.js";

// Lien reçu par e-mail après l'inscription : confirme l'adresse (jeton à usage unique).
export default function VerifyEmail() {
  const { t } = useI18n();
  const [sp] = useSearchParams();
  const token = sp.get("token") ?? "";
  const [state, setState] = useState(token ? "pending" : "invalid"); // pending | ok | invalid | unavailable
  const done = useRef(false);
  const signedIn = useAuth((a) => a.status === "authenticated");

  useEffect(() => {
    document.title = t("meta.sign");
    if (!token || done.current) return;
    done.current = true;
    window.history.replaceState(null, "", "/verify-email"); // le jeton ne reste pas dans l'adresse
    request("/auth/email/verify", { method: "POST", body: { token } }).then((r) => setState(r.ok ? "ok" : r.status === 0 || r.status >= 500 ? "unavailable" : "invalid"));
  }, [token, t]);

  return (
    <div className="mx-2.5 mt-2.5 flex justify-center py-16">
      <section className="w-full max-w-md rounded-[4px] bg-white p-8 text-center shadow-card" aria-live="polite">
        <h1 className="text-[28px] font-bold">{t(`verify.${state}.title`)}</h1>
        <p className="mt-3 text-gray-700">{t(`verify.${state}.text`)}</p>
        {state !== "pending" && (
          <Link to={signedIn ? "/profil" : "/sign"} className="mt-6 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">
            {t(signedIn ? "verify.toAccount" : "reset.backSign")}
          </Link>
        )}
      </section>
    </div>
  );
}
