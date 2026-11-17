import type { GuidePrefs } from "./types";

export const DEFAULT_PREFS: GuidePrefs = { enabled: true, animations: "auto", seen: [] };

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

type Storage_ = Pick<Storage, "getItem" | "setItem">;

/**
 * Préférences du guide, stockées par membre (clé = id). `useSyncExternalStore`-compatible :
 * `get` renvoie une référence stable tant que la valeur ne change pas.
 * Stockage local pour l'instant ; un endpoint de préférences du gateway le remplacera sans changer cette API.
 */
export function createPrefsStore(key: string, storage?: Storage_ | null) {
  const listeners = new Set<() => void>();
  let memory: string | null = null; // repli si le stockage est indisponible (navigation privée…)
  let cache: { raw: string | null; value: GuidePrefs } = { raw: null, value: DEFAULT_PREFS };

  const store = (): Storage_ | null => {
    if (storage !== undefined) return storage;
    try { return typeof window === "undefined" ? null : window.localStorage; } catch { return null; }
  };
  const read = () => {
    try { return store()?.getItem(key) ?? memory; } catch { return memory; }
  };
  const emit = () => listeners.forEach((l) => l());

  const get = (): GuidePrefs => {
    const raw = read();
    if (raw !== cache.raw) cache = { raw, value: parsePrefs(raw) };
    return cache.value;
  };
  const set = (patch: Partial<GuidePrefs>) => {
    const raw = JSON.stringify({ ...get(), ...patch });
    memory = raw;
    try { store()?.setItem(key, raw); } catch { /* repli mémoire */ }
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
  return { get, set, subscribe };
}
