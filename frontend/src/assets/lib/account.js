// @ts-check
// Compte du client : photo de profil, couverture, appareils connectés, historique des connexions.
// Tous les appels passent par useAuth().call (renouvellement automatique du jeton d'accès sur 401).
import { useEffect, useState } from "react";
import { useAuth } from "../store/authStore.js";

const call = (/** @type {string} */ path, /** @type {any} */ opts) => useAuth.getState().call(path, opts);

// ── Images (photo de profil, couverture) ────────────────────────────────────────────────────────────────────

/** Cadrage et dimensions d'envoi : carré pour la photo, bandeau 3:1 pour la couverture. */
const SHAPES = { avatar: { w: 512, h: 512 }, cover: { w: 1500, h: 500 } };
const MAX_BYTES = 600 * 1024; // limite du serveur

/**
 * Redimensionne et recadre (au centre) l'image choisie, puis la réencode en WebP (ou JPEG si le navigateur ne sait
 * pas produire du WebP). Le réencodage retire aussi les métadonnées de la photo (dont sa position GPS).
 * @param {File} file @param {"avatar" | "cover"} kind @returns {Promise<Blob>}
 */
export async function prepareImage(file, kind) {
  if (!/^image\/(png|jpeg|webp|gif|heic|heif|avif)$/.test(file.type)) throw new Error("type");
  const bmp = await createImageBitmap(file);
  const { w, h } = SHAPES[kind];
  const ratio = w / h;
  let sw = bmp.width;
  let sh = bmp.width / ratio;
  if (sh > bmp.height) {
    sh = bmp.height;
    sw = bmp.height * ratio;
  }
  const scale = Math.min(1, w / sw); // jamais d'agrandissement
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(sw * scale);
  canvas.height = Math.round(sh * scale);
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas");
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(bmp, (bmp.width - sw) / 2, (bmp.height - sh) / 2, sw, sh, 0, 0, canvas.width, canvas.height);
  bmp.close();
  for (const q of [0.86, 0.75, 0.62, 0.5]) {
    let blob = await new Promise((res) => canvas.toBlob(res, "image/webp", q));
    if (!blob || blob.type !== "image/webp") blob = await new Promise((res) => canvas.toBlob(res, "image/jpeg", q));
    if (blob && blob.size <= MAX_BYTES) return blob;
  }
  throw new Error("size");
}

/** @param {Blob} blob @returns {Promise<string>} contenu en base64 (sans préfixe data:) */
const toBase64 = (blob) =>
  new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result).split(",")[1] ?? "");
    r.onerror = () => reject(r.error);
    r.readAsDataURL(blob);
  });

/** Envoie l'image puis recharge le compte (la date de l'image sert de clé de cache). @param {"avatar"|"cover"} kind @param {File} file */
export async function uploadMyImage(kind, file) {
  const blob = await prepareImage(file, kind);
  const r = await call(`/me/images/${kind}`, { method: "PUT", body: { data: await toBase64(blob) } });
  if (r.ok) await refreshMe();
  return r;
}

/** @param {"avatar"|"cover"} kind */
export async function deleteMyImage(kind) {
  const r = await call(`/me/images/${kind}`, { method: "DELETE" });
  if (r.ok) await refreshMe();
  return r;
}

export async function refreshMe() {
  const r = await call("/me");
  if (r.ok) useAuth.getState().setUser(r.data);
  return r;
}

/** Cache des images lues (clé : type + date de modification) : une seule lecture par version. @type {Map<string, Promise<string | null>>} */
const imageCache = new Map();

/**
 * Adresse (data:) de la photo ou de la couverture du compte connecté ; null s'il n'y en a pas.
 * @param {"avatar" | "cover"} kind @returns {string | null}
 */
export function useMyImage(kind) {
  const stamp = useAuth((s) => (s.user ? s.user[`${kind}UpdatedAt`] : null));
  const [url, setUrl] = useState(/** @type {string | null} */ (null));
  useEffect(() => {
    if (!stamp) {
      setUrl(null);
      return undefined;
    }
    const key = `${kind}:${stamp}`;
    if (!imageCache.has(key)) {
      imageCache.set(key, call(`/me/images/${kind}`).then((r) => (r.ok ? `data:${r.data.contentType};base64,${r.data.data}` : null)));
    }
    let live = true;
    imageCache.get(key)?.then((u) => live && setUrl(u));
    return () => {
      live = false;
    };
  }, [kind, stamp]);
  return url;
}

// ── Sécurité : appareils et historique ─────────────────────────────────────────────────────────────────────

export const fetchSessions = () => call("/me/sessions");
/** @param {string} id */
export const revokeSession = (id) => call(`/me/sessions/${encodeURIComponent(id)}`, { method: "DELETE" });
export const revokeOtherSessions = () => call("/me/sessions/revoke-others", { method: "POST" });
/** @param {number} page */
export const fetchLoginActivity = (page = 1) => call(`/me/login-activity?page=${page}&pageSize=10`);

/** Libellé de lieu approximatif : « Abidjan, Côte d'Ivoire ». @param {{city?: string, country?: string} | null | undefined} l */
export const placeLabel = (l) => (l ? [l.city, l.country].filter(Boolean).join(", ") : "");
