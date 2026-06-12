package ai

import "github.com/google/uuid"

// DetectedItem is one item the vision model identified in capture photos. The
// JSON field names are part of the API contract and mirror entity fields.
type DetectedItem struct {
	Name          string  `json:"name"`
	Quantity      float64 `json:"quantity"`
	Description   string  `json:"description"`
	Manufacturer  string  `json:"manufacturer"`
	ModelNumber   string  `json:"modelNumber"`
	SerialNumber  string  `json:"serialNumber"`
	PurchasePrice float64 `json:"purchasePrice"`
	PurchaseFrom  string  `json:"purchaseFrom"`
	Notes         string  `json:"notes"`
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
