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

        <p v-else-if="suggestions.length === 0" class="py-4 text-sm text-muted-foreground">
          {{ $t("ai.fill.no_suggestions") }}
        </p>

        <div v-else class="flex flex-col gap-3">
          <div v-for="s in suggestions" :key="s.field" class="rounded-md border p-3">
            <div class="flex items-center gap-2">
              <Checkbox :id="`fill-${s.field}`" v-model="accepted[s.field]" />
              <Label class="cursor-pointer font-medium" :for="`fill-${s.field}`">
                {{ fieldLabel(s.field) }}
              </Label>
            </div>
            <div class="mt-2 grid grid-cols-1 gap-1 text-sm">
              <div v-if="s.current" class="text-muted-foreground line-through">{{ s.current }}</div>
              <div>{{ s.suggested }}</div>
            </div>
          </div>
        </div>

        <DialogFooter v-if="suggestions.length > 0">
          <Button variant="outline" :disabled="applying" @click="acceptAll">
            {{ $t("ai.fill.accept_all") }}
          </Button>
          <Button :disabled="acceptedCount === 0 || applying" @click="apply">
            <MdiLoading v-if="applying" class="animate-spin" />
            {{ $t("ai.fill.apply", { count: acceptedCount }) }}
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
  import type { AiFieldSuggestion, EntityOut } from "~~/lib/api/types/data-contracts";
  import { suggestionsToPatch } from "~~/composables/use-ai-capture-types";

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
  const accepted = ref<Record<string, boolean>>({});

  const acceptedCount = computed(() => suggestions.value.filter(s => accepted.value[s.field]).length);

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

  async function fetchSuggestions() {
    loading.value = true;
    suggestions.value = [];
    accepted.value = {};
    try {
      const { data, error } = await api.ai.suggest(props.item.id, overwrite.value);
      if (error || !data) {
        toast.error(t("ai.fill.failed"));
        closeDialog(DialogID.AiFill);
        return;
      }
      suggestions.value = data.suggestions;
      accepted.value = Object.fromEntries(data.suggestions.map(s => [s.field, true]));
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

  function acceptAll() {
    accepted.value = Object.fromEntries(suggestions.value.map(s => [s.field, true]));
  }

  async function apply() {
    applying.value = true;
    try {
      const patch = suggestionsToPatch(
        props.item.id,
        suggestions.value.filter(s => accepted.value[s.field])
      );

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
