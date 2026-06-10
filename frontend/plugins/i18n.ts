/* eslint-disable @typescript-eslint/no-explicit-any */
import type { CompileError, MessageContext } from "vue-i18n";
import { createI18n } from "vue-i18n";
import { IntlMessageFormat } from "intl-messageformat";

export default defineNuxtPlugin(({ vueApp }) => {
  // The requested languages come from the browser on the client and from the
  // Accept-Language header during SSR, so both sides resolve the same locale.
  function requestedLanguages(): readonly string[] {
    if (import.meta.client) {
      return navigator.languages;
    }

    const { "accept-language": acceptLanguage } = useRequestHeaders(["accept-language"]);
    if (!acceptLanguage) {
      return [];
    }

    return acceptLanguage
      .split(",")
      .map(part => part.split(";")[0]!.trim())
      .filter(Boolean);
  }

  function checkDefaultLanguage() {
    let matched = null;
    const languages = Object.getOwnPropertyNames(messages());
    const requested = requestedLanguages();
    const matching = requested.filter(lang => languages.some(l => l.toLowerCase() === lang.toLowerCase()));
    if (matching.length > 0) {
      matched = matching[0];
    }
    if (!matched) {
      const languagePartials = requested.map(lang => lang.split("-")[0]!.toLowerCase());
      languages.forEach(lang => {
        if (languagePartials.includes(lang.toLowerCase())) {
          matched ??= lang;
        }
      });
    }
    return matched;
  }
  const preferences = useViewPreferences();
  const i18n = createI18n({
    fallbackLocale: "en",
    globalInjection: true,
    legacy: false,
    locale: preferences.value.language || checkDefaultLanguage() || "en",
    messageCompiler,
    messages: messages(),
  });
  vueApp.use(i18n);

  watch(
    () => preferences.value.language,
    language => {
      if (!language) {
        return;
      }

      i18n.global.locale.value = language;
    }
  );

  return {
    provide: {
      i18nGlobal: i18n.global,
    },
  };
});

export const messages = () => {
  const messages: Record<string, any> = {};
  const modules = import.meta.glob("~//locales/**.json", { eager: true });
  for (const path in modules) {
    const key = path.slice(9, -5);
    messages[key] = modules[path];
  }
  return messages;
};

export const messageCompiler: (
  message: string | any,
  {
    locale,
    key,
    onError,
  }: {
    locale: any;
    key: any;
    onError: any;
  }
) => (ctx: MessageContext) => unknown = (message, { locale, key, onError }) => {
  if (typeof message === "string") {
    /**
     * You can tune your message compiler performance more with your cache strategy or also memoization at here
     */
    const formatter = new IntlMessageFormat(message, locale);
    return (ctx: MessageContext) => {
      return formatter.format(ctx.values);
    };
  } else {
    /**
     * for AST.
     * If you would like to support it,
     * You need to transform locale messages such as `json`, `yaml`, etc. with the bundle plugin.
     */
    if (onError) {
      onError(new Error("not support for AST") as CompileError);
    }
    return () => key;
  }
};
