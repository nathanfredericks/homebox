<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from "@/components/ui/dialog";
  import { Button } from "@/components/ui/button";
  import { Checkbox } from "@/components/ui/checkbox";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import { useDialog } from "@/components/ui/dialog-provider";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import { toast } from "@/components/ui/sonner";
  import AssetLabel from "@/components/Label/AssetLabel.vue";
  import MdiLoading from "~icons/mdi/loading";
  import type { EntitySummary } from "~~/lib/api/types/data-contracts";
  import {
    type LabelData,
    calculateGridData,
    chunkIntoPages,
    clampSkipLabels,
    expandByQuantity,
    fmtAssetID,
    getQRCodeUrl,
    hasAssetID,
    labelTargetUrl,
  } from "~~/lib/labels";
  import { openPrintWindow, printLabelSheet, renderNodeToPng } from "~~/lib/labels/render";

  const { t } = useI18n();
  const { registerOpenDialogCallback } = useDialog();

  const { settings, sansFontFamily, monoFontFamily, resolvedBaseURL } = useLabelSettings();

  const items = ref<EntitySummary[]>([]);
  const printing = ref(false);

  onMounted(() => {
    const cleanup = registerOpenDialogCallback(DialogID.PrintLabels, params => {
      items.value = params.items;
    });
    onUnmounted(cleanup);
  });

  type BulkLabel = LabelData & { quantity?: number };

  function toLabelData(item: EntitySummary): BulkLabel {
    return {
      url: getQRCodeUrl(labelTargetUrl(resolvedBaseURL.value, item)),
      assetID: hasAssetID(item.assetId) ? fmtAssetID(item.assetId) : null,
      name: item.name,
      location: item.parent?.name ?? null,
      quantity: item.quantity,
    };
  }

  const labels = computed<BulkLabel[]>(() =>
    expandByQuantity(items.value.map(toLabelData), settings.value.labelPerQuantity)
  );

  const grid = computed(() =>
    calculateGridData({
      measure: settings.value.measure,
      page: {
        height: settings.value.pageHeight,
        width: settings.value.pageWidth,
        pageTopPadding: settings.value.pageTopPadding,
        pageBottomPadding: settings.value.pageBottomPadding,
        pageLeftPadding: settings.value.pageLeftPadding,
        pageRightPadding: settings.value.pageRightPadding,
      },
      cardHeight: settings.value.cardHeight,
      cardWidth: settings.value.cardWidth,
    })
  );

  const pages = computed(() => {
    if (!grid.value) {
      return [];
    }
    return chunkIntoPages(labels.value, grid.value, clampSkipLabels(Number(settings.value.skipLabels), grid.value));
  });

  // sequential offscreen rasterization; one render per unique label url
  const renderTarget = ref<InstanceType<typeof AssetLabel> | null>(null);
  const currentLabel = ref<BulkLabel | null>(null);

  async function renderAllLabels(): Promise<Map<string, string>> {
    const images = new Map<string, string>();

    for (const label of labels.value) {
      if (images.has(label.url)) {
        continue;
      }

      currentLabel.value = label;
      await nextTick();

      const el = renderTarget.value?.$el as HTMLElement | undefined;
      if (!el) {
        throw new Error("label render target missing");
      }

      images.set(label.url, await renderNodeToPng(el));
    }

    currentLabel.value = null;
    return images;
  }

  async function print() {
    if (!grid.value) {
      toast.error(t("reports.label_generator.toast.page_too_small_card"));
      return;
    }

    const printWindow = openPrintWindow();
    if (printWindow === null) {
      return;
    }

    printing.value = true;
    try {
      const images = await renderAllLabels();
      printLabelSheet(printWindow, pages.value, grid.value, images);
    } catch (err) {
      console.error("Failed to print labels:", err);
      printWindow.close();
      toast.error(t("components.global.label_maker.toast.print_failed"));
    } finally {
      printing.value = false;
    }
  }
</script>

<template>
  <Dialog :dialog-id="DialogID.PrintLabels">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>
          {{ $t("components.item.print_labels.title") }}
        </DialogTitle>
        <DialogDescription>
          {{ $t("components.item.print_labels.description") }}
        </DialogDescription>
      </DialogHeader>

      <ClientOnly>
        <div v-if="labels.length > 0" class="flex justify-center overflow-auto rounded-md border p-4">
          <AssetLabel
            :name="labels[0]!.name"
            :asset-id="labels[0]!.assetID"
            :location="labels[0]!.location"
            :qr-url="labels[0]!.url"
            :width="settings.cardWidth"
            :height="settings.cardHeight"
            :measure="settings.measure"
            :bordered="settings.bordered"
            :show-location="settings.printLocationRow"
            :sans-font-family="sansFontFamily"
            :mono-font-family="monoFontFamily"
          />
        </div>

        <!-- offscreen render target used while rasterizing -->
        <div aria-hidden="true" style="position: fixed; left: -10000px; top: 0; pointer-events: none">
          <AssetLabel
            v-if="currentLabel"
            ref="renderTarget"
            :name="currentLabel.name"
            :asset-id="currentLabel.assetID"
            :location="currentLabel.location"
            :qr-url="currentLabel.url"
            :width="settings.cardWidth"
            :height="settings.cardHeight"
            :measure="settings.measure"
            :bordered="settings.bordered"
            :show-location="settings.printLocationRow"
            :sans-font-family="sansFontFamily"
            :mono-font-family="monoFontFamily"
          />
        </div>
      </ClientOnly>

      <div class="flex flex-col gap-4">
        <div class="flex w-full max-w-xs flex-col gap-1">
          <Label for="printLabelsSkip">
            {{ $t("reports.label_generator.skip_first_labels") }}
          </Label>
          <Input
            id="printLabelsSkip"
            v-model="settings.skipLabels"
            type="number"
            :min="0"
            :max="grid ? Math.max(0, grid.rows * grid.cols - 1) : undefined"
            :step="1"
          />
        </div>
        <div class="flex items-center gap-2">
          <Checkbox id="printLabelsPerQuantity" v-model="settings.labelPerQuantity" />
          <Label class="cursor-pointer" for="printLabelsPerQuantity">
            {{ $t("reports.label_generator.label_per_quantity") }}
          </Label>
        </div>
        <p class="text-sm text-muted-foreground">
          {{
            $t("components.item.print_labels.summary", {
              items: items.length,
              labels: labels.length,
              pages: pages.length,
            })
          }}
        </p>
        <NuxtLink
          to="/collection/tools/label-generator"
          class="text-sm text-primary underline-offset-4 hover:underline"
        >
          {{ $t("components.global.label_maker.configure_settings") }}
        </NuxtLink>
      </div>

      <DialogFooter>
        <Button type="submit" :disabled="printing || labels.length === 0" @click="print">
          <MdiLoading v-if="printing" class="animate-spin" />
          {{ $t("components.item.print_labels.print") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
