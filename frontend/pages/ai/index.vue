<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import { Button } from "@/components/ui/button";
  import { Label } from "@/components/ui/label";
  import { Checkbox } from "@/components/ui/checkbox";
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import LocationSelector from "~/components/Location/Selector.vue";
  import PhotoStep, { type CapturePhoto } from "~/components/AI/PhotoStep.vue";
  import DetectedItemCard from "~/components/AI/DetectedItemCard.vue";
  import CorrectionBar from "~/components/AI/CorrectionBar.vue";
  import CreateProgress, { type CreateResult } from "~/components/AI/CreateProgress.vue";
  import MdiCreation from "~icons/mdi/creation";
  import MdiArrowLeft from "~icons/mdi/arrow-left";
  import MdiLoading from "~icons/mdi/loading";
  import type { EntityCreate, EntitySummary, EntityTypeSummary } from "~~/lib/api/types/data-contracts";
  import { AttachmentTypes } from "~~/lib/api/types/non-generated";
  import type { EditableDetectedItem } from "~~/composables/use-ai-capture-types";

  definePageMeta({
    middleware: ["auth"],
  });

  const { t } = useI18n();
  useHead({ title: t("ai.capture.title") });

  const api = useUserApi();
  const { canAny, can } = usePermissions();

  // This page must not exist for users without the grant or when AI is off.
  if (!canAny("ai", "view")) {
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

  const MAX_PHOTOS = 8;

  type Step = "location" | "photos" | "review" | "create";
  const step = ref<Step>("location");

  // --- Step 1: location -----------------------------------------------------
  const location = ref<EntitySummary | null>(null);

  // --- Step 2: photos -------------------------------------------------------
  const photos = ref<CapturePhoto[]>([]);
  const singleItem = ref(false);

  function addPhotos(files: File[]) {
    for (const file of files) {
      photos.value.push({ file, url: URL.createObjectURL(file) });
    }
  }

  function removePhoto(index: number) {
    const [removed] = photos.value.splice(index, 1);
    if (removed) URL.revokeObjectURL(removed.url);
  }

  onUnmounted(() => {
    for (const photo of photos.value) {
      URL.revokeObjectURL(photo.url);
    }
  });

  // Photos are downscaled client-side before upload when the browser can
  // decode them (HEIC falls through unchanged; the server decodes it).
  async function compressPhoto(file: File): Promise<Blob> {
    try {
      const bitmap = await createImageBitmap(file);
      const maxEdge = 1568;
      const scale = Math.min(1, maxEdge / Math.max(bitmap.width, bitmap.height));
      const canvas = document.createElement("canvas");
      canvas.width = Math.round(bitmap.width * scale);
      canvas.height = Math.round(bitmap.height * scale);
      canvas.getContext("2d")!.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
      bitmap.close();
      const blob = await new Promise<Blob | null>(resolve => canvas.toBlob(resolve, "image/jpeg", 0.85));
      return blob ?? file;
    } catch {
      return file;
    }
  }

  // --- Step 3: analyze + review ---------------------------------------------
  const analyzing = ref(false);
  const items = ref<EditableDetectedItem[]>([]);

  function toEditable(detected: EditableDetectedItem[] | null): void {
    items.value = (detected ?? []).map(item => ({
      ...item,
      include: true,
      photoIdx: photos.value.map((_, i) => i),
    }));
  }

  async function analyze(feedback?: string) {
    analyzing.value = true;
    try {
      const compressed = await Promise.all(photos.value.map(p => compressPhoto(p.file)));
      const { data, error } = await api.ai.analyze(compressed, {
        singleItem: singleItem.value,
        ...(feedback
          ? {
              feedback,
              priorItems: items.value.map(({ include: _i, photoIdx: _p, duplicate: _d, ...rest }) => rest),
            }
          : {}),
      });
      if (error || !data) {
        toast.error(t("ai.capture.analyze_failed"));
        return;
      }
      toEditable(data.items as EditableDetectedItem[]);
      if (!data.items.length) {
        toast.info(t("ai.capture.nothing_detected"));
      }
      step.value = "review";
    } finally {
      analyzing.value = false;
    }
  }

  const includedItems = computed(() => items.value.filter(i => i.include));

  // --- Step 4: create -------------------------------------------------------
  const creating = ref(false);
  const results = ref<CreateResult[]>([]);

  // Items created from capture use the collection's default (first) item type.
  const itemEntityType = ref<EntityTypeSummary | null>(null);
  onMounted(async () => {
    const { data, error } = await api.entityTypes.getAll();
    if (!error && data) {
      itemEntityType.value = data.filter(et => !et.isLocation)[0] ?? null;
    }
  });

  async function createAll() {
    if (!location.value) return;
    step.value = "create";
    creating.value = true;
    results.value = includedItems.value.map(item => ({ name: item.name, status: "pending" }));

    for (const [idx, item] of includedItems.value.entries()) {
      const result = results.value[idx]!;
      try {
        // entityTypeId is omitted when no item type is known: the backend
        // falls back to the collection's default item type.
        const payload = {
          parentId: location.value.id,
          name: item.name,
          quantity: item.quantity || 1,
          description: item.description,
          tagIds: [],
          ...(itemEntityType.value ? { entityTypeId: itemEntityType.value.id } : {}),
          serialNumber: item.serialNumber,
          modelNumber: item.modelNumber,
          manufacturer: item.manufacturer,
          notes: item.notes,
        } as EntityCreate;
        const { data: created, error } = await api.items.create(payload);
        if (error || !created) {
          result.status = "failed";
          continue;
        }
        result.id = created.id;

        if (item.purchasePrice > 0 || item.purchaseFrom) {
          await api.items.patch(created.id, {
            id: created.id,
            ...(item.purchasePrice > 0 ? { purchasePrice: item.purchasePrice } : {}),
            ...(item.purchaseFrom ? { purchaseFrom: item.purchaseFrom } : {}),
          });
        }

        for (const [n, photoIndex] of item.photoIdx.entries()) {
          const photo = photos.value[photoIndex];
          if (!photo) continue;
          await api.items.attachments.add(
            created.id,
            photo.file,
            photo.file.name || `photo-${n + 1}.jpg`,
            AttachmentTypes.Photo,
            n === 0
          );
        }

        result.status = "created";
      } catch {
        result.status = "failed";
      }
    }
    creating.value = false;
  }

  function restart() {
    for (const photo of photos.value) {
      URL.revokeObjectURL(photo.url);
    }
    photos.value = [];
    items.value = [];
    results.value = [];
    singleItem.value = false;
    step.value = "location";
  }

  const createdCount = computed(() => results.value.filter(r => r.status === "created").length);
</script>

<template>
  <BaseContainer class="mb-6 flex flex-col gap-4">
    <BaseCard>
      <template #title>
        <BaseSectionHeader>
          <MdiCreation class="mr-2" />
          <span>{{ $t("ai.capture.title") }}</span>
          <template #description>{{ $t("ai.capture.description") }}</template>
        </BaseSectionHeader>
      </template>

      <div class="flex flex-col gap-4 border-t p-4 sm:px-6">
        <!-- Step 1: pick a location -->
        <template v-if="step === 'location'">
          <LocationSelector v-model="location" />
          <Button :disabled="!location" @click="step = 'photos'">
            {{ $t("ai.capture.next") }}
          </Button>
          <NuxtLink
            v-if="canAny('items', 'edit')"
            to="/ai/review"
            class="text-sm text-primary underline-offset-4 hover:underline"
          >
            {{ $t("ai.review.title") }} →
          </NuxtLink>
        </template>

        <!-- Step 2: capture photos -->
        <template v-else-if="step === 'photos'">
          <p class="text-sm text-muted-foreground">
            {{ $t("ai.capture.target_location", { location: location?.name }) }}
          </p>
          <PhotoStep :photos="photos" :max-photos="MAX_PHOTOS" @add="addPhotos" @remove="removePhoto" />
          <div class="flex items-center gap-2">
            <Checkbox id="aiSingleItem" v-model="singleItem" />
            <Label class="cursor-pointer" for="aiSingleItem">{{ $t("ai.capture.single_item") }}</Label>
          </div>
          <div class="flex gap-2">
            <Button variant="outline" @click="step = 'location'">
              <MdiArrowLeft />
              {{ $t("ai.capture.back") }}
            </Button>
            <Button class="grow" :disabled="photos.length === 0 || analyzing" @click="analyze()">
              <MdiLoading v-if="analyzing" class="animate-spin" />
              <MdiCreation v-else />
              {{ analyzing ? $t("ai.capture.analyzing") : $t("ai.capture.analyze") }}
            </Button>
          </div>
        </template>

        <!-- Step 3: review detected items -->
        <template v-else-if="step === 'review'">
          <p class="text-sm text-muted-foreground">
            {{ $t("ai.capture.review_hint", { count: items.length, location: location?.name }) }}
          </p>

          <DetectedItemCard
            v-for="(item, idx) in items"
            :key="idx"
            v-model="items[idx]!"
            :index="idx"
            :photos="photos"
          />

          <CorrectionBar :busy="analyzing" @correct="feedback => analyze(feedback)" />

          <div class="flex gap-2">
            <Button variant="outline" @click="step = 'photos'">
              <MdiArrowLeft />
              {{ $t("ai.capture.back") }}
            </Button>
            <Button
              v-if="can('items', 'create')"
              class="grow"
              :disabled="includedItems.length === 0 || analyzing"
              @click="createAll"
            >
              {{ $t("ai.capture.create_items", { count: includedItems.length }) }}
            </Button>
          </div>
        </template>

        <!-- Step 4: creation progress -->
        <template v-else>
          <CreateProgress :results="results" />
          <p v-if="!creating" class="text-sm text-muted-foreground">
            {{ $t("ai.capture.done_summary", { count: createdCount }) }}
          </p>
          <Button v-if="!creating" @click="restart">
            {{ $t("ai.capture.scan_more") }}
          </Button>
        </template>
      </div>
    </BaseCard>
  </BaseContainer>
</template>
