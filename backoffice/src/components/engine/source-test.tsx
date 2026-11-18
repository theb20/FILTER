"use client";

import { useState, useTransition } from "react";
import { CircleAlert, CircleCheck, FlaskConical } from "lucide-react";
import { btnCls, inputCls, selectCls } from "@/components/ui/ui";
import { testSource } from "@/app/(admin)/engine/sources/actions";
import type { SourceTestResult } from "@/lib/types";

const MICROS = 1_000_000;
const fmt = (micros: number, currency: string) => `${new Intl.NumberFormat("fr-FR", { maximumFractionDigits: 6 }).format(micros / MICROS)} ${currency}`;

/** Essai réel de la source : un appel facturé, avec la clé lue dans l'environnement du service. */
export default function SourceTest({ id, errorLabels }: { id: string; errorLabels: Record<string, string> }) {
  const [query, setQuery] = useState("filtre hydraulique");
  // Pays facultatif : Brave n'en couvre qu'une liste (la Côte d'Ivoire n'y est pas et fait échouer l'appel en 422).
  const [country, setCountry] = useState("");
  const [res, setRes] = useState<SourceTestResult | null>(null);
  const [err, setErr] = useState("");
  const [pending, start] = useTransition();

  const run = () => start(async () => {
    setErr(""); setRes(null);
    try {
      const r = await testSource(id, query, country);
      if (r.error) setErr(r.error); else setRes(r.result ?? null);
    } catch { setErr("Service indisponible ou session expirée. Réessayez."); }
  });

  return (
    <div data-guide="source-test" className="space-y-3">
      <div className="flex flex-wrap items-end gap-2">
        <label className="flex-1 text-sm font-medium text-night">Requête de test
          <input value={query} onChange={(e) => setQuery(e.target.value)} maxLength={200} className={inputCls} placeholder="ex. filtre hydraulique" />
        </label>
        <label className="text-sm font-medium text-night">Pays
          <select value={country} onChange={(e) => setCountry(e.target.value)} className={`${selectCls} mt-1.5 block`}>
            <option value="">Aucun (défaut de la source)</option>
            {["FR", "BE", "CH", "CA", "US", "GB", "DE", "ES", "IT", "NL", "PT"].map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </label>
        <button type="button" onClick={run} disabled={pending} className={`${btnCls} disabled:opacity-50`}>
          <FlaskConical size={16} aria-hidden /> {pending ? "Appel en cours…" : "Tester la source"}
        </button>
      </div>
      <p className="text-xs text-slate">Cet essai lance un vrai appel : il est facturé par la source et décompté du budget. Toutes les sources ne couvrent pas tous les pays : Brave refuse « CI », laissez « Aucun » en cas de doute.</p>
      {err && <p role="alert" className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{err}</p>}
      {res && (
        <div role="status" className={`rounded-lg border p-3 text-sm ${res.ok ? "border-ok/30 bg-ok/5" : "border-danger/30 bg-danger/5"}`}>
          <p className={`flex items-center gap-2 font-semibold ${res.ok ? "text-[#007a52]" : "text-danger"}`}>
            {res.ok ? <CircleCheck size={16} aria-hidden /> : <CircleAlert size={16} aria-hidden />}
            {res.ok ? `Appel réussi : ${res.resultCount} résultat(s)` : "Appel en échec"}
          </p>
          {!res.ok && res.errorKind && <p className="mt-1 text-night">{errorLabels[res.errorKind] ?? res.errorKind}</p>}
          <p className="mt-1 text-slate">{res.durationMs} ms · coût {fmt(res.costMicros, res.currency)}</p>
          {res.sample.length > 0 && (
            <ul className="mt-2 list-disc space-y-0.5 pl-5 text-night">
              {res.sample.slice(0, 5).map((s, i) => <li key={i} className="truncate">{s}</li>)}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
