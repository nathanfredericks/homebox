<script setup lang="ts">
  import "@fontsource/open-sans/400.css";
  import "@fontsource/open-sans/700.css";
  import "@fontsource/geist-mono/400.css";
  import { MapPin } from "lucide-vue-next";

  withDefaults(
    defineProps<{
      name: string;
      assetId?: string | null;
      location?: string | null;
      qrUrl: string;
      width: number;
      height: number;
      measure: string;
      bordered?: boolean;
      showLocation?: boolean;
      sansFontFamily: string;
      monoFontFamily: string;
    }>(),
    {
      assetId: null,
      location: null,
      bordered: false,
      showLocation: true,
    }
  );
</script>

<template>
  <div
    class="flex items-center border-2"
    :class="{
      'border-black': bordered,
      'border-transparent': !bordered,
    }"
    :style="{
      height: `${height}${measure}`,
      width: `${width}${measure}`,
      gap: '0.1in',
      padding: '0.1in',
      fontFamily: sansFontFamily,
      background: 'white',
      color: 'black',
    }"
  >
    <div class="flex items-center">
      <img
        :src="qrUrl"
        :style="{
          minWidth: `${height - 0.2}${measure}`,
          width: `${height - 0.2}${measure}`,
          height: `${height - 0.2}${measure}`,
        }"
      />
    </div>
    <div class="flex flex-1 flex-col justify-center overflow-hidden">
      <div v-if="assetId" class="text-xs" :style="{ fontFamily: monoFontFamily }">#{{ assetId }}</div>
      <div class="line-clamp-2 overflow-hidden text-xs font-bold">{{ name }}</div>
      <div v-if="showLocation && location" class="flex items-center gap-0.5 text-xs">
        <MapPin :size="12" class="shrink-0" />
        <span>{{ location }}</span>
      </div>
    </div>
  </div>
</template>
