import type {
  AiAnalyzedItem,
  AiCustomFieldSuggestion,
  AiFieldSuggestion,
  EntityFieldData,
  EntityPatch,
} from "~~/lib/api/types/data-contracts";

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
      case "quantity":
        patch.quantity = Number(s.suggested);
        break;
      case "purchaseDate":
        patch.purchaseDate = s.suggested;
        break;
    }
  }
  return patch;
}

/**
 * Converts a date-only string into the RFC3339 instant EntityFieldData.timeValue
 * expects on the wire (Go time.Time rejects bare dates).
 */
export function dateToTimeValue(date: string): string {
  if (!date) return "";
  return /^\d{4}-\d{2}-\d{2}$/.test(date) ? `${date}T00:00:00Z` : date;
}

/** Converts accepted custom-field suggestions into typed patch field entries. */
export function customFieldSuggestionsToFields(accepted: AiCustomFieldSuggestion[]): EntityFieldData[] {
  return accepted.map(s => {
    // id/timeValue null: Go treats JSON null as "unset" while "" fails to
    // parse as a UUID/time — same casting idiom as the item edit page.
    const field = {
      id: null,
      type: s.type,
      name: s.name,
      textValue: "",
      numberValue: 0,
      booleanValue: false,
      timeValue: null as string | null,
    } as unknown as EntityFieldData;
    switch (s.type) {
      case "number":
        field.numberValue = Number(s.suggested) || 0;
        break;
      case "boolean":
        field.booleanValue = s.suggested === "true";
        break;
      case "time":
        if (s.suggested) field.timeValue = dateToTimeValue(s.suggested);
        break;
      default:
        field.textValue = s.suggested;
    }
    return field;
  });
}
