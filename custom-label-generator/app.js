// =============================================================================
// Constants
// =============================================================================

const LABEL_BLANK_LINE = "_______________";

const PIN_ICON_PATH = `<path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0"></path><circle cx="12" cy="10" r="3"></circle>`;

// =============================================================================
// Default Configuration
// =============================================================================

const displayProperties = {
  baseURL: "",
  measure: "in",
  cardHeight: 1,
  cardWidth: 2.63,
  pageWidth: 8.5,
  pageHeight: 11,
  pageTopPadding: 0.5,
  pageBottomPadding: 0.5,
  pageLeftPadding: 0.19,
  pageRightPadding: 0.19,
};

// =============================================================================
// State
// =============================================================================

let loadedData = null;
let printLocationRow = true;
let skipLabels = 0;
let jsonInputTimeout;
let regenerateTimeout;

let out = {
  measure: "in",
  cols: 0,
  rows: 0,
  gapY: 0,
  gapX: 0,
  card: { width: 0, height: 0 },
  page: { width: 0, height: 0, pt: 0, pb: 0, pl: 0, pr: 0 },
};

// =============================================================================
// Pure Functions
// =============================================================================

function fmtAssetID(aid) {
  aid = aid.toString();
  let aidStr = aid.toString().padStart(6, "0");
  aidStr = aidStr.slice(0, 3) + "-" + aidStr.slice(3);
  return aidStr;
}

function calculateGridData(input) {
  const { page, cardHeight, cardWidth } = input;

  const measureRegex = /in|cm|mm/;
  const measure = measureRegex.test(input.measure) ? input.measure : "in";

  const availablePageWidth = page.width - page.pageLeftPadding - page.pageRightPadding;
  const availablePageHeight = page.height - page.pageTopPadding - page.pageBottomPadding;

  if (availablePageWidth < cardWidth || availablePageHeight < cardHeight || isNaN(availablePageWidth) || isNaN(availablePageHeight) || cardWidth <= 0 || cardHeight <= 0) {
    showStatus("Page too small for label size", "error");
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

function getQRCodeUrl(assetID) {
  let origin = displayProperties.baseURL.trim();
  if (origin.endsWith("/")) {
    origin = origin.slice(0, -1);
  }
  return `${origin}/a/${assetID}`;
}

function getItem(item) {
  const assetID = fmtAssetID(item.assetId);
  return {
    url: getQRCodeUrl(assetID),
    assetID: item.assetId,
    name: item.name || LABEL_BLANK_LINE,
    location: item.location?.name || LABEL_BLANK_LINE,
  };
}

function expandByQuantity(items) {
  const expanded = [];
  for (const item of items) {
    const qty = parseInt(item.quantity, 10) || 1;
    for (let i = 0; i < qty; i++) {
      expanded.push({ ...item });
    }
  }
  return expanded;
}

function validateJSON(json) {
  if (!json || typeof json !== "object") {
    return { valid: false, error: "Invalid JSON: must be an object" };
  }
  if (!Array.isArray(json.headers)) {
    return { valid: false, error: "Missing 'headers' array" };
  }
  if (!Array.isArray(json.data)) {
    return { valid: false, error: "Missing 'data' array" };
  }
  if (json.data.length === 0) {
    return { valid: false, error: "Data array is empty" };
  }
  for (let i = 0; i < json.data.length; i++) {
    const item = json.data[i];
    if (typeof item.assetId === "undefined") {
      return { valid: false, error: `Item ${i + 1} missing 'assetId'` };
    }
  }
  return { valid: true };
}

// =============================================================================
// QR Code Generation
// =============================================================================

async function generateQRCode(url) {
  return new Promise((resolve) => {
    const typeNumber = 0;
    const errorCorrectionLevel = "H";
    const qr = qrcode(typeNumber, errorCorrectionLevel);
    qr.addData(url);
    qr.make();
    const cellSize = 4;
    const qrDataUrl = qr.createDataURL(cellSize, 0);

    const img = new Image();
    img.onload = () => {
      const canvas = document.createElement("canvas");
      canvas.width = img.width;
      canvas.height = img.height;
      const ctx = canvas.getContext("2d");
      ctx.drawImage(img, 0, 0);
      resolve(canvas.toDataURL());
    };
    img.src = qrDataUrl;
  });
}

// =============================================================================
// Page Rendering
// =============================================================================

async function calcPages() {
  if (!loadedData) {
    return;
  }

  readSettings();

  const gridResult = calculateGridData({
    measure: displayProperties.measure,
    page: {
      height: displayProperties.pageHeight,
      width: displayProperties.pageWidth,
      pageTopPadding: displayProperties.pageTopPadding,
      pageBottomPadding: displayProperties.pageBottomPadding,
      pageLeftPadding: displayProperties.pageLeftPadding,
      pageRightPadding: displayProperties.pageRightPadding,
    },
    cardHeight: displayProperties.cardHeight,
    cardWidth: displayProperties.cardWidth,
  });

  if (!gridResult) {
    return;
  }
  out = gridResult;

  const expandedItems = expandByQuantity(loadedData.data);
  const items = expandedItems.map(getItem);

  const blankLabels = Array(skipLabels).fill(null);
  const allItems = [...blankLabels, ...items];

  if (items.length === 0) {
    document.getElementById("labelOutput").innerHTML = "";
    return;
  }

  const perPage = out.rows * out.cols;
  const pages = [];
  const itemsCopy = [...allItems];

  while (itemsCopy.length > 0) {
    const page = { rows: [] };

    for (let i = 0; i < perPage; i++) {
      const item = itemsCopy.shift();
      if (typeof item === "undefined") {
        break;
      }

      if (i % out.cols === 0) {
        page.rows.push({ items: [] });
      }

      page.rows[page.rows.length - 1].items.push(item);
    }

    pages.push(page);
  }

  await renderPages(pages, "labelOutput");
}

async function renderPages(pages, outputId) {
  const output = document.getElementById(outputId);
  output.innerHTML = "";

  for (const page of pages) {
    const section = document.createElement("section");
    section.className = "page";
    section.style.cssText = `
      box-sizing: border-box;
      padding-top: ${out.page.pt}${out.measure};
      padding-bottom: ${out.page.pb}${out.measure};
      padding-left: ${out.page.pl}${out.measure};
      padding-right: ${out.page.pr}${out.measure};
      width: ${out.page.width}${out.measure};
      height: ${out.page.height}${out.measure};
      background: white;
      color: black;
    `;

    for (const row of page.rows) {
      const rowDiv = document.createElement("div");
      rowDiv.className = "label-row";
      rowDiv.style.cssText = `
        display: flex;
        break-inside: avoid;
        column-gap: ${out.gapX}${out.measure};
        row-gap: ${out.gapY}${out.measure};
      `;

      for (const item of row.items) {
        const labelDiv = document.createElement("div");
        labelDiv.className = "label";
        labelDiv.style.cssText = `
          display: flex;
          align-items: center;
          gap: 0.1in;
          border: 2px solid transparent;
          height: ${out.card.height}${out.measure};
          width: ${out.card.width}${out.measure};
          padding: 0.1in;
          font-family: 'Open Sans', sans-serif;
        `;

        if (item) {
          const qrContainer = document.createElement("div");
          qrContainer.style.cssText = `
            display: flex;
            align-items: center;
          `;

          const qrImg = document.createElement("img");
          const qrSize = out.card.height - 0.2;
          qrImg.style.cssText = `
            min-width: ${qrSize}${out.measure};
            width: ${qrSize}${out.measure};
            height: ${qrSize}${out.measure};
          `;
          qrImg.src = await generateQRCode(item.url);
          qrContainer.appendChild(qrImg);

          const textContainer = document.createElement("div");
          textContainer.style.cssText = `
            display: flex;
            flex-direction: column;
            justify-content: center;
            flex: 1;
            overflow: hidden;
          `;

          const assetIdDiv = document.createElement("div");
          assetIdDiv.style.cssText = `font-size: 0.75rem; line-height: 1rem; font-family: 'Geist Mono', monospace;`;
          assetIdDiv.textContent = '#' + item.assetID;
          textContainer.appendChild(assetIdDiv);

          const nameDiv = document.createElement("div");
          nameDiv.style.cssText = `
            overflow: hidden;
            font-size: 0.75rem;
            line-height: 1rem;
            font-weight: bold;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
          `;
          nameDiv.textContent = item.name;
          textContainer.appendChild(nameDiv);

          if (printLocationRow) {
            const locationDiv = document.createElement("div");
            locationDiv.style.cssText = `
              display: flex;
              align-items: center;
              gap: 0.125rem;
              font-size: 0.75rem;
              line-height: 1rem;
            `;

            const pinIcon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
            pinIcon.setAttribute("width", "12");
            pinIcon.setAttribute("height", "12");
            pinIcon.setAttribute("viewBox", "0 0 24 24");
            pinIcon.setAttribute("fill", "none");
            pinIcon.setAttribute("stroke", "currentColor");
            pinIcon.setAttribute("stroke-width", "2");
            pinIcon.setAttribute("stroke-linecap", "round");
            pinIcon.setAttribute("stroke-linejoin", "round");
            pinIcon.style.flexShrink = "0";
            pinIcon.innerHTML = PIN_ICON_PATH;

            const locationText = document.createElement("span");
            locationText.textContent = item.location;
            locationDiv.appendChild(pinIcon);
            locationDiv.appendChild(locationText);
            textContainer.appendChild(locationDiv);
          }

          labelDiv.appendChild(qrContainer);
          labelDiv.appendChild(textContainer);
        }

        rowDiv.appendChild(labelDiv);
      }

      section.appendChild(rowDiv);
    }

    output.appendChild(section);
  }
}

// =============================================================================
// Settings & UI
// =============================================================================

function readSettings() {
  const getNum = (id, fallback) => {
    const val = parseFloat(document.getElementById(id)?.value);
    return isNaN(val) ? fallback : val;
  };
  displayProperties.measure = document.getElementById("measureType")?.value || "in";
  displayProperties.cardHeight = getNum("labelHeight", 1);
  displayProperties.cardWidth = getNum("labelWidth", 2.63);
  displayProperties.pageWidth = getNum("pageWidth", 8.5);
  displayProperties.pageHeight = getNum("pageHeight", 11);
  displayProperties.pageTopPadding = getNum("pageTopPadding", 0.52);
  displayProperties.pageBottomPadding = getNum("pageBottomPadding", 0.42);
  displayProperties.pageLeftPadding = getNum("pageLeftPadding", 0.25);
  displayProperties.pageRightPadding = getNum("pageRightPadding", 0.1);
  displayProperties.baseURL = document.getElementById("baseURL")?.value?.trim() || "";
  printLocationRow = document.getElementById("printLocationRow")?.checked ?? true;
  skipLabels = Math.max(0, Math.floor(getNum("skipLabels", 0)));
}

function showStatus(message, type) {
  const el = document.getElementById("statusMsg");
  el.textContent = message;
  el.className = type === "success" ? "small text-success" : "small text-danger";
}

function loadJSON(jsonString) {
  try {
    const json = JSON.parse(jsonString);
    const validation = validateJSON(json);
    if (!validation.valid) {
      showStatus(validation.error, "error");
      loadedData = null;
      return;
    }
    loadedData = json;
    localStorage.setItem("labelGeneratorJson", jsonString);
    calcPages();
  } catch (e) {
    showStatus("Invalid JSON: " + e.message, "error");
    loadedData = null;
  }
}

function updateQRExample() {
  const baseURL = document.getElementById("baseURL").value.trim().replace(/\/$/, "");
  document.getElementById("qrExample").textContent = `${baseURL}/a/{asset_id}`;
}

function resetToDefaults() {
  document.getElementById("measureType").value = "in";
  document.getElementById("labelHeight").value = "1";
  document.getElementById("labelWidth").value = "2.63";
  document.getElementById("pageWidth").value = "8.5";
  document.getElementById("pageHeight").value = "11";
  document.getElementById("pageTopPadding").value = "0.5";
  document.getElementById("pageBottomPadding").value = "0.5";
  document.getElementById("pageLeftPadding").value = "0.19";
  document.getElementById("pageRightPadding").value = "0.19";
}

function debouncedRegenerate() {
  clearTimeout(regenerateTimeout);
  regenerateTimeout = setTimeout(() => {
    if (loadedData) calcPages();
  }, 500);
}

// =============================================================================
// Event Handlers
// =============================================================================

function initEventListeners() {
  document.getElementById("jsonInput").addEventListener("input", () => {
    clearTimeout(jsonInputTimeout);
    jsonInputTimeout = setTimeout(() => {
      const jsonString = document.getElementById("jsonInput").value.trim();
      if (jsonString) {
        loadJSON(jsonString);
      }
    }, 500);
  });

  document.getElementById("fileInput").addEventListener("change", (e) => {
    const file = e.target.files[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (event) => {
        document.getElementById("jsonInput").value = event.target.result;
        loadJSON(event.target.result);
      };
      reader.readAsText(file);
    }
  });

  document.getElementById("baseURL").addEventListener("input", () => {
    updateQRExample();
    debouncedRegenerate();
  });

  const settingInputs = [
    "measureType", "labelHeight", "labelWidth", "pageWidth", "pageHeight",
    "pageTopPadding", "pageBottomPadding", "pageLeftPadding", "pageRightPadding",
    "printLocationRow", "skipLabels"
  ];

  settingInputs.forEach(id => {
    const el = document.getElementById(id);
    if (el) {
      el.addEventListener("change", debouncedRegenerate);
      el.addEventListener("input", debouncedRegenerate);
    }
  });
}

// =============================================================================
// Initialization
// =============================================================================

resetToDefaults();
updateQRExample();
readSettings();
initEventListeners();

const storedJson = localStorage.getItem("labelGeneratorJson");
const textareaJson = document.getElementById("jsonInput").value.trim();

if (storedJson) {
  document.getElementById("jsonInput").value = storedJson;
  loadJSON(storedJson);
} else if (textareaJson) {
  loadJSON(textareaJson);
}
