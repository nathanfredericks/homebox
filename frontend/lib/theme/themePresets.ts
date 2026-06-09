import type { DaisyTheme } from "../data/themes";

/**
 * Curated MUI theme presets replacing the 31 legacy DaisyUI themes.
 *
 * Each preset defines a primary/secondary seed colour used by both the light
 * and dark colour schemes (MUI derives the rest). `dark` marks presets whose
 * identity is dark-leaning so the preset-class FOUC script / theme picker can
 * default them to the dark scheme. Legacy DaisyUI preference values are mapped
 * to the nearest preset by `mapLegacyTheme()` on first load.
 */
export interface ThemePreset {
  id: string;
  label: string;
  primary: string;
  secondary: string;
  /** Whether this preset reads as a dark identity by default. */
  dark?: boolean;
}

export const themePresets = [
  { id: "homebox", label: "Homebox", primary: "#5b7f67", secondary: "#8aa17d" },
  { id: "garden", label: "Garden", primary: "#5c7f67", secondary: "#a3b18a" },
  { id: "emerald", label: "Emerald", primary: "#10b981", secondary: "#34d399" },
  { id: "corporate", label: "Corporate", primary: "#4b6bfb", secondary: "#7b92b2" },
  { id: "retro", label: "Retro", primary: "#ef9995", secondary: "#a4cbb4" },
  { id: "nord", label: "Nord", primary: "#5e81ac", secondary: "#81a1c1" },
  { id: "synthwave", label: "Synthwave", primary: "#e779c1", secondary: "#58c7f3", dark: true },
  { id: "dracula", label: "Dracula", primary: "#bd93f9", secondary: "#ff79c6", dark: true },
] as const satisfies readonly ThemePreset[];

export type ThemePresetId = (typeof themePresets)[number]["id"];

export const DEFAULT_PRESET_ID: ThemePresetId = "homebox";

const presetById = new Map<string, ThemePreset>(themePresets.map(p => [p.id, p]));

export function getPreset(id: string | undefined): ThemePreset {
  return (id && presetById.get(id)) || presetById.get(DEFAULT_PRESET_ID)!;
}

export function isDarkPreset(id: string | undefined): boolean {
  return !!getPreset(id).dark;
}

/**
 * Map any of the 31 legacy DaisyUI theme names to the nearest curated preset.
 * Unknown values fall back to the Homebox default. Used to migrate existing
 * users' persisted `theme` preference on first load after the migration.
 */
const legacyThemeMap: Record<DaisyTheme, ThemePresetId> = {
  homebox: "homebox",
  garden: "garden",
  forest: "garden",
  light: "homebox",
  cupcake: "retro",
  bumblebee: "retro",
  emerald: "emerald",
  corporate: "corporate",
  business: "corporate",
  wireframe: "corporate",
  synthwave: "synthwave",
  cyberpunk: "synthwave",
  retro: "retro",
  valentine: "retro",
  halloween: "dracula",
  aqua: "nord",
  winter: "nord",
  lofi: "corporate",
  pastel: "retro",
  fantasy: "retro",
  black: "dracula",
  dark: "dracula",
  night: "nord",
  luxury: "dracula",
  dracula: "dracula",
  cmyk: "corporate",
  autumn: "retro",
  acid: "synthwave",
  lemonade: "garden",
  coffee: "retro",
};

export function mapLegacyTheme(theme: string | undefined): ThemePresetId {
  if (!theme) {
    return DEFAULT_PRESET_ID;
  }
  if (presetById.has(theme)) {
    return theme as ThemePresetId;
  }
  return legacyThemeMap[theme as DaisyTheme] ?? DEFAULT_PRESET_ID;
}
