"use client";

import { startTransition, useActionState } from "react";
import { useFormStatus } from "react-dom";
import { btnCls, btnDangerCls, btnGhostCls, Notice } from "./ui";

export type FormState = { error?: string; ok?: string } | undefined;

function Submit({ children, variant, busy, guide }: { children: React.ReactNode; variant?: "danger" | "ghost"; busy?: boolean; guide?: string }) {
  const { pending: formPending } = useFormStatus();
  const pending = busy ?? formPending;
  const cls = variant === "danger" ? btnDangerCls : variant === "ghost" ? btnGhostCls : btnCls;
  return <button type="submit" disabled={pending} data-guide={guide} className={cls}>{pending ? "Patientez…" : children}</button>;
}

/** Formulaire relié à une action serveur : affiche l'erreur renvoyée et CONSERVE la saisie (React 19 vide sinon les champs après chaque soumission). */
export function ActionForm({ action, submit, children, className = "space-y-4" }: { action: (s: FormState, f: FormData) => Promise<FormState>; submit: string; children: React.ReactNode; className?: string }) {
  const [state, run, isPending] = useActionState(action, undefined);
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        const data = new FormData(e.currentTarget);
        startTransition(() => run(data));
      }}
      data-guide="form"
      className={className}
    >
      {state?.error && <Notice kind="error">{state.error}</Notice>}
      {state?.ok && <Notice kind="ok">{state.ok}</Notice>}
      {children}
      <Submit busy={isPending} guide="form-submit">{submit}</Submit>
    </form>
  );
}

/** Bouton d'action (sans champ) avec confirmation optionnelle ; renvoie l'erreur sous le bouton. */
export function ActionButton({ action, label, confirm, variant }: { action: (s: FormState, f: FormData) => Promise<FormState>; label: string; confirm?: string; variant?: "danger" | "ghost" }) {
  const [state, run] = useActionState(action, undefined);
  return (
    <form action={run} onSubmit={(e) => { if (confirm && !window.confirm(confirm)) e.preventDefault(); }}>
      <Submit variant={variant}>{label}</Submit>
      {state?.error && <p role="alert" className="mt-1 max-w-xs text-xs text-danger">{state.error}</p>}
    </form>
  );
}
