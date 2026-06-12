package ai

import (
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

// DetectedItem is one item the vision model identified in capture photos. The
// JSON field names are part of the API contract and mirror entity fields.
// Fields disabled in the admin AI settings come back as zero values.
type DetectedItem struct {
	Name          string  `json:"name"`
	Quantity      float64 `json:"quantity"`
	Description   string  `json:"description"`
	Manufacturer  string  `json:"manufacturer"`
	ModelNumber   string  `json:"modelNumber"`
	SerialNumber  string  `json:"serialNumber"`
	PurchasePrice float64 `json:"purchasePrice"`
	PurchaseFrom  string  `json:"purchaseFrom"`
	// PurchaseDate is "YYYY-MM-DD" or empty.
	PurchaseDate string `json:"purchaseDate"`
	Notes        string `json:"notes"`
	// TagIDs only ever contains tags that exist in the group; hallucinated
	// IDs are filtered out before the result leaves the service.
	TagIDs []uuid.UUID `json:"tagIds"`
	// Fields are values for the entity type's template custom fields,
	// validated against the template (unknown names are dropped).
	Fields []repo.EntityFieldData `json:"fields"`
}

// DuplicateMatch points at an existing entity with the same serial number.
type DuplicateMatch struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	SerialNumber string    `json:"serialNumber"`
}

// AnalyzedItem is a DetectedItem annotated with duplicate detection results.
type AnalyzedItem struct {
	DetectedItem
	Duplicate *DuplicateMatch `json:"duplicate,omitempty" extensions:"x-nullable,x-omitempty"`
}

// AnalyzeOptions tunes a capture analysis run.
type AnalyzeOptions struct {
	// SingleItem hints that all photos show one item from multiple angles.
	SingleItem bool `json:"singleItem"`
	// Hint is free-form user context about the photos ("everything here is
	// from the garage shelf, receipt in photo 2"), added to the user message.
	Hint string `json:"hint"`
	// EntityTypeID selects whose template defines the custom fields to
	// extract. Zero falls back to the group's default item type.
	EntityTypeID uuid.UUID `json:"entityTypeId"`
	// Feedback re-runs the analysis with user corrections applied to PriorItems.
	Feedback   string         `json:"feedback"`
	PriorItems []DetectedItem `json:"priorItems"`
}

// FieldSuggestion is one proposed value for an existing item's field.
type FieldSuggestion struct {
	Field     string `json:"field"`
	Current   string `json:"current"`
	Suggested string `json:"suggested"`
}

// TagSuggestion proposes adding an existing tag to the item.
type TagSuggestion struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// CustomFieldSuggestion proposes a value for one of the item's template
// custom fields. Values are string-rendered; Type tells the client how to
// coerce ("text", "number", "boolean" or "time").
type CustomFieldSuggestion struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Current   string `json:"current"`
	Suggested string `json:"suggested"`
}

// SuggestResult groups everything the model proposed for an existing item.
type SuggestResult struct {
	Suggestions  []FieldSuggestion       `json:"suggestions"`
	Tags         []TagSuggestion         `json:"tags"`
	CustomFields []CustomFieldSuggestion `json:"customFields"`
}

// modelItem is the raw shape the model returns for one item. Tag IDs arrive
// as plain strings (a hallucinated non-UUID must not fail the whole decode)
// and custom fields as a name->value object; both are validated and converted
// into the DetectedItem contract by the service.
type modelItem struct {
	Name          string         `json:"name"`
	Quantity      float64        `json:"quantity"`
	Description   string         `json:"description"`
	Manufacturer  string         `json:"manufacturer"`
	ModelNumber   string         `json:"modelNumber"`
	SerialNumber  string         `json:"serialNumber"`
	PurchasePrice float64        `json:"purchasePrice"`
	PurchaseFrom  string         `json:"purchaseFrom"`
	PurchaseDate  string         `json:"purchaseDate"`
	Notes         string         `json:"notes"`
	TagIDs        []string       `json:"tagIds"`
	CustomFields  map[string]any `json:"customFields"`
}
