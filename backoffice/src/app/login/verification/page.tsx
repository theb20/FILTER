import { redirect } from "next/navigation";
import { AuthShell } from "@/components/auth-shell";
import { CodeForm } from "@/components/code-form";
import { getChallenge } from "@/lib/session";
import { verify } from "../actions";

export default async function VerificationPage() {
  if (!(await getChallenge())) redirect("/login");
  return (
    <AuthShell>
      <div className="space-y-5">
        <div>
          <h1 className="text-2xl font-bold text-night">Vérification en deux étapes</h1>
          <p className="mt-1 text-sm text-slate">Saisissez le code à 6 chiffres de votre application d&apos;authentification, ou un code de secours.</p>
        </div>
        <CodeForm action={verify} label="Code de vérification" />
        <a href="/login" className="block text-center text-sm text-link hover:underline">Retour à la connexion</a>
      </div>
    </AuthShell>
  );
}
