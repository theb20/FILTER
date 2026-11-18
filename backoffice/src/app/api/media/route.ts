import { NextResponse } from "next/server";
import { adminAct, errorMessage } from "@/lib/admin";

export const dynamic = "force-dynamic";

/** Téléversement d'une image par le personnel : le navigateur envoie le fichier ici (même origine), le jeton reste côté serveur. */
export async function POST(req: Request) {
  const form = await req.formData().catch(() => null);
  const file = form?.get("file");
  if (!(file instanceof File)) return NextResponse.json({ error: "Aucun fichier reçu." }, { status: 400 });
  if (file.size > 5 * 1024 * 1024) return NextResponse.json({ error: "Image trop lourde (5 Mo au maximum)." }, { status: 413 });
  const dataBase64 = Buffer.from(await file.arrayBuffer()).toString("base64");
  const r = await adminAct<{ url: string }>("POST", "/admin/uploads", { body: { filename: file.name || "image", dataBase64 } });
  if (!r.ok) return NextResponse.json({ error: errorMessage(r) }, { status: r.status === 401 ? 401 : 422 });
  return NextResponse.json({ url: r.data.url });
}
