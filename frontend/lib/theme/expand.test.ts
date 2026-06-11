import { describe, expect, it } from "vitest";
import { expandThemeColors, hexToHsl, hslToHex } from "./expand";

const homebox = {
  background: "#ffffff",
  foreground: "#333333",
  primary: "#5c7f67",
  secondary: "#2d2f28",
  accent: "#ecf4e7",
  destructive: "#f87272",
  radius: "0.5rem",
};

function lightnessOf(triple: string): number {
  return parseFloat(triple.split(" ")[2]!);
}

describe("hex/hsl conversion", () => {
  it("round-trips", () => {
    for (const hex of ["#ffffff", "#000000", "#5c7f67", "#f87272", "#09e9f1"]) {
      expect(hslToHex(hexToHsl(hex))).toBe(hex);
    }
  });
});

describe("expandThemeColors", () => {
  const vars = expandThemeColors(homebox);

  it("produces the full variable set", () => {
    const expected = [
      "background",
      "background-accent",
      "foreground",
      "card",
      "card-foreground",
      "popover",
      "popover-foreground",
      "muted",
      "muted-foreground",
      "border",
      "input",
      "ring",
      "primary",
      "primary-foreground",
      "secondary",
      "secondary-foreground",
      "accent",
      "accent-foreground",
      "destructive",
      "destructive-foreground",
      "sidebar-background",
      "sidebar-foreground",
      "sidebar-primary",
      "sidebar-primary-foreground",
      "sidebar-accent",
      "sidebar-accent-foreground",
      "sidebar-border",
      "sidebar-ring",
      "radius",
    ];
    expect(Object.keys(vars).sort()).toEqual(expected.sort());
  });

  it("anchors close to the original hand-tuned homebox palette", () => {
    // Original values: muted 90%, border/accent 81%, primary-fg light (89%),
    // destructive-fg dark (14%). Derivations should land in the same zones.
    expect(vars.background).toBe("0 0% 100%");
    expect(vars.card).toBe(vars.background);
    expect(vars.popover).toBe(vars.background);
    expect(lightnessOf(vars.muted!)).toBeGreaterThan(85);
    expect(lightnessOf(vars.muted!)).toBeLessThan(95);
    expect(lightnessOf(vars.border!)).toBeGreaterThan(75);
    expect(lightnessOf(vars.border!)).toBeLessThan(88);
    expect(vars.ring).toBe(vars.primary);
    // Dark-ish green primary gets a light foreground...
    expect(lightnessOf(vars["primary-foreground"]!)).toBeGreaterThan(80);
    // ...while the light red destructive gets a dark one (matches original).
    expect(lightnessOf(vars["destructive-foreground"]!)).toBeLessThan(25);
  });

  it("lightens neutral shades on near-black backgrounds", () => {
    const black = expandThemeColors({ ...homebox, background: "#000000" });
    expect(lightnessOf(black.muted!)).toBeGreaterThan(0);
    expect(lightnessOf(black.border!)).toBeGreaterThan(lightnessOf(black.muted!));
  });

  it("defaults radius", () => {
    const noRadius = expandThemeColors({ ...homebox, radius: undefined });
    expect(noRadius.radius).toBe("0.5rem");
  });
});
