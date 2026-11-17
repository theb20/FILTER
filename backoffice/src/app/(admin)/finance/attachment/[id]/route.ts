import { NextResponse, type NextRequest } from "next/server";
import { API_URL } from "@/lib/api";
import { getAccess } from "@/lib/session";

// Téléchargement d'une pièce justificative : le jeton reste côté serveur ; le type est celui vérifié par le service sur le CONTENU.
export async function GET(req: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  if (!/^[0-9a-f-]{36}$/i.test(id)) return new NextResponse("Pièce inconnue", { status: 404 });
  const token = await getAccess();
  if (!token) return NextResponse.redirect(new URL("/login", req.url));
  let res: Response;
  try {
    res = await fetch(`${API_URL}/v1/admin/finance/attachments/${id}`, { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" });
  } catch {
    return new NextResponse("Service indisponible", { status: 503 });
  }
  if (res.status === 403) return new NextResponse("Vous n'avez pas la permission de consulter cette pièce.", { status: 403 });
  if (!res.ok) return new NextResponse("Pièce introuvable", { status: res.status === 404 ? 404 : 502 });
  const j = (await res.json()) as { attachment: { filename: string; contentType: string }; dataBase64: string };
  const bytes = Buffer.from(j.dataBase64, "base64");
  const name = encodeURIComponent(j.attachment.filename);
  return new NextResponse(bytes, {
    headers: {
      "Content-Type": j.attachment.contentType,
      "Content-Disposition": `attachment; filename*=UTF-8''${name}`,
      "X-Content-Type-Options": "nosniff",
      "Content-Security-Policy": "default-src 'none'; sandbox",
      "Cache-Control": "private, no-store",
    },
  });
}
