import { describe, expect, it, vi } from "vitest";
import { clampIndex, computePlacement, hasPermission, resolveSteps, tourForPath } from "./logic";
import { createPrefsStore, DEFAULT_PREFS, parsePrefs } from "./prefs-store";
import { TOURS } from "./content";
import { SCENE_META } from "./scenes/meta";
import type { GuideTour } from "./types";

const mem = () => {
  const m = new Map<string, string>();
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v) };
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
    const broken = { getItem: () => { throw new Error("x"); }, setItem: () => { throw new Error("x"); } };
    const s = createPrefsStore("k", broken);
    s.set({ enabled: false });
    expect(s.get().enabled).toBe(false);
  });
});

describe("couverture du backoffice", () => {
  // Toutes les pages du menu et leurs fiches doivent avoir une visite.
  const pages = ["/dashboard", "/customers", "/customers/abc", "/companies", "/companies/abc", "/products", "/products/new", "/products/abc", "/categories", "/categories/new", "/categories/abc",
    "/attributes/new", "/attributes/abc", "/brands", "/brands/new", "/brands/abc", "/sectors", "/sectors/new", "/sectors/abc", "/staff", "/staff/new", "/staff/abc", "/roles", "/roles/new", "/roles/abc",
    "/audit", "/settings", "/account",
    ...["shipping-zones", "promo-codes", "reviews", "quote-requests", "faq", "pages", "articles", "blog-categories", "blog-tags", "authors", "contact-messages", "newsletter-subscribers"].flatMap((c) => [`/content/${c}`, `/content/${c}/abc`]),
    "/content/settings/site", "/content/settings/commerce"];
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
