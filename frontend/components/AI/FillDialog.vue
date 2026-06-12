<template>
  <div v-if="visible">
    <Button variant="outline" class="w-9 md:w-auto" :aria-label="$t('ai.fill.button')" @click="openDialog">
      <MdiCreation />
      <span class="hidden md:inline">{{ $t("ai.fill.button") }}</span>
    </Button>

    <Dialog :dialog-id="DialogID.AiFill">
      <DialogContent class="max-h-[85vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t("ai.fill.title") }}</DialogTitle>
          <DialogDescription>{{ $t("ai.fill.description") }}</DialogDescription>
        </DialogHeader>

        <div class="flex items-center gap-2">
          <Checkbox id="aiFillOverwrite" v-model="overwrite" :disabled="loading" />
          <Label class="cursor-pointer" for="aiFillOverwrite">{{ $t("ai.fill.overwrite") }}</Label>
        </div>

        <div v-if="loading" class="flex items-center justify-center gap-2 py-8 text-muted-foreground">
          <MdiLoading class="size-5 animate-spin" />
          {{ $t("ai.capture.analyzing") }}
        </div>

        <p v-else-if="!hasSuggestions" class="py-4 text-sm text-muted-foreground">
          {{ $t("ai.fill.no_suggestions") }}
        </p>

        <SuggestionList
          v-else
          ref="listRef"
          :suggestions="suggestions"
          :tags="tagSuggestions"
          :custom-fields="customFieldSuggestions"
        />

        <DialogFooter v-if="hasSuggestions && !loading">
          <Button variant="outline" :disabled="applying" @click="listRef?.acceptAll()">
            {{ $t("ai.fill.accept_all") }}
          </Button>
          <Button :disabled="(listRef?.selectedCount ?? 0) === 0 || applying" @click="apply">
            <MdiLoading v-if="applying" class="animate-spin" />
            {{ $t("ai.fill.apply", { count: listRef?.selectedCount ?? 0 }) }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import { Button } from "@/components/ui/button";
  import { Checkbox } from "@/components/ui/checkbox";
  import { Label } from "@/components/ui/label";
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from "@/components/ui/dialog";
  import { useDialog } from "@/components/ui/dialog-provider";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import MdiCreation from "~icons/mdi/creation";
  import MdiLoading from "~icons/mdi/loading";
  import SuggestionList from "~/components/AI/SuggestionList.vue";
  import type {
    AiCustomFieldSuggestion,
    AiFieldSuggestion,
    AiTagSuggestion,
    EntityOut,
  } from "~~/lib/api/types/data-contracts";
  import { customFieldSuggestionsToFields, suggestionsToPatch } from "~~/composables/use-ai-capture-types";

  const props = defineProps<{
    item: EntityOut;
  }>();

  const emit = defineEmits<{ applied: [] }>();

  const { t } = useI18n();
  const api = useUserApi();
  const { can } = usePermissions();
  const { aiEnabled } = useAi();
  const { activeDialog, openDialog: openProviderDialog, closeDialog } = useDialog();

  const hasPhotos = computed(() =>
    (props.item.attachments ?? []).some(a => a.type === "photo" || a.type === "receipt")
  );
  const visible = computed(() => aiEnabled.value && can("items", "edit") && hasPhotos.value);

  const open = computed(() => activeDialog.value === DialogID.AiFill);
  const loading = ref(false);
  const applying = ref(false);
  const overwrite = ref(false);
  const suggestions = ref<AiFieldSuggestion[]>([]);
  const tagSuggestions = ref<AiTagSuggestion[]>([]);
  const customFieldSuggestions = ref<AiCustomFieldSuggestion[]>([]);

  const listRef = ref<InstanceType<typeof SuggestionList> | null>(null);

  const hasSuggestions = computed(
    () => suggestions.value.length > 0 || tagSuggestions.value.length > 0 || customFieldSuggestions.value.length > 0
  );

  async function fetchSuggestions() {
    loading.value = true;
    suggestions.value = [];
    tagSuggestions.value = [];
    customFieldSuggestions.value = [];
    try {
      const { data, error } = await api.ai.suggest(props.item.id, overwrite.value);
      if (error || !data) {
        toast.error(t("ai.fill.failed"));
        closeDialog(DialogID.AiFill);
        return;
      }
      suggestions.value = data.suggestions;
      tagSuggestions.value = data.tags ?? [];
      customFieldSuggestions.value = data.customFields ?? [];
    } finally {
      loading.value = false;
    }
  }

  function openDialog() {
    openProviderDialog(DialogID.AiFill);
    fetchSuggestions();
  }

  watch(overwrite, () => {
    if (open.value && !loading.value) fetchSuggestions();
  });

  async function apply() {
    const selection = listRef.value?.selection;
    if (!selection) return;

    applying.value = true;
    try {
      const patch = suggestionsToPatch(props.item.id, selection.fields);
      if (selection.tagIds.length) {
        // The patch endpoint syncs tags to the exact list, so accepted
        // suggestions are unioned with the item's current tags.
        const current = (props.item.tags ?? []).map(tag => tag.id);
        patch.tagIds = [...new Set([...current, ...selection.tagIds])];
      }
      if (selection.customFields.length) {
        patch.fields = customFieldSuggestionsToFields(selection.customFields);
      }

      const { error } = await api.items.patch(props.item.id, patch);
      if (error) {
        toast.error(t("ai.fill.apply_failed"));
        return;
      }
      toast.success(t("ai.fill.applied"));
      closeDialog(DialogID.AiFill);
      emit("applied");
    } finally {
      applying.value = false;
    }
  }
</script>
