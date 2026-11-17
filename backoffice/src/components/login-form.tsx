"use client";

import { useActionState } from "react";
import { login } from "@/app/login/actions";
import { inputClass } from "./auth-shell";
import { FormError } from "./form-error";
import { SubmitButton } from "./submit-button";

export function LoginForm({ expired }: { expired?: boolean }) {
  const [state, action] = useActionState(login, undefined);
  return (
    <form action={action} className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-night">Connexion</h1>
        <p className="mt-1 text-sm text-slate">Identifiez-vous avec votre compte professionnel.</p>
      </div>
      {expired && !state && <FormError message="Votre session a expiré. Reconnectez-vous." />}
      <FormError message={state?.error} />
      <label className="block text-sm font-medium text-night">
        Adresse e-mail
        <input type="email" name="email" autoComplete="username" required autoFocus defaultValue={state?.email} className={inputClass} />
      </label>
      <label className="block text-sm font-medium text-night">
        Mot de passe
        <input type="password" name="password" autoComplete="current-password" required className={inputClass} />
      </label>
      <SubmitButton>Se connecter</SubmitButton>
    </form>
  );
}
