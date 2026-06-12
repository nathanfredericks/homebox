<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import { Button } from "@/components/ui/button";
  import { Checkbox } from "@/components/ui/checkbox";
  import { Label } from "@/components/ui/label";
  import MdiLoading from "~icons/mdi/loading";
  import SectionCard from "~/components/Admin/Settings/SectionCard.vue";
  import SettingField from "~/components/Admin/Settings/SettingField.vue";
  import type { FieldDef } from "~/components/Admin/Settings/SettingField.vue";
  import type { SettingsSection } from "~~/lib/api/classes/admin-settings";
  import type { AdminSettingsOut } from "~~/lib/api/types/data-contracts";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();

  useHead({ title: t("admin.tabs.settings") });

  const api = useUserApi();
  const confirm = useConfirm();
  const { can } = usePermissions();

  // This page must not exist for users without the grant.
  if (!can("site_settings", "view")) {
    await navigateTo("/home", { replace: true });
  }

  const canEdit = computed(() => can("site_settings", "edit"));

  // Mirrors algolia.RecordFields on the backend; an empty allowlist means all.
  const algoliaRecordFields = [
    "assetId",
    "name",
    "description",
    "quantity",
    "insured",
    "archived",
    "purchasePrice",
    "location",
    "tags",
    "soldTime",
    "thumbnailUrl",
    "createdAt",
    "updatedAt",
    "lifetimeWarranty",
    "manufacturer",
    "modelNumber",
    "serialNumber",
    "purchaseFrom",
    "purchaseTime",
    "soldTo",
    "soldPrice",
    "soldNotes",
    "notes",
    "warrantyDetails",
    "warrantyExpires",
  ];

  type SectionDef = {
    id: SettingsSection;
    fields: FieldDef[];
  };

  const sectionDefs = computed<SectionDef[]>(() => [
    {
      id: "options",
      fields: [
        { key: "allowLocalLogin", label: t("admin.settings.options.allow_local_login"), type: "boolean" },
        { key: "autoIncrementAssetId", label: t("admin.settings.options.auto_increment_asset_id"), type: "boolean" },
        {
          key: "hostname",
          label: t("admin.settings.options.hostname"),
          type: "text",
          placeholder: "https://homebox.example.com",
          help: t("admin.settings.options.hostname_help"),
        },
      ],
    },
    {
      id: "thumbnail",
      fields: [
        { key: "enabled", label: t("admin.settings.thumbnail.enabled"), type: "boolean" },
        { key: "width", label: t("admin.settings.thumbnail.width"), type: "number" },
        { key: "height", label: t("admin.settings.thumbnail.height"), type: "number" },
      ],
    },
    {
      id: "mailer",
      fields: [
        { key: "host", label: t("admin.settings.mailer.host"), type: "text", placeholder: "smtp.example.com" },
        { key: "port", label: t("admin.settings.mailer.port"), type: "number", placeholder: "587" },
        { key: "username", label: t("admin.settings.mailer.username"), type: "text" },
        { key: "password", label: t("admin.settings.mailer.password"), type: "secret" },
        { key: "from", label: t("admin.settings.mailer.from"), type: "text", placeholder: "homebox@example.com" },
      ],
    },
    {
      id: "barcode",
      fields: [
        { key: "tokenBarcodespider", label: t("admin.settings.barcode.token_barcodespider"), type: "secret" },
        {
          key: "openFoodFactsContact",
          label: t("admin.settings.barcode.openfoodfacts_contact"),
          type: "text",
          help: t("admin.settings.barcode.openfoodfacts_contact_help"),
        },
      ],
    },
    {
      id: "labelmaker",
      fields: [
        {
          key: "baseUrl",
          label: t("admin.settings.labelmaker.base_url"),
          type: "text",
          placeholder: "https://homebox.example.com",
          help: t("admin.settings.labelmaker.base_url_help"),
        },
        {
          key: "measure",
          label: t("admin.settings.labelmaker.measure"),
          type: "text",
          placeholder: "in",
          help: t("admin.settings.labelmaker.measure_help"),
        },
        { key: "cardWidth", label: t("admin.settings.labelmaker.card_width"), type: "number" },
        { key: "cardHeight", label: t("admin.settings.labelmaker.card_height"), type: "number" },
        { key: "pageWidth", label: t("admin.settings.labelmaker.page_width"), type: "number" },
        { key: "pageHeight", label: t("admin.settings.labelmaker.page_height"), type: "number" },
        { key: "pageTopPadding", label: t("admin.settings.labelmaker.page_top_padding"), type: "number" },
        { key: "pageBottomPadding", label: t("admin.settings.labelmaker.page_bottom_padding"), type: "number" },
        { key: "pageLeftPadding", label: t("admin.settings.labelmaker.page_left_padding"), type: "number" },
        { key: "pageRightPadding", label: t("admin.settings.labelmaker.page_right_padding"), type: "number" },
        { key: "sansFont", label: t("admin.settings.labelmaker.sans_font"), type: "googleFont" },
        { key: "monoFont", label: t("admin.settings.labelmaker.mono_font"), type: "googleFont" },
        { key: "bordered", label: t("admin.settings.labelmaker.bordered"), type: "boolean" },
        { key: "printLocationRow", label: t("admin.settings.labelmaker.print_location_row"), type: "boolean" },
        { key: "labelPerQuantity", label: t("admin.settings.labelmaker.label_per_quantity"), type: "boolean" },
      ],
    },
    {
      id: "notifier",
      fields: [
        {
          key: "allowNets",
          label: t("admin.settings.notifier.allow_nets"),
          type: "list",
          help: t("admin.settings.notifier.nets_help"),
        },
        {
          key: "blockNets",
          label: t("admin.settings.notifier.block_nets"),
          type: "list",
          help: t("admin.settings.notifier.nets_help"),
        },
        { key: "blockLocalhost", label: t("admin.settings.notifier.block_localhost"), type: "boolean" },
        { key: "blockLocalNets", label: t("admin.settings.notifier.block_local_nets"), type: "boolean" },
        { key: "blockBogonNets", label: t("admin.settings.notifier.block_bogon_nets"), type: "boolean" },
        { key: "blockCloudMetadata", label: t("admin.settings.notifier.block_cloud_metadata"), type: "boolean" },
      ],
    },
    {
      id: "algolia",
      fields: [
        { key: "enabled", label: t("admin.settings.algolia.enabled"), type: "boolean" },
        { key: "appId", label: t("admin.settings.algolia.app_id"), type: "text" },
        {
          key: "adminApiKey",
          label: t("admin.settings.algolia.admin_api_key"),
          type: "secret",
          help: t("admin.settings.algolia.admin_api_key_help"),
        },
        { key: "indexName", label: t("admin.settings.algolia.index_name"), type: "text" },
        {
          key: "publicImageUrls",
          label: t("admin.settings.algolia.public_image_urls"),
          type: "boolean",
        },
        {
          key: "publicBaseUrl",
          label: t("admin.settings.algolia.public_base_url"),
          type: "text",
          placeholder: "https://homebox.example.com",
          help: t("admin.settings.algolia.public_base_url_help"),
        },
        {
          key: "reindexInterval",
          label: t("admin.settings.algolia.reindex_interval"),
          type: "text",
          placeholder: "24h",
          help: t("admin.settings.algolia.reindex_interval_help"),
        },
      ],
    },
    {
      id: "ai",
      fields: [
        { key: "enabled", label: t("admin.settings.ai.enabled"), type: "boolean" },
        {
          key: "baseUrl",
          label: t("admin.settings.ai.base_url"),
          type: "text",
          placeholder: "https://api.openai.com/v1",
          help: t("admin.settings.ai.base_url_help"),
        },
        { key: "apiKey", label: t("admin.settings.ai.api_key"), type: "secret" },
        {
          key: "model",
          label: t("admin.settings.ai.model"),
          type: "text",
          placeholder: "gpt-4o-mini",
          help: t("admin.settings.ai.model_help"),
        },
        {
          key: "extraInstructions",
          label: t("admin.settings.ai.extra_instructions"),
          type: "text",
          help: t("admin.settings.ai.extra_instructions_help"),
        },
      ],
    },
  ]);

  type SectionState = Record<string, unknown>;

  const forms = reactive<Record<string, SectionState>>({});
  const initial = ref<Record<string, SectionState>>({});
  const saving = reactive<Record<string, boolean>>({});
  const reindexing = ref(false);

  function applyResponse(doc: AdminSettingsOut) {
    const sections = doc.settings as unknown as Record<string, SectionState>;
    for (const def of sectionDefs.value) {
      forms[def.id] = { ...(sections[def.id] ?? {}) };
    }
    initial.value = JSON.parse(JSON.stringify(forms));
  }

  // SSR-first load; failures render the inline error block.
  const { data: loaded, status } = await useAsyncData("admin-settings", async () => {
    const res = await api.adminSettings.get();
    if (res.error || !res.data) return null;
    return res.data;
  });

  watch(
    loaded,
    doc => {
      if (doc) applyResponse(doc);
    },
    { immediate: true }
  );

  const loading = computed(() => status.value === "pending");
  const loadFailed = computed(() => status.value !== "pending" && !loaded.value);

  function setField(section: SettingsSection, key: string, value: unknown) {
    const state = forms[section];
    if (state) state[key] = value;
  }

  const isDirty = (section: SettingsSection) =>
    JSON.stringify(forms[section] ?? {}) !== JSON.stringify(initial.value[section] ?? {});

  const dirtyKeys = (section: SettingsSection): SectionState => {
    const cur = forms[section] ?? {};
    const base = initial.value[section] ?? {};
    const out: SectionState = {};
    for (const [k, v] of Object.entries(cur)) {
      if (JSON.stringify(v) !== JSON.stringify(base[k])) out[k] = v;
    }
    return out;
  };

  async function saveSection(section: SettingsSection) {
    const payload = dirtyKeys(section);
    if (!Object.keys(payload).length) return;

    saving[section] = true;
    try {
      const res = await api.adminSettings.updateSection(section, payload);
      if (res.error || !res.data) {
        toast.error(t("admin.settings.save_failed"));
        return;
      }
      applyResponse(res.data);
      toast.success(t("admin.settings.saved"));
    } finally {
      saving[section] = false;
    }
  }

  async function resetSection(section: SettingsSection) {
    const result = await confirm.open(t("admin.settings.reset_confirm"));
    if (result.isCanceled) return;

    saving[section] = true;
    try {
      const res = await api.adminSettings.resetSection(section);
      if (res.error || !res.data) {
        toast.error(t("admin.settings.save_failed"));
        return;
      }
      applyResponse(res.data);
      toast.success(t("admin.settings.reset_done"));
    } finally {
      saving[section] = false;
    }
  }

  // Algolia field allowlist: CSV string in the payload, checkbox set in the UI.
  const algoliaSelectedFields = computed<Set<string>>(() => {
    const csv = String(forms.algolia?.fields ?? "");
    if (!csv.trim()) return new Set(algoliaRecordFields);
    return new Set(
      csv
        .split(",")
        .map(s => s.trim())
        .filter(Boolean)
    );
  });

  function toggleAlgoliaField(field: string, checked: boolean) {
    const next = new Set(algoliaSelectedFields.value);
    if (checked) {
      next.add(field);
    } else {
      next.delete(field);
    }
    if (!forms.algolia) return;
    // All fields selected = empty allowlist = send everything.
    forms.algolia.fields =
      next.size === algoliaRecordFields.length ? "" : algoliaRecordFields.filter(f => next.has(f)).join(",");
  }

  const algoliaEnabledSaved = computed(() => initial.value.algolia?.enabled === true);

  async function reindexNow() {
    reindexing.value = true;
    try {
      const res = await api.adminSettings.algoliaReindex();
      if (res.error) {
        toast.error(t("admin.settings.algolia.reindex_failed"));
      } else {
        toast.success(t("admin.settings.algolia.reindex_started"));
      }
    } finally {
      reindexing.value = false;
    }
  }
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
      {{ $t("global.loading") }}
    </div>

    <div v-else-if="loadFailed" class="rounded-md border bg-card p-4 text-sm text-muted-foreground">
      {{ $t("admin.settings.load_failed") }}
    </div>

    <template v-else>
      <SectionCard
        v-for="def in sectionDefs"
        :key="def.id"
        :title="$t(`admin.settings.${def.id}.title`)"
        :description="$t(`admin.settings.${def.id}.description`)"
        :saving="saving[def.id] === true"
        :dirty="isDirty(def.id)"
        :can-edit="canEdit"
        @save="saveSection(def.id)"
        @reset="resetSection(def.id)"
      >
        <SettingField
          v-for="field in def.fields"
          :key="field.key"
          :def="field"
          :model-value="forms[def.id]?.[field.key]"
          :view-only="!canEdit"
          @update:model-value="v => setField(def.id, field.key, v)"
        />

        <template v-if="def.id === 'algolia'">
          <div class="space-y-2">
            <Label class="px-1">{{ $t("admin.settings.algolia.fields") }}</Label>
            <p class="px-1 text-xs text-muted-foreground">{{ $t("admin.settings.algolia.fields_help") }}</p>
            <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
              <div v-for="field in algoliaRecordFields" :key="field" class="flex items-center gap-2">
                <Checkbox
                  v-if="canEdit"
                  :id="`algolia-field-${field}`"
                  :model-value="algoliaSelectedFields.has(field)"
                  @update:model-value="checked => toggleAlgoliaField(field, checked === true)"
                />
                <span v-else class="text-xs">{{ algoliaSelectedFields.has(field) ? "✓" : "✗" }}</span>
                <Label :for="`algolia-field-${field}`" class="font-mono text-xs">{{ field }}</Label>
              </div>
            </div>
          </div>
        </template>

        <template v-if="def.id === 'algolia' && canEdit && algoliaEnabledSaved" #actions>
          <Button variant="outline" size="sm" :disabled="reindexing" @click="reindexNow">
            <MdiLoading v-if="reindexing" class="mr-1 size-4 animate-spin" />
            {{ $t("admin.settings.algolia.reindex_now") }}
          </Button>
        </template>
      </SectionCard>
    </template>
  </div>
</template>
