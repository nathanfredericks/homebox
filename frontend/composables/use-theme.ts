import type { ComputedRef } from "vue";
import type { DaisyTheme } from "~~/lib/data/themes";

export interface UseTheme {
  /** Active built-in theme slug, or "custom" when a custom theme is active. */
  theme: ComputedRef<string>;
}

/**
 * Read-only view of the instance-wide active theme. Theming is administered
 * site-wide (admin settings → theming); there is no per-user theme, so this
 * exposes no setter. Application to the DOM happens in use-instance-theme.
 */
export function useTheme(): UseTheme {
  const { slug } = useInstanceTheme();
  return { theme: slug };
}

/**
 * True when the active theme is one of the given built-in slugs. Custom
 * themes never match; use useIsDarkTheme() for light/dark behavior.
 */
export function useIsThemeInList(list: DaisyTheme[]) {
  const { theme } = useTheme();
  return computed(() => list.includes(theme.value as DaisyTheme));
}
