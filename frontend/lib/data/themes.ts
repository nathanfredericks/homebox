import type { CoreColors } from "~~/lib/theme/expand";

export type DaisyTheme =
  | "homebox"
  | "light"
  | "dark"
  | "cupcake"
  | "bumblebee"
  | "emerald"
  | "corporate"
  | "synthwave"
  | "retro"
  | "cyberpunk"
  | "valentine"
  | "halloween"
  | "garden"
  | "forest"
  | "aqua"
  | "lofi"
  | "pastel"
  | "fantasy"
  | "wireframe"
  | "black"
  | "luxury"
  | "dracula"
  | "cmyk"
  | "autumn"
  | "business"
  | "acid"
  | "lemonade"
  | "night"
  | "coffee"
  | "winter";

// ThemeSpec is a built-in theme: six core colors + radius, from which every
// CSS variable is derived (lib/theme/expand.ts). Custom themes created in the
// admin theming area share the same shape server-side. The core colors were
// extracted from the original hand-written DaisyUI-derived palettes (MIT,
// (c) 2020 Pouya Saadeghi, converted by tonya (c) 2025).
export type ThemeSpec = {
  label: string;
  value: DaisyTheme;
  colors: CoreColors;
  radius: string;
};

export const themes: ThemeSpec[] = [
  {
    label: "Homebox",
    value: "homebox",
    colors: {
      background: "#ffffff",
      foreground: "#333333",
      primary: "#5c7f67",
      secondary: "#2d2f28",
      accent: "#ecf4e7",
      destructive: "#f87272",
    },
    radius: "0.5rem",
  },
  {
    label: "Garden",
    value: "garden",
    colors: {
      background: "#e9e7e7",
      foreground: "#100f0f",
      primary: "#5c7f67",
      secondary: "#5d5656",
      accent: "#ecf4e7",
      destructive: "#f87272",
    },
    radius: "0.5rem",
  },
  {
    label: "Light",
    value: "light",
    colors: {
      background: "#ffffff",
      foreground: "#1f2937",
      primary: "#570df8",
      secondary: "#3d4451",
      accent: "#f000b8",
      destructive: "#f87272",
    },
    radius: "0.5rem",
  },
  {
    label: "Cupcake",
    value: "cupcake",
    colors: {
      background: "#faf7f5",
      foreground: "#291334",
      primary: "#65c3c8",
      secondary: "#291334",
      accent: "#ef9fbc",
      destructive: "#f87272",
    },
    radius: "1.9rem",
  },
  {
    label: "Bumblebee",
    value: "bumblebee",
    colors: {
      background: "#ffffff",
      foreground: "#333333",
      primary: "#e0a82e",
      secondary: "#18182f",
      accent: "#f9d72f",
      destructive: "#f87272",
    },
    radius: "0.5rem",
  },
  {
    label: "Emerald",
    value: "emerald",
    colors: {
      background: "#ffffff",
      foreground: "#333c4d",
      primary: "#66cc8a",
      secondary: "#333c4d",
      accent: "#377cfb",
      destructive: "#f87272",
    },
    radius: "0.5rem",
  },
  {
    label: "Corporate",
    value: "corporate",
    colors: {
      background: "#ffffff",
      foreground: "#181a2a",
      primary: "#4b6bfb",
      secondary: "#181a2a",
      accent: "#7b92b2",
      destructive: "#f87272",
    },
    radius: "0.125rem",
  },
  {
    label: "Synthwave",
    value: "synthwave",
    colors: {
      background: "#2d1b69",
      foreground: "#f9f7fd",
      primary: "#e779c1",
      secondary: "#20134e",
      accent: "#58c7f3",
      destructive: "#e24056",
    },
    radius: "0.5rem",
  },
  {
    label: "Retro",
    value: "retro",
    colors: {
      background: "#e4d8b4",
      foreground: "#282425",
      primary: "#ef9995",
      secondary: "#7d7259",
      accent: "#a4cbb4",
      destructive: "#dc2828",
    },
    radius: "0.4rem",
  },
  {
    label: "Cyberpunk",
    value: "cyberpunk",
    colors: {
      background: "#ffee00",
      foreground: "#333000",
      primary: "#ff7598",
      secondary: "#423f00",
      accent: "#75d1f0",
      destructive: "#f87272",
    },
    radius: "0",
  },
  {
    label: "Valentine",
    value: "valentine",
    colors: {
      background: "#f0d6e8",
      foreground: "#632c3b",
      primary: "#e96d7b",
      secondary: "#af4670",
      accent: "#a992f7",
      destructive: "#dc2828",
    },
    radius: "1.9rem",
  },
  {
    label: "Halloween",
    value: "halloween",
    colors: {
      background: "#212121",
      foreground: "#d4d4d4",
      primary: "#f28c18",
      secondary: "#1b1d1d",
      accent: "#6d3a9c",
      destructive: "#dc2828",
    },
    radius: "0.5rem",
  },
  {
    label: "Forest",
    value: "forest",
    colors: {
      background: "#171212",
      foreground: "#d7cccc",
      primary: "#1eb854",
      secondary: "#110e0e",
      accent: "#1fd65f",
      destructive: "#f87272",
    },
    radius: "1.9rem",
  },
  {
    label: "Aqua",
    value: "aqua",
    colors: {
      background: "#345ca8",
      foreground: "#c7dbff",
      primary: "#09e9f1",
      secondary: "#3b8bc4",
      accent: "#966fb3",
      destructive: "#dc2828",
    },
    radius: "0.5rem",
  },
  {
    label: "Lofi",
    value: "lofi",
    colors: {
      background: "#ffffff",
      foreground: "#000000",
      primary: "#0d0d0d",
      secondary: "#000000",
      accent: "#1a1919",
      destructive: "#de1b8d",
    },
    radius: "0.125rem",
  },
  {
    label: "Pastel",
    value: "pastel",
    colors: {
      background: "#ffffff",
      foreground: "#333333",
      primary: "#d1c1d7",
      secondary: "#70acc7",
      accent: "#f6cbd1",
      destructive: "#f87272",
    },
    radius: "1.9rem",
  },
  {
    label: "Fantasy",
    value: "fantasy",
    colors: {
      background: "#ffffff",
      foreground: "#1f2937",
      primary: "#6e0b75",
      secondary: "#1f2937",
      accent: "#007ebd",
      destructive: "#f87272",
    },
    radius: "0.5rem",
  },
  {
    label: "Wireframe",
    value: "wireframe",
    colors: {
      background: "#ffffff",
      foreground: "#333333",
      primary: "#b8b8b8",
      secondary: "#ebebeb",
      accent: "#b8b8b8",
      destructive: "#ff0000",
    },
    radius: "0.2rem",
  },
  {
    label: "Black",
    value: "black",
    colors: {
      background: "#000000",
      foreground: "#cccccc",
      primary: "#343232",
      secondary: "#272626",
      accent: "#343232",
      destructive: "#ff0000",
    },
    radius: "0",
  },
  {
    label: "Luxury",
    value: "luxury",
    colors: {
      background: "#09090b",
      foreground: "#dca54c",
      primary: "#ffffff",
      secondary: "#171618",
      accent: "#152747",
      destructive: "#ff7070",
    },
    radius: "0.5rem",
  },
  {
    label: "Dracula",
    value: "dracula",
    colors: {
      background: "#272935",
      foreground: "#f8f8f2",
      primary: "#ff7ac6",
      secondary: "#414558",
      accent: "#bf95f9",
      destructive: "#ff5757",
    },
    radius: "0.5rem",
  },
  {
    label: "Cmyk",
    value: "cmyk",
    colors: {
      background: "#ffffff",
      foreground: "#333333",
      primary: "#44adee",
      secondary: "#1a1a1a",
      accent: "#e9498c",
      destructive: "#ea4034",
    },
    radius: "0.5rem",
  },
  {
    label: "Autumn",
    value: "autumn",
    colors: {
      background: "#f2f2f2",
      foreground: "#303030",
      primary: "#8c0327",
      secondary: "#836b5d",
      accent: "#d75050",
      destructive: "#e01a2e",
    },
    radius: "0.5rem",
  },
  {
    label: "Business",
    value: "business",
    colors: {
      background: "#212121",
      foreground: "#d1d1d1",
      primary: "#1c4f82",
      secondary: "#23282f",
      accent: "#7d919b",
      destructive: "#ab3d30",
    },
    radius: "0.125rem",
  },
  {
    label: "Acid",
    value: "acid",
    colors: {
      background: "#fafafa",
      foreground: "#333333",
      primary: "#ff00f2",
      secondary: "#191a3e",
      accent: "#ff7300",
      destructive: "#e60400",
    },
    radius: "1rem",
  },
  {
    label: "Lemonade",
    value: "lemonade",
    colors: {
      background: "#ffffff",
      foreground: "#333333",
      primary: "#529b03",
      secondary: "#191a3e",
      accent: "#e9e92f",
      destructive: "#f2b6b5",
    },
    radius: "0.5rem",
  },
  {
    label: "Night",
    value: "night",
    colors: {
      background: "#0f1729",
      foreground: "#b3c5ef",
      primary: "#3abff8",
      secondary: "#1d283a",
      accent: "#828df8",
      destructive: "#fb6f84",
    },
    radius: "0.5rem",
  },
  {
    label: "Coffee",
    value: "coffee",
    colors: {
      background: "#211720",
      foreground: "#746d63",
      primary: "#dc944c",
      secondary: "#120c12",
      accent: "#263f40",
      destructive: "#fc9783",
    },
    radius: "0.5rem",
  },
  {
    label: "Winter",
    value: "winter",
    colors: {
      background: "#ffffff",
      foreground: "#394e6a",
      primary: "#057aff",
      secondary: "#021431",
      accent: "#463aa1",
      destructive: "#e58b8b",
    },
    radius: "0.5rem",
  },
];

export function builtinTheme(slug: string): ThemeSpec | undefined {
  return themes.find(t => t.value === slug);
}

export const darkThemes: DaisyTheme[] = [
  "synthwave",
  "retro",
  "cyberpunk",
  "valentine",
  "halloween",
  "forest",
  "aqua",
  "black",
  "luxury",
  "dracula",
  "business",
  "night",
  "coffee",
];
