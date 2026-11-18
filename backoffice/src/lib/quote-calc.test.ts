import { describe, expect, it } from "vitest";
import { computeTotals, lineTotal, readiness, type QDraft } from "./quote-calc";

describe("chiffrage du devis (même résultat que le service)", () => {
  it("calcule les totaux comme le test Go", () => {
    const t = computeTotals([
      { reference: "A", quantity: 2, unitPrice: 1000, discountPct: 10 },
      { reference: "B", quantity: 1, unitPrice: 500 },
      { reference: "C", quantity: 1, unitPrice: 9999, optional: true },
      { reference: "D", quantity: 3 },
    ], 5, 100, 18);
    expect(t).toMatchObject({ subtotal: 2300, discountAmount: 115, vatAmount: 411, total: 2696, priced: 2, missing: 1 });
  });
  it("autres frais et acompte comme le service", () => {
    const t = computeTotals([{ reference: "A", quantity: 10, unitPrice: 1000 }], 0, 500, 18, 500, 3000);
    expect(t).toMatchObject({ subtotal: 10000, vatAmount: 1980, total: 12980, remaining: 9980 });
  });
  it("une ligne sans prix n'a pas de total", () => expect(lineTotal({ reference: "X", quantity: 4 })).toBeNull());
  it("n'empêche l'envoi que sur les règles du service", () => {
    const ok: QDraft = { number: "DEV-1", version: "1", incoterm: "", correspondent: "", other: 0, acompte: 0, annexes: [], lines: [{ reference: "A", quantity: 1, unitPrice: 100 }], discountPct: 0, shipping: 0, vatRate: 18, validUntil: "2099-01-01", deliveryPlace: "", deliveryDelay: "", paymentTerms: "", intro: "Bonjour", internalNote: "", followUpAt: "" };
    expect(readiness(ok).blocking).toEqual([]);
    // Sans message d'accompagnement : conseillé, jamais bloquant.
    expect(readiness({ ...ok, intro: " " }).blocking).toEqual([]);
    expect(readiness({ ...ok, intro: " " }).advice.length).toBeGreaterThan(0);
    expect(readiness({ ...ok, validUntil: "2001-01-01" }).blocking[0]).toMatch(/dépassée/);
    expect(readiness({ ...ok, validUntil: "" }).blocking[0]).toMatch(/validité/);
    expect(readiness({ ...ok, number: "" }).blocking[0]).toMatch(/numéro/);
    expect(readiness({ ...ok, lines: [{ reference: "A", quantity: 1 }] }).blocking.length).toBeGreaterThan(0);
    // Une ligne optionnelle non chiffrée ne bloque pas, mais il faut au moins une ligne chiffrée.
    expect(readiness({ ...ok, lines: [{ reference: "A", quantity: 1, unitPrice: 100 }, { reference: "B", quantity: 1, optional: true }] }).blocking).toEqual([]);
  });
});
