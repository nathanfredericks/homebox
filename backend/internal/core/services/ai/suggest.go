package ai

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/pkgs/llm"
	"gocloud.dev/blob"
)

// suggestMaxPhotos caps how many of an item's photos are sent to the model.
const suggestMaxPhotos = 4

// suggestionOrder fixes the field order suggestions are returned in.
var suggestionOrder = []string{
	"name", "description", "manufacturer", "modelNumber",
	"serialNumber", "purchasePrice", "purchaseFrom", "notes",
}

// SuggestForItem analyzes an existing item's photo attachments and returns
// per-field suggestions. With overwrite false only empty fields get
// suggestions; with overwrite true filled fields may be replaced.
func (s *Service) SuggestForItem(ctx context.Context, gid, itemID uuid.UUID, overwrite bool) ([]FieldSuggestion, error) {
	client, extra, err := s.client()
	if err != nil {
		return nil, err
	}

	item, err := s.repos.Entities.GetOneByGroup(ctx, gid, itemID)
	if err != nil {
		return nil, err
	}

	images, err := s.loadItemPhotos(ctx, item.Attachments)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, ErrNoPhotos
	}

	parts := make([]llm.ContentPart, 0, len(images)+1)
	parts = append(parts, llm.Text("Fill in the catalog fields for the item in these photos."))
	for _, img := range images {
		parts = append(parts, llm.ImageJPEG(img))
	}

	var suggested map[string]string
	if err := client.ChatJSON(ctx, buildSuggestSystem(extra), parts, "item_fields", suggestSchema, &suggested); err != nil {
		return nil, err
	}

	current := map[string]string{
		"name":          item.Name,
		"description":   item.Description,
		"manufacturer":  item.Manufacturer,
		"modelNumber":   item.ModelNumber,
		"serialNumber":  item.SerialNumber,
		"purchasePrice": formatPrice(item.PurchasePrice),
		"purchaseFrom":  item.PurchaseFrom,
		"notes":         item.Notes,
	}

	return filterSuggestions(current, suggested, overwrite), nil
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
		if field == "purchasePrice" {
			// The model returns price as a string; drop unparseable values so
			// the apply step never sends garbage to the number field.
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				continue
			}
		}
		out = append(out, FieldSuggestion{Field: field, Current: current[field], Suggested: value})
	}
	return out
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
