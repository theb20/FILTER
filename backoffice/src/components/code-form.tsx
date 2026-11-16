"use client";

import { useActionState } from "react";
import { inputClass } from "./auth-shell";
import { FormError } from "./form-error";
import { SubmitButton } from "./submit-button";
import type { FormState } from "@/app/login/actions";

export function CodeForm({ action, label }: { action: (s: FormState, f: FormData) => Promise<FormState>; label: string }) {
  const [state, run] = useActionState(action, undefined);
  return (
    <form action={run} className="space-y-4">
      <FormError message={state?.error} />
      <label className="block text-sm font-medium text-night">
        {label}
        <input
          name="code"
          inputMode="text"
          autoComplete="one-time-code"
          required
          autoFocus
          placeholder="123456"
          className={`${inputClass} tracking-[0.3em]`}
        />
      </label>
      <SubmitButton>Valider</SubmitButton>
    </form>
  );
}
