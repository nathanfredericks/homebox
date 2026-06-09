"use client";

import { useRef, type ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import type { i18n as I18nInstance } from "i18next";
import { createI18n } from "./config";
import { detectBrowserLanguage, FALLBACK_LANGUAGE } from "./languages";
import { PREFERENCES_KEY } from "../theme/useThemePreset";

/**
 * Resolve the initial language: the user's persisted `language` preference if
 * set, otherwise the browser-detected locale (matching the legacy plugin).
 */
function resolveInitialLanguage(): string {
  if (typeof window !== "undefined") {
    try {
      const raw = window.localStorage.getItem(PREFERENCES_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as { language?: string };
        if (parsed.language) {
          return parsed.language;
        }
      }
    } catch {
      // ignore malformed preferences
    }
  }
  return detectBrowserLanguage();
}

/**
 * Provides a lazily-initialized i18next instance to the React tree. The
 * instance is created once per mount; the active language is later driven by
 * the preferences hook (via `i18n.changeLanguage`), owned by the shell.
 */
export function I18nProvider({ children }: { children: ReactNode }) {
  const instanceRef = useRef<I18nInstance | null>(null);
  if (!instanceRef.current) {
    instanceRef.current = createI18n(typeof window === "undefined" ? FALLBACK_LANGUAGE : resolveInitialLanguage());
  }

  return <I18nextProvider i18n={instanceRef.current}>{children}</I18nextProvider>;
}
