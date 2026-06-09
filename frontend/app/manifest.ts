import type { MetadataRoute } from "next";

/**
 * PWA manifest, ported from the legacy nuxt.config.ts `pwa.manifest`. Values
 * (name, start_url, theme color, icons) are preserved so an upgrade over an
 * installed Nuxt build keeps the same identity.
 */
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Homebox",
    short_name: "Homebox",
    description: "Home Inventory App",
    theme_color: "#5b7f67",
    start_url: "/home",
    display: "standalone",
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
        purpose: "maskable",
      },
    ],
  };
}
