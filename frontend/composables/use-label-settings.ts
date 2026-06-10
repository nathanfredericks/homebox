import type { LabelMakerPreferences } from "./use-preferences";

export function useLabelSettings() {
  const preferences = useViewPreferences();

  const settings = computed<LabelMakerPreferences>(() => preferences.value.labelmaker);

  const sansFontFamily = computed(() =>
    settings.value.sansFont === "open-sans" ? "'Open Sans', sans-serif" : "ui-sans-serif, system-ui, sans-serif"
  );
  const monoFontFamily = computed(() =>
    settings.value.monoFont === "geist-mono"
      ? "'Geist Mono', monospace"
      : "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
  );

  // useRequestURL works during SSR; window.location does not
  const origin = useRequestURL().origin;
  const resolvedBaseURL = computed(() => {
    let base = settings.value.baseURL?.trim() || origin;
    if (base.endsWith("/")) {
      base = base.slice(0, -1);
    }
    return base;
  });

  return { settings, sansFontFamily, monoFontFamily, resolvedBaseURL };
}
