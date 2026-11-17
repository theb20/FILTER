import { LockKeyhole } from "lucide-react";
import { LinkButton } from "@/components/ui/ui";

export const metadata = { title: "Accès refusé" };

export default function Denied() {
  return (
    <div className="mx-auto mt-16 max-w-md rounded-xl border border-line bg-white p-8 text-center shadow-card">
      <span className="mx-auto grid h-12 w-12 place-items-center rounded-full bg-action/20 text-[#7a5a00]"><LockKeyhole className="h-6 w-6" aria-hidden /></span>
      <h1 className="mt-4 text-xl font-bold text-night">Accès refusé</h1>
      <p className="mt-2 text-sm text-slate">Vous n&apos;avez pas la permission de consulter cette page. Demandez l&apos;accès à un administrateur.</p>
      <div className="mt-6"><LinkButton href="/dashboard">Retour au tableau de bord</LinkButton></div>
    </div>
  );
}
