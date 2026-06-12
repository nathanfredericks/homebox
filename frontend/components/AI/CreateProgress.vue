<template>
  <div class="flex flex-col gap-2">
    <div
      v-for="result in results"
      :key="result.name"
      class="flex items-center justify-between gap-2 rounded-md border p-3 text-sm"
    >
      <span class="truncate font-medium">{{ result.name }}</span>
      <span v-if="result.status === 'pending'" class="text-muted-foreground">
        <MdiLoading class="size-4 animate-spin" />
      </span>
      <NuxtLink
        v-else-if="result.status === 'created'"
        :to="`/item/${result.id}`"
        class="inline-flex items-center gap-1 text-primary hover:underline"
      >
        <MdiCheckCircle class="size-4" />
        {{ $t("ai.capture.view_item") }}
      </NuxtLink>
      <span v-else class="inline-flex items-center gap-1 text-destructive">
        <MdiAlertCircle class="size-4" />
        {{ $t("ai.capture.create_failed") }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
  import MdiLoading from "~icons/mdi/loading";
  import MdiCheckCircle from "~icons/mdi/check-circle";
  import MdiAlertCircle from "~icons/mdi/alert-circle";

  export type CreateResult = {
    name: string;
    status: "pending" | "created" | "failed";
    id?: string;
  };

  defineProps<{ results: CreateResult[] }>();
</script>
