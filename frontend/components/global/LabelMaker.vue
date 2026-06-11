<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { type QueryValue, route } from "../../lib/api/base/urls";
  import PageQRCode from "./PageQRCode.vue";
  import {
    type LabelData,
    calculateGridData,
    chunkIntoPages,
    clampSkipLabels,
    fmtAssetID,
    getQRCodeUrl,
    hasAssetID,
    labelTargetUrl,
  } from "~~/lib/labels";
  import { openPrintWindow, printLabelSheet, renderNodeToPng } from "~~/lib/labels/render";
  import AssetLabel from "@/components/Label/AssetLabel.vue";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import { toast } from "@/components/ui/sonner";
  import MdiLoading from "~icons/mdi/loading";
  import MdiPrinterPos from "~icons/mdi/printer-pos";
  import MdiFileDownload from "~icons/mdi/file-download";

  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from "@/components/ui/dialog";
  import { useDialog } from "@/components/ui/dialog-provider";
  import { Button, ButtonGroup } from "@/components/ui/button";
  import { Checkbox } from "@/components/ui/checkbox";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

  const { t } = useI18n();
  const { openDialog, closeDialog } = useDialog();

  const props = withDefaults(
    defineProps<{
      type: "item" | "asset" | "location";
      id: string;
      name: string;
      assetId?: string | null;
      location?: string | null;
      quantity?: number;
    }>(),
    {
      assetId: null,
      location: null,
      quantity: 1,
    }
  );

  const pubApi = usePublicApi();

  const { data: status } = await useAsyncData("label-maker-status", async () => {
    const { data, error } = await pubApi.status();
    if (error) {
      toast.error(t("components.global.label_maker.toast.load_status_failed"));
      return;
    }

    return data;
  });

  const { settings, sansFontFamily, monoFontFamily, resolvedBaseURL } = useLabelSettings();

  const labelAssetId = computed(() =>
    props.type !== "location" && hasAssetID(props.assetId) ? fmtAssetID(props.assetId!) : null
  );

  const qrUrl = computed(() =>
    getQRCodeUrl(
      labelTargetUrl(
        resolvedBaseURL.value,
        { id: props.id, assetId: props.assetId },
        props.type === "location" ? "location" : "item"
      )
    )
  );

  const copies = computed(() =>
    props.type !== "location" && settings.value.labelPerQuantity ? Math.max(1, Math.floor(props.quantity)) : 1
  );

  const labelRef = ref<InstanceType<typeof AssetLabel> | null>(null);
  const serverPrinting = ref(false);
  const rendering = ref(false);

  async function renderLabel(): Promise<string | null> {
    const el = labelRef.value?.$el as HTMLElement | undefined;
    if (!el) {
      return null;
    }
    return await renderNodeToPng(el);
  }

  // Print on the configured label sheet (same page setup as the label
  // generator) so a partially used sheet can be reused via "skip first labels".
  async function browserPrint() {
    const grid = calculateGridData({
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
    });

    if (grid === null) {
      toast.error(t("reports.label_generator.toast.page_too_small_card"));
      return;
    }

    const printWindow = openPrintWindow();
    if (printWindow === null) {
      return;
    }

    rendering.value = true;
    try {
      const dataUrl = await renderLabel();
      if (!dataUrl) {
        printWindow.close();
        return;
      }

      const labelData: LabelData = {
        url: qrUrl.value,
        name: props.name,
        assetID: labelAssetId.value,
        location: props.location,
      };
      const labels = Array.from({ length: copies.value }, () => labelData);
      const pages = chunkIntoPages(labels, grid, clampSkipLabels(Number(settings.value.skipLabels), grid));

      printLabelSheet(printWindow, pages, grid, new Map([[labelData.url, dataUrl]]));
    } catch (err) {
      console.error("Failed to print labels:", err);
      printWindow.close();
      toast.error(t("components.global.label_maker.toast.print_failed"));
    } finally {
      rendering.value = false;
    }
  }

  async function serverPrint() {
    serverPrinting.value = true;
    try {
      await fetch(getLabelUrl(true));
    } catch (err) {
      console.error("Failed to print labels:", err);
      serverPrinting.value = false;
      toast.error(t("components.global.label_maker.toast.print_failed"));
      return;
    }

    toast.success(t("components.global.label_maker.toast.print_success"));
    closeDialog(DialogID.PrintLabel);
    serverPrinting.value = false;
  }

  async function downloadLabel() {
    rendering.value = true;
    try {
      const dataUrl = await renderLabel();
      if (!dataUrl) {
        return;
      }

      const link = document.createElement("a");
      link.download = `label-${props.id}.png`;
      link.href = dataUrl;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err) {
      console.error("Failed to render label:", err);
      toast.error(t("components.global.label_maker.toast.print_failed"));
    } finally {
      rendering.value = false;
    }
  }

  function getLabelUrl(print: boolean): string {
    const { selectedId } = useCollections();
    const params: Record<string, QueryValue> = { print };

    if (selectedId.value) {
      params.tenant = selectedId.value;
    }

    if (props.type === "item") {
      return route(`/labelmaker/entity/${props.id}`, params);
    } else if (props.type === "location") {
      return route(`/labelmaker/location/${props.id}`, params);
    } else if (props.type === "asset") {
      return route(`/labelmaker/asset/${props.assetId ?? props.id}`, params);
    } else {
      throw new Error(`Unexpected labelmaker type ${props.type}`);
    }
  }
</script>

<template>
  <div>
    <Dialog :dialog-id="DialogID.PrintLabel">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {{ $t("components.global.label_maker.print") }}
          </DialogTitle>
          <DialogDescription>
            {{ $t("components.global.label_maker.confirm_description") }}
          </DialogDescription>
        </DialogHeader>
        <ClientOnly>
          <div class="flex justify-center overflow-auto rounded-md border p-4">
            <AssetLabel
              :name="name"
              :asset-id="labelAssetId"
              :location="type !== 'location' ? location : null"
              :qr-url="qrUrl"
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
        <div class="flex w-full max-w-xs flex-col gap-1">
          <Label for="labelMakerSkip">
            {{ $t("reports.label_generator.skip_first_labels") }}
          </Label>
          <Input id="labelMakerSkip" v-model="settings.skipLabels" type="number" :min="0" :step="1" />
        </div>
        <div v-if="type !== 'location' && quantity > 1" class="flex items-center gap-2">
          <Checkbox id="labelMakerPerQuantity" v-model="settings.labelPerQuantity" />
          <Label class="cursor-pointer" for="labelMakerPerQuantity">
            {{ $t("components.global.label_maker.label_per_quantity", { quantity }) }}
          </Label>
        </div>
        <NuxtLink
          to="/collection/labels"
          class="text-sm text-primary underline-offset-4 hover:underline"
          @click="closeDialog(DialogID.PrintLabel)"
        >
          {{ $t("components.global.label_maker.configure_settings") }}
        </NuxtLink>
        <DialogFooter>
          <ButtonGroup>
            <Button v-if="status?.labelPrinting || false" type="submit" :disabled="serverPrinting" @click="serverPrint">
              <MdiLoading v-if="serverPrinting" class="animate-spin" />
              {{ $t("components.global.label_maker.server_print") }}
            </Button>
            <Button type="submit" :disabled="rendering" @click="browserPrint">
              <MdiLoading v-if="rendering" class="animate-spin" />
              {{ $t("components.global.label_maker.browser_print") }}
            </Button>
          </ButtonGroup>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- offscreen render target so download/print work without the dialog markup -->
    <ClientOnly>
      <div aria-hidden="true" style="position: fixed; left: -10000px; top: 0; pointer-events: none">
        <AssetLabel
          ref="labelRef"
          :name="name"
          :asset-id="labelAssetId"
          :location="type !== 'location' ? location : null"
          :qr-url="qrUrl"
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

    <TooltipProvider :delay-duration="0">
      <ButtonGroup>
        <Button variant="outline" disabled class="disabled:opacity-100">
          {{ $t("components.global.label_maker.titles") }}
        </Button>

        <Tooltip>
          <TooltipTrigger as-child>
            <Button size="icon" :disabled="rendering" @click="downloadLabel">
              <MdiFileDownload name="mdi-file-download" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {{ $t("components.global.label_maker.download") }}
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger as-child>
            <Button size="icon" @click="openDialog(DialogID.PrintLabel)">
              <MdiPrinterPos name="mdi-printer-pos" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {{ $t("components.global.label_maker.browser_print") }}
          </TooltipContent>
        </Tooltip>

        <PageQRCode />
      </ButtonGroup>
    </TooltipProvider>
  </div>
</template>
