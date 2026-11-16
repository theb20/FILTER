import { AuthShell, inputClass } from "@/components/auth-shell";
import { ActionForm } from "@/components/ui/client";
import { acceptInvitation } from "./actions";

export default async function InvitationPage({ searchParams }: { searchParams: Promise<{ token?: string }> }) {
  const { token } = await searchParams;
  return (
    <AuthShell>
      {!token ? (
        <p className="text-sm text-slate">Lien d&apos;invitation incomplet. Utilisez le lien reçu par e-mail.</p>
      ) : (
        <div className="space-y-5">
          <div>
            <h1 className="text-2xl font-bold text-night">Bienvenue</h1>
            <p className="mt-1 text-sm text-slate">Choisissez votre mot de passe (12 caractères minimum). Vous enregistrerez ensuite votre application d&apos;authentification.</p>
          </div>
          <ActionForm action={acceptInvitation.bind(null, token)} submit="Continuer">
            <label className="block text-sm font-medium text-night">Mot de passe<input name="password" type="password" autoComplete="new-password" minLength={12} required className={inputClass} /></label>
            <label className="block text-sm font-medium text-night">Confirmer<input name="confirm" type="password" autoComplete="new-password" required className={inputClass} /></label>
          </ActionForm>
        </div>
      )}
    </AuthShell>
  );
}
