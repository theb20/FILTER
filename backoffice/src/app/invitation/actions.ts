"use server";

import { redirect } from "next/navigation";
import { gateway } from "@/lib/api";
import { setChallenge } from "@/lib/session";
import type { FormState } from "@/components/ui/client";

type Accepted = { status: string; challengeToken: string | null };

export async function acceptInvitation(token: string, _: FormState, form: FormData): Promise<FormState> {
  const password = String(form.get("password") ?? "");
  if (password.length < 12) return { error: "Le mot de passe doit contenir au moins 12 caractères." };
  if (password !== String(form.get("confirm") ?? "")) return { error: "La confirmation ne correspond pas au mot de passe." };
  const r = await gateway<Accepted>("/admin/auth/invitation/accept", { body: { token, password } });
  if (!r.ok) {
    if (r.status === 429) return { error: `Trop de tentatives. Réessayez dans ${r.retryAfter ?? 60} s.` };
    if (r.code === "INVALID_CODE") return { error: "Cette invitation est invalide ou a expiré. Demandez-en une nouvelle à un administrateur." };
    return { error: r.fields?.[0]?.message ?? r.message ?? "Une erreur est survenue." };
  }
  if (!r.data.challengeToken) return { error: "Réponse inattendue du serveur." };
  await setChallenge(r.data.challengeToken);
  redirect("/login/activation"); // enchaîne sur l'enregistrement de la double authentification
}
