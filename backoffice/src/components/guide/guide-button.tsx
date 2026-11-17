"use client";

import { CircleHelp } from "lucide-react";
import { useGuide } from "./guide-provider";

/** « Découvrir cette section » : visible seulement si le guide est activé et que la page a une visite. */
export function GuideButton() {
  const { prefs, current, active, start } = useGuide();
  if (!prefs.enabled || !current) return null;
  return (
    <button
      type="button"
      data-guide="guide-help"
      onClick={() => start(current.id)}
      disabled={Boolean(active)}
      aria-label="Découvrir cette section"
      className="inline-flex items-center gap-2 rounded-lg border border-line bg-white px-3 py-1.5 text-sm font-medium text-night transition hover:bg-page disabled:opacity-50"
    >
      <span className="relative">
        <CircleHelp className="h-4 w-4 text-link" aria-hidden />
        {!prefs.seen.includes(current.id) && <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-action ring-2 ring-white" aria-hidden />}
      </span>
      <span className="hidden md:inline">Découvrir cette section</span>
    </button>
  );
}
