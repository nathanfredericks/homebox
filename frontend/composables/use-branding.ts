import { useAppStatus, useInstanceTheme } from "./use-instance-theme";

export type BrandingSocialLink = {
  icon: string; // github | mastodon | discord | docs | link
  label: string;
  /** i18n key used for default links when label is empty. */
  labelKey?: string;
  url: string;
};

export const DEFAULT_SOCIAL_LINKS: BrandingSocialLink[] = [
  { icon: "github", label: "", labelKey: "global.github", url: "https://github.com/sysadminsmedia/homebox" },
  { icon: "mastodon", label: "", labelKey: "global.follow_dev", url: "https://noc.social/@sysadminszone" },
  { icon: "discord", label: "", labelKey: "global.join_discord", url: "https://discord.gg/aY4DCkpNA9" },
  { icon: "docs", label: "", labelKey: "global.read_docs", url: "https://homebox.software/en/" },
];

export const DEFAULT_APP_NAME = "HomeBox";

/**
 * Whitelabel branding of the active instance theme, with stock HomeBox
 * defaults when a built-in theme is active or a field is left empty.
 */
export function useBranding() {
  const status = useAppStatus();
  const { isCustom } = useInstanceTheme();

  const theming = computed(() => status.value?.theming);
  const branding = computed(() => (isCustom.value ? theming.value?.branding : undefined));

  const appName = computed(() => branding.value?.appName || DEFAULT_APP_NAME);
  const hasCustomName = computed(() => !!branding.value?.appName);
  const loginSubtitle = computed(() => branding.value?.loginSubtitle || "");

  const socialLinks = computed<BrandingSocialLink[]>(() => {
    const links = branding.value?.socialLinks;
    if (!isCustom.value || !links) {
      return DEFAULT_SOCIAL_LINKS;
    }
    return links;
  });

  const assetUrl = (kind: "nav-logo" | "sidebar-logo" | "login-icon") =>
    `/api/v1/theming/assets/${kind}?v=${theming.value?.version ?? 0}`;

  const navLogoUrl = computed(() => (isCustom.value && theming.value?.assets?.navLogo ? assetUrl("nav-logo") : null));
  const sidebarLogoUrl = computed(() =>
    isCustom.value && theming.value?.assets?.sidebarLogo ? assetUrl("sidebar-logo") : null
  );
  const loginIconUrl = computed(() =>
    isCustom.value && theming.value?.assets?.loginIcon ? assetUrl("login-icon") : null
  );

  return { isCustom, appName, hasCustomName, loginSubtitle, socialLinks, navLogoUrl, sidebarLogoUrl, loginIconUrl };
}
