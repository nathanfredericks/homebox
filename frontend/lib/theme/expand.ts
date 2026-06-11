/**
 * A theme's palette is defined by six core colors (hex) plus a corner radius;
 * every CSS variable the app consumes is derived from those here. This is the
 * single source of truth for the derivation — the global style emission, the
 * theme picker preview cards and the admin editor live preview all call it.
 */

export type CoreColors = {
  background: string;
  foreground: string;
  primary: string;
  secondary: string;
  accent: string;
  destructive: string;
};

export type ThemeColors = CoreColors & { radius?: string };

export const CORE_COLOR_KEYS: (keyof CoreColors)[] = [
  "background",
  "foreground",
  "primary",
  "secondary",
  "accent",
  "destructive",
];

type Hsl = { h: number; s: number; l: number };

export function hexToHsl(hex: string): Hsl {
  const m = hex.trim().match(/^#?([0-9a-f]{6})$/i);
  if (!m) {
    return { h: 0, s: 0, l: 0 };
  }
  const r = parseInt(m[1]!.slice(0, 2), 16) / 255;
  const g = parseInt(m[1]!.slice(2, 4), 16) / 255;
  const b = parseInt(m[1]!.slice(4, 6), 16) / 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const l = (max + min) / 2;

  if (max === min) {
    return { h: 0, s: 0, l: l * 100 };
  }

  const d = max - min;
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
  let h: number;
  switch (max) {
    case r:
      h = (g - b) / d + (g < b ? 6 : 0);
      break;
    case g:
      h = (b - r) / d + 2;
      break;
    default:
      h = (r - g) / d + 4;
  }

  return { h: h * 60, s: s * 100, l: l * 100 };
}

export function hslToHex({ h, s, l }: Hsl): string {
  const sn = s / 100;
  const ln = l / 100;
  const a = sn * Math.min(ln, 1 - ln);
  const f = (n: number) => {
    const k = (n + h / 30) % 12;
    const c = ln - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
    return Math.round(255 * c)
      .toString(16)
      .padStart(2, "0");
  };
  return `#${f(0)}${f(8)}${f(4)}`;
}

function triple({ h, s, l }: Hsl): string {
  const round = (n: number) => Math.round(n * 10) / 10;
  return `${round(h)} ${round(s)}% ${round(l)}%`;
}

/**
 * Neutral shade variants (borders, muted surfaces) move lightness one or two
 * steps away from the background: darker for light/mid backgrounds, lighter
 * for dark ones — mirroring how the original hand-written themes were tuned
 * (white 100% → 90% → 81%; black 0% → 5% → 10%; dracula 18% → 23% → 28%).
 * The direction is fixed up front so consecutive steps never bounce back
 * across the threshold and collapse into the background.
 */
function shade(color: Hsl, steps: number): Hsl {
  const direction = color.l > 25 ? -1 : 1;
  let l = color.l;
  for (let i = 0; i < steps; i++) {
    l += direction * (4 + l * 0.05);
  }
  return { ...color, l: Math.min(100, Math.max(0, l)) };
}

/** WCAG relative luminance of a hex color (0..1). */
function luminance(hex: string): number {
  const m = hex.trim().match(/^#?([0-9a-f]{6})$/i);
  if (!m) {
    return 0;
  }
  const channel = (v: number) => {
    const c = v / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  };
  const r = channel(parseInt(m[1]!.slice(0, 2), 16));
  const g = channel(parseInt(m[1]!.slice(2, 4), 16));
  const b = channel(parseInt(m[1]!.slice(4, 6), 16));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

/**
 * Readable foreground for a brand color: same hue, lightness pushed to
 * whichever extreme wins on WCAG contrast. The 0.26 luminance threshold is
 * where dark text (L≈16%) starts beating light text (L≈88%).
 */
function contrastOf(hex: string): Hsl {
  const base = hexToHsl(hex);
  if (luminance(hex) > 0.26) {
    return { ...base, l: 16 };
  }
  return { ...base, l: 88 };
}

/**
 * Expands the six core colors + radius into the full CSS variable set, keyed
 * by variable name without the leading `--`. Color values are HSL triples
 * ("H S% L%") matching how main.css and tailwind consume them.
 */
export function expandThemeColors(theme: ThemeColors): Record<string, string> {
  const background = hexToHsl(theme.background);
  const foreground = hexToHsl(theme.foreground);
  const primary = hexToHsl(theme.primary);
  const secondary = hexToHsl(theme.secondary);
  const accent = hexToHsl(theme.accent);
  const destructive = hexToHsl(theme.destructive);

  const shade1 = triple(shade(background, 1));
  const shade2 = triple(shade(background, 2));
  const fg = triple(foreground);
  const primaryFg = triple(contrastOf(theme.primary));

  return {
    background: triple(background),
    "background-accent": shade2,
    foreground: fg,

    card: triple(background),
    "card-foreground": fg,
    popover: triple(background),
    "popover-foreground": fg,
    muted: shade1,
    "muted-foreground": fg,
    border: shade2,
    input: shade2,

    primary: triple(primary),
    "primary-foreground": primaryFg,
    secondary: triple(secondary),
    "secondary-foreground": triple(contrastOf(theme.secondary)),
    accent: triple(accent),
    "accent-foreground": triple(contrastOf(theme.accent)),
    destructive: triple(destructive),
    "destructive-foreground": triple(contrastOf(theme.destructive)),
    ring: triple(primary),

    "sidebar-background": shade1,
    "sidebar-foreground": fg,
    "sidebar-primary": triple(primary),
    "sidebar-primary-foreground": primaryFg,
    "sidebar-accent": shade2,
    "sidebar-accent-foreground": fg,
    "sidebar-border": shade2,
    "sidebar-ring": shade2,

    radius: theme.radius || "0.5rem",
  };
}

/** Renders expanded variables as a CSS declaration block body. */
export function themeCssDeclarations(theme: ThemeColors): string {
  return Object.entries(expandThemeColors(theme))
    .map(([name, value]) => `--${name}: ${value};`)
    .join(" ");
}

/** Inline style object form, for preview cards bound via `:style`. */
export function themeStyleVars(theme: ThemeColors): Record<string, string> {
  return Object.fromEntries(Object.entries(expandThemeColors(theme)).map(([name, value]) => [`--${name}`, value]));
}
