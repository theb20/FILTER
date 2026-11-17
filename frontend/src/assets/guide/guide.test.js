import { describe, expect, it, vi } from "vitest";
import { execSync } from "node:child_process";
import { clampIndex, computePlacement, resolveSteps, tourForPath } from "./logic.js";
import { createPrefsStore, DEFAULT_PREFS, parsePrefs } from "./prefsStore.js";
import { TOURS } from "./content/tours.js";
import en from "./i18n/en.js";
import fr from "./i18n/fr.js";
import es from "./i18n/es.js";

const mem = () => {
  const m = new Map();
  return { getItem: (k) => m.get(k) ?? null, setItem: (k, v) => void m.set(k, v) };
};

describe("resolveSteps", () => {
  const steps = [{ id: "a" }, { id: "b", target: "cart" }, { id: "c", target: "absent" }];
  it("garde les étapes centrées et celles dont la cible est visible, saute les autres", () => {
    expect(resolveSteps(steps, (t) => t === "cart").map((s) => s.id)).toEqual(["a", "b"]);
  });
  it("renvoie une visite vide si rien n'est visible, sans erreur", () => {
    expect(resolveSteps([steps[2]], () => false)).toEqual([]);
  });
});

describe("clampIndex / computePlacement", () => {
  const view = { width: 1000, height: 700 };
  const bubble = { width: 360, height: 200 };
  it("borne l'index", () => {
    expect(clampIndex(-1, 3)).toBe(0);
    expect(clampIndex(9, 3)).toBe(2);
    expect(clampIndex(0, 0)).toBe(0);
  });
  it("place sous la cible, ou au-dessus si le bas est plein", () => {
    expect(computePlacement({ x: 400, y: 100, width: 100, height: 40 }, bubble, view).side).toBe("bottom");
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

describe("préférences", () => {
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
  });
  it("les préférences d'un compte n'affectent pas celles d'un autre ni celles d'un visiteur", () => {
    const st = mem();
    createPrefsStore("ff-guide:a", st).set({ enabled: false });
    expect(createPrefsStore("ff-guide:b", st).get().enabled).toBe(true);
    expect(createPrefsStore("ff-guide:guest", st).get().enabled).toBe(true);
  });
  it("fonctionne même si le stockage lève une erreur", () => {
    const broken = { getItem: () => { throw new Error("x"); }, setItem: () => { throw new Error("x"); } };
    const s = createPrefsStore("k", broken);
    s.set({ enabled: false });
    expect(s.get().enabled).toBe(false);
  });
});

describe("couverture de la boutique", () => {
  const pages = ["/", "/category", "/card", "/basket", "/quote", "/contact", "/profil", "/blog", "/blog/mon-article", "/brands", "/terms", "/refund", "/privacy"];
  it.each(pages)("%s a une visite", (p) => expect(tourForPath(TOURS, p)).toBeDefined());
  it("chaque page n'a qu'une seule visite", () => {
    for (const p of pages) expect(TOURS.filter((t) => t.match(p)).length).toBe(1);
  });
  it("une page sans guide (connexion, 404) n'en a pas", () => {
    expect(tourForPath(TOURS, "/sign")).toBeUndefined();
    expect(tourForPath(TOURS, "/inconnu")).toBeUndefined();
  });
  it("identifiants de visites et d'étapes uniques", () => {
    expect(new Set(TOURS.map((t) => t.id)).size).toBe(TOURS.length);
    for (const t of TOURS) expect(new Set(t.steps.map((s) => s.id)).size).toBe(t.steps.length);
  });
  it("aucune visite ne se lance toute seule (le visiteur garde la main)", () => {
    expect(TOURS.some((t) => t.auto)).toBe(false);
  });
  it("les visites qui exigent des données ne se lancent pas depuis le profil", () => {
    const startable = TOURS.filter((t) => t.match(t.path)).map((t) => t.id);
    expect(startable).not.toContain("product");
    expect(startable).not.toContain("post");
  });
});

describe("textes en trois langues", () => {
  const keys = TOURS.flatMap((t) => [`guide.${t.id}.title`, `guide.${t.id}.summary`, ...t.steps.flatMap((s) => [s.titleKey, s.textKey])]);
  const scenes = ["guide.scene.filter-anatomy", ...["media", "core", "caps"].map((k) => `guide.scene.filter-anatomy.${k}`)];
  it.each([["en", en], ["fr", fr], ["es", es]])("%s : toutes les clés utilisées existent et ne sont pas vides", (_, d) => {
    for (const k of [...keys, ...scenes]) expect(d[k]?.length, k).toBeGreaterThan(3);
  });
  it("fr et es ont exactement les mêmes clés que en", () => {
    expect(Object.keys(fr).sort()).toEqual(Object.keys(en).sort());
    expect(Object.keys(es).sort()).toEqual(Object.keys(en).sort());
  });
  it("les variables {n} et {total} sont présentes dans chaque langue", () => {
    for (const d of [en, fr, es]) expect(d["guide.ui.stepOf"]).toMatch(/\{n\}.*\{total\}/);
  });
});

describe("repères dans le code", () => {
  it("chaque cible de visite existe dans les composants", () => {
    const src = execSync(`grep -rhoE '(data-guide|guide)="[a-z-]+"' src`, { cwd: process.cwd() }).toString();
    const known = new Set([...src.matchAll(/"([a-z-]+)"/g)].map((m) => m[1]));
    const missing = TOURS.flatMap((t) => t.steps.filter((s) => s.target && !known.has(s.target)).map((s) => `${t.id}:${s.target}`));
    expect(missing).toEqual([]);
  });
});
