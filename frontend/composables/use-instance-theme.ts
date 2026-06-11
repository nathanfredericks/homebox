import type { Ref } from "vue";
import type { APISummary } from "~~/lib/api/types/data-contracts";
import { builtinTheme, darkThemes, type DaisyTheme } from "~~/lib/data/themes";
import { expandThemeColors, hexToHsl, type ThemeColors } from "~~/lib/theme/expand";
import { DEFAULT_FONT, fontFamilyValue, useGoogleFont } from "./use-google-font";

/**
 * Shared /status fetch (SSR-resolved, one request per page load). The login
 * page and the theming pipeline both consume this key.
 */
export function useAppStatus(): Ref<APISummary | null> {
  const api = usePublicApi();
  const { data } = useAsyncData("app-status", async () => {
    const { data } = await api.status();
    return data;
  });
  return data as Ref<APISummary | null>;
}

const HOMEBOX_SLUG = "homebox";

/**
 * The site-wide active theme, admin-configured (there is no per-user theme).
 * Resolves the /status theming block to a renderable spec: built-in slugs are
 * looked up locally, custom themes carry their colors in the payload.
 */
export function useInstanceTheme() {
  const status = useAppStatus();

  const theming = computed(() => status.value?.theming);
  const active = computed(() => theming.value?.active || `builtin:${HOMEBOX_SLUG}`);
  const isCustom = computed(() => active.value.startsWith("custom:") && !!theming.value?.colors);

  /** Built-in slug driving the data-theme attribute; "custom" otherwise. */
  const slug = computed<string>(() => {
    if (isCustom.value) {
      return "custom";
    }
    const name = active.value.replace(/^builtin:/, "");
    return builtinTheme(name) ? name : HOMEBOX_SLUG;
  });

  const colors = computed<ThemeColors>(() => {
    if (isCustom.value) {
      const t = theming.value!;
      return { ...(t.colors as ThemeColors), radius: t.radius };
    }
    const spec = builtinTheme(slug.value) ?? builtinTheme(HOMEBOX_SLUG)!;
    return { ...spec.colors, radius: spec.radius };
  });

  const fontSans = computed(() => (isCustom.value && theming.value?.fontSans) || DEFAULT_FONT);
  const fontMono = computed(() => (isCustom.value && theming.value?.fontMono) || DEFAULT_FONT);

  const isDark = computed(() => {
    if (!isCustom.value) {
      return darkThemes.includes(slug.value as DaisyTheme);
    }
    return hexToHsl(colors.value.background).l < 40;
  });

  /** Full `:root` declaration body for the active theme (colors + fonts). */
  const cssDeclarations = computed(() => {
    const vars = expandThemeColors(colors.value);
    const decls = Object.entries(vars).map(([name, value]) => `--${name}: ${value};`);
    decls.push(`--font-sans: ${fontFamilyValue(fontSans.value, "sans")};`);
    decls.push(`--font-mono: ${fontFamilyValue(fontMono.value, "mono")};`);
    return decls.join(" ");
  });

  return { active, isCustom, slug, colors, fontSans, fontMono, isDark, cssDeclarations };
}

/**
 * Applies the active instance theme globally: inline `:root` style, the
 * data-theme attribute (use-css-var's ThemeObserver watches it) and the
 * Google Fonts stylesheets. Rendered through useHead so SSR HTML already
 * carries the theme — no flash of unstyled content. Call once from app.vue.
 */
export function useApplyInstanceTheme() {
  const instance = useInstanceTheme();

  useGoogleFont(instance.fontSans);
  useGoogleFont(instance.fontMono);

  useHead({
    htmlAttrs: {
      "data-theme": instance.slug,
    },
    style: computed(() => [
      {
        key: "instance-theme",
        // :root[data-theme] outranks main.css's `:root,.homebox` defaults
        // regardless of stylesheet order (the attribute is set right above).
        innerHTML: `:root[data-theme] { ${instance.cssDeclarations.value} }`,
      },
    ]),
  });

  return instance;
}

/** True when the active theme reads as dark (built-in list or background luminance). */
export function useIsDarkTheme() {
  return useInstanceTheme().isDark;
}
