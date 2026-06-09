/**
 * Framework-agnostic reader for the small slice of view preferences the API
 * layer needs. The full reactive preferences hook lives in `hooks/use-preferences`
 * (owned by the shell); this module exists so `lib/api` can resolve the active
 * collection (tenant) for attachment/QR URLs without a framework dependency,
 * reading the same localStorage key the rest of the app uses.
 *
 * The `{ value: ... }` accessor shape mirrors the old Nuxt `useViewPreferences()`
 * composable so the `lib/api` call site keeps the same `prefs.value.collectionId`
 * access.
 */

export const PREFERENCES_KEY = "homebox/preferences/location";

export interface ApiViewPreferences {
  collectionId?: string | null;
}

function read(): ApiViewPreferences {
  if (typeof window === "undefined") {
    return {};
  }
  try {
    const raw = window.localStorage.getItem(PREFERENCES_KEY);
    return raw ? (JSON.parse(raw) as ApiViewPreferences) : {};
  } catch {
    return {};
  }
}

/** Returns the current view preferences behind a `.value` accessor. */
export function getViewPreferences(): { readonly value: ApiViewPreferences } {
  return {
    get value() {
      return read();
    },
  };
}
