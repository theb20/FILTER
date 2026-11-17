import Link from "next/link";
import { btnCls } from "@/components/ui/ui";

export default function NotFound() {
  return (
    <main className="grid min-h-screen place-items-center bg-page p-6">
      <div className="max-w-md rounded-xl border border-line bg-white p-10 text-center shadow-card">
        <p className="text-5xl font-bold tracking-tight text-action-dark">404</p>
        <h1 className="mt-3 text-xl font-bold text-night">Page introuvable</h1>
        <p className="mt-2 text-sm text-slate">Cette page n&apos;existe pas ou a été supprimée.</p>
        <Link href="/dashboard" className={`${btnCls} mt-6`}>Retour au tableau de bord</Link>
      </div>
    </main>
  );
}
