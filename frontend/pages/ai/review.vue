<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import { Button } from "@/components/ui/button";
  import { Checkbox } from "@/components/ui/checkbox";
  import { Label } from "@/components/ui/label";
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import LocationSelector from "~/components/Location/Selector.vue";
  import MdiCreation from "~icons/mdi/creation";
  import MdiLoading from "~icons/mdi/loading";
  import type { AiFieldSuggestion, EntitySummary } from "~~/lib/api/types/data-contracts";
  import { suggestionsToPatch } from "~~/composables/use-ai-capture-types";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();
  useHead({ title: t("ai.review.title") });

  const api = useUserApi();
  const { canAny } = usePermissions();

  // Batch review edits items, so it requires the items edit grant on top of
  // the AI surface; the page must not exist otherwise.
  if (!canAny("ai", "view") || !canAny("items", "edit")) {
    await navigateTo("/home", { replace: true });
  }
  const { data: status } = await useAsyncData("ai-status", async () => {
    const { data, error } = await api.ai.status();
    if (error || !data) return false;
    return data.enabled;
  });
  if (status.value !== true) {
    await navigateTo("/home", { replace: true });
  }

  const location = ref<EntitySummary | null>(null);
  const overwrite = ref(false);

  type Phase = "pick" | "review" | "done";
  const phase = ref<Phase>("pick");

  const queue = ref<EntitySummary[]>([]);
  const position = ref(0);
  const current = computed(() => queue.value[position.value] ?? null);

  const loading = ref(false);
  const applying = ref(false);
  const suggestions = ref<AiFieldSuggestion[]>([]);
  const appliedCount = ref(0);
  const skippedCount = ref(0);

  const FIELD_LABELS: Record<string, string> = {
    name: "components.item.create_modal.item_name",
    description: "components.item.create_modal.item_description",
    manufacturer: "global.manufacturer",
    modelNumber: "global.model_number",
    serialNumber: "global.serial_number",
    purchasePrice: "global.purchase_price",
    purchaseFrom: "global.purchased_from",
    notes: "global.notes",
  };

  function fieldLabel(field: string): string {
    const key = FIELD_LABELS[field];
    return key ? t(key) : field;
  }

  async function start() {
    if (!location.value) return;
    loading.value = true;
    try {
      const { data, error } = await api.items.getAll({
        parentIds: [location.value.id],
        onlyWithPhoto: true,
        pageSize: 100,
      });
      if (error || !data) {
        toast.error(t("ai.fill.failed"));
        return;
      }
      queue.value = data.items ?? [];
      position.value = 0;
      appliedCount.value = 0;
      skippedCount.value = 0;
      if (queue.value.length === 0) {
        toast.info(t("ai.review.no_items"));
        return;
      }
      phase.value = "review";
      await fetchCurrent();
    } finally {
      loading.value = false;
    }
  }

  async function fetchCurrent() {
    if (!current.value) {
      phase.value = "done";
      return;
    }
    loading.value = true;
    suggestions.value = [];
    try {
      const { data, error } = await api.ai.suggest(current.value.id, overwrite.value);
      if (error || !data) {
        // Unanalyzable item (e.g. unreadable photos) — skip it silently.
        await next(true);
        return;
      }
      suggestions.value = data.suggestions;
      if (data.suggestions.length === 0) {
        await next(true);
      }
    } finally {
      loading.value = false;
    }
  }

  async function next(skipped: boolean) {
    if (skipped) {
      skippedCount.value++;
    }
    position.value++;
    if (position.value >= queue.value.length) {
      phase.value = "done";
      return;
    }
    await fetchCurrent();
  }

  async function approve() {
    if (!current.value) return;
    applying.value = true;
    try {
      const patch = suggestionsToPatch(current.value.id, suggestions.value);
      const { error } = await api.items.patch(current.value.id, patch);
      if (error) {
        toast.error(t("ai.fill.apply_failed"));
        return;
      }
      appliedCount.value++;
      await next(false);
    } finally {
      applying.value = false;
    }
  }

  function restart() {
    phase.value = "pick";
    queue.value = [];
    suggestions.value = [];
  }
</script>

<template>
  <BaseContainer class="mb-6 flex flex-col gap-4">
    <BaseCard>
      <template #title>
        <BaseSectionHeader>
          <MdiCreation class="mr-2" />
          <span>{{ $t("ai.review.title") }}</span>
          <template #description>{{ $t("ai.review.description") }}</template>
        </BaseSectionHeader>
      </template>

      <div class="flex flex-col gap-4 border-t p-4 sm:px-6">
        <!-- Pick a location -->
        <template v-if="phase === 'pick'">
          <LocationSelector v-model="location" />
          <div class="flex items-center gap-2">
            <Checkbox id="aiReviewOverwrite" v-model="overwrite" />
            <Label class="cursor-pointer" for="aiReviewOverwrite">{{ $t("ai.fill.overwrite") }}</Label>
          </div>
          <Button :disabled="!location || loading" @click="start">
            <MdiLoading v-if="loading" class="animate-spin" />
            {{ $t("ai.review.start") }}
          </Button>
        </template>

        <!-- Review queue -->
        <template v-else-if="phase === 'review'">
          <p class="text-sm text-muted-foreground">
            {{ $t("ai.review.progress", { current: position + 1, total: queue.length }) }}
          </p>

          <h2 class="text-lg font-medium">
            <NuxtLink v-if="current" :to="`/item/${current.id}`" class="hover:underline" target="_blank">
              {{ current.name }}
            </NuxtLink>
          </h2>

          <div v-if="loading" class="flex items-center justify-center gap-2 py-8 text-muted-foreground">
            <MdiLoading class="size-5 animate-spin" />
            {{ $t("ai.capture.analyzing") }}
          </div>

          <template v-else>
            <div v-for="s in suggestions" :key="s.field" class="rounded-md border p-3">
              <div class="font-medium">{{ fieldLabel(s.field) }}</div>
              <div class="mt-1 grid grid-cols-1 gap-1 text-sm">
                <div v-if="s.current" class="text-muted-foreground line-through">{{ s.current }}</div>
                <div>{{ s.suggested }}</div>
              </div>
            </div>

            <div class="flex gap-2">
              <Button variant="outline" class="grow" :disabled="applying" @click="next(true)">
                {{ $t("ai.review.skip") }}
              </Button>
              <Button class="grow" :disabled="applying || suggestions.length === 0" @click="approve">
                <MdiLoading v-if="applying" class="animate-spin" />
                {{ $t("ai.review.approve") }}
              </Button>
            </div>
          </template>
        </template>

        <!-- Done -->
        <template v-else>
          <p class="text-sm">
            {{ $t("ai.review.done", { applied: appliedCount, skipped: skippedCount }) }}
          </p>
          <Button @click="restart">
            {{ $t("ai.review.start") }}
          </Button>
        </template>
      </div>
    </BaseCard>
  </BaseContainer>
</template>
