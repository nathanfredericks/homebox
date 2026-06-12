package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/pkgs/llm"
	"gocloud.dev/blob"
)

// suggestMaxPhotos caps how many of an item's photos are sent to the model.
const suggestMaxPhotos = 4

// suggestionOrder fixes the field order suggestions are returned in.
var suggestionOrder = []string{
	"name", "quantity", "description", "manufacturer", "modelNumber",
	"serialNumber", "purchasePrice", "purchaseFrom", "purchaseDate", "notes",
}

// SuggestForItem analyzes an existing item's photo attachments and returns
// per-field suggestions plus proposed tags and custom-field values. With
// overwrite false only empty fields get suggestions; with overwrite true
// filled fields may be replaced.
func (s *Service) SuggestForItem(ctx context.Context, gid, itemID uuid.UUID, overwrite bool) (SuggestResult, error) {
	result := SuggestResult{
		Suggestions:  []FieldSuggestion{},
		Tags:         []TagSuggestion{},
		CustomFields: []CustomFieldSuggestion{},
	}

	client, conf, err := s.client()
	if err != nil {
		return result, err
	}

	item, err := s.repos.Entities.GetOneByGroup(ctx, gid, itemID)
	if err != nil {
		return result, err
	}

	images, err := s.loadItemPhotos(ctx, item.Attachments)
	if err != nil {
		return result, err
	}
	if len(images) == 0 {
		return result, ErrNoPhotos
	}

	tags := s.groupTags(ctx, gid, conf)
	var customFields []repo.TemplateField
	if item.EntityType != nil {
		customFields = s.templateFields(ctx, gid, conf, item.EntityType.DefaultTemplateID)
	}

	current := map[string]string{
		"name":          item.Name,
		"quantity":      formatQuantity(item.Quantity),
		"description":   item.Description,
		"manufacturer":  item.Manufacturer,
		"modelNumber":   item.ModelNumber,
		"serialNumber":  item.SerialNumber,
		"purchasePrice": formatPrice(item.PurchasePrice),
		"purchaseFrom":  item.PurchaseFrom,
		"purchaseDate":  formatDate(item.PurchaseDate.Time()),
		"notes":         item.Notes,
	}
	currentJSON, _ := json.Marshal(current)

	parts := make([]llm.ContentPart, 0, len(images)+1)
	parts = append(parts, llm.Text("Fill in the catalog fields for the item in these photos."))
	for _, img := range images {
		parts = append(parts, llm.ImageJPEG(img))
	}

	system := buildSuggestSystem(conf, tags, customFields, string(currentJSON))
	schema := buildSuggestSchema(conf.Fields, len(tags) > 0, customFields)

	var raw struct {
		Name          string         `json:"name"`
		Quantity      string         `json:"quantity"`
		Description   string         `json:"description"`
		Manufacturer  string         `json:"manufacturer"`
		ModelNumber   string         `json:"modelNumber"`
		SerialNumber  string         `json:"serialNumber"`
		PurchasePrice string         `json:"purchasePrice"`
		PurchaseFrom  string         `json:"purchaseFrom"`
		PurchaseDate  string         `json:"purchaseDate"`
		Notes         string         `json:"notes"`
		TagIDs        []string       `json:"tagIds"`
		CustomFields  map[string]any `json:"customFields"`
	}
	if err := client.ChatJSON(ctx, system, parts, "item_fields", schema, &raw); err != nil {
		return result, err
	}

	suggested := map[string]string{
		"name":          raw.Name,
		"quantity":      raw.Quantity,
		"description":   raw.Description,
		"manufacturer":  raw.Manufacturer,
		"modelNumber":   raw.ModelNumber,
		"serialNumber":  raw.SerialNumber,
		"purchasePrice": raw.PurchasePrice,
		"purchaseFrom":  raw.PurchaseFrom,
		"purchaseDate":  validDate(raw.PurchaseDate),
		"notes":         raw.Notes,
	}
	result.Suggestions = filterSuggestions(current, suggested, overwrite)
	result.Tags = tagSuggestions(tags, item.Tags, raw.TagIDs)
	result.CustomFields = customFieldSuggestions(customFields, item.Fields, raw.CustomFields, overwrite)
	return result, nil
}

// filterSuggestions keeps model outputs that are usable and allowed: non-empty
// values that differ from the current one, and (without overwrite) only for
// fields that are currently empty.
func filterSuggestions(current, suggested map[string]string, overwrite bool) []FieldSuggestion {
	out := []FieldSuggestion{}
	for _, field := range suggestionOrder {
		value := suggested[field]
		if emptyish(value) || value == current[field] {
			continue
		}
		if !overwrite && current[field] != "" && current[field] != "0" {
			continue
		}
		switch field {
		case "purchasePrice", "quantity":
			// The model returns numbers as strings; drop unparseable values so
			// the apply step never sends garbage to a number field.
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				continue
			}
		}
		out = append(out, FieldSuggestion{Field: field, Current: current[field], Suggested: value})
	}
	return out
}

// tagSuggestions keeps proposed tags that exist in the group and are not
// already on the item.
func tagSuggestions(groupTags []repo.TagSummary, itemTags []repo.TagSummary, proposed []string) []TagSuggestion {
	byID := make(map[uuid.UUID]repo.TagSummary, len(groupTags))
	for _, t := range groupTags {
		byID[t.ID] = t
	}
	has := make(map[uuid.UUID]bool, len(itemTags))
	for _, t := range itemTags {
		has[t.ID] = true
	}

	out := []TagSuggestion{}
	seen := map[uuid.UUID]bool{}
	for _, rawID := range proposed {
		id, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil || has[id] || seen[id] {
			continue
		}
		tag, ok := byID[id]
		if !ok {
			continue
		}
		seen[id] = true
		out = append(out, TagSuggestion{ID: tag.ID, Name: tag.Name})
	}
	return out
}

// customFieldSuggestions diffs proposed custom-field values against the
// item's current ones, honoring the overwrite flag.
func customFieldSuggestions(templateFields []repo.TemplateField, itemFields []repo.EntityFieldData, proposed map[string]any, overwrite bool) []CustomFieldSuggestion {
	currentByName := make(map[string]repo.EntityFieldData, len(itemFields))
	for _, f := range itemFields {
		currentByName[f.Name] = f
	}

	out := []CustomFieldSuggestion{}
	for _, tf := range templateFields {
		value, ok := proposed[tf.Name]
		if !ok {
			continue
		}
		data, ok := coerceFieldValue(tf, value)
		if !ok {
			continue
		}
		suggested := renderFieldValue(data)

		var current string
		if cur, ok := currentByName[tf.Name]; ok {
			current = renderFieldValue(cur)
		}
		if suggested == current {
			continue
		}
		if !overwrite && current != "" && current != "0" && current != "false" {
			continue
		}
		out = append(out, CustomFieldSuggestion{
			Name:      tf.Name,
			Type:      tf.Type,
			Current:   current,
			Suggested: suggested,
		})
	}
	return out
}

// renderFieldValue string-renders a custom field value for diffing and
// display. Type-specific zero values render as "" so they read as unset.
func renderFieldValue(f repo.EntityFieldData) string {
	switch f.Type {
	case "number":
		if f.NumberValue == 0 {
			return ""
		}
		return strconv.Itoa(f.NumberValue)
	case "boolean":
		if !f.BooleanValue {
			return ""
		}
		return "true"
	case "time":
		return formatDate(f.TimeValue)
	default:
		return f.TextValue
	}
}

// loadItemPhotos reads the item's photo/receipt attachments from blob storage
// (primary photo first) and prepares them for the model.
func (s *Service) loadItemPhotos(ctx context.Context, attachments []repo.ItemAttachment) ([][]byte, error) {
	photos := make([]repo.ItemAttachment, 0, len(attachments))
	for _, a := range attachments {
		if a.Type == "photo" || a.Type == "receipt" {
			photos = append(photos, a)
		}
	}
	sort.SliceStable(photos, func(i, j int) bool { return photos[i].Primary && !photos[j].Primary })
	if len(photos) > suggestMaxPhotos {
		photos = photos[:suggestMaxPhotos]
	}
	if len(photos) == 0 {
		return nil, nil
	}

	bucket, err := blob.OpenBucket(ctx, s.repos.Attachments.GetConnString())
	if err != nil {
		return nil, fmt.Errorf("ai: opening attachment bucket: %w", err)
	}
	defer func() { _ = bucket.Close() }()

	out := make([][]byte, 0, len(photos))
	for _, photo := range photos {
		raw, err := readBlob(ctx, bucket, s.repos.Attachments.GetFullPath(photo.Path))
		if err != nil {
			// A single unreadable photo should not block the others.
			continue
		}
		prepared, err := prepareImage(raw)
		if err != nil {
			continue
		}
		out = append(out, prepared)
	}
	return out, nil
}

func readBlob(ctx context.Context, bucket *blob.Bucket, path string) ([]byte, error) {
	r, err := bucket.NewReader(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

func formatPrice(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func formatQuantity(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// formatDate renders a date as YYYY-MM-DD, or "" for the zero time.
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
