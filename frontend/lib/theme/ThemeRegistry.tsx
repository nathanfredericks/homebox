"use client";

import { useMemo, type ReactNode } from "react";
import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import { AppRouterCacheProvider } from "@mui/material-nextjs/v15-appRouter";
import { createTheme } from "./createTheme";
import { useThemePreset } from "./useThemePreset";

/**
 * Wraps the app in the MUI theme derived from the active preset, re-memoizing
 * only when the preset changes. The light/dark scheme is handled by CSS
 * variables + the `class` colour-scheme selector (toggled by InitColorSchemeScript
 * and the theme picker), so this component only needs to track the preset.
 *
 * AppRouterCacheProvider integrates Emotion's cache with the App Router so
 * styles are emitted correctly during streaming.
 */
export function ThemeRegistry({ children }: { children: ReactNode }) {
  const presetId = useThemePreset();
  const theme = useMemo(() => createTheme(presetId), [presetId]);

  return (
    <AppRouterCacheProvider options={{ key: "mui" }}>
      <ThemeProvider theme={theme} defaultMode="system">
        <CssBaseline />
        {children}
      </ThemeProvider>
    </AppRouterCacheProvider>
  );
}
