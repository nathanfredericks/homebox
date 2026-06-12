<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
  import { Button } from "@/components/ui/button";
  import { Badge } from "@/components/ui/badge";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
  import { toast } from "@/components/ui/sonner";
  import MdiContentCopy from "~icons/mdi/content-copy";
  import MdiDelete from "~icons/mdi/delete";
  import MdiPencil from "~icons/mdi/pencil";
  import MdiPlus from "~icons/mdi/plus";
  import ThemePicker, { type ThemePickerEntry } from "~/components/App/ThemePicker.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import { themes as builtinThemes, builtinTheme } from "~~/lib/data/themes";
  import { DEFAULT_FONT } from "~~/composables/use-google-font";
  import type { ThemeOut } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: t("admin.tabs.theming") });

  const api = useUserApi();
  const confirm = useConfirm();
  const { can } = usePermissions();

  const customThemes = ref<ThemeOut[]>([]);
  const active = ref<string>("builtin:homebox");
  const deleting = ref<Record<string, boolean>>({});

  // Fetched during SSR so the picker and table render without a loading
  // state. The local refs stay because activation mutates them optimistically.
  const { data: themingData, refresh: load } = await useAsyncData("admin-theming", async () => {
    const [themesRes, activeRes] = await Promise.all([api.theming.getAll(), api.theming.getActive()]);
    if (themesRes.error || activeRes.error) {
      return null;
    }
    return { themes: themesRes.data ?? [], active: activeRes.data?.active || "builtin:homebox" };
  });

  watch(
    themingData,
    data => {
      if (data) {
        customThemes.value = [...data.themes];
        active.value = data.active;
      }
    },
    { immediate: true }
  );

  const pickerEntries = computed<ThemePickerEntry[]>(() => [
    ...builtinThemes.map(spec => ({
      id: `builtin:${spec.value}`,
      label: spec.label,
      colors: { ...spec.colors, radius: spec.radius },
    })),
    ...customThemes.value.map(theme => ({
      id: `custom:${theme.id}`,
      label: theme.name,
      colors: { ...(theme.colors as ThemePickerEntry["colors"]), radius: theme.radius },
    })),
  ]);

  async function activate(pointer: string) {
    if (pointer === active.value || !can("theming", "edit")) {
      return;
    }
    const previous = active.value;
    active.value = pointer;
    const { error } = await api.theming.setActive(pointer);
    if (error) {
      active.value = previous;
      toast.error(t("errors.api_failure") + String(error));
      return;
    }
    toast.success(t("admin.theming.activated"));
    // The active theme rides on /status; refetch so the new theme applies live.
    await refreshNuxtData("app-status");
  }

  function fontSummary(theme: ThemeOut): string {
    const sans = theme.fontSans && theme.fontSans !== DEFAULT_FONT ? theme.fontSans : t("admin.theming.system_font");
    const mono = theme.fontMono && theme.fontMono !== DEFAULT_FONT ? theme.fontMono : t("admin.theming.system_font");
    return `${sans} · ${mono}`;
  }

  async function createTheme() {
    const seed = builtinTheme("homebox")!;
    const { data, error } = await api.theming.create({
      name: t("admin.theming.new_theme_name"),
      colors: { ...seed.colors },
      radius: seed.radius,
      fontSans: "",
      fontMono: "",
      branding: { appName: "", loginSubtitle: "", socialLinks: [] },
    });
    if (error || !data) {
      toast.error(t("errors.api_failure") + String(error));
      return;
    }
    await navigateTo(`/admin/theming/${data.id}`);
  }

  async function duplicateTheme(theme: ThemeOut) {
    const { data, error } = await api.theming.create({
      name: `${theme.name} (${t("admin.theming.copy_suffix")})`,
      colors: { ...theme.colors },
      radius: theme.radius,
      fontSans: theme.fontSans,
      fontMono: theme.fontMono,
      branding: theme.branding,
    });
    if (error || !data) {
      toast.error(t("errors.api_failure") + String(error));
      return;
    }
    toast.success(t("admin.theming.created"));
    await load();
  }

  async function deleteTheme(theme: ThemeOut) {
    const result = await confirm.open(t("admin.theming.delete_confirm", { name: theme.name }));
    if (result.isCanceled) return;

    deleting.value = { ...deleting.value, [theme.id]: true };
    try {
      const { error } = await api.theming.delete(theme.id);
      if (error) {
        toast.error(t("errors.api_failure") + String(error));
        return;
      }
      customThemes.value = customThemes.value.filter(item => item.id !== theme.id);
      toast.success(t("admin.theming.deleted"));
    } finally {
      deleting.value = { ...deleting.value, [theme.id]: false };
    }
  }
</script>

<template>
  <div class="space-y-6">
    <div>
      <BaseSectionHeader>
        <span>{{ $t("admin.theming.active_theme") }}</span>
        <template #description>{{ $t("admin.theming.active_theme_sub") }}</template>
      </BaseSectionHeader>
      <ThemePicker :model-value="active" :entries="pickerEntries" @update:model-value="activate" />
    </div>

    <div class="space-y-4">
      <div class="flex items-end justify-between">
        <BaseSectionHeader>
          <span>{{ $t("admin.theming.custom_themes") }}</span>
          <template #description>{{ $t("admin.theming.custom_themes_sub") }}</template>
        </BaseSectionHeader>
        <Button v-if="can('theming', 'create')" size="sm" @click="createTheme">
          <MdiPlus class="mr-1 size-4" />
          {{ $t("admin.theming.new_theme") }}
        </Button>
      </div>

      <div v-if="customThemes.length === 0" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
        {{ $t("admin.theming.empty") }}
      </div>

      <div v-else class="scroll-bg overflow-x-auto rounded-md border bg-card">
        <Table class="min-w-[560px]">
          <TableHeader>
            <TableRow>
              <TableHead>{{ $t("admin.theming.name") }}</TableHead>
              <TableHead>{{ $t("admin.theming.fonts") }}</TableHead>
              <TableHead>{{ $t("admin.theming.app_name") }}</TableHead>
              <TableHead class="w-36 text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="theme in customThemes" :key="theme.id">
              <TableCell>
                <div class="flex items-center gap-2">
                  {{ theme.name }}
                  <Badge v-if="active === `custom:${theme.id}`" variant="default">
                    {{ $t("admin.theming.active") }}
                  </Badge>
                </div>
              </TableCell>
              <TableCell class="text-muted-foreground">{{ fontSummary(theme) }}</TableCell>
              <TableCell class="text-muted-foreground">{{ theme.branding?.appName }}</TableCell>
              <TableCell>
                <div class="ml-auto flex justify-end gap-1">
                  <TooltipProvider :delay-duration="0">
                    <Tooltip v-if="can('theming', 'edit')">
                      <TooltipTrigger as-child>
                        <Button
                          variant="outline"
                          size="icon"
                          :aria-label="$t('global.edit')"
                          @click="navigateTo(`/admin/theming/${theme.id}`)"
                        >
                          <MdiPencil class="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{{ $t("global.edit") }}</TooltipContent>
                    </Tooltip>
                    <Tooltip v-if="can('theming', 'create')">
                      <TooltipTrigger as-child>
                        <Button
                          variant="outline"
                          size="icon"
                          :aria-label="$t('admin.theming.duplicate')"
                          @click="duplicateTheme(theme)"
                        >
                          <MdiContentCopy class="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{{ $t("admin.theming.duplicate") }}</TooltipContent>
                    </Tooltip>
                    <Tooltip v-if="can('theming', 'delete') && active !== `custom:${theme.id}`">
                      <TooltipTrigger as-child>
                        <Button
                          variant="destructive"
                          size="icon"
                          :aria-label="$t('global.delete')"
                          :disabled="deleting[theme.id]"
                          @click="deleteTheme(theme)"
                        >
                          <MdiDelete class="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{{ $t("global.delete") }}</TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>
  </div>
</template>
