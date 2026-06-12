<template>
  <div class="flex flex-col gap-4">
    <div class="grid grid-cols-3 gap-2 sm:grid-cols-4">
      <!-- aspect-square is a Tailwind core utility; the eslint plugin
      misses it because the aspect-ratio plugin is also installed. -->
      <!-- eslint-disable tailwindcss/no-custom-classname -->
      <div v-for="(photo, idx) in photos" :key="photo.url" class="aspect-square group relative">
        <img :src="photo.url" class="size-full rounded-md border object-cover" />
        <Button
          type="button"
          variant="destructive"
          size="icon"
          class="absolute right-1 top-1 size-6"
          @click="$emit('remove', idx)"
        >
          <MdiClose class="size-4" />
        </Button>
      </div>

      <button
        v-if="photos.length < maxPhotos"
        type="button"
        class="aspect-square flex flex-col items-center justify-center gap-1 rounded-md border border-dashed text-muted-foreground hover:bg-accent"
        @click="fileInput?.click()"
      >
        <MdiCameraPlus class="size-8" />
        <span class="text-xs">{{ $t("ai.capture.add_photos") }}</span>
      </button>
    </div>

    <input
      ref="fileInput"
      type="file"
      accept="image/*"
      capture="environment"
      multiple
      class="hidden"
      @change="onFilesPicked"
    />

    <p class="text-sm text-muted-foreground">
      {{ $t("ai.capture.photo_hint", { max: maxPhotos }) }}
    </p>
  </div>
</template>

<script setup lang="ts">
  import { Button } from "@/components/ui/button";
  import MdiClose from "~icons/mdi/close";
  import MdiCameraPlus from "~icons/mdi/camera-plus";

  export type CapturePhoto = {
    file: File;
    url: string;
  };

  const props = defineProps<{
    photos: CapturePhoto[];
    maxPhotos: number;
  }>();

  const emit = defineEmits<{
    add: [files: File[]];
    remove: [index: number];
  }>();

  const fileInput = ref<HTMLInputElement | null>(null);

  function onFilesPicked(event: Event) {
    const input = event.target as HTMLInputElement;
    const files = Array.from(input.files ?? []).slice(0, props.maxPhotos - props.photos.length);
    if (files.length) {
      emit("add", files);
    }
    input.value = "";
  }
</script>
