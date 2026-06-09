import {
  mdiTagOutline,
  mdiTreeOutline,
  mdiBagSuitcaseOutline,
  mdiBedOutline,
  mdiCountertopOutline,
  mdiBookOpenVariantOutline,
  mdiLaptop,
  mdiToolboxOutline,
  mdiFolderOutline,
  mdiDresserOutline,
  mdiLightbulbOutline,
  mdiPowerPlugOutline,
  mdiWrenchOutline,
  mdiDumbbell,
  mdiSofaOutline,
  mdiPaletteOutline,
} from "@mdi/js";

/**
 * The set of selectable icons for locations/tags. `path` is the raw MDI SVG
 * path string (24x24 viewBox) consumed by the SvgIcon wrapper / MUI SvgIcon.
 * Names are kept identical to the legacy unplugin-icons keys so existing
 * persisted icon preferences keep resolving.
 */
export const availableIcons = [
  { name: "tag-outline", path: mdiTagOutline },
  { name: "tree-outline", path: mdiTreeOutline },
  { name: "bag-suitcase-outline", path: mdiBagSuitcaseOutline },
  { name: "bed-outline", path: mdiBedOutline },
  { name: "kitchen-counter-outline", path: mdiCountertopOutline },
  { name: "book-open-variant-outline", path: mdiBookOpenVariantOutline },
  { name: "laptop", path: mdiLaptop },
  { name: "sofa-outline", path: mdiSofaOutline },
  { name: "toolbox-outline", path: mdiToolboxOutline },
  { name: "file-cabinet-outline", path: mdiFolderOutline },
  { name: "dresser-outline", path: mdiDresserOutline },
  { name: "lightbulb-outline", path: mdiLightbulbOutline },
  { name: "power-plug-outline", path: mdiPowerPlugOutline },
  { name: "wrench-outline", path: mdiWrenchOutline },
  { name: "dumbbell", path: mdiDumbbell },
  { name: "palette-outline", path: mdiPaletteOutline },
] as const;

export type IconName = (typeof availableIcons)[number]["name"];

export const defaultIcon = mdiTagOutline;

/** Resolve an icon name to its MDI SVG path, falling back to the default. */
export function getIconPath(iconName: string | undefined): string {
  if (!iconName) {
    return defaultIcon;
  }
  const icon = availableIcons.find(i => i.name === iconName);
  return icon ? icon.path : defaultIcon;
}
