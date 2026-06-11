import { $fetch } from "ofetch";

const DEFAULT_APP_NAME = "HomeBox";
const DEFAULT_BACKGROUND_COLOR = "#ffffff";
const DEFAULT_THEME_COLOR = "#5b7f67";

type StatusResponse = {
  theming?: {
    branding?: { appName?: string };
    colors?: { primary?: string; background?: string };
  };
};

export default defineEventHandler(async event => {
  const config = useRuntimeConfig();

  let appName = DEFAULT_APP_NAME;
  let backgroundColor = DEFAULT_BACKGROUND_COLOR;
  let themeColor = DEFAULT_THEME_COLOR;

  try {
    const status = await $fetch<StatusResponse>(`${config.apiHost}/api/v1/status`);
    if (status.theming?.branding?.appName) {
      appName = status.theming.branding.appName;
    }
    if (status.theming?.colors?.background) {
      backgroundColor = status.theming.colors.background;
    }
    if (status.theming?.colors?.primary) {
      themeColor = status.theming.colors.primary;
    }
  } catch {
    // Fall back to defaults if status fetch fails
  }

  const manifest = {
    name: appName,
    short_name: appName,
    description: `${appName} - Home Inventory`,
    start_url: "/home",
    display: "standalone",
    background_color: backgroundColor,
    theme_color: themeColor,
    lang: "en",
    scope: "/",
    icons: [
      {
        src: "/pwa-192x192.png",
        sizes: "192x192",
        type: "image/png",
      },
      {
        src: "/pwa-512x512.png",
        sizes: "512x512",
        type: "image/png",
      },
      {
        src: "/pwa-512x512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "any maskable",
      },
    ],
  };

  setHeader(event, "Content-Type", "application/manifest+json");
  return manifest;
});
