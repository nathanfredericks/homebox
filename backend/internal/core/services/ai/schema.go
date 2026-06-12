package ai

import (
	"encoding/json"
	"strings"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

// Schemas are built per request: disabled fields are omitted entirely, and
// the custom-fields object mirrors the entity type's template. Strict
// structured output requires every property listed in "required" and
// additionalProperties:false, so optionality is expressed through zero values
// ("", 0, false, []) rather than null unions.

// scalarTypes maps field keys to their JSON type in the analyze schema. The
// suggest schema types everything as string (values are diffed as text).
var scalarTypes = map[string]string{
	"name": "string", "quantity": "number", "description": "string",
	"manufacturer": "string", "modelNumber": "string", "serialNumber": "string",
	"purchasePrice": "number", "purchaseFrom": "string", "purchaseDate": "string",
	"notes": "string",
}

func prop(t string) map[string]any { return map[string]any{"type": t} }

// schemaCustomFields builds the typed object schema for template fields.
// Returns nil when there is nothing to extract.
func schemaCustomFields(c config.AIFieldConfs, fields []repo.TemplateField) map[string]any {
	if !fieldEnabled(c, "customFields") || len(fields) == 0 {
		return nil
	}
	props := map[string]any{}
	required := []string{}
	seen := map[string]bool{}
	for _, f := range fields {
		name := strings.TrimSpace(f.Name)
		// Skip unusable names; dedupe case-insensitively so providers never
		// see two properties differing only in case.
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		switch f.Type {
		case "number":
			props[name] = prop("number")
		case "boolean":
			props[name] = prop("boolean")
		default: // text and time are both strings on the wire
			props[name] = prop("string")
		}
		required = append(required, name)
	}
	if len(props) == 0 {
		return nil
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// itemProperties assembles the per-item schema properties shared by both
// modes. stringScalars forces every scalar to "string" (suggest mode).
func itemProperties(c config.AIFieldConfs, includeTags bool, custom map[string]any, stringScalars bool) map[string]any {
	props := map[string]any{}
	for _, key := range fieldOrder {
		if !fieldEnabled(c, key) {
			continue
		}
		t := scalarTypes[key]
		if stringScalars {
			t = "string"
		}
		props[key] = prop(t)
	}
	if includeTags {
		props["tagIds"] = map[string]any{"type": "array", "items": prop("string")}
	}
	if custom != nil {
		props["customFields"] = custom
	}
	return props
}

func requiredKeys(props map[string]any) []string {
	keys := make([]string, 0, len(props))
	// fieldOrder first for stable, readable schemas; extras after.
	for _, key := range fieldOrder {
		if _, ok := props[key]; ok {
			keys = append(keys, key)
		}
	}
	for _, key := range []string{"tagIds", "customFields"} {
		if _, ok := props[key]; ok {
			keys = append(keys, key)
		}
	}
	return keys
}

// buildAnalyzeSchema is the response schema for capture detection: an items
// array of typed objects.
func buildAnalyzeSchema(c config.AIFieldConfs, includeTags bool, customFields []repo.TemplateField) json.RawMessage {
	props := itemProperties(c, includeTags, schemaCustomFields(c, customFields), false)
	item := map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             requiredKeys(props),
		"additionalProperties": false,
	}
	schema, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{"type": "array", "items": item},
		},
		"required":             []string{"items"},
		"additionalProperties": false,
	})
	return schema
}

// buildSuggestSchema is the response schema for field suggestions: one flat
// object with string scalars.
func buildSuggestSchema(c config.AIFieldConfs, includeTags bool, customFields []repo.TemplateField) json.RawMessage {
	props := itemProperties(c, includeTags, schemaCustomFields(c, customFields), true)
	schema, _ := json.Marshal(map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             requiredKeys(props),
		"additionalProperties": false,
	})
	return schema
}
