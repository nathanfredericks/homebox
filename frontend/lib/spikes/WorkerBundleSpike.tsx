"use client";

import { useEffect } from "react";

/**
 * Spike (foundation): verifies that the zxing-wasm barcode-detector worker
 * bundles under `next build` via the standard `new Worker(new URL(...))`
 * pattern. Unused in the running app — the advanced-features agent wires the
 * worker into the scanner pipeline. Kept as a build-time bundling smoke test.
 */
export function WorkerBundleSpike() {
  useEffect(() => {
    const worker = new Worker(new URL("../../workers/barcode-detector.ts", import.meta.url), { type: "module" });
    return () => worker.terminate();
  }, []);

  return null;
}
