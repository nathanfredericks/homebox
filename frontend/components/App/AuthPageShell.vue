<script setup lang="ts">
  import type { Component } from "vue";
  import MdiGithub from "~icons/mdi/github";
  import MdiDiscord from "~icons/mdi/discord";
  import MdiFolder from "~icons/mdi/folder";
  import MdiMastodon from "~icons/mdi/mastodon";
  import MdiLinkVariant from "~icons/mdi/link-variant";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
  import LanguageSelector from "~/components/App/LanguageSelector.vue";
  import AppLogo from "~/components/App/Logo.vue";

  const branding = useBranding();

  const SOCIAL_ICONS: Record<string, Component> = {
    github: MdiGithub,
    mastodon: MdiMastodon,
    discord: MdiDiscord,
    docs: MdiFolder,
    link: MdiLinkVariant,
  };

  function socialIcon(icon: string) {
    return SOCIAL_ICONS[icon] ?? MdiLinkVariant;
  }

  const isEvilAccentTheme = useIsThemeInList([
    "bumblebee",
    "corporate",
    "forest",
    "pastel",
    "wireframe",
    "black",
    "dracula",
    "autumn",
    "acid",
  ]);
  const isEvilForegroundTheme = useIsThemeInList(["light", "aqua", "fantasy", "autumn", "night"]);
  const isLofiTheme = useIsThemeInList(["lofi"]);
</script>

<template>
  <div class="relative flex min-h-screen flex-col">
    <div class="pointer-events-none absolute top-0 z-0 min-w-full fill-primary">
      <div class="flex min-h-[20vh] flex-col bg-primary" />
      <svg
        class="fill-primary drop-shadow-xl"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 1440 320"
        preserveAspectRatio="none"
      >
        <path
          fill-opacity="1"
          d="M0,32L80,69.3C160,107,320,181,480,181.3C640,181,800,107,960,117.3C1120,128,1280,224,1360,272L1440,320L1440,0L1360,0C1280,0,1120,0,960,0C800,0,640,0,480,0C320,0,160,0,80,0L0,0Z"
        />
      </svg>
    </div>
    <div class="relative z-10">
      <header
        class="mx-auto p-4 sm:flex sm:items-end sm:p-6 lg:p-14"
        :class="{
          'text-accent': !isEvilAccentTheme,
          'text-white': isLofiTheme,
        }"
      >
        <div class="z-10">
          <h2
            v-if="branding.hasCustomName.value || branding.loginIconUrl.value"
            class="mt-1 flex items-center gap-3 text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl"
          >
            <img
              v-if="branding.loginIconUrl.value"
              :src="branding.loginIconUrl.value"
              :alt="branding.appName.value"
              class="-mb-2 size-12 object-contain sm:size-14"
            />
            {{ branding.appName.value }}
          </h2>
          <h2 v-else class="mt-1 flex text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
            HomeB
            <AppLogo class="-mb-4 w-12" />
            x
          </h2>
          <p
            class="ml-1 text-lg"
            :class="{
              'text-foreground': !isEvilForegroundTheme,
              'text-white': isLofiTheme,
            }"
          >
            {{ branding.loginSubtitle.value || $t("index.tagline") }}
          </p>
        </div>
        <TooltipProvider :delay-duration="0">
          <div class="z-10 ml-auto mt-6 flex items-center gap-4 sm:mt-0">
            <Tooltip v-for="(link, i) in branding.socialLinks.value" :key="i">
              <TooltipTrigger as-child>
                <a :href="link.url" target="_blank" rel="noopener noreferrer">
                  <component :is="socialIcon(link.icon)" class="size-8" />
                </a>
              </TooltipTrigger>
              <TooltipContent>{{ link.label || (link.labelKey ? $t(link.labelKey) : link.url) }}</TooltipContent>
            </Tooltip>

            <LanguageSelector class="z-10 text-primary" :expanded="false" />
          </div>
        </TooltipProvider>
      </header>
      <div class="grid min-h-[50vh] p-6 sm:place-items-center">
        <div>
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>
