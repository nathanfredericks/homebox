<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Button } from "@/components/ui/button";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
  import { Separator } from "@/components/ui/separator";
  import { toast } from "@/components/ui/sonner";
  import MdiArrowLeft from "~icons/mdi/arrow-left";
  import MdiClose from "~icons/mdi/close";
  import MdiPlus from "~icons/mdi/plus";
  import MdiUpload from "~icons/mdi/upload";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import FormGoogleFontSelect from "~/components/Form/GoogleFontSelect.vue";
  import { themes as builtinThemes } from "~~/lib/data/themes";
  import { themeStyleVars, CORE_COLOR_KEYS, type CoreColors } from "~~/lib/theme/expand";
  import type { ThemeAssetKind } from "~~/lib/api/classes/theming";
  import type { SchemaSocialLink, ThemeOut } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: t("admin.tabs.theming") });

  const api = useUserApi();
  const route = useRoute();
  const themeId = computed(() => route.params.id as string);

  const saving = ref(false);
  const theme = ref<ThemeOut | null>(null);
  const activePointer = ref<string>("");

  const form = reactive({
    name: "",
    colors: {
      background: "#ffffff",
      foreground: "#333333",
      primary: "#5c7f67",
      secondary: "#2d2f28",
      accent: "#ecf4e7",
      destructive: "#f87272",
    } as CoreColors,
    radiusRem: 0.5,
    fontSans: "default",
    fontMono: "default",
    appName: "",
    loginSubtitle: "",
    socialLinks: [] as SchemaSocialLink[],
  });

  const SOCIAL_ICON_OPTIONS = ["github", "mastodon", "discord", "docs", "link"];

  function applyTheme(data: ThemeOut) {
    theme.value = data;
    form.name = data.name;
    form.colors = { ...(data.colors as unknown as CoreColors) };
    form.radiusRem = parseFloat(data.radius) || 0;
    form.fontSans = data.fontSans || "default";
    form.fontMono = data.fontMono || "default";
    form.appName = data.branding?.appName ?? "";
    form.loginSubtitle = data.branding?.loginSubtitle ?? "";
    form.socialLinks = (data.branding?.socialLinks ?? []).map(link => ({ ...link }));
  }

  // Fetched during SSR so the editor renders without a loading state. The
  // theme/activePointer refs stay because saves and asset uploads update them
  // in place.
  const { data: themeData } = await useAsyncData(`admin-theme-${themeId.value}`, async () => {
    const [themeRes, activeRes] = await Promise.all([api.theming.get(themeId.value), api.theming.getActive()]);
    if (themeRes.error || !themeRes.data) {
      return null;
    }
    return { theme: themeRes.data, active: activeRes.data?.active ?? "" };
  });

  // Unknown theme: back to the list.
  if (!themeData.value) {
    await navigateTo("/admin/theming", { replace: true });
  }

  watch(
    themeData,
    data => {
      if (data) {
        applyTheme(data.theme);
        activePointer.value = data.active;
      }
    },
    { immediate: true }
  );

  const isActive = computed(() => activePointer.value === `custom:${themeId.value}`);

  const radius = computed(() => `${form.radiusRem}rem`);

  const previewVars = computed(() => themeStyleVars({ ...form.colors, radius: radius.value }));

  function seedFromBuiltin(slug: unknown) {
    const spec = builtinThemes.find(entry => entry.value === slug);
    if (!spec) return;
    form.colors = { ...spec.colors };
    form.radiusRem = parseFloat(spec.radius) || 0;
  }

  async function save() {
    saving.value = true;
    try {
      const { data, error } = await api.theming.update(themeId.value, {
        name: form.name.trim(),
        colors: { ...form.colors },
        radius: radius.value,
        fontSans: form.fontSans === "default" ? "" : form.fontSans,
        fontMono: form.fontMono === "default" ? "" : form.fontMono,
        branding: {
          appName: form.appName.trim(),
          loginSubtitle: form.loginSubtitle.trim(),
          socialLinks: form.socialLinks.filter(link => link.url.trim() !== ""),
        },
      });
      if (error || !data) {
        toast.error(t("errors.api_failure") + String(error));
        return;
      }
      applyTheme(data);
      toast.success(t("admin.theming.saved"));
      if (isActive.value) {
        await refreshNuxtData("app-status");
      }
    } finally {
      saving.value = false;
    }
  }

  // ---------------------------------------------------------------------------
  // Branding assets

  type AssetSlot = {
    kind: ThemeAssetKind;
    labelKey: string;
    has: (theme: ThemeOut) => boolean;
  };

  const assetSlots: AssetSlot[] = [
    { kind: "nav-logo", labelKey: "admin.theming.nav_logo", has: th => th.assets.navLogo },
    { kind: "sidebar-logo", labelKey: "admin.theming.sidebar_logo", has: th => th.assets.sidebarLogo },
    { kind: "login-icon", labelKey: "admin.theming.login_icon", has: th => th.assets.loginIcon },
  ];

  const fileInputs = ref<Record<string, HTMLInputElement | null>>({});

  function assetPreviewUrl(slot: AssetSlot): string | null {
    if (!theme.value || !slot.has(theme.value)) {
      return null;
    }
    return api.theming.assetUrl(themeId.value, slot.kind, new Date(theme.value.updatedAt).getTime());
  }

  async function onAssetPicked(slot: AssetSlot, event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = "";
    if (!file) return;

    const { data, error } = await api.theming.uploadAsset(themeId.value, slot.kind, file);
    if (error || !data) {
      toast.error(t("errors.api_failure") + String(error));
      return;
    }
    theme.value = data;
    toast.success(t("admin.theming.asset_uploaded"));
    if (isActive.value) {
      await refreshNuxtData("app-status");
    }
  }

  async function removeAsset(slot: AssetSlot) {
    const { data, error } = await api.theming.deleteAsset(themeId.value, slot.kind);
    if (error || !data) {
      toast.error(t("errors.api_failure") + String(error));
      return;
    }
    theme.value = data;
    toast.success(t("admin.theming.asset_removed"));
    if (isActive.value) {
      await refreshNuxtData("app-status");
    }
  }

  function addSocialLink() {
    form.socialLinks.push({ icon: "link", label: "", url: "" });
  }

  function removeSocialLink(index: number) {
    form.socialLinks.splice(index, 1);
  }
</script>

<template>
  <div v-if="theme" class="space-y-4">
    <Button variant="ghost" size="sm" @click="navigateTo('/admin/theming')">
      <MdiArrowLeft class="mr-1 size-4" />
      {{ $t("admin.theming.back") }}
    </Button>

    <div class="grid gap-6 lg:grid-cols-2">
      <div class="space-y-6">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="flex flex-col gap-1">
            <Label for="theme-name">{{ $t("admin.theming.name") }}</Label>
            <Input id="theme-name" v-model="form.name" />
          </div>
          <div class="flex flex-col gap-1">
            <Label for="seed-from">{{ $t("admin.theming.seed_from") }}</Label>
            <Select id="seed-from" @update:model-value="seedFromBuiltin">
              <SelectTrigger>
                <SelectValue :placeholder="$t('admin.theming.seed_from_placeholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="spec in builtinThemes" :key="spec.value" :value="spec.value">
                  {{ spec.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div>
          <BaseSectionHeader>
            <span>{{ $t("admin.theming.colors") }}</span>
            <template #description>{{ $t("admin.theming.colors_sub") }}</template>
          </BaseSectionHeader>
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
            <div v-for="key in CORE_COLOR_KEYS" :key="key" class="flex flex-col gap-1">
              <Label :for="`color-${key}`">{{ $t(`admin.theming.color_names.${key}`) }}</Label>
              <input
                :id="`color-${key}`"
                v-model="form.colors[key]"
                type="color"
                class="h-10 w-full cursor-pointer rounded-md border bg-background p-1"
              />
            </div>
          </div>
          <div class="mt-4 flex flex-col gap-1">
            <Label for="theme-radius"> {{ $t("admin.theming.radius") }} ({{ radius }}) </Label>
            <input id="theme-radius" v-model.number="form.radiusRem" type="range" min="0" max="1.5" step="0.05" />
          </div>
        </div>

        <Separator />

        <div>
          <BaseSectionHeader>
            <span>{{ $t("admin.theming.fonts") }}</span>
            <template #description>{{ $t("admin.theming.fonts_sub") }}</template>
          </BaseSectionHeader>
          <div class="grid gap-4 sm:grid-cols-2">
            <FormGoogleFontSelect v-model="form.fontSans" :label="$t('admin.theming.font_sans')" />
            <FormGoogleFontSelect v-model="form.fontMono" :label="$t('admin.theming.font_mono')" />
          </div>
        </div>

        <Separator />

        <div class="space-y-4">
          <BaseSectionHeader>
            <span>{{ $t("admin.theming.branding") }}</span>
            <template #description>{{ $t("admin.theming.branding_sub") }}</template>
          </BaseSectionHeader>

          <div class="grid gap-4 sm:grid-cols-2">
            <div class="flex flex-col gap-1">
              <Label for="brand-app-name">{{ $t("admin.theming.app_name") }}</Label>
              <Input id="brand-app-name" v-model="form.appName" placeholder="HomeBox" />
            </div>
            <div class="flex flex-col gap-1">
              <Label for="brand-subtitle">{{ $t("admin.theming.login_subtitle") }}</Label>
              <Input id="brand-subtitle" v-model="form.loginSubtitle" :placeholder="$t('index.tagline')" />
            </div>
          </div>

          <div class="grid gap-4 sm:grid-cols-3">
            <div v-for="slot in assetSlots" :key="slot.kind" class="flex flex-col gap-2">
              <Label>{{ $t(slot.labelKey) }}</Label>
              <div class="flex h-20 items-center justify-center rounded-md border bg-muted/40 p-2">
                <img
                  v-if="assetPreviewUrl(slot)"
                  :src="assetPreviewUrl(slot)!"
                  :alt="$t(slot.labelKey)"
                  class="max-h-full max-w-full object-contain"
                />
                <span v-else class="text-xs text-muted-foreground">{{ $t("admin.theming.no_image") }}</span>
              </div>
              <input
                :ref="el => (fileInputs[slot.kind] = el as HTMLInputElement | null)"
                type="file"
                accept=".png,.jpg,.jpeg,.webp,.gif,.svg,.ico"
                class="hidden"
                @change="onAssetPicked(slot, $event)"
              />
              <div class="flex gap-1">
                <Button variant="outline" size="sm" class="flex-1" @click="fileInputs[slot.kind]?.click()">
                  <MdiUpload class="mr-1 size-4" />
                  {{ $t("admin.theming.upload") }}
                </Button>
                <Button
                  v-if="theme && slot.has(theme)"
                  variant="destructive"
                  size="icon"
                  :aria-label="$t('admin.theming.remove')"
                  @click="removeAsset(slot)"
                >
                  <MdiClose class="size-4" />
                </Button>
              </div>
            </div>
          </div>

          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <Label>{{ $t("admin.theming.social_links") }}</Label>
              <Button variant="outline" size="sm" @click="addSocialLink">
                <MdiPlus class="mr-1 size-4" />
                {{ $t("admin.theming.add_link") }}
              </Button>
            </div>
            <p class="text-xs text-muted-foreground">{{ $t("admin.theming.social_links_sub") }}</p>
            <div
              v-for="(link, index) in form.socialLinks"
              :key="index"
              class="flex flex-wrap items-center gap-2 sm:flex-nowrap"
            >
              <Select v-model="link.icon">
                <SelectTrigger class="w-32 shrink-0">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="icon in SOCIAL_ICON_OPTIONS" :key="icon" :value="icon">
                    {{ $t(`admin.theming.social_icons.${icon}`) }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <Input v-model="link.label" class="w-40" :placeholder="$t('admin.theming.link_label')" />
              <Input v-model="link.url" class="flex-1" placeholder="https://" />
              <Button
                variant="ghost"
                size="icon"
                :aria-label="$t('admin.theming.remove')"
                @click="removeSocialLink(index)"
              >
                <MdiClose class="size-4" />
              </Button>
            </div>
          </div>
        </div>

        <div class="flex gap-2">
          <Button :disabled="saving || !form.name.trim()" @click="save">
            {{ $t("global.save") }}
          </Button>
          <Button variant="outline" @click="navigateTo('/admin/theming')">
            {{ $t("global.cancel") }}
          </Button>
        </div>
      </div>

      <div class="lg:sticky lg:top-4 lg:self-start">
        <BaseSectionHeader>
          <span>{{ $t("admin.theming.preview") }}</span>
        </BaseSectionHeader>
        <div :style="previewVars" class="overflow-hidden rounded-lg border" style="border-radius: var(--radius)">
          <div class="flex items-center gap-2 bg-sidebar p-3 text-sm font-bold text-foreground">
            <div class="size-6 rounded-full bg-background-accent" />
            {{ form.appName || "HomeBox" }}
          </div>
          <div class="space-y-3 bg-background p-4 text-foreground">
            <div class="text-lg font-bold">{{ form.name || $t("admin.theming.preview") }}</div>
            <p class="text-sm">{{ form.loginSubtitle || $t("index.tagline") }}</p>
            <div class="border bg-card p-3 text-sm text-card-foreground shadow-sm" style="border-radius: var(--radius)">
              {{ $t("admin.theming.preview_card") }}
            </div>
            <div class="flex flex-wrap gap-2">
              <span
                class="bg-primary px-3 py-1 text-sm font-medium text-primary-foreground"
                style="border-radius: var(--radius)"
              >
                {{ $t("admin.theming.color_names.primary") }}
              </span>
              <span
                class="bg-secondary px-3 py-1 text-sm font-medium text-secondary-foreground"
                style="border-radius: var(--radius)"
              >
                {{ $t("admin.theming.color_names.secondary") }}
              </span>
              <span
                class="bg-accent px-3 py-1 text-sm font-medium text-accent-foreground"
                style="border-radius: var(--radius)"
              >
                {{ $t("admin.theming.color_names.accent") }}
              </span>
              <span
                class="bg-destructive px-3 py-1 text-sm font-medium text-destructive-foreground"
                style="border-radius: var(--radius)"
              >
                {{ $t("admin.theming.color_names.destructive") }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
