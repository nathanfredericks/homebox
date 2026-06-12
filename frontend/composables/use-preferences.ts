import type { Ref } from "vue";
import type { NuxtApp } from "nuxt/app";
import type { EntitySummary } from "~/lib/api/types/data-contracts";

export type ViewType = "table" | "card";

export type DuplicateSettings = {
  copyMaintenance: boolean;
  copyAttachments: boolean;
  copyCustomFields: boolean;
  copyPrefixOverride: string | null;
};

// Per-print-job inputs only: the sheet layout itself (dimensions, fonts,
// style) is instance-wide admin configuration served by /labelmaker/settings.
export type LabelMakerPreferences = {
  assetRange: number;
  assetRangeMax: number;
  skipLabels: number;
};

export type LocationViewPreferences = {
  showDetails: boolean;
  showEmpty: boolean;
  editorAdvancedView: boolean;
  itemDisplayView: ViewType;
  itemsPerTablePage: number;
  tableHeaders?: {
    value: keyof EntitySummary;
    enabled: boolean;
  }[];
  displayLegacyHeader: boolean;
  legacyImageFit: boolean;
  language?: string | null;
  overrideFormatLocale?: string | null;
  collectionId?: string | null;
  duplicateSettings: DuplicateSettings;
  shownMultiTabWarning: boolean;
  quickActions: {
    enabled: boolean;
  };
  labelmaker: LabelMakerPreferences;
};
export type PreferenceSyncConfig = Partial<Record<keyof LocationViewPreferences, boolean>>;

const DEFAULT_PREFERENCES: LocationViewPreferences = {
  showDetails: true,
  showEmpty: true,
  editorAdvancedView: false,
  itemDisplayView: "card",
  itemsPerTablePage: 10,
  displayLegacyHeader: false,
  legacyImageFit: false,
  language: null,
  overrideFormatLocale: null,
  duplicateSettings: {
    copyMaintenance: false,
    copyAttachments: true,
    copyCustomFields: true,
    copyPrefixOverride: null,
  },
  shownMultiTabWarning: false,
  quickActions: {
    enabled: true,
  },
  labelmaker: {
    assetRange: 1,
    assetRangeMax: 91,
    skipLabels: 0,
  },
};
let syncConfig: PreferenceSyncConfig = {
  itemDisplayView: false,
  shownMultiTabWarning: false,
};

let syncInitialized = false;

const preferenceKeys = Object.keys(DEFAULT_PREFERENCES) as (keyof LocationViewPreferences)[];

const PREFERENCE_COOKIE_OPTS = {
  path: "/",
  sameSite: "lax",
  maxAge: 60 * 60 * 24 * 365,
} as const;

// Bulk preferences live in localStorage (browser-only). The few fields the
// server needs to render correct HTML — language (SSR locale) and
// collectionId (X-Tenant header) — are mirrored to cookies so SSR sees them.
// The theme is no longer per-user: the instance-wide active theme comes from
// /status (use-instance-theme).
function createPreferences(): Ref<LocationViewPreferences> {
  const localeCookie = useCookie<string | null>("hb.locale", { ...PREFERENCE_COOKIE_OPTS, default: () => null });
  const collectionCookie = useCookie<string | null>("hb.collection", {
    ...PREFERENCE_COOKIE_OPTS,
    default: () => null,
  });

  if (import.meta.server) {
    // cast needed because the auto-imported global Ref and the "vue" Ref types
    // don't unify in this codebase
    return useState<LocationViewPreferences>("preferences", () => ({
      ...DEFAULT_PREFERENCES,
      ...(localeCookie.value ? { language: localeCookie.value } : {}),
      ...(collectionCookie.value ? { collectionId: collectionCookie.value } : {}),
    })) as unknown as Ref<LocationViewPreferences>;
  }

  const stored = useLocalStorage("homebox/preferences/location", DEFAULT_PREFERENCES, {
    mergeDefaults: true,
  }) as unknown as Ref<LocationViewPreferences>;

  watch(
    () => [stored.value.language, stored.value.collectionId] as const,
    ([language, collectionId]) => {
      localeCookie.value = language ?? null;
      collectionCookie.value = collectionId ?? null;
    },
    { immediate: true }
  );

  return stored;
}

function forEachSyncedPreference(callback: (key: keyof LocationViewPreferences) => void) {
  for (const key of preferenceKeys) {
    if (syncConfig[key] !== false) {
      callback(key);
    }
  }
}

function buildSyncedSettings(preferences: LocationViewPreferences): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  forEachSyncedPreference(key => {
    payload[key] = preferences[key];
  });
  return payload;
}

function mergeSyncedSettings(
  settings: Record<string, unknown>,
  preferences: LocationViewPreferences
): LocationViewPreferences {
  const nextPreferences = { ...preferences };

  forEachSyncedPreference(key => {
    if (key in settings) {
      const defaultValue = DEFAULT_PREFERENCES[key];
      const serverValue = settings[key];
      // deep-merge object preferences with defaults so server snapshots
      // written before a subkey existed don't drop it
      if (
        defaultValue !== null &&
        typeof defaultValue === "object" &&
        !Array.isArray(defaultValue) &&
        serverValue !== null &&
        typeof serverValue === "object" &&
        !Array.isArray(serverValue)
      ) {
        nextPreferences[key] = { ...defaultValue, ...serverValue } as never;
      } else {
        nextPreferences[key] = serverValue as never;
      }
    }
  });

  return nextPreferences;
}

export function configureViewPreferenceSync(config: PreferenceSyncConfig) {
  syncConfig = {
    ...syncConfig,
    ...config,
  };
}

async function refreshViewPreferencesFromServer(preferences: Ref<LocationViewPreferences>) {
  const auth = useAuthContext();
  if (!auth.isAuthorized()) {
    return;
  }

  const api = useUserApi();
  const { data, error } = await api.user.getSettings();
  if (error || !data?.item) {
    return;
  }

  preferences.value = mergeSyncedSettings(data.item, preferences.value);
}
export function useViewPreferencesSync() {
  if (syncInitialized || !import.meta.client) {
    return;
  }

  syncInitialized = true;

  const auth = useAuthContext();
  const preferences = useViewPreferences();
  let pauseServerSaves = true;
  let applyingServerSnapshot = false;
  let saveInFlight = false;
  let refreshInFlight = false;
  let refreshRequested = false;
  let localRevision = 0;
  let syncedRevision = 0;
  let retryTimer: ReturnType<typeof setTimeout> | null = null;

  const scheduleRetry = () => {
    if (retryTimer !== null) {
      return;
    }

    retryTimer = setTimeout(() => {
      retryTimer = null;
      void saveToServer();
    }, 1000);
  };

  const markDirty = () => {
    localRevision += 1;
    queueSaveToServer();
  };

  const saveToServer = async () => {
    if (saveInFlight || pauseServerSaves || !auth.isAuthorized()) {
      return;
    }

    saveInFlight = true;

    const api = useUserApi();
    try {
      while (syncedRevision < localRevision && !pauseServerSaves && auth.isAuthorized()) {
        const targetRevision = localRevision;
        const { error } = await api.user.setSettings(buildSyncedSettings(preferences.value));
        if (error) {
          scheduleRetry();
          return;
        }

        syncedRevision = targetRevision;
      }
    } finally {
      saveInFlight = false;

      if (syncedRevision < localRevision && !pauseServerSaves) {
        void saveToServer();
      }
    }
  };

  const queueSaveToServer = useDebounceFn(() => {
    void saveToServer();
  }, 400);

  const refreshFromServer = async () => {
    refreshRequested = true;
    if (refreshInFlight) {
      return;
    }

    refreshInFlight = true;
    try {
      while (refreshRequested) {
        refreshRequested = false;

        pauseServerSaves = true;
        applyingServerSnapshot = true;
        try {
          await refreshViewPreferencesFromServer(preferences);
        } finally {
          applyingServerSnapshot = false;
        }
        pauseServerSaves = false;

        if (syncedRevision < localRevision) {
          await saveToServer();
        }
      }
    } finally {
      refreshInFlight = false;
    }
  };

  watch(
    preferences,
    () => {
      if (applyingServerSnapshot) {
        return;
      }

      markDirty();
    },
    { deep: true }
  );

  watch(
    () => auth.token,
    token => {
      if (!token) {
        pauseServerSaves = true;
        syncedRevision = localRevision;
        return;
      }

      void refreshFromServer();
    },
    { immediate: true }
  );

  onServerEvent(ServerEvent.UserMutation, () => {
    void refreshFromServer();
  });
}

export function useViewPreferences(): Ref<LocationViewPreferences> {
  const nuxtApp = useNuxtApp() as NuxtApp & { _preferences?: Ref<LocationViewPreferences> };
  nuxtApp._preferences ??= createPreferences();
  return nuxtApp._preferences;
}
