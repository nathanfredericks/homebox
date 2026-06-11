import type { MaybeRefOrGetter } from "vue";

export const DEFAULT_FONT = "default";

function css2Url(family: string, weights: number[]): string {
  const fam = encodeURIComponent(family).replace(/%20/g, "+");
  return `https://fonts.googleapis.com/css2?family=${fam}:wght@${weights.join(";")}&display=swap`;
}

function isRealFamily(family: string | null | undefined): family is string {
  return !!family && family !== DEFAULT_FONT;
}

/**
 * Loads a Google Font by injecting its css2 stylesheet link (SSR-rendered via
 * useHead, deduped by family). The `crossorigin="anonymous"` attribute is
 * required: it makes the stylesheet's cssRules readable so html-to-image can
 * embed the font when rasterizing labels.
 *
 * `ensureLoaded()` resolves once the font faces are actually committed to
 * `document.fonts`, which must happen before rasterizing to PNG.
 */
export function useGoogleFont(family: MaybeRefOrGetter<string | null | undefined>, weights: number[] = [400, 700]) {
  useHead({
    link: computed(() => {
      const fam = toValue(family);
      if (!isRealFamily(fam)) {
        return [];
      }
      return [
        { key: "gfonts-preconnect", rel: "preconnect", href: "https://fonts.googleapis.com" },
        {
          key: "gfonts-preconnect-static",
          rel: "preconnect",
          href: "https://fonts.gstatic.com",
          crossorigin: "anonymous" as const,
        },
        {
          key: `gfont-${fam}`,
          rel: "stylesheet",
          href: css2Url(fam, weights),
          crossorigin: "anonymous" as const,
        },
      ];
    }),
  });

  async function ensureLoaded(): Promise<void> {
    if (import.meta.server) {
      return;
    }
    const fam = toValue(family);
    if (!isRealFamily(fam)) {
      return;
    }

    await waitForStylesheet(css2Url(fam, weights));
    await Promise.all(weights.map(w => document.fonts.load(`${w} 16px "${fam}"`)));
    await document.fonts.ready;
  }

  return { ensureLoaded };
}

/**
 * Resolves once the <link> for the given href has loaded its stylesheet (or
 * after a timeout, so an offline instance degrades to fallback fonts instead
 * of hanging).
 */
function waitForStylesheet(href: string, timeoutMs = 5000): Promise<void> {
  return new Promise(resolve => {
    const link = document.querySelector<HTMLLinkElement>(`link[href="${href}"]`);
    if (!link || link.sheet) {
      resolve();
      return;
    }

    const timer = setTimeout(() => {
      cleanup();
      resolve();
    }, timeoutMs);

    const cleanup = () => {
      clearTimeout(timer);
      link.removeEventListener("load", onDone);
      link.removeEventListener("error", onDone);
    };
    const onDone = () => {
      cleanup();
      resolve();
    };

    link.addEventListener("load", onDone);
    link.addEventListener("error", onDone);
  });
}

/**
 * CSS font-family value for a label/platform font preference: quotes the
 * Google Font family and keeps the system stack as fallback.
 */
export function fontFamilyValue(family: string | null | undefined, kind: "sans" | "mono"): string {
  const fallback =
    kind === "sans"
      ? "ui-sans-serif, system-ui, sans-serif"
      : "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";
  if (!isRealFamily(family)) {
    return fallback;
  }
  return `"${family}", ${fallback}`;
}
