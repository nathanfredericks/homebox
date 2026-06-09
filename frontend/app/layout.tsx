import type { Metadata, Viewport } from "next";
import InitColorSchemeScript from "@mui/material/InitColorSchemeScript";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "Homebox",
  description: "Home Inventory App",
  icons: {
    icon: "/favicon.svg",
  },
};

export const viewport: Viewport = {
  themeColor: "#5b7f67",
};

/**
 * Inline, render-blocking script that applies the persisted theme preset class
 * to <html> before hydration to avoid a flash of the default preset. Mirrors
 * the legacy public/set-theme.js, reading the same preferences key. Light/dark
 * scheme is handled separately by InitColorSchemeScript below.
 */
const themePresetScript = `
(function () {
  try {
    var raw = localStorage.getItem("homebox/preferences/location");
    if (!raw) return;
    var theme = JSON.parse(raw).theme;
    if (theme) document.documentElement.classList.add("theme-" + theme);
  } catch (e) {}
})();
`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <InitColorSchemeScript attribute="class" defaultMode="system" />
        <script dangerouslySetInnerHTML={{ __html: themePresetScript }} />
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
