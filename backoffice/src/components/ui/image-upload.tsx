"use client";
/* eslint-disable @next/next/no-img-element */

import { useRef, useState } from "react";
import { ImagePlus, Loader2, X } from "lucide-react";

/** Réduit une photo avant l'envoi (côté le plus long : 1600 px, JPEG) ; un PNG léger est gardé tel quel (transparence des logos et cachets). */
async function prepare(file: File): Promise<File> {
  if (file.type === "image/png" && file.size < 1.5 * 1024 * 1024) return file;
  if (!/^image\/(png|jpeg|webp)$/.test(file.type)) return file;
  const bmp = await createImageBitmap(file).catch(() => null);
  if (!bmp) return file;
  const scale = Math.min(1, 1600 / Math.max(bmp.width, bmp.height));
  if (scale === 1 && file.size < 1.5 * 1024 * 1024) return file;
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(bmp.width * scale); canvas.height = Math.round(bmp.height * scale);
  const ctx = canvas.getContext("2d");
  if (!ctx) return file;
  ctx.fillStyle = "#fff"; ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(bmp, 0, 0, canvas.width, canvas.height);
  const blob = await new Promise<Blob | null>((ok) => canvas.toBlob(ok, "image/jpeg", 0.85));
  return blob ? new File([blob], file.name.replace(/\.\w+$/, "") + ".jpg", { type: "image/jpeg" }) : file;
}

/** Envoie une image et renvoie son adresse. Lève une Error au message affichable. */
export async function uploadImage(file: File): Promise<string> {
  if (!/^image\/(png|jpeg|webp)$/.test(file.type)) throw new Error("Format non pris en charge : PNG, JPEG ou WebP.");
  const body = new FormData();
  body.append("file", await prepare(file));
  const res = await fetch("/api/media", { method: "POST", body });
  const json = (await res.json().catch(() => null)) as { url?: string; error?: string } | null;
  if (!res.ok || !json?.url) throw new Error(json?.error ?? (res.status === 401 ? "Session expirée : reconnectez-vous." : "Téléversement impossible."));
  return json.url;
}

const btn = "inline-flex items-center gap-1.5 rounded-lg border border-line bg-white px-3 py-1.5 text-sm font-medium text-night transition hover:bg-page disabled:opacity-50";

/** Un champ image : aperçu, « Téléverser », adresse modifiable et retrait. `name` le rend utilisable dans un formulaire classique. */
export function ImagePicker({ value, onChange, name, label, compact = false, disabled = false }: { value: string; onChange?: (url: string) => void; name?: string; label?: string; compact?: boolean; disabled?: boolean }) {
  const [url, setUrl] = useState(value);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const input = useRef<HTMLInputElement>(null);
  const set = (v: string) => { setUrl(v); onChange?.(v); };
  const pick = async (f: File) => {
    setBusy(true); setErr("");
    try { set(await uploadImage(f)); } catch (e) { setErr(e instanceof Error ? e.message : "Téléversement impossible."); } finally { setBusy(false); }
  };
  return (
    <div className="space-y-1.5">
      {label && <p className="text-sm font-medium text-night">{label}</p>}
      <div className="flex items-center gap-3">
        {url ? <img src={url} alt="" className={`${compact ? "h-9 w-9" : "h-16 w-16"} shrink-0 rounded-md border border-line bg-white object-contain`} /> : <span className={`${compact ? "h-9 w-9" : "h-16 w-16"} shrink-0 rounded-md border border-dashed border-line bg-page`} aria-hidden />}
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
          {!disabled && <button type="button" className={btn} onClick={() => input.current?.click()} disabled={busy}>{busy ? <Loader2 size={15} className="animate-spin" aria-hidden /> : <ImagePlus size={15} aria-hidden />}{url ? "Remplacer" : "Téléverser"}</button>}
          {url && !disabled && <button type="button" className={btn} onClick={() => set("")} aria-label="Retirer l'image"><X size={15} aria-hidden /></button>}
          {!compact && <input name={name} value={url} onChange={(e) => set(e.target.value)} disabled={disabled} placeholder="ou collez une adresse (https://…)" className="min-w-[180px] flex-1 rounded-lg border border-line bg-white px-3 py-1.5 text-sm text-night focus:border-link focus:outline-none focus:ring-2 focus:ring-link/20" />}
          {compact && name && <input type="hidden" name={name} value={url} />}
        </div>
      </div>
      <input ref={input} type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={(e) => { const f = e.target.files?.[0]; if (f) void pick(f); e.target.value = ""; }} />
      {err && <p role="alert" className="text-xs text-danger">{err}</p>}
    </div>
  );
}
