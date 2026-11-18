import { describe, expect, it } from "vitest";
import { localized, matchHeaders, norm, optLocalized, parseFulfillment, parsePriceMode, toInt } from "./bulk-shared";

const cols = [{ id: "slug", title: "Identifiant d'URL" }, { id: "nameFr", title: "Nom (FR)", aliases: ["Nom"] }, { id: "priceAmount", title: "Prix HT" }];

describe("édition en masse", () => {
  it("norm ignore accents, casse et ponctuation", () => expect(norm("  Identifiant d'URL ")).toBe("identifiantdurl"));
  it("associe les en-têtes aux colonnes", () => expect(matchHeaders(["Nom", "prix ht", "Inconnu", "SLUG"], cols)).toEqual(["nameFr", "priceAmount", null, "slug"]));
  it("toInt", () => {
    expect(toInt("1 250")).toBe(1250);
    expect(toInt("")).toBeNull();
    expect(toInt(null)).toBeNull();
    expect(toInt("12,5")).toBeNaN();
    expect(toInt(-3)).toBeNaN();
  });
  it("modes de prix et de commande", () => {
    expect(parsePriceMode("Sur devis")).toBe("on_quote");
    expect(parsePriceMode("")).toBe("fixed");
    expect(parsePriceMode("gratuit")).toBeNull();
    expect(parseFulfillment("Sur commande")).toBe("made_to_order");
    expect(parseFulfillment("stock")).toBe("stock");
    expect(parseFulfillment("x")).toBeNull();
  });
  it("textes par langue", () => {
    expect(localized("Filtre", "", "Filtro")).toEqual({ fr: "Filtre", es: "Filtro" });
    expect(optLocalized("", "", "")).toBeNull();
  });
});
