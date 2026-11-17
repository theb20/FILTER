import { ChevronDown } from "lucide-react";

// Accordéon de questions fréquentes (balises <details>, accessible au clavier).
// items : [{ q, a }] déjà traduits. Réutilisable (fiche produit, panier…).
export default function FaqAccordion({ items, defaultOpen = 0 }) {
  return (
    <div className="divide-y divide-gray-200">
      {items.map((it, i) => (
        <details key={it.q} open={i === defaultOpen} className="group py-3">
          <summary className="flex cursor-pointer list-none items-center justify-between gap-3 font-medium">
            {it.q}
            <ChevronDown size={18} className="shrink-0 text-gray-500 transition-transform group-open:rotate-180" aria-hidden="true" />
          </summary>
          <p className="mt-2 text-[15px] text-gray-700">{it.a}</p>
        </details>
      ))}
    </div>
  );
}
