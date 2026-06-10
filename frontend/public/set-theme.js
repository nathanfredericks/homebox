// Migration shim: themes used to live only in localStorage, which SSR cannot
// read. Apply the stored theme before paint and seed the hb.theme cookie so
// the next request is server-rendered with the right theme. Remove once
// existing sessions have picked up the cookie.
try {
  const stored = JSON.parse(localStorage.getItem("homebox/preferences/location") || "null");
  const theme = stored && stored.theme;
  if (theme) {
    document.documentElement.setAttribute("data-theme", theme);
    document.documentElement.classList.add("theme-" + theme);
    if (!document.cookie.split("; ").some(c => c.startsWith("hb.theme="))) {
      document.cookie = "hb.theme=" + encodeURIComponent(theme) + "; path=/; max-age=31536000; samesite=lax";
    }
  }
} catch (e) {
  console.error("Failed to set theme", e);
}
