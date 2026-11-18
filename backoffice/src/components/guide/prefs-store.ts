import type { GuidePrefs } from "./types";

export const DEFAULT_PREFS: GuidePrefs = { enabled: true, animations: "auto", seen: [] };

export const isDefaultPrefs = (p: GuidePrefs) => p.enabled === DEFAULT_PREFS.enabled && p.animations === DEFAULT_PREFS.animations && p.seen.length === 0;

/** Lecture tolérante : une valeur corrompue ou absente redonne les valeurs par défaut, jamais une erreur. */
export function parsePrefs(raw: string | null | undefined): GuidePrefs {
  if (!raw) return DEFAULT_PREFS;
  try {
    const v = JSON.parse(raw) as Partial<GuidePrefs>;
    return {
      enabled: typeof v.enabled === "boolean" ? v.enabled : DEFAULT_PREFS.enabled,
      animations: v.animations === "off" ? "off" : "auto",
      seen: Array.isArray(v.seen) ? v.seen.filter((s): s is string => typeof s === "string") : [],
    };
  } catch {
    return DEFAULT_PREFS;
  }
}

type Storage_ = Pick<Storage, "getItem" | "setItem" | "removeItem">;
/** Envoi au serveur : `false` ou une exception = échec (les changements restent « à envoyer »). */
export type PrefsSync = { push: (p: GuidePrefs) => Promise<boolean | void> };

/**
 * Préférences du guide, par membre. Le NAVIGATEUR sert de cache (affichage immédiat, hors ligne) ; le SERVEUR fait foi
 * (synchronisation entre appareils). Compatible `useSyncExternalStore` : `get` renvoie une référence stable tant que la valeur ne change pas.
 *
 * - `set` écrit en local, marque « à envoyer » puis envoie (anti-rebond) ; en cas d'échec le marquage reste.
 * - `hydrate(distant)` : si des changements locaux n'ont pas été envoyés, ils gagnent et sont envoyés ; sinon la valeur du serveur remplace le cache.
 *   Premier lancement sur un navigateur qui avait déjà des préférences locales (avant la synchronisation) : les visites vues sont réunies.
 */
export function createPrefsStore(key: string, storage?: Storage_ | null, sync?: PrefsSync, debounceMs = 600) {
  const listeners = new Set<() => void>();
  const memory = new Map<string, string>(); // repli si le stockage est indisponible (navigation privée…)
  let cache: { raw: string | null; value: GuidePrefs } = { raw: null, value: DEFAULT_PREFS };
  let timer: ReturnType<typeof setTimeout> | undefined;
  let inflight = false;
  let again = false;

  const store = (): Storage_ | null => {
    if (storage !== undefined) return storage;
    try { return typeof window === "undefined" ? null : window.localStorage; } catch { return null; }
  };
  const read = (k: string) => { try { return store()?.getItem(k) ?? memory.get(k) ?? null; } catch { return memory.get(k) ?? null; } };
  const write = (k: string, v: string | null) => {
    if (v === null) memory.delete(k); else memory.set(k, v);
    try { if (v === null) store()?.removeItem(k); else store()?.setItem(k, v); } catch { /* repli mémoire */ }
  };
  const emit = () => listeners.forEach((l) => l());
  const dirtyKey = `${key}:dirty`, syncedKey = `${key}:synced`;

  const get = (): GuidePrefs => {
    const raw = read(key);
    if (raw !== cache.raw) cache = { raw, value: parsePrefs(raw) };
    return cache.value;
  };

  const flush = async () => {
    if (!sync) return;
    if (inflight) { again = true; return; }
    const snapshot = get();
    inflight = true;
    let ok = false;
    try { ok = (await sync.push(snapshot)) !== false; } catch { ok = false; }
    inflight = false;
    if (ok) {
      write(syncedKey, "1");
      if (get() === snapshot) write(dirtyKey, null); // rien n'a changé pendant l'envoi
    }
    if (again) { again = false; schedule(); }
  };
  const schedule = () => {
    if (!sync) return;
    clearTimeout(timer);
    timer = setTimeout(() => { void flush(); }, debounceMs);
  };

  const set = (patch: Partial<GuidePrefs>) => {
    write(key, JSON.stringify({ ...get(), ...patch }));
    if (sync) { write(dirtyKey, "1"); schedule(); }
    emit();
  };

  const hydrate = (remote: GuidePrefs) => {
    const local = get();
    const dirty = read(dirtyKey) === "1";
    const neverSynced = read(syncedKey) !== "1";
    if (sync && (dirty || (neverSynced && !isDefaultPrefs(local)))) {
      if (!dirty) {
        // Préférences locales d'avant la synchronisation : on réunit les visites vues, le choix local fait foi pour le reste.
        const seen = [...new Set([...remote.seen, ...local.seen])];
        write(key, JSON.stringify({ ...local, seen }));
        write(dirtyKey, "1");
        emit();
      }
      void flush();
      return;
    }
    write(key, JSON.stringify(remote));
    write(syncedKey, "1");
    emit();
  };

  const subscribe = (cb: () => void) => {
    listeners.add(cb);
    const onStorage = (e: StorageEvent) => e.key === key && cb(); // synchronisation entre onglets
    if (typeof window !== "undefined") window.addEventListener("storage", onStorage);
    return () => {
      listeners.delete(cb);
      if (typeof window !== "undefined") window.removeEventListener("storage", onStorage);
    };
  };
  return { get, set, subscribe, hydrate, flush };
}
