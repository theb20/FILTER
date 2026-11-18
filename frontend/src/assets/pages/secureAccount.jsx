import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ShieldAlert, ShieldCheck } from "lucide-react";
import { request } from "../lib/api.js";
import { useAuth } from "../store/authStore.js";
import { useI18n } from "../i18n/i18nContext.js";

// Lien « ce n'était pas moi » de l'e-mail d'alerte de nouvelle connexion : tous les appareils sont déconnectés,
// puis le titulaire choisit un nouveau mot de passe. Le jeton est à usage unique (valable 7 jours).
export default function SecureAccount() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const [state, setState] = useState(/** @type {"busy" | "done" | "invalid" | "error"} */ ("busy"));
  const sent = useRef(false);

  useEffect(() => {
    document.title = t("meta.page", { title: t("secure.title") });
    if (sent.current) return; // une seule tentative (mode strict de React : double effet en développement)
    sent.current = true;
    if (!token) return setState("invalid");
    request("/auth/login-alert/secure", { method: "POST", body: { token } }).then((r) => {
      if (r.ok) {
        useAuth.setState({ status: "anonymous", user: null, accessToken: null }); // cette session-ci est aussi révoquée
        setState("done");
      } else setState(r.status === 400 || r.status === 422 ? "invalid" : "error");
    });
  }, [token, t]);

  const ok = state === "done";
  return (
    <section className="mx-auto mt-6 max-w-xl rounded-[10px] bg-white p-8 text-center shadow-card" aria-live="polite">
      <span className={`mx-auto flex h-14 w-14 items-center justify-center rounded-full ${ok ? "bg-green-50 text-green-700" : "bg-amber-50 text-amber-700"}`}>
        {ok ? <ShieldCheck size={28} aria-hidden="true" /> : <ShieldAlert size={28} aria-hidden="true" />}
      </span>
      <h1 className="mt-4 text-[26px] font-bold text-night">{t("secure.title")}</h1>
      {state === "busy" && <p className="mt-3 text-gray-600" role="status">{t("secure.busy")}</p>}
      {state === "done" && (
        <>
          <p className="mt-3 text-gray-700">{t("secure.done")}</p>
          <Link to="/reset" className="mt-6 inline-flex rounded-[8px] bg-action px-5 py-2.5 font-bold text-black hover:bg-action-dark">{t("secure.reset")}</Link>
        </>
      )}
      {state === "invalid" && (
        <>
          <p className="mt-3 text-gray-700">{t("secure.invalid")}</p>
          <Link to="/sign" className="mt-6 inline-flex rounded-[8px] bg-night px-5 py-2.5 font-medium text-white hover:opacity-90">{t("secure.signin")}</Link>
        </>
      )}
      {state === "error" && <p className="mt-3 text-red-700">{t("profile.err.generic")}</p>}
    </section>
  );
}
