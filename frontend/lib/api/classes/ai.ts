import { BaseAPI, route } from "../base";
import type { AIAnalyzeResult, AIStatus, AISuggestResult, AiAnalyzedItem } from "../types/data-contracts";

/** A detected item without the duplicate annotation (ai.DetectedItem). */
export type DetectedItem = Omit<AiAnalyzedItem, "duplicate">;

/** Options for an analyze run; mirrors ai.AnalyzeOptions on the backend. */
export type AnalyzeOptions = {
  /** All photos show one item from multiple angles. */
  singleItem?: boolean;
  /** Re-run with user corrections applied to priorItems. */
  feedback?: string;
  priorItems?: DetectedItem[];
};

export class AiApi extends BaseAPI {
  /** Whether AI features are enabled on this instance. */
  status() {
    return this.http.get<AIStatus>({ url: route("/ai/status") });
  }

  /** Run vision detection over capture photos. */
  analyze(images: (File | Blob)[], options: AnalyzeOptions = {}) {
    const formData = new FormData();
    for (const image of images) {
      formData.append("images", image);
    }
    if (Object.keys(options).length) {
      formData.append("options", JSON.stringify(options));
    }

    return this.http.post<FormData, AIAnalyzeResult>({
      url: route("/ai/analyze"),
      data: formData,
    });
  }

  /** Suggest field values for an existing item from its photos. */
  suggest(id: string, overwrite = false) {
    return this.http.post<{ overwrite: boolean }, AISuggestResult>({
      url: route(`/ai/entities/${id}/suggest`),
      body: { overwrite },
    });
  }
}
