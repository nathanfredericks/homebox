import type { GridOutput, LabelPage } from "./index";

async function waitForImages(el: HTMLElement) {
  const images = Array.from(el.querySelectorAll("img"));
  await Promise.all(
    images.map(img =>
      img.decode().catch(() => {
        /* ignore broken images; capture proceeds without them */
      })
    )
  );
}

export async function renderNodeToPng(el: HTMLElement, pixelRatio = 4): Promise<string> {
  const { toPng } = await import("html-to-image");
  await document.fonts.ready;
  await waitForImages(el);
  return toPng(el, { pixelRatio, backgroundColor: "#ffffff" });
}

// Opens a blank popup synchronously so it isn't blocked; content is injected
// later, after rasterization finishes.
export function openPrintWindow(): Window | null {
  return window.open("", "_blank", "popup=true");
}

function writeAndPrint(printWindow: Window, body: string, pageSizeCss: string) {
  printWindow.document.open();
  printWindow.document.write(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<style>
  @page { margin: 0; size: ${pageSizeCss}; }
  html, body { margin: 0; padding: 0; background: white; }
  * { box-sizing: border-box; }
</style>
</head>
<body>${body}</body>
</html>`);
  printWindow.document.close();

  const doPrint = async () => {
    const images = Array.from(printWindow.document.querySelectorAll("img"));
    await Promise.all(images.map(img => img.decode().catch(() => {})));
    printWindow.focus();
    printWindow.print();
  };
  void doPrint();
}

export function printLabelSheet(
  printWindow: Window,
  pages: LabelPage[],
  grid: GridOutput,
  images: Map<string, string>
) {
  const m = grid.measure;
  const cellStyle = `width:${grid.card.width}${m};height:${grid.card.height}${m};`;

  const body = pages
    .map((page, pi) => {
      const rows = page.rows
        .map(row => {
          const cells = row.items
            .map(item => {
              const src = item ? images.get(item.url) : null;
              if (src) {
                return `<img src="${src}" style="display:block;${cellStyle}" />`;
              }
              return `<div style="${cellStyle}"></div>`;
            })
            .join("");
          return `<div style="display:flex;column-gap:${grid.gapX}${m};row-gap:${grid.gapY}${m};break-inside:avoid;">${cells}</div>`;
        })
        .join("");

      const breakStyle = pi < pages.length - 1 ? "break-after:page;" : "";
      return `<section style="width:${grid.page.width}${m};height:${grid.page.height}${m};padding:${grid.page.pt}${m} ${grid.page.pr}${m} ${grid.page.pb}${m} ${grid.page.pl}${m};background:white;${breakStyle}">${rows}</section>`;
    })
    .join("");

  writeAndPrint(printWindow, body, `${grid.page.width}${m} ${grid.page.height}${m}`);
}
