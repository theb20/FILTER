import { useId } from "react";
import { useGuide } from "./guideContext.js";
import { useI18n } from "../i18n/i18nContext.js";

function Switch({ checked, onChange, label, hint }) {
  const id = useId();
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <p className="font-medium" id={id}>{label}</p>
        <p className="text-sm text-gray-600">{hint}</p>
      </div>
      <button
        type="button" role="switch" aria-checked={checked} aria-labelledby={id} onClick={() => onChange(!checked)}
        className={`relative mt-0.5 h-6 w-11 shrink-0 rounded-full transition ${checked ? "bg-trust" : "bg-gray-300"}`}
      >
        <span className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${checked ? "translate-x-5" : ""}`} />
      </button>
    </div>
  );
}

/** Réglages du guide : enregistrés tout de suite, propres au compte. Affichés dans le profil et dans le pied de page. */
export default function GuideSettings() {
  const { t } = useI18n();
  const { prefs, tours, setEnabled, setAnimations, start, resetSeen } = useGuide();
  return (
    <div className="space-y-5">
      <Switch checked={prefs.enabled} onChange={setEnabled} label={t("guide.settings.enabled")} hint={t("guide.settings.enabledHint")} />
      <Switch checked={prefs.animations === "auto"} onChange={(v) => setAnimations(v ? "auto" : "off")} label={t("guide.settings.animations")} hint={t("guide.settings.animationsHint")} />
      <div>
        <div className="mb-2 flex items-center justify-between gap-3">
          <p className="font-medium">{t("guide.settings.tours")}</p>
          <button type="button" onClick={resetSeen} className="text-sm text-link hover:underline">{t("guide.settings.reset")}</button>
        </div>
        <ul className="max-h-80 divide-y divide-gray-200 overflow-y-auto rounded-[4px] border border-gray-200">
          {tours.filter((x) => x.match(x.path)).map((x) => (
            <li key={x.id} className="flex items-center justify-between gap-3 px-3 py-2.5">
              <div className="min-w-0">
                <p className="font-medium">
                  {t(`guide.${x.id}.title`)}{" "}
                  <span className={`ml-1 text-xs font-normal ${prefs.seen.includes(x.id) ? "text-trust" : "text-gray-600"}`}>
                    · {prefs.seen.includes(x.id) ? t("guide.settings.seen") : t("guide.settings.unseen")}
                  </span>
                </p>
                <p className="truncate text-sm text-gray-600">{t(`guide.${x.id}.summary`)}</p>
              </div>
              <button type="button" onClick={() => start(x.id)} className="shrink-0 rounded-[3px] border border-gray-300 px-3 py-1.5 text-sm font-medium transition hover:bg-gray-50">{t("guide.settings.launch")}</button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
