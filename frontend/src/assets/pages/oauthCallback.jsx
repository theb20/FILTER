import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../store/authStore.js";
import { useI18n } from "../i18n/i18nContext.js";

// Retour de Google : le serveur renvoie ici avec `?code=` (échange à usage unique) ou `?error=`.
export default function OAuthCallback() {
  const { t } = useI18n();
  const [sp] = useSearchParams();
  const navigate = useNavigate();
  const exchange = useAuth((a) => a.exchangeOAuth);
  const [error, setError] = useState(sp.get("error") ? "auth.err.oauth" : "");
  const done = useRef(false); // le code ne sert qu'une fois : jamais deux échanges (double rendu de développement)

  useEffect(() => {
    document.title = t("meta.sign");
    const code = sp.get("code");
    if (done.current || sp.get("error")) return;
    done.current = true;
    if (!code) { setError("auth.err.oauth"); return; }
    window.history.replaceState(null, "", "/oauth/callback"); // le code ne reste ni dans l'historique ni dans l'adresse
    exchange(code).then((r) => (r.ok ? navigate("/profil", { replace: true }) : setError(r.error === "auth.err.generic" ? "auth.err.oauth" : r.error)));
  }, [sp, exchange, navigate, t]);

  return (
    <div className="mx-2.5 mt-2.5 flex justify-center py-16">
      <section className="w-full max-w-md rounded-[4px] bg-white p-8 text-center shadow-card" aria-live="polite">
        {error ? (
          <>
            <h1 className="text-[28px] font-bold">{t("auth.oauth.failed")}</h1>
            <p role="alert" className="mt-3 text-gray-700">{t(error)}</p>
            <Link to="/sign" className="mt-6 inline-flex rounded-[3px] bg-action px-5 py-2.5 font-bold hover:bg-action-dark">{t("reset.backSign")}</Link>
          </>
        ) : (
          <p role="status">{t("auth.oauth.pending")}</p>
        )}
      </section>
    </div>
  );
}
