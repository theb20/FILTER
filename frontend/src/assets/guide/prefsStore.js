export const DEFAULT_PREFS = { enabled: true, animations: "auto", seen: [] };

/** Lecture tolérante : une valeur corrompue ou absente redonne les valeurs par défaut, jamais une erreur. */
export function parsePrefs(raw) {
  if (!raw) return DEFAULT_PREFS;
  try {
    const v = JSON.parse(raw);
    return {
      enabled: typeof v.enabled === "boolean" ? v.enabled : DEFAULT_PREFS.enabled,
      animations: v.animations === "off" ? "off" : "auto",
      seen: Array.isArray(v.seen) ? v.seen.filter((s) => typeof s === "string") : [],
    };
  } catch {
    return DEFAULT_PREFS;
  }
}

/**
 * Préférences du guide, stockées par visiteur (clé = compte, ou « guest »). Compatible `useSyncExternalStore` :
 * `get` renvoie une référence stable tant que la valeur ne change pas. `storage` est injectable pour les tests.
 */
export function createPrefsStore(key, storage) {
  const listeners = new Set();
  let memory = null; // repli si le stockage est indisponible (navigation privée…)
  let cache = { raw: null, value: DEFAULT_PREFS };

  const store = () => {
    if (storage !== undefined) return storage;
    try { return typeof window === "undefined" ? null : window.localStorage; } catch { return null; }
  };
  const read = () => { try { return store()?.getItem(key) ?? memory; } catch { return memory; } };

  const get = () => {
    const raw = read();
    if (raw !== cache.raw) cache = { raw, value: parsePrefs(raw) };
    return cache.value;
  };
  const set = (patch) => {
    const raw = JSON.stringify({ ...get(), ...patch });
    memory = raw;
    try { store()?.setItem(key, raw); } catch { /* repli mémoire */ }
    listeners.forEach((l) => l());
  };
  const subscribe = (cb) => {
    listeners.add(cb);
    const onStorage = (e) => e.key === key && cb(); // synchronisation entre onglets
    if (typeof window !== "undefined") window.addEventListener("storage", onStorage);
    return () => { listeners.delete(cb); if (typeof window !== "undefined") window.removeEventListener("storage", onStorage); };
  };
  return { get, set, subscribe };
}
