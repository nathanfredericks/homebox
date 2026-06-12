<template>
  <div class="rounded-lg border bg-card p-4" :class="{ 'opacity-50': !item.include }">
    <div class="mb-3 flex items-start justify-between gap-2">
      <div class="flex items-center gap-2">
        <Switch :id="`include-${index}`" v-model="item.include" />
        <Label :for="`include-${index}`" class="font-medium">
          {{ item.include ? $t("ai.capture.included") : $t("ai.capture.excluded") }}
        </Label>
      </div>
      <NuxtLink
        v-if="item.duplicate"
        :to="`/item/${item.duplicate.id}`"
        class="inline-flex items-center gap-1 rounded-full bg-destructive/10 px-2 py-0.5 text-xs font-medium text-destructive hover:underline"
      >
        <MdiAlert class="size-3.5" />
        {{ $t("ai.capture.duplicate_of", { name: item.duplicate.name }) }}
      </NuxtLink>
    </div>

    <div v-if="item.include" class="flex flex-col gap-3">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <FormTextField v-model="item.name" :label="$t('components.item.create_modal.item_name')" required />
        <FormTextField v-model.number="item.quantity" type="number" :label="$t('global.quantity')" :min="1" />
      </div>

      <FormTextArea v-model="item.description" :label="$t('components.item.create_modal.item_description')" />

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <FormTextField v-model="item.manufacturer" :label="$t('global.manufacturer')" />
        <FormTextField v-model="item.modelNumber" :label="$t('global.model_number')" />
        <FormTextField v-model="item.serialNumber" :label="$t('global.serial_number')" />
        <FormTextField v-model="item.purchaseFrom" :label="$t('global.purchased_from')" />
        <FormTextField v-model.number="item.purchasePrice" type="number" :label="$t('global.purchase_price')" />
        <FormTextField v-model="purchaseDate" type="date" :label="$t('global.purchase_date')" />
      </div>

      <TagSelector v-if="tags.length" v-model="item.tagIds" :tags="tags" />

      <FormTextArea v-model="item.notes" :label="$t('global.notes')" />

      <div v-if="item.fields?.length" class="flex flex-col gap-1.5">
        <Label class="px-1">{{ $t("items.custom_fields") }}</Label>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <template v-for="(field, fi) in item.fields" :key="`cf-${fi}`">
            <div v-if="field.type === 'boolean'" class="flex items-center gap-2 pt-2">
              <Switch :id="`cf-${index}-${fi}`" v-model="field.booleanValue" />
              <Label :for="`cf-${index}-${fi}`">{{ field.name }}</Label>
            </div>
            <FormTextField
              v-else-if="field.type === 'number'"
              v-model.number="field.numberValue"
              type="number"
              :label="field.name"
            />
            <FormTextField
              v-else-if="field.type === 'time'"
              :model-value="(field.timeValue ?? '').slice(0, 10)"
              type="date"
              :label="field.name"
              @update:model-value="(v: string | number) => (field.timeValue = dateToTimeValue(String(v ?? '')))"
            />
            <FormTextField v-else v-model="field.textValue" :label="field.name" :max-length="500" />
          </template>
        </div>
      </div>

      <div v-if="photos.length > 1" class="flex flex-col gap-1.5">
        <Label class="px-1">{{ $t("ai.capture.attach_photos") }}</Label>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="(photo, pi) in photos"
            :key="photo.url"
            type="button"
            class="relative size-16 overflow-hidden rounded-md border-2"
            :class="item.photoIdx.includes(pi) ? 'border-primary' : 'border-transparent opacity-60'"
            @click="togglePhoto(pi)"
          >
            <img :src="photo.url" class="size-full object-cover" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { Label } from "@/components/ui/label";
  import { Switch } from "@/components/ui/switch";
  import FormTextField from "~/components/Form/TextField.vue";
  import FormTextArea from "~/components/Form/TextArea.vue";
  import TagSelector from "~/components/Tag/Selector.vue";
  import MdiAlert from "~icons/mdi/alert";
  import type { TagOut } from "~~/lib/api/types/data-contracts";
  import type { CapturePhoto } from "./PhotoStep.vue";
  import { dateToTimeValue, type EditableDetectedItem } from "~~/composables/use-ai-capture-types";

  withDefaults(
    defineProps<{
      index: number;
      photos: CapturePhoto[];
      tags?: TagOut[];
    }>(),
    { tags: () => [] }
  );

  const item = defineModel<EditableDetectedItem>({ required: true });

  // The model returns purchaseDate as a date-only string, which is exactly
  // what a date input speaks.
  const purchaseDate = computed({
    get: () => String(item.value.purchaseDate ?? "").slice(0, 10),
    set: (v: string) => {
      item.value.purchaseDate = v;
    },
  });

  function togglePhoto(idx: number) {
    const set = new Set(item.value.photoIdx);
    if (set.has(idx)) {
      set.delete(idx);
    } else {
      set.add(idx);
    }
    item.value.photoIdx = [...set].sort((a, b) => a - b);
  }
</script>
