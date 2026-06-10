import { route } from "../api/base";

export type LabelData = {
  url: string;
  name: string;
  assetID: string | null;
  location: string | null;
};

export type GridInput = {
  measure: string;
  page: {
    height: number;
    width: number;
    pageTopPadding: number;
    pageBottomPadding: number;
    pageLeftPadding: number;
    pageRightPadding: number;
  };
  cardHeight: number;
  cardWidth: number;
};

export type GridOutput = {
  measure: string;
  cols: number;
  rows: number;
  gapY: number;
  gapX: number;
  card: {
    width: number;
    height: number;
  };
  page: {
    width: number;
    height: number;
    pt: number;
    pb: number;
    pl: number;
    pr: number;
  };
};

export type LabelRow = {
  items: Array<LabelData | null>;
};

export type LabelPage = {
  rows: LabelRow[];
};

export function fmtAssetID(aid: number | string): string {
  let aidStr = aid.toString().replace(/\D/g, "").padStart(6, "0");
  aidStr = aidStr.slice(0, 3) + "-" + aidStr.slice(3);
  return aidStr;
}

export function hasAssetID(assetId: string | null | undefined): boolean {
  return !!assetId && assetId !== "0" && assetId !== "000-000";
}

export function getQRCodeUrl(target: string): string {
  return route(`/qrcode`, { data: encodeURIComponent(target) });
}

export function labelTargetUrl(
  baseURL: string,
  entity: { id: string; assetId?: string | null },
  type: "item" | "location" = "item"
): string {
  if (type === "location") {
    return `${baseURL}/location/${entity.id}`;
  }
  if (hasAssetID(entity.assetId)) {
    return `${baseURL}/a/${fmtAssetID(entity.assetId!)}`;
  }
  return `${baseURL}/item/${entity.id}`;
}

export function calculateGridData(input: GridInput): GridOutput | null {
  const { page, cardHeight, cardWidth } = input;

  const measureRegex = /in|cm|mm/;
  const measure = measureRegex.test(input.measure) ? input.measure : "in";

  const availablePageWidth = page.width - page.pageLeftPadding - page.pageRightPadding;
  const availablePageHeight = page.height - page.pageTopPadding - page.pageBottomPadding;

  if (availablePageWidth < cardWidth || availablePageHeight < cardHeight) {
    return null;
  }

  const cols = Math.floor(availablePageWidth / cardWidth);
  const rows = Math.floor(availablePageHeight / cardHeight);
  const gapX = (availablePageWidth - cols * cardWidth) / (cols - 1);
  const gapY = (page.height - rows * cardHeight) / (rows - 1);

  return {
    measure,
    cols,
    rows,
    gapX,
    gapY,
    card: {
      width: cardWidth,
      height: cardHeight,
    },
    page: {
      width: page.width,
      height: page.height,
      pt: page.pageTopPadding,
      pb: page.pageBottomPadding,
      pl: page.pageLeftPadding,
      pr: page.pageRightPadding,
    },
  };
}

export function clampSkipLabels(skipLabels: number, grid: GridOutput): number {
  const perPage = grid.rows * grid.cols;
  const maxSkipLabels = Math.max(0, perPage - 1);
  const raw = Number(skipLabels);
  return Number.isFinite(raw) ? Math.min(maxSkipLabels, Math.max(0, Math.floor(raw))) : 0;
}

export function chunkIntoPages(items: LabelData[], grid: GridOutput, skipLabels: number): LabelPage[] {
  if (items.length === 0) {
    return [];
  }

  const perPage = grid.rows * grid.cols;
  const skip = clampSkipLabels(skipLabels, grid);

  const itemsCopy: Array<LabelData | null> = [...items];
  if (skip > 0) {
    itemsCopy.unshift(...Array.from({ length: skip }, () => null));
  }

  const calc: LabelPage[] = [];
  while (itemsCopy.length > 0) {
    const page: LabelPage = {
      rows: [],
    };

    for (let i = 0; i < perPage; i++) {
      const item = itemsCopy.shift();
      if (typeof item === "undefined") {
        break;
      }

      if (i % grid.cols === 0) {
        page.rows.push({
          items: [],
        });
      }

      page.rows[page.rows.length - 1]!.items.push(item ?? null);
    }

    calc.push(page);
  }

  return calc;
}

export function expandByQuantity<T extends { quantity?: number }>(items: T[], enabled: boolean): T[] {
  if (!enabled) {
    return items;
  }

  const expanded: T[] = [];
  for (const item of items) {
    const copies = Math.max(1, Math.floor(item.quantity ?? 1));
    for (let i = 0; i < copies; i++) {
      expanded.push(item);
    }
  }
  return expanded;
}
