import { useState } from "react";
import { Info, X } from "lucide-react";
import { SITE } from "../config/site.js";

// Affiché seulement si un site régional est configuré (SITE.regionalSite).
export default function GeoBanner() {
  const [open, setOpen] = useState(true);
  const regional = SITE.regionalSite;
  if (!open || !regional) return null;

  return (
    <div className="bg-info" role="region" aria-label="Regional site">
      <div className="mx-auto flex max-w-[1500px] items-center gap-3 px-4 py-2 text-sm lg:px-6">
        <Info size={18} aria-hidden="true" className="shrink-0" />
        <p className="flex-1">Looks like you are in {regional.country}. Would you like to visit our {regional.country} site?</p>
        <a href={regional.url} className="rounded-[3px] bg-teal px-3 py-1.5 font-medium text-white hover:opacity-90">
          Yes, take me there
        </a>
        <button type="button" aria-label="Dismiss" onClick={() => setOpen(false)}>
          <X size={18} />
        </button>
      </div>
    </div>
  );
}
