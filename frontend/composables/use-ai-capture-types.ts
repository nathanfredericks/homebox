import type { AiAnalyzedItem, AiFieldSuggestion, EntityPatch } from "~~/lib/api/types/data-contracts";

/** A detected item being reviewed/edited before creation. */
export type EditableDetectedItem = AiAnalyzedItem & {
  include: boolean;
  /** Indexes into the session's captured photos to attach on create. */
  photoIdx: number[];
};

/** Builds the PATCH payload applying the given AI field suggestions. */
export function suggestionsToPatch(id: string, suggestions: AiFieldSuggestion[]): EntityPatch {
  const patch: EntityPatch = { id };
  for (const s of suggestions) {
    switch (s.field) {
      case "name":
        patch.name = s.suggested;
        break;
      case "description":
        patch.description = s.suggested;
        break;
      case "manufacturer":
        patch.manufacturer = s.suggested;
        break;
      case "modelNumber":
        patch.modelNumber = s.suggested;
        break;
      case "serialNumber":
        patch.serialNumber = s.suggested;
        break;
      case "purchaseFrom":
        patch.purchaseFrom = s.suggested;
        break;
      case "notes":
        patch.notes = s.suggested;
        break;
      case "purchasePrice":
        patch.purchasePrice = Number(s.suggested);
        break;
    }
  }
  return patch;
}
