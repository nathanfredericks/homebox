<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import DOMPurify from "isomorphic-dompurify";
  import {
    type LabelData,
    type LabelPage,
    calculateGridData,
    chunkIntoPages,
    clampSkipLabels,
    expandByQuantity,
    fmtAssetID,
    getQRCodeUrl,
  } from "~~/lib/labels";
  import AssetLabel from "@/components/Label/AssetLabel.vue";
  import FormGoogleFontSelect from "~/components/Form/GoogleFontSelect.vue";
  import { toast } from "@/components/ui/sonner";
  import { Separator } from "@/components/ui/separator";
  import { Button } from "@/components/ui/button";
  import { Label } from "@/components/ui/label";
  import { Input } from "@/components/ui/input";
  import { Checkbox } from "@/components/ui/checkbox";
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import MdiPrinter from "~icons/mdi/printer";

  const { t } = useI18n();
  const { appName } = useBranding();

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: t("reports.label_generator.title"),
  });

  const api = useUserApi();

  const { settings, sansFontFamily, monoFontFamily, resolvedBaseURL } = useLabelSettings();

  const labelBlankLine = "_______________";

  const baseURLModel = computed({
    get: () => resolvedBaseURL.value,
    set: (value: string) => {
      settings.value.baseURL = value;
    },
  });

  interface InputDef {
    label: string;
    ref:
      | "assetRange"
      | "assetRangeMax"
      | "skipLabels"
      | "measure"
      | "cardHeight"
      | "cardWidth"
      | "pageWidth"
      | "pageHeight"
      | "pageTopPadding"
      | "pageBottomPadding"
      | "pageLeftPadding"
      | "pageRightPadding";
    type?: "number" | "text";
    min?: number;
    step?: number;
  }

  const propertyInputs = computed<InputDef[]>(() => {
    return [
      {
        label: t("reports.label_generator.asset_start"),
        ref: "assetRange",
      },
      {
        label: t("reports.label_generator.asset_end"),
        ref: "assetRangeMax",
      },
      {
        label: t("reports.label_generator.skip_first_labels"),
        ref: "skipLabels",
        min: 0,
        step: 1,
      },
      {
        label: t("reports.label_generator.measure_type"),
        ref: "measure",
        type: "text",
      },
      {
        label: t("reports.label_generator.label_height"),
        ref: "cardHeight",
      },
      {
        label: t("reports.label_generator.label_width"),
        ref: "cardWidth",
      },
      {
        label: t("reports.label_generator.page_width"),
        ref: "pageWidth",
      },
      {
        label: t("reports.label_generator.page_height"),
        ref: "pageHeight",
      },
      {
        label: t("reports.label_generator.page_top_padding"),
        ref: "pageTopPadding",
      },
      {
        label: t("reports.label_generator.page_bottom_padding"),
        ref: "pageBottomPadding",
      },
      {
        label: t("reports.label_generator.page_left_padding"),
        ref: "pageLeftPadding",
      },
      {
        label: t("reports.label_generator.page_right_padding"),
        ref: "pageRightPadding",
      },
    ];
  });

  type GeneratorItem = LabelData & { quantity?: number };

  function getItem(
    n: number,
    item: { assetId: string; name: string; quantity?: number; parent?: { name: string } | null } | null
  ): GeneratorItem {
    // format n into - seperated string with leading zeros
    const assetID = fmtAssetID(item?.assetId ?? n + 1);

    return {
      url: getQRCodeUrl(`${resolvedBaseURL.value}/a/${assetID}`),
      assetID: item?.assetId ?? assetID,
      name: item?.name ?? labelBlankLine,
      location: item?.parent?.name ?? labelBlankLine,
      quantity: item?.quantity,
    };
  }

  const { data: allFields } = await useAsyncData(async () => {
    const { data, error } = await api.items.getAll({ orderBy: "assetId" });

    if (error) {
      return {
        items: [],
      };
    }

    return data;
  });

  const items = computed<GeneratorItem[]>(() => {
    if (settings.value.assetRange > settings.value.assetRangeMax) {
      return [];
    }

    const diff = settings.value.assetRangeMax - settings.value.assetRange;

    if (diff > 999) {
      return [];
    }

    const items: GeneratorItem[] = [];
    for (let i = settings.value.assetRange - 1; i < settings.value.assetRangeMax - 1; i++) {
      const item = allFields?.value?.items?.[i];
      items.push(getItem(i, (item as Parameters<typeof getItem>[1]) ?? null));
    }
    return expandByQuantity(items, settings.value.labelPerQuantity);
  });

  const pages = ref<LabelPage[]>([]);

  const out = ref({
    measure: "in",
    cols: 0,
    rows: 0,
    gapY: 0,
    gapX: 0,
    card: {
      width: 0,
      height: 0,
    },
    page: {
      width: 0,
      height: 0,
      pt: 0,
      pb: 0,
      pl: 0,
      pr: 0,
    },
  });

  function calcPages() {
    // Set Out Dimensions
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

    out.value = grid;

    const skipLabels = clampSkipLabels(Number(settings.value.skipLabels), grid);
    if (Number(settings.value.skipLabels) !== skipLabels) {
      settings.value.skipLabels = skipLabels;
    }

    pages.value = chunkIntoPages(items.value, grid, skipLabels);
  }

  onMounted(() => {
    calcPages();
  });
</script>

<template>
  <BaseContainer class="m-0 flex flex-col gap-4 px-0 print:hidden">
    <BaseCard>
      <template #title>
        <BaseSectionHeader>
          <MdiPrinter class="mr-2" />
          <span> {{ $t("reports.label_generator.title") }} </span>
          <template #description> {{ $t("reports.label_generator.instruction_1", { appName }) }} </template>
        </BaseSectionHeader>
      </template>
      <div class="border-t p-4 sm:px-6">
        <div class="prose max-w-none">
          <p>
            {{ $t("reports.label_generator.instruction_2", { appName }) }}
          </p>
          <h4>{{ $t("reports.label_generator.tips") }}</h4>
          <ul>
            <li v-html="DOMPurify.sanitize($t('reports.label_generator.tip_1'))" />
            <li v-html="DOMPurify.sanitize($t('reports.label_generator.tip_2'))" />
            <li v-html="DOMPurify.sanitize($t('reports.label_generator.tip_3'))" />
          </ul>
        </div>
        <Separator class="my-4" />
        <div class="grid grid-cols-2 gap-3">
          <div v-for="(prop, i) in propertyInputs" :key="i" class="flex w-full max-w-xs flex-col">
            <Label :for="`input-${prop.ref}`">
              {{ prop.label }}
            </Label>
            <Input
              :id="`input-${prop.ref}`"
              v-model="settings[prop.ref]"
              :type="prop.type ? prop.type : 'number'"
              :min="prop.min"
              :max="prop.ref === 'skipLabels' ? Math.max(0, out.rows * out.cols - 1) : undefined"
              :step="prop.type === 'text' ? undefined : (prop.step ?? 0.01)"
              :placeholder="$t('reports.label_generator.input_placeholder')"
              class="w-full max-w-xs"
            />
          </div>
          <div class="flex w-full max-w-xs flex-col">
            <Label for="input-baseURL">
              {{ $t("reports.label_generator.base_url") }}
            </Label>
            <Input
              id="input-baseURL"
              v-model="baseURLModel"
              type="text"
              :placeholder="$t('reports.label_generator.input_placeholder')"
              class="w-full max-w-xs"
            />
          </div>
          <div class="flex w-full max-w-xs flex-col">
            <FormGoogleFontSelect
              v-model="settings.sansFont"
              :label="$t('reports.label_generator.sans_serif_font')"
              class="w-full max-w-xs"
            />
          </div>
          <div class="flex w-full max-w-xs flex-col">
            <FormGoogleFontSelect
              v-model="settings.monoFont"
              :label="$t('reports.label_generator.monospace_font')"
              class="w-full max-w-xs"
            />
          </div>
        </div>
        <div class="max-w-xs">
          <div class="flex items-center gap-2 py-4">
            <Checkbox id="borderedLabels" v-model="settings.bordered" />
            <Label class="cursor-pointer" for="borderedLabels">
              {{ $t("reports.label_generator.bordered_labels") }}
            </Label>
          </div>
          <div class="flex items-center gap-2 py-4">
            <Checkbox id="printLocationRow" v-model="settings.printLocationRow" />
            <Label class="cursor-pointer" for="printLocationRow">
              {{ $t("reports.label_generator.print_location_row") }}
            </Label>
          </div>
          <div class="flex items-center gap-2 py-4">
            <Checkbox id="labelPerQuantity" v-model="settings.labelPerQuantity" />
            <Label class="cursor-pointer" for="labelPerQuantity">
              {{ $t("reports.label_generator.label_per_quantity") }}
            </Label>
          </div>
        </div>

        <div>
          <p class="text-sm text-muted-foreground">
            {{ $t("reports.label_generator.qr_code_example") }} {{ resolvedBaseURL }}/a/{asset_id}
          </p>
          <Button size="lg" class="mt-4 w-full" @click="calcPages">
            {{ $t("reports.label_generator.generate_page") }}
          </Button>
        </div>
      </div>
    </BaseCard>
  </BaseContainer>
  <div class="flex flex-col items-center">
    <section
      v-for="(page, pi) in pages"
      :key="pi"
      class="box-border border-2 print:border-none"
      :class="{ 'print:break-after-page': pi < pages.length - 1 }"
      :style="{
        paddingTop: `${out.page.pt}${out.measure}`,
        paddingBottom: `${out.page.pb}${out.measure}`,
        paddingLeft: `${out.page.pl}${out.measure}`,
        paddingRight: `${out.page.pr}${out.measure}`,
        width: `${out.page.width}${out.measure}`,
        height: `${out.page.height}${out.measure}`,
        background: `white`,
        color: `black`,
      }"
    >
      <div
        v-for="(row, ri) in page.rows"
        :key="ri"
        class="flex break-inside-avoid"
        :style="{
          columnGap: `${out.gapX}${out.measure}`,
          rowGap: `${out.gapY}${out.measure}`,
        }"
      >
        <template v-for="(item, idx) in row.items" :key="idx">
          <AssetLabel
            v-if="item"
            :name="item.name"
            :asset-id="item.assetID"
            :location="item.location"
            :qr-url="item.url"
            :width="out.card.width"
            :height="out.card.height"
            :measure="out.measure"
            :bordered="settings.bordered"
            :show-location="settings.printLocationRow"
            :sans-font-family="sansFontFamily"
            :mono-font-family="monoFontFamily"
          />
          <div
            v-else
            :style="{
              height: `${out.card.height}${out.measure}`,
              width: `${out.card.width}${out.measure}`,
            }"
          />
        </template>
      </div>
    </section>
  </div>
</template>
