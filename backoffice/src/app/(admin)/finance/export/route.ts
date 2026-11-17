import { NextResponse, type NextRequest } from "next/server";
import { API_URL } from "@/lib/api";
import { getAccess } from "@/lib/session";

// Téléchargement CSV : le jeton reste côté serveur ; le navigateur n'appelle jamais le gateway directement.
const ROUTES: Record<string, { path: string; params: string[]; file: string }> = {
  "trial-balance": { path: "/admin/finance/reports/trial-balance/export", params: ["from", "to"], file: "balance" },
  ledger: { path: "/admin/finance/reports/general-ledger/export", params: ["accountId", "from", "to"], file: "grand-livre" },
};

export async function GET(req: NextRequest) {
  const spec = ROUTES[req.nextUrl.searchParams.get("kind") ?? ""];
  if (!spec) return new NextResponse("Export inconnu", { status: 404 });
  const token = await getAccess();
  if (!token) return NextResponse.redirect(new URL("/login", req.url));
  const qs = spec.params.map((k) => [k, req.nextUrl.searchParams.get(k)] as const).filter(([, v]) => v).map(([k, v]) => `${k}=${encodeURIComponent(v as string)}`).join("&");
  let res: Response;
  try {
    res = await fetch(`${API_URL}/v1${spec.path}?${qs}`, { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" });
  } catch {
    return new NextResponse("Service indisponible", { status: 503 });
  }
  if (res.status === 403) return new NextResponse("Vous n'avez pas la permission d'exporter.", { status: 403 });
  if (!res.ok) return new NextResponse("Export impossible", { status: res.status });
  const stamp = new Date().toISOString().slice(0, 10);
  return new NextResponse(res.body, { headers: { "Content-Type": "text/csv; charset=utf-8", "Content-Disposition": `attachment; filename="${spec.file}-${stamp}.csv"`, "Cache-Control": "no-store" } });
}
