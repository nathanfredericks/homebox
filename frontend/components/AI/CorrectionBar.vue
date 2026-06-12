<template>
  <form class="flex items-end gap-2" @submit.prevent="submit">
    <div class="flex grow flex-col gap-1.5">
      <Label for="ai-correction" class="px-1">{{ $t("ai.capture.correction_label") }}</Label>
      <Input
        id="ai-correction"
        v-model="feedback"
        :placeholder="$t('ai.capture.correction_placeholder')"
        :disabled="busy"
      />
    </div>
    <Button type="submit" variant="outline" :disabled="busy || !feedback.trim()">
      <MdiLoading v-if="busy" class="animate-spin" />
      <MdiRefresh v-else />
      {{ $t("ai.capture.reanalyze") }}
    </Button>
  </form>
</template>

<script setup lang="ts">
  import { Button } from "@/components/ui/button";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import MdiLoading from "~icons/mdi/loading";
  import MdiRefresh from "~icons/mdi/refresh";

  defineProps<{ busy: boolean }>();
  const emit = defineEmits<{ correct: [feedback: string] }>();

  const feedback = ref("");

  function submit() {
    const text = feedback.value.trim();
    if (!text) return;
    emit("correct", text);
    feedback.value = "";
  }
</script>
