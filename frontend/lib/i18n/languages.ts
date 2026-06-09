/**
 * The set of locales shipped in `frontend/locales/*.json` (Weblate-managed).
 * Keep in sync with that directory; the list drives lazy loading and browser
 * language detection. Order is irrelevant.
 */
export const SUPPORTED_LANGUAGES = [
  "af-ZA",
  "ar-AA",
  "bs-BA",
  "ca",
  "cs-CZ",
  "da-DK",
  "de",
  "el-GR",
  "en",
  "en@pirate",
  "es",
  "fi-FI",
  "fr",
  "hi-IN",
  "hr-HR",
  "hu",
  "id-ID",
  "it",
  "ja-JP",
  "kmr",
  "ko-KR",
  "lb-LU",
  "lt-LT",
  "nb-NO",
  "nl",
  "pl",
  "pt-BR",
  "pt-PT",
  "ro-RO",
  "ru",
  "sk-SK",
  "sl",
  "sq-AL",
  "sr-RS",
  "sv",
  "th-TH",
  "tr",
  "uk-UA",
  "vi-VN",
  "zh-CN",
  "zh-HK",
  "zh-MO",
  "zh-TW",
] as const;

export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

export const FALLBACK_LANGUAGE: SupportedLanguage = "en";

/**
 * Resolve the best initial locale, mirroring the legacy Nuxt i18n plugin:
 * first an exact (case-insensitive) match of any `navigator.languages` entry
 * against a supported locale, then a match on the base language subtag, else
 * the fallback.
 */
export function detectBrowserLanguage(): SupportedLanguage {
  if (typeof navigator === "undefined") {
    return FALLBACK_LANGUAGE;
  }
  const supported = SUPPORTED_LANGUAGES as readonly string[];
  const browserLangs = navigator.languages?.length ? navigator.languages : [navigator.language];

  for (const lang of browserLangs) {
    const exact = supported.find(l => l.toLowerCase() === lang.toLowerCase());
    if (exact) {
      return exact as SupportedLanguage;
    }
  }

  const base = (navigator.language || "").split("-")[0].toLowerCase();
  const partial = supported.find(l => l.toLowerCase() === base);
  return (partial as SupportedLanguage) ?? FALLBACK_LANGUAGE;
}
