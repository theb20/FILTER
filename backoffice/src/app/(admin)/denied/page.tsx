import Link from "next/link";

export default function Denied() {
  return (
    <div className="mx-auto mt-16 max-w-md rounded-lg border border-line bg-white p-8 text-center shadow-card">
      <h1 className="text-xl font-bold text-night">Accès refusé</h1>
      <p className="mt-2 text-sm text-slate">Vous n&apos;avez pas la permission de consulter cette page. Demandez l&apos;accès à un administrateur.</p>
      <Link href="/dashboard" className="mt-5 inline-block rounded bg-action px-4 py-2 font-medium text-night hover:bg-action-dark">Retour au tableau de bord</Link>
    </div>
  );
}
