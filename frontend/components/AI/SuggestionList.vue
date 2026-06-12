<template>
  <div class="flex flex-col gap-3">
    <div v-for="s in suggestions" :key="s.field" class="rounded-md border p-3">
      <div class="flex items-center gap-2">
        <Checkbox :id="`sugg-${uid}-${s.field}`" v-model="accepted[s.field]" />
        <Label class="cursor-pointer font-medium" :for="`sugg-${uid}-${s.field}`">
          {{ fieldLabel(s.field) }}
        </Label>
      </div>
      <div class="mt-2 grid grid-cols-1 gap-1 text-sm">
        <div v-if="s.current" class="text-muted-foreground line-through">{{ s.current }}</div>
        <!-- AI output is a starting point, not the answer: every accepted
             value stays editable until it is applied. -->
        <Input
          v-model="edited[s.field]"
          :type="inputType(s.field)"
          :disabled="!accepted[s.field]"
          :aria-label="$t('ai.fill.edit_value', { field: fieldLabel(s.field) })"
        />
      </div>
    </div>

    <div v-if="tags.length" class="rounded-md border p-3">
      <p class="mb-2 text-sm font-medium">{{ $t("ai.fill.suggested_tags") }}</p>
      <div class="flex flex-col gap-2">
        <div v-for="tag in tags" :key="tag.id" class="flex items-center gap-2">
          <Checkbox :id="`sugg-${uid}-tag-${tag.id}`" v-model="acceptedTags[tag.id]" />
          <Label class="cursor-pointer" :for="`sugg-${uid}-tag-${tag.id}`">{{ tag.name }}</Label>
        </div>
      </div>
    </div>

    <div v-if="customFields.length" class="rounded-md border p-3">
      <p class="mb-2 text-sm font-medium">{{ $t("ai.fill.suggested_custom_fields") }}</p>
      <div class="flex flex-col gap-3">
        <div v-for="cf in customFields" :key="cf.name">
          <div class="flex items-center gap-2">
            <Checkbox :id="`sugg-${uid}-cf-${cf.name}`" v-model="acceptedCustom[cf.name]" />
            <Label class="cursor-pointer font-medium" :for="`sugg-${uid}-cf-${cf.name}`">{{ cf.name }}</Label>
          </div>
          <div v-if="cf.type !== 'boolean'" class="mt-2 grid grid-cols-1 gap-1 text-sm">
            <div v-if="cf.current" class="text-muted-foreground line-through">{{ cf.current }}</div>
            <Input
              v-model="editedCustom[cf.name]"
              :type="cf.type === 'number' ? 'number' : cf.type === 'time' ? 'date' : 'text'"
              :disabled="!acceptedCustom[cf.name]"
              :aria-label="$t('ai.fill.edit_value', { field: cf.name })"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { Checkbox } from "@/components/ui/checkbox";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import type { AiCustomFieldSuggestion, AiFieldSuggestion, AiTagSuggestion } from "~~/lib/api/types/data-contracts";

  /** What the user accepted, with their edits applied. */
  export type SuggestionSelection = {
    fields: AiFieldSuggestion[];
    tagIds: string[];
    customFields: AiCustomFieldSuggestion[];
  };

  const props = defineProps<{
    suggestions: AiFieldSuggestion[];
    tags: AiTagSuggestion[];
    customFields: AiCustomFieldSuggestion[];
  }>();

  const { t } = useI18n();
  const uid = useId();

  const accepted = ref<Record<string, boolean>>({});
  const edited = ref<Record<string, string>>({});
  const acceptedTags = ref<Record<string, boolean>>({});
  const acceptedCustom = ref<Record<string, boolean>>({});
  const editedCustom = ref<Record<string, string>>({});

  // Everything starts accepted with the model's value; a new result set
  // (refetch, next item in the queue) resets the working state.
  watch(
    () => [props.suggestions, props.tags, props.customFields] as const,
    ([suggestions, tags, customFields]) => {
      accepted.value = Object.fromEntries(suggestions.map(s => [s.field, true]));
      edited.value = Object.fromEntries(suggestions.map(s => [s.field, s.suggested]));
      acceptedTags.value = Object.fromEntries(tags.map(tag => [tag.id, true]));
      acceptedCustom.value = Object.fromEntries(customFields.map(cf => [cf.name, true]));
      editedCustom.value = Object.fromEntries(customFields.map(cf => [cf.name, cf.suggested]));
    },
    { immediate: true }
  );

  const FIELD_LABELS: Record<string, string> = {
    name: "components.item.create_modal.item_name",
    description: "components.item.create_modal.item_description",
    quantity: "global.quantity",
    manufacturer: "global.manufacturer",
    modelNumber: "global.model_number",
    serialNumber: "global.serial_number",
    purchasePrice: "global.purchase_price",
    purchaseFrom: "global.purchased_from",
    purchaseDate: "global.purchase_date",
    notes: "global.notes",
  };

  function fieldLabel(field: string): string {
    const key = FIELD_LABELS[field];
    return key ? t(key) : field;
  }

  function inputType(field: string): string {
    if (field === "purchasePrice" || field === "quantity") return "number";
    if (field === "purchaseDate") return "date";
    return "text";
  }

  const selection = computed<SuggestionSelection>(() => ({
    fields: props.suggestions
      .filter(s => accepted.value[s.field] && (edited.value[s.field] ?? "").trim() !== "")
      .map(s => ({ ...s, suggested: edited.value[s.field]!.trim() })),
    tagIds: props.tags.filter(tag => acceptedTags.value[tag.id]).map(tag => tag.id),
    customFields: props.customFields
      .filter(
        cf =>
          acceptedCustom.value[cf.name] && (cf.type === "boolean" || (editedCustom.value[cf.name] ?? "").trim() !== "")
      )
      .map(cf => (cf.type === "boolean" ? { ...cf } : { ...cf, suggested: editedCustom.value[cf.name]!.trim() })),
  }));

  const selectedCount = computed(
    () => selection.value.fields.length + selection.value.tagIds.length + selection.value.customFields.length
  );

  function acceptAll() {
    accepted.value = Object.fromEntries(props.suggestions.map(s => [s.field, true]));
    acceptedTags.value = Object.fromEntries(props.tags.map(tag => [tag.id, true]));
    acceptedCustom.value = Object.fromEntries(props.customFields.map(cf => [cf.name, true]));
  }

  defineExpose({ selection, selectedCount, acceptAll });
</script>
