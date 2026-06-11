import type { LabelMakerPreferences } from "./use-preferences";
import { fontFamilyValue, useGoogleFont } from "./use-google-font";

// Pre-Google-Fonts preferences stored the only two bundled font ids; map them
// onto their catalog family names so existing settings keep working.
const LEGACY_FONTS: Record<string, string> = {
  "open-sans": "Open Sans",
  "geist-mono": "Geist Mono",
};

function resolveFamily(stored: string): string {
  return LEGACY_FONTS[stored] ?? stored;
}

export function useLabelSettings() {
  const preferences = useViewPreferences();

  const settings = computed<LabelMakerPreferences>(() => preferences.value.labelmaker);

  const sansFamily = computed(() => resolveFamily(settings.value.sansFont));
  const monoFamily = computed(() => resolveFamily(settings.value.monoFont));

  const sansFont = useGoogleFont(sansFamily);
  const monoFont = useGoogleFont(monoFamily);

  const sansFontFamily = computed(() => fontFamilyValue(sansFamily.value, "sans"));
  const monoFontFamily = computed(() => fontFamilyValue(monoFamily.value, "mono"));

  // Labels are rasterized with html-to-image; the picked fonts must be fully
  // committed to document.fonts before rendering or the PNGs fall back.
  async function ensureFontsLoaded() {
    await Promise.all([sansFont.ensureLoaded(), monoFont.ensureLoaded()]);
  }

  // useRequestURL works during SSR; window.location does not
  const origin = useRequestURL().origin;
  const resolvedBaseURL = computed(() => {
    let base = settings.value.baseURL?.trim() || origin;
    if (base.endsWith("/")) {
      base = base.slice(0, -1);
    }
    return base;
  });

  return { settings, sansFontFamily, monoFontFamily, ensureFontsLoaded, resolvedBaseURL };
}
