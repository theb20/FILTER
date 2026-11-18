export const DEFAULT_PREFS = { enabled: true, animations: "auto", seen: [] };

export const isDefaultPrefs = (p) => p.enabled === DEFAULT_PREFS.enabled && p.animations === DEFAULT_PREFS.animations && p.seen.length === 0;

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
 * Préférences du guide, par visiteur (compte connecté ou « guest »). Le NAVIGATEUR sert de cache (affichage immédiat) ;
 * pour un compte connecté, le SERVEUR fait foi (synchronisation entre appareils). Compatible `useSyncExternalStore`.
 *
 * - `set` écrit en local ; avec `sync`, marque « à envoyer » puis envoie (anti-rebond). Un échec laisse le marquage.
 * - `hydrate(distant)` : des changements locaux non envoyés gagnent et sont envoyés ; sinon le serveur remplace le cache.
 *   Premier lancement avec des préférences locales d'avant la synchronisation : les visites vues sont réunies.
 *
 * `sync` = { push(prefs) → Promise<boolean> } ; `storage` est injectable pour les tests.
 */
export function createPrefsStore(key, storage, sync, debounceMs = 600) {
  const listeners = new Set();
  const memory = new Map(); // repli si le stockage est indisponible (navigation privée…)
  let cache = { raw: null, value: DEFAULT_PREFS };
  let timer;
  let inflight = false;
  let again = false;
  const dirtyKey = `${key}:dirty`, syncedKey = `${key}:synced`;

  const store = () => {
    if (storage !== undefined) return storage;
    try { return typeof window === "undefined" ? null : window.localStorage; } catch { return null; }
  };
  const read = (k) => { try { return store()?.getItem(k) ?? memory.get(k) ?? null; } catch { return memory.get(k) ?? null; } };
  const write = (k, v) => {
    if (v === null) memory.delete(k); else memory.set(k, v);
    try { if (v === null) store()?.removeItem(k); else store()?.setItem(k, v); } catch { /* repli mémoire */ }
  };
  const emit = () => listeners.forEach((l) => l());

  const get = () => {
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
    timer = setTimeout(() => { flush(); }, debounceMs);
  };

  const set = (patch) => {
    write(key, JSON.stringify({ ...get(), ...patch }));
    if (sync) { write(dirtyKey, "1"); schedule(); }
    emit();
  };

  const hydrate = (remote) => {
    const local = get();
    const dirty = read(dirtyKey) === "1";
    const neverSynced = read(syncedKey) !== "1";
    if (sync && (dirty || (neverSynced && !isDefaultPrefs(local)))) {
      if (!dirty) {
        const seen = [...new Set([...remote.seen, ...local.seen])];
        write(key, JSON.stringify({ ...local, seen }));
        write(dirtyKey, "1");
        emit();
      }
      flush();
      return;
    }
    write(key, JSON.stringify(remote));
    write(syncedKey, "1");
    emit();
  };

  const subscribe = (cb) => {
    listeners.add(cb);
    const onStorage = (e) => e.key === key && cb(); // synchronisation entre onglets
    if (typeof window !== "undefined") window.addEventListener("storage", onStorage);
    return () => { listeners.delete(cb); if (typeof window !== "undefined") window.removeEventListener("storage", onStorage); };
  };
  return { get, set, subscribe, hydrate, flush };
}
