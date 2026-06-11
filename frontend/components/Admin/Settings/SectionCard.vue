<script setup lang="ts">
  import { Badge } from "@/components/ui/badge";
  import { Button } from "@/components/ui/button";
  import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
  import MdiLoading from "~icons/mdi/loading";

  defineProps<{
    title: string;
    description?: string;
    /** True when this section has a database override (vs pure environment). */
    overridden: boolean;
    saving: boolean;
    dirty: boolean;
    canEdit: boolean;
  }>();

  defineEmits<{ save: []; reset: [] }>();
</script>

<template>
  <Card>
    <CardHeader>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <CardTitle>{{ title }}</CardTitle>
        <Badge :variant="overridden ? 'default' : 'secondary'">
          {{ overridden ? $t("admin.settings.source_database") : $t("admin.settings.source_environment") }}
        </Badge>
      </div>
      <CardDescription v-if="description">{{ description }}</CardDescription>
    </CardHeader>

    <CardContent class="space-y-4">
      <slot />
    </CardContent>

    <CardFooter v-if="canEdit" class="flex justify-end gap-2">
      <slot name="actions" />
      <Button v-if="overridden" variant="outline" size="sm" :disabled="saving" @click="$emit('reset')">
        {{ $t("admin.settings.reset_to_environment") }}
      </Button>
      <Button size="sm" :disabled="saving || !dirty" @click="$emit('save')">
        <MdiLoading v-if="saving" class="mr-1 size-4 animate-spin" />
        {{ $t("global.save") }}
      </Button>
    </CardFooter>
  </Card>
</template>
