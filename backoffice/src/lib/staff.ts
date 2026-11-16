import "server-only";
import { gateway } from "./api";
import { getAccess, getRefresh } from "./session";

export type StaffMe = { id: string; email: string; name: string; roles: string[]; permissions: string[]; twoFactorEnabled: boolean };

/** Profil du personnel connecté. `needsRefresh` : le jeton d'accès a expiré mais une session de rafraîchissement existe. */
export async function currentStaff(): Promise<{ me?: StaffMe; needsRefresh?: boolean }> {
  const token = await getAccess();
  if (token) {
    const r = await gateway<StaffMe>("/admin/me", { method: "GET", token });
    if (r.ok) return { me: r.data };
  }
  return { needsRefresh: Boolean(await getRefresh()) };
}
