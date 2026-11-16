import { NextRequest, NextResponse } from "next/server";
import { gateway } from "@/lib/api";
import { clearSession, getRefresh, setSession, type StaffTokens } from "@/lib/session";

// Renouvelle la session (rotation du jeton de rafraîchissement) puis revient à la page demandée.
export async function GET(req: NextRequest) {
  const next = req.nextUrl.searchParams.get("next") ?? "/dashboard";
  const safeNext = next.startsWith("/") && !next.startsWith("//") ? next : "/dashboard"; // jamais de redirection vers un autre site
  const refresh = await getRefresh();
  if (refresh) {
    const r = await gateway<StaffTokens>("/admin/auth/refresh", { body: { refreshToken: refresh } });
    if (r.ok) {
      await setSession(r.data);
      return NextResponse.redirect(new URL(safeNext, req.url));
    }
  }
  await clearSession();
  return NextResponse.redirect(new URL("/login", req.url));
}
