"use client";

import { useEffect } from "react";
import { AlertTriangle, RotateCw } from "lucide-react";
import { btnCls, btnGhostCls } from "@/components/ui/ui";
import Link from "next/link";

export default function ErrorPage({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => { console.error(error); }, [error]);
  return (
    <div role="alert" className="mx-auto mt-16 max-w-md rounded-xl border border-line bg-white p-8 text-center shadow-card">
      <span className="mx-auto grid h-12 w-12 place-items-center rounded-full bg-danger/10 text-danger"><AlertTriangle className="h-6 w-6" aria-hidden /></span>
      <h1 className="mt-4 text-xl font-bold text-night">Une erreur est survenue</h1>
      <p className="mt-2 text-sm text-slate">Le service est peut-être momentanément indisponible. Réessayez ; si le problème persiste, contactez un administrateur.</p>
      <div className="mt-6 flex justify-center gap-2">
        <button type="button" onClick={() => retry()} className={btnCls}><RotateCw className="h-4 w-4" aria-hidden />Réessayer</button>
        <Link href="/dashboard" className={btnGhostCls}>Tableau de bord</Link>
      </div>
    </div>
  );
}
