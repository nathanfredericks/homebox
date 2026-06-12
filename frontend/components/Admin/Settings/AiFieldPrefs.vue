<template>
  <div class="space-y-2">
    <Label class="px-1">{{ $t("admin.settings.ai.fields_title") }}</Label>
    <p class="px-1 text-xs text-muted-foreground">{{ $t("admin.settings.ai.fields_help") }}</p>

    <div class="space-y-3">
      <div v-for="key in FIELD_KEYS" :key="key" class="flex flex-col gap-1.5">
        <div class="flex items-center gap-3">
          <template v-if="viewOnly">
            <span class="text-xs">{{ isEnabled(key) ? "✓" : "✗" }}</span>
            <span class="text-sm text-muted-foreground">{{ fieldLabel(key) }}</span>
          </template>
          <template v-else>
            <Switch
              :id="`ai-field-${key}`"
              :model-value="isEnabled(key)"
              :disabled="key === 'name'"
              @update:model-value="v => setEnabled(key, v === true)"
            />
            <Label :for="`ai-field-${key}`">{{ fieldLabel(key) }}</Label>
          </template>
        </div>
        <template v-if="isEnabled(key)">
          <p v-if="viewOnly && instruction(key)" class="px-1 text-xs">{{ instruction(key) }}</p>
          <Input
            v-else-if="!viewOnly"
            :model-value="instruction(key)"
            :placeholder="defaultRule(key)"
            :aria-label="$t('admin.settings.ai.fields_instruction', { field: fieldLabel(key) })"
            autocomplete="off"
            @update:model-value="v => setInstruction(key, String(v))"
          />
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import { Switch } from "@/components/ui/switch";

  /** Mirrors config.AIFieldConf on the backend. */
  type FieldPref = { enabled: boolean; instruction: string };
  type FieldPrefs = Record<string, FieldPref | undefined>;

  // Mirrors config.AIFieldConfs; the name toggle is rendered locked because
  // the backend always keeps the name field on.
  const FIELD_KEYS = [
    "name",
    "quantity",
    "description",
    "manufacturer",
    "modelNumber",
    "serialNumber",
    "purchasePrice",
    "purchaseFrom",
    "purchaseDate",
    "notes",
    "tags",
    "customFields",
  ] as const;

  const props = defineProps<{
    modelValue: unknown;
    viewOnly: boolean;
  }>();

  const emit = defineEmits<{ "update:modelValue": [value: FieldPrefs] }>();

  const { t } = useI18n();

  const prefs = computed<FieldPrefs>(() => (props.modelValue ?? {}) as FieldPrefs);

  const isEnabled = (key: string) => prefs.value[key]?.enabled !== false;
  const instruction = (key: string) => prefs.value[key]?.instruction ?? "";

  // Always emit the complete fields object: the settings page tracks dirtiness
  // and saves per top-level section key, and the backend replaces the stored
  // "fields" override wholesale.
  function update(key: string, patch: Partial<FieldPref>) {
    const next: FieldPrefs = {};
    for (const k of FIELD_KEYS) {
      next[k] = { enabled: isEnabled(k), instruction: instruction(k) };
    }
    next[key] = { enabled: isEnabled(key), instruction: instruction(key), ...patch };
    emit("update:modelValue", next);
  }

  const setEnabled = (key: string, enabled: boolean) => update(key, { enabled });
  const setInstruction = (key: string, instr: string) => update(key, { instruction: instr });

  const fieldLabel = (key: string) => t(`admin.settings.ai.fields.${key}`);
  // Shows the built-in prompt rule so admins know what they're overriding.
  // Duplicated from ai.defaultFieldRules on the backend — keep in sync.
  const defaultRule = (key: string) => t(`admin.settings.ai.fields.${key}_default`);
</script>
