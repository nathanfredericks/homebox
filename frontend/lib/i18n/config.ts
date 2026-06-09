"use client";

import i18next, { type i18n as I18nInstance } from "i18next";
import { initReactI18next } from "react-i18next";
import ICU from "i18next-icu";
import resourcesToBackend from "i18next-resources-to-backend";
import { FALLBACK_LANGUAGE, SUPPORTED_LANGUAGES } from "./languages";

/**
 * Create and initialize the i18next instance. Locales are loaded lazily from
 * the byte-identical Weblate JSON files in `frontend/locales/`; only the active
 * (and fallback) language is fetched. i18next-icu uses the same
 * intl-messageformat engine the Vue app used, so the ICU plural/select syntax
 * in those files renders unchanged.
 */
export function createI18n(initialLanguage: string): I18nInstance {
  const instance = i18next.createInstance();

  instance
    .use(ICU)
    .use(
      resourcesToBackend((language: string) => {
        // The translation files use the locale name verbatim as the filename,
        // including non-standard tags like `en@pirate`.
        return import(`../../locales/${language}.json`);
      })
    )
    .use(initReactI18next)
    .init({
      lng: initialLanguage,
      fallbackLng: FALLBACK_LANGUAGE,
      supportedLngs: SUPPORTED_LANGUAGES as readonly string[] as string[],
      // Keys are nested objects accessed by dotted path (e.g. "global.add"),
      // matching how vue-i18n resolved them, so keep the default "." separator.
      // The locale files carry a single namespace, so disable namespace
      // splitting to avoid treating a ":" in a value/key as a namespace marker.
      nsSeparator: false,
      interpolation: {
        // ICU handles formatting; React escapes output on render.
        escapeValue: false,
      },
      react: {
        useSuspense: false,
      },
    });

  return instance;
}
