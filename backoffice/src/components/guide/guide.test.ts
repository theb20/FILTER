import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clampIndex, computePlacement, hasPermission, resolveSteps, tourForPath } from "./logic";
import { createPrefsStore, DEFAULT_PREFS, parsePrefs, type PrefsSync } from "./prefs-store";
import type { GuidePrefs } from "./types";
import { TOURS } from "./content";
import { SCENE_META } from "./scenes/meta";
import type { GuideTour } from "./types";

const mem = () => {
  const m = new Map<string, string>();
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v), removeItem: (k: string) => void m.delete(k) };
};

describe("resolveSteps", () => {
  const steps = [{ id: "a", title: "", text: "" }, { id: "b", target: "nav", title: "", text: "" }, { id: "c", target: "absent", title: "", text: "" }];
  it("garde les étapes centrées et celles dont la cible est visible, saute les autres", () => {
    expect(resolveSteps(steps, (t) => t === "nav").map((s) => s.id)).toEqual(["a", "b"]);
  });
  it("renvoie une visite vide si rien n'est visible, sans erreur", () => {
    expect(resolveSteps([steps[2]], () => false)).toEqual([]);
  });
});

describe("permissions et registre", () => {
  const tour = { ...TOURS[0], permission: "catalog:read" } as GuideTour;
  it("joker et permission exacte", () => {
    expect(hasPermission(["*"], "x")).toBe(true);
    expect(hasPermission(["catalog:read"], "catalog:read")).toBe(true);
    expect(hasPermission([], "catalog:read")).toBe(false);
    expect(hasPermission([], undefined)).toBe(true);
  });
  it("tourForPath respecte l'URL et la permission", () => {
    expect(tourForPath([tour], "/dashboard", ["catalog:read"])).toBe(tour);
    expect(tourForPath([tour], "/dashboard", [])).toBeUndefined();
    expect(tourForPath([tour], "/products", ["*"])).toBeUndefined();
  });
  it("les identifiants de visite et d'étape sont uniques", () => {
    expect(new Set(TOURS.map((t) => t.id)).size).toBe(TOURS.length);
    for (const t of TOURS) expect(new Set(t.steps.map((s) => s.id)).size).toBe(t.steps.length);
  });
});

describe("clampIndex", () => {
  it("borne l'index", () => {
    expect(clampIndex(-1, 3)).toBe(0);
    expect(clampIndex(9, 3)).toBe(2);
    expect(clampIndex(0, 0)).toBe(0);
  });
});

describe("computePlacement", () => {
  const view = { width: 1000, height: 700 };
  const bubble = { width: 360, height: 200 };
  it("place sous la cible quand il y a la place", () => {
    const p = computePlacement({ x: 400, y: 100, width: 100, height: 40 }, bubble, view);
    expect(p.side).toBe("bottom");
    expect(p.y).toBeGreaterThan(140);
  });
  it("passe au-dessus si le bas est plein", () => {
    expect(computePlacement({ x: 400, y: 600, width: 100, height: 40 }, bubble, view).side).toBe("top");
  });
  it("reste toujours entièrement dans la fenêtre", () => {
    for (const t of [{ x: 0, y: 0, width: 50, height: 50 }, { x: 950, y: 650, width: 50, height: 50 }, { x: 0, y: 0, width: 260, height: 700 }]) {
      const p = computePlacement(t, bubble, view);
      expect(p.x).toBeGreaterThanOrEqual(0);
      expect(p.y).toBeGreaterThanOrEqual(0);
      expect(p.x + bubble.width).toBeLessThanOrEqual(view.width);
      expect(p.y + bubble.height).toBeLessThanOrEqual(view.height);
    }
  });
});

describe("prefs", () => {
  it("valeurs par défaut si vide ou corrompu", () => {
    expect(parsePrefs(null)).toBe(DEFAULT_PREFS);
    expect(parsePrefs("{pas du json")).toBe(DEFAULT_PREFS);
    expect(parsePrefs('{"enabled":"oui","seen":[1,"a"]}')).toEqual({ enabled: true, animations: "auto", seen: ["a"] });
  });
  it("persiste, notifie et garde une référence stable", () => {
    const s = createPrefsStore("k", mem());
    const cb = vi.fn();
    s.subscribe(cb);
    const first = s.get();
    expect(s.get()).toBe(first);
    s.set({ enabled: false });
    expect(cb).toHaveBeenCalledTimes(1);
    expect(s.get().enabled).toBe(false);
    s.set({ seen: ["dashboard"] });
    expect(s.get()).toEqual({ enabled: false, animations: "auto", seen: ["dashboard"] });
  });
  it("les préférences d'un membre n'affectent pas celles d'un autre", () => {
    const st = mem();
    createPrefsStore("ff-guide:a", st).set({ enabled: false });
    expect(createPrefsStore("ff-guide:b", st).get().enabled).toBe(true);
    expect(createPrefsStore("ff-guide:a", st).get().enabled).toBe(false);
  });
  it("fonctionne même si le stockage lève une erreur", () => {
    const broken = { getItem: () => { throw new Error("x"); }, setItem: () => { throw new Error("x"); }, removeItem: () => { throw new Error("x"); } };
    const s = createPrefsStore("k", broken);
    s.set({ enabled: false });
    expect(s.get().enabled).toBe(false);
  });
});

describe("couverture du backoffice", () => {
  // Toutes les pages du menu et leurs fiches doivent avoir une visite.
  const pages = ["/dashboard", "/customers", "/customers/abc", "/companies", "/companies/abc", "/products", "/products/bulk", "/products/new", "/products/abc", "/categories", "/categories/bulk", "/categories/new", "/categories/abc",
    "/attributes/new", "/attributes/abc", "/brands", "/brands/new", "/brands/abc", "/sectors", "/sectors/new", "/sectors/abc", "/staff", "/staff/new", "/staff/abc", "/roles", "/roles/new", "/roles/abc",
    "/audit", "/settings", "/account", "/quotes", "/quotes/abc", "/engine/sources", "/engine/sources/new", "/engine/sources/abc", "/engine/costs",
    ...["shipping-zones", "promo-codes", "reviews", "quote-requests", "faq", "hero-cards", "home-partners", "pages", "articles", "blog-categories", "blog-tags", "authors", "contact-messages", "newsletter-subscribers"].flatMap((c) => [`/content/${c}`, `/content/${c}/abc`]),
    "/content/settings/site", "/content/settings/home", "/content/settings/commerce"];
  it.each(pages)("%s a une visite", (p) => expect(tourForPath(TOURS, p, ["*"])).toBeDefined());
  it("chaque page n'a qu'une seule visite (pas d'ambiguïté)", () => {
    for (const p of pages) expect(TOURS.filter((t) => t.match(p)).length).toBe(1);
  });
  it("une page inconnue n'en a pas", () => expect(tourForPath(TOURS, "/inconnu", ["*"])).toBeUndefined());
  it("les visites respectent les permissions", () => {
    expect(tourForPath(TOURS, "/audit", ["catalog:read"])).toBeUndefined();
    expect(tourForPath(TOURS, "/audit", ["audit:read"])?.id).toBe("audit");
    expect(tourForPath(TOURS, "/account", [])?.id).toBe("account");
  });
  it("toutes les cibles data-guide existent dans le code", async () => {
    const { execSync } = await import("node:child_process");
    const src = execSync(`grep -rhoE 'guide="[a-z-]+"|data-guide="[a-z-]+"|guide: ?"[a-z-]+"' src --include=*.tsx`, { cwd: process.cwd() }).toString();
    const known = new Set([...src.matchAll(/"([a-z-]+)"/g)].map((m) => m[1]));
    const missing = TOURS.flatMap((t) => t.steps.filter((s) => s.target && !known.has(s.target)).map((s) => `${t.id}:${s.target}`));
    expect(missing).toEqual([]);
  });
});

describe("scènes 3D", () => {
  it("chaque étape avec scène référence une scène connue et légendée", () => {
    const used = TOURS.flatMap((t) => t.steps.filter((s) => s.scene).map((s) => s.scene!));
    expect(used.length).toBeGreaterThanOrEqual(7);
    for (const id of used) expect(SCENE_META[id]?.caption.length).toBeGreaterThan(20);
  });
  it("les trois scènes du backoffice sont utilisées", () => {
    const used = new Set(TOURS.flatMap((t) => t.steps.map((s) => s.scene)));
    for (const id of Object.keys(SCENE_META)) expect(used.has(id as never)).toBe(true);
  });
});

describe("synchronisation avec le serveur", () => {
  beforeEach(() => { vi.useFakeTimers(); });
  afterEach(() => { vi.useRealTimers(); });
  const remote = (over: Partial<GuidePrefs> = {}): GuidePrefs => ({ enabled: true, animations: "auto", seen: [], ...over });
  const fakeSync = () => {
    const calls: GuidePrefs[] = [];
    let result: boolean | "throw" = true;
    const sync: PrefsSync = { push: async (p) => { calls.push(p); if (result === "throw") throw new Error("réseau"); return result; } };
    return { sync, calls, set ok(v: boolean | "throw") { result = v; } };
  };

  it("les changements rapprochés partent en un seul envoi (anti-rebond)", async () => {
    const f = fakeSync();
    const s = createPrefsStore("k", mem(), f.sync, 600);
    s.set({ seen: ["a"] }); s.set({ seen: ["a", "b"] }); s.set({ enabled: false });
    expect(f.calls).toHaveLength(0);
    await vi.advanceTimersByTimeAsync(700);
    expect(f.calls).toHaveLength(1);
    expect(f.calls[0]).toEqual({ enabled: false, animations: "auto", seen: ["a", "b"] });
  });

  it("le serveur fait foi : sans changement local, il remplace le cache sans renvoyer", async () => {
    const f = fakeSync();
    const st = mem();
    const s = createPrefsStore("k", st, f.sync);
    s.hydrate(remote({ enabled: false, seen: ["x"] }));
    await vi.advanceTimersByTimeAsync(2000);
    expect(s.get()).toEqual({ enabled: false, animations: "auto", seen: ["x"] });
    expect(f.calls).toHaveLength(0);
  });

  it("un envoi échoué reste à envoyer et gagne sur le serveur au prochain chargement", async () => {
    const f = fakeSync();
    const st = mem();
    const s1 = createPrefsStore("k", st, f.sync);
    s1.hydrate(remote());
    f.ok = false;
    s1.set({ seen: ["dashboard"] });
    await vi.advanceTimersByTimeAsync(700);
    expect(f.calls).toHaveLength(1);
    // Nouveau chargement de page : le serveur ne connaît pas encore « dashboard ».
    f.ok = true;
    const s2 = createPrefsStore("k", st, f.sync);
    s2.hydrate(remote());
    await vi.advanceTimersByTimeAsync(10);
    expect(s2.get().seen).toEqual(["dashboard"]); // le choix local n'est pas perdu
    expect(f.calls.at(-1)?.seen).toEqual(["dashboard"]); // et il est renvoyé
    // une fois envoyé, le serveur refait foi
    const s3 = createPrefsStore("k", st, f.sync);
    s3.hydrate(remote({ seen: ["dashboard", "audit"] }));
    expect(s3.get().seen).toEqual(["dashboard", "audit"]);
  });

  it("une exception réseau est traitée comme un échec, sans casser l'interface", async () => {
    const f = fakeSync();
    const s = createPrefsStore("k", mem(), f.sync);
    f.ok = "throw";
    s.set({ enabled: false });
    await vi.advanceTimersByTimeAsync(700);
    expect(s.get().enabled).toBe(false);
  });

  it("migration : des préférences locales d'avant la synchronisation sont réunies avec celles du serveur", async () => {
    const f = fakeSync();
    const st = mem();
    createPrefsStore("k", st).set({ enabled: false, seen: ["a", "b"] }); // ancien cache, sans synchronisation
    const s = createPrefsStore("k", st, f.sync);
    s.hydrate(remote({ seen: ["b", "c"] }));
    await vi.advanceTimersByTimeAsync(10);
    expect(s.get().enabled).toBe(false);
    expect([...s.get().seen].sort()).toEqual(["a", "b", "c"]);
    expect(f.calls.at(-1)?.seen.sort()).toEqual(["a", "b", "c"]);
  });

  it("un changement pendant l'envoi n'est pas perdu", async () => {
    let release: (v: boolean) => void = () => {};
    const calls: GuidePrefs[] = [];
    const sync: PrefsSync = { push: (p) => { calls.push(p); return new Promise<boolean>((r) => { release = r; }); } };
    const s = createPrefsStore("k", mem(), sync, 100);
    s.set({ seen: ["a"] });
    await vi.advanceTimersByTimeAsync(150);
    s.set({ seen: ["a", "b"] }); // pendant que le premier envoi est en cours
    await vi.advanceTimersByTimeAsync(150);
    release(true);
    await vi.advanceTimersByTimeAsync(200);
    expect(calls.at(-1)?.seen).toEqual(["a", "b"]);
  });
});
