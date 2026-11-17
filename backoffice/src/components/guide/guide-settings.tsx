"use client";

import { useGuide } from "./guide-provider";
import { btnGhostCls } from "@/components/ui/ui";

function Switch({ checked, onChange, label, hint }: { checked: boolean; onChange: (v: boolean) => void; label: string; hint: string }) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <p className="text-sm font-medium text-night" id={`sw-${label}`}>{label}</p>
        <p className="text-sm text-slate">{hint}</p>
      </div>
      <button
        type="button" role="switch" aria-checked={checked} aria-labelledby={`sw-${label}`} onClick={() => onChange(!checked)}
        className={`relative mt-0.5 h-6 w-11 shrink-0 rounded-full transition ${checked ? "bg-ok" : "bg-slate/30"}`}
      >
        <span className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${checked ? "translate-x-5" : ""}`} />
      </button>
    </div>
  );
}

/** Réglages du guide, affichés dans « Mon compte ». Les choix sont enregistrés tout de suite, par membre. */
export function GuideSettings() {
  const { prefs, tours, setEnabled, setAnimations, start, resetSeen } = useGuide();
  return (
    <div className="space-y-5">
      <Switch checked={prefs.enabled} onChange={setEnabled} label="Guide visuel" hint="Affiche le bouton « Découvrir cette section » sur chaque page (une pastille jaune signale les visites pas encore faites) et lance la visite d'accueil à la première connexion." />
      <Switch checked={prefs.animations === "auto"} onChange={(v) => setAnimations(v ? "auto" : "off")} label="Animations du guide" hint="Désactivées automatiquement si votre système demande de réduire les animations." />
      <div>
        <div className="mb-2 flex items-center justify-between gap-3">
          <p className="text-sm font-medium text-night">Visites disponibles</p>
          <button type="button" onClick={resetSeen} className="text-sm text-link hover:underline">Tout marquer comme non vu</button>
        </div>
        <ul className="max-h-96 divide-y divide-line overflow-y-auto rounded-lg border border-line">
          {tours.filter((t) => t.match(t.path)).map((t) => (
            <li key={t.id} className="flex items-center justify-between gap-3 px-3 py-2.5">
              <div className="min-w-0">
                <p className="text-sm font-medium text-night">{t.title} <span className={`ml-1 text-xs font-normal ${prefs.seen.includes(t.id) ? "text-ok" : "text-slate"}`}>{prefs.seen.includes(t.id) ? "· vue" : "· non vue"}</span></p>
                <p className="truncate text-xs text-slate">{t.summary}</p>
              </div>
              <button type="button" onClick={() => start(t.id)} className={`${btnGhostCls} shrink-0 !px-3 !py-1.5`}>Lancer</button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
