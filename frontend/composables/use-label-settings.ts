import type { LabelMakerPreferences } from "./use-preferences";
import { fontFamilyValue, useGoogleFont } from "./use-google-font";
import type { LabelLayout } from "~~/lib/api/classes/labelmaker";

// Mirrors the backend defaults in config.LabelMakerConf; used until the
// instance settings load (and as the SSR fallback).
export const DEFAULT_LABEL_LAYOUT: LabelLayout = {
  baseUrl: "",
  measure: "in",
  cardWidth: 2.63,
  cardHeight: 1,
  pageWidth: 8.5,
  pageHeight: 11,
  pageTopPadding: 0.52,
  pageBottomPadding: 0.42,
  pageLeftPadding: 0.25,
  pageRightPadding: 0.1,
  sansFont: "default",
  monoFont: "default",
  bordered: false,
  printLocationRow: true,
  labelPerQuantity: false,
};

// Per-session override of the instance-wide label-per-quantity default; a
// print-job choice, deliberately not persisted anywhere.
const labelPerQuantityOverride = ref<boolean | null>(null);

/**
 * Label rendering settings. The sheet layout (card/page dimensions, fonts,
 * style) is instance-wide and admin-managed; only per-print-job inputs (asset
 * range, skip-first-N) remain per-user preferences.
 */
export function useLabelSettings() {
  const api = useUserApi();

  const { data: loaded } = useAsyncData("labelmaker-settings", async () => {
    const { data, error } = await api.labelmaker.settings();
    if (error || !data) return null;
    return data;
  });

  const layout = computed<LabelLayout>(() => ({ ...DEFAULT_LABEL_LAYOUT, ...(loaded.value ?? {}) }));

  // Per-print-job inputs (writable, persisted per user).
  const preferences = useViewPreferences();
  const job = computed<LabelMakerPreferences>(() => preferences.value.labelmaker);

  const labelPerQuantity = computed<boolean>({
    get: () => labelPerQuantityOverride.value ?? layout.value.labelPerQuantity,
    set: value => {
      labelPerQuantityOverride.value = value;
    },
  });

  const sansFont = useGoogleFont(computed(() => layout.value.sansFont));
  const monoFont = useGoogleFont(computed(() => layout.value.monoFont));

  const sansFontFamily = computed(() => fontFamilyValue(layout.value.sansFont, "sans"));
  const monoFontFamily = computed(() => fontFamilyValue(layout.value.monoFont, "mono"));

  // Labels are rasterized with html-to-image; the picked fonts must be fully
  // committed to document.fonts before rendering or the PNGs fall back.
  async function ensureFontsLoaded() {
    await Promise.all([sansFont.ensureLoaded(), monoFont.ensureLoaded()]);
  }

  // useRequestURL works during SSR; window.location does not
  const origin = useRequestURL().origin;
  const resolvedBaseURL = computed(() => {
    let base = layout.value.baseUrl?.trim() || origin;
    if (base.endsWith("/")) {
      base = base.slice(0, -1);
    }
    return base;
  });

  return { layout, job, labelPerQuantity, sansFontFamily, monoFontFamily, ensureFontsLoaded, resolvedBaseURL };
}
