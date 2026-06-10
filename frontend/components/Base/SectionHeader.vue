<template>
  <div class="pb-3">
    <CardTitle class="flex items-center">
      <slot />
    </CardTitle>
    <slot name="subtitle" />
    <!-- div, not CardDescription: CardDescription renders a <p>, and slots
    here contain block elements, which is invalid inside <p> and breaks SSR
    hydration -->
    <div v-if="$slots.description" class="text-sm text-muted-foreground">
      <slot name="description" />
    </div>
    <div v-if="$slots.after">
      <slot name="after" />
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { CardTitle } from "@/components/ui/card";
</script>
