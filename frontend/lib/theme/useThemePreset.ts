"use client";

import { useSyncExternalStore } from "react";
import { DEFAULT_PRESET_ID, mapLegacyTheme, type ThemePresetId } from "./themePresets";

/**
 * The preferences blob persisted by the app under this localStorage key. The
 * shape is owned by the (later) `use-preferences` hook; here we only read the
 * `theme` field to resolve the active preset. Key is preserved from the Vue app
 * so existing users keep their selection.
 */
export const PREFERENCES_KEY = "homebox/preferences/location";

function readPresetFromStorage(): ThemePresetId {
  if (typeof window === "undefined") {
    return DEFAULT_PRESET_ID;
  }
  try {
    const raw = window.localStorage.getItem(PREFERENCES_KEY);
    if (!raw) {
      return DEFAULT_PRESET_ID;
    }
    const parsed = JSON.parse(raw) as { theme?: string };
    return mapLegacyTheme(parsed.theme);
  } catch {
    return DEFAULT_PRESET_ID;
  }
}

function subscribe(callback: () => void): () => void {
  if (typeof window === "undefined") {
    return () => {};
  }
  // `storage` fires for cross-tab writes; a custom event lets same-tab writers
  // (the preferences hook) notify the registry immediately.
  window.addEventListener("storage", callback);
  window.addEventListener("homebox:preferences", callback);
  return () => {
    window.removeEventListener("storage", callback);
    window.removeEventListener("homebox:preferences", callback);
  };
}

/**
 * Subscribe to the active theme preset id. Returns the default during SSR and
 * the initial client render, then the persisted value once hydrated.
 */
export function useThemePreset(): ThemePresetId {
  return useSyncExternalStore(subscribe, readPresetFromStorage, () => DEFAULT_PRESET_ID);
}
