import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Database, Lock } from "lucide-react";
import { useI18n } from "../../i18n/i18nContext.js";
import { useNotify } from "../../configurations/notificationContext.js";
import { useAuth } from "../../store/authStore.js";
import { btnDark, card, cardTitle, fmtDate } from "./ui.js";

const POLICY_VERSION = "1.0.0"; // version de la politique de cookies affichée par le bandeau du site
const PURPOSES = ["functional", "analytics", "marketing"];

// Onglet « Confidentialité » : consentements enregistrés sur le compte (preuve horodatée côté serveur, alignée
// avec le bandeau de cookies de ce navigateur) et droits sur les données.
export default function PrivacyPanel() {
  const { t, lang } = useI18n();
  const notify = useNotify();
  const call = useAuth((s) => s.call);
  const [prefs, setPrefs] = useState(/** @type {Record<string, boolean> | null} */ (null));
  const [recordedAt, setRecordedAt] = useState(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    call("/me/consents").then((r) => {
      if (r.ok && r.data) {
        setPrefs({ functional: !!r.data.purposes.functional, analytics: !!r.data.purposes.analytics, marketing: !!r.data.purposes.marketing });
        setRecordedAt(r.data.recordedAt);
      } else {
        // Aucun choix sur le compte : on reprend celui du bandeau de ce navigateur, sinon rien n'est présumé accepté.
        let local = null;
        try { local = JSON.parse(localStorage.getItem("ff_cookie_consent") ?? "null")?.preferences ?? null; } catch { /* illisible */ }
        setPrefs({ functional: !!local?.functional, analytics: !!local?.analytics, marketing: !!local?.marketing });
      }
    });
  }, [call]);

  const save = async () => {
    setBusy(true);
    const purposes = { necessary: true, ...prefs };
    const r = await call("/me/consents", { method: "PUT", body: { purposes, policyVersion: POLICY_VERSION, locale: lang } });
    setBusy(false);
    if (!r.ok) return notify(t("profile.err.generic"), "error");
    setRecordedAt(r.data?.recordedAt ?? new Date().toISOString());
    try { // même format que le bandeau de cookies : ce navigateur applique aussitôt le choix
      localStorage.setItem("ff_cookie_consent", JSON.stringify({ preferences: purposes, consented: !!(prefs.functional || prefs.analytics || prefs.marketing), decisionDate: new Date().toISOString() }));
    } catch { /* stockage indisponible : le choix reste enregistré sur le compte */ }
    notify(t("profile.priv.saved"), "success");
  };

  return (
    <div className="space-y-4">
      <section className={card} aria-labelledby="consent-title">
        <h2 id="consent-title" className={cardTitle}><Lock size={20} aria-hidden="true" /> {t("profile.priv.title")}</h2>
        <p className="mt-1 text-sm text-gray-600">{t("profile.priv.lead")}</p>
        {prefs === null ? <p className="mt-4 text-sm text-gray-500" role="status">{t("loading")}</p> : (
          <>
            <ul className="mt-4 divide-y divide-gray-100">
              <li className="flex items-start justify-between gap-4 py-3">
                <span><span className="font-semibold text-night">{t("cookie.necessary")}</span><span className="block text-sm text-gray-600">{t("cookie.necessaryD")}</span></span>
                <span className="shrink-0 text-sm font-medium text-gray-500">{t("profile.priv.always")}</span>
              </li>
              {PURPOSES.map((k) => (
                <li key={k} className="flex items-start justify-between gap-4 py-3">
                  <label htmlFor={`consent-${k}`} className="cursor-pointer"><span className="font-semibold text-night">{t(`cookie.${k}`)}</span><span className="block text-sm text-gray-600">{t(`cookie.${k}D`)}</span></label>
                  <button id={`consent-${k}`} type="button" role="switch" aria-checked={prefs[k]} onClick={() => setPrefs((p) => ({ ...p, [k]: !p[k] }))}
                    className={`relative mt-1 h-6 w-11 shrink-0 rounded-full transition ${prefs[k] ? "bg-green-600" : "bg-gray-300"}`}>
                    <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-all ${prefs[k] ? "left-[22px]" : "left-0.5"}`} />
                  </button>
                </li>
              ))}
            </ul>
            <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
              <p className="text-xs text-gray-500">{recordedAt ? t("profile.priv.recorded", { date: fmtDate(lang, recordedAt, { dateStyle: "long", timeStyle: "short" }) }) : t("profile.priv.never")}</p>
              <button type="button" onClick={save} disabled={busy} className={btnDark}>{busy ? t("profile.form.saving") : t("profile.priv.save")}</button>
            </div>
          </>
        )}
      </section>

      <section className={card} aria-labelledby="rights-title">
        <h2 id="rights-title" className={cardTitle}><Database size={20} aria-hidden="true" /> {t("profile.priv.rightsTitle")}</h2>
        <ul className="mt-3 list-disc space-y-1.5 pl-5 text-sm text-gray-700">
          <li>{t("profile.priv.rightsHistory")}</li>
          <li>{t("profile.priv.rightsImages")}</li>
          <li>{t("profile.priv.rightsExport")}</li>
        </ul>
        <div className="mt-4 flex flex-wrap gap-3">
          <Link to="/contact" className={btnDark}>{t("profile.priv.contact")}</Link>
          <Link to="/privacy" className="inline-flex items-center px-2 text-sm text-link hover:underline">{t("profile.priv.policy")}</Link>
        </div>
      </section>
    </div>
  );
}
