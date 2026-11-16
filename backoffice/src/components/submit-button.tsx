"use client";

import { useFormStatus } from "react-dom";
import { buttonClass } from "./auth-shell";

export function SubmitButton({ children }: { children: React.ReactNode }) {
  const { pending } = useFormStatus();
  return (
    <button type="submit" disabled={pending} className={buttonClass}>
      {pending ? "Patientez…" : children}
    </button>
  );
}
