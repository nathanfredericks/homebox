import { createTheme as muiCreateTheme, type Theme } from "@mui/material/styles";
import { getPreset } from "./themePresets";

/**
 * Build the MUI theme for a given preset. CSS variables are enabled with a
 * `class`-based colour-scheme selector so the light/dark scheme is toggled by
 * adding `.light` / `.dark` to <html> (set pre-hydration by InitColorSchemeScript
 * to avoid FOUC). The preset only seeds primary/secondary; MUI derives the
 * full light and dark palettes from those seeds.
 */
export function createTheme(presetId: string | undefined): Theme {
  const preset = getPreset(presetId);

  return muiCreateTheme({
    cssVariables: {
      colorSchemeSelector: "class",
    },
    colorSchemes: {
      light: {
        palette: {
          primary: { main: preset.primary },
          secondary: { main: preset.secondary },
        },
      },
      dark: {
        palette: {
          primary: { main: preset.primary },
          secondary: { main: preset.secondary },
        },
      },
    },
    shape: {
      borderRadius: 8,
    },
    components: {
      MuiButton: {
        defaultProps: {
          disableElevation: true,
        },
      },
    },
  });
}
