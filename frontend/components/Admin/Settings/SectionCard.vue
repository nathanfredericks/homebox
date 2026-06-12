<script setup lang="ts">
  import { Button } from "@/components/ui/button";
  import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
  import MdiLoading from "~icons/mdi/loading";

  defineProps<{
    title: string;
    description?: string;
    saving: boolean;
    dirty: boolean;
    canEdit: boolean;
  }>();

  defineEmits<{ save: []; reset: [] }>();
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>{{ title }}</CardTitle>
      <CardDescription v-if="description">{{ description }}</CardDescription>
    </CardHeader>

    <CardContent class="space-y-4">
      <slot />
    </CardContent>

    <CardFooter v-if="canEdit" class="flex justify-end gap-2">
      <slot name="actions" />
      <Button variant="outline" size="sm" :disabled="saving" @click="$emit('reset')">
        {{ $t("admin.settings.reset_to_defaults") }}
      </Button>
      <Button size="sm" :disabled="saving || !dirty" @click="$emit('save')">
        <MdiLoading v-if="saving" class="mr-1 size-4 animate-spin" />
        {{ $t("global.save") }}
      </Button>
    </CardFooter>
  </Card>
</template>
