<template>
  <DialogProvider>
    <ClientOnly>
      <Toaster class="pointer-events-auto" />
    </ClientOnly>

    <NuxtLayout>
      <Html :lang="locale" />
      <Link rel="icon" type="image/svg" href="/favicon.svg" />
      <Link rel="apple-touch-icon" href="/apple-touch-icon.png" size="180x180" />
      <Link rel="mask-icon" href="/mask-icon.svg" color="#5b7f67" />
      <Meta name="theme-color" content="#5b7f67" />
      <Link rel="manifest" href="/manifest.webmanifest" />
      <NuxtPage />
    </NuxtLayout>
  </DialogProvider>
</template>

<script lang="ts" setup>
  import { useI18n } from "vue-i18n";
  import { DialogProvider } from "@/components/ui/dialog-provider";
  import { Toaster } from "@/components/ui/sonner";

  const { locale } = useI18n();

  // Resolve the instance-wide theme + branding during SSR so the first paint
  // already carries the active palette, fonts and app name.
  useApplyInstanceTheme();
  const { appName } = useBranding();
  useHead({
    titleTemplate: title => (title ? `${appName.value} | ${title}` : appName.value),
  });

  useViewPreferencesSync();
</script>
