package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/pkgs/llm"
)

// AnalyzeImages runs item detection over capture photos and annotates results
// with duplicate matches by serial number. A non-empty opts.Feedback re-runs
// the analysis as a correction round over opts.PriorItems.
func (s *Service) AnalyzeImages(ctx context.Context, gid uuid.UUID, images [][]byte, opts AnalyzeOptions) ([]AnalyzedItem, error) {
	client, conf, err := s.client()
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("ai: no images supplied")
	}
	if len(images) > MaxImages {
		images = images[:MaxImages]
	}

	tags := s.groupTags(ctx, gid, conf)
	customFields := s.templateFieldsForType(ctx, gid, conf, opts.EntityTypeID)

	parts := make([]llm.ContentPart, 0, len(images)+1)
	parts = append(parts, llm.Text(buildAnalyzeUserText(opts.Hint)))
	for _, raw := range images {
		prepared, err := prepareImage(raw)
		if err != nil {
			return nil, err
		}
		parts = append(parts, llm.ImageJPEG(prepared))
	}

	system := buildAnalyzeSystem(opts, conf, tags, customFields)
	schema := buildAnalyzeSchema(conf.Fields, len(tags) > 0, customFields)

	var result struct {
		Items []modelItem `json:"items"`
	}
	if err := client.ChatJSON(ctx, system, parts, "detected_items", schema, &result); err != nil {
		return nil, err
	}

	tagSet := tagIDSet(tags)
	defaultTag := defaultTagID(conf, tagSet)

	out := make([]AnalyzedItem, 0, len(result.Items))
	for _, mi := range result.Items {
		item := convertModelItem(mi, tagSet, defaultTag, customFields)
		analyzed := AnalyzedItem{DetectedItem: item}
		if !emptyish(item.SerialNumber) {
			analyzed.Duplicate = s.findDuplicate(ctx, gid, item.SerialNumber)
		}
		out = append(out, analyzed)
	}
	return out, nil
}

// tagIDSet indexes group tags for validating model output.
func tagIDSet(tags []repo.TagSummary) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool, len(tags))
	for _, t := range tags {
		set[t.ID] = true
	}
	return set
}

// defaultTagID parses the configured default tag and verifies it exists in
// the group; anything else means "no default tag".
func defaultTagID(c config.AIConf, tagSet map[uuid.UUID]bool) uuid.UUID {
	id, err := uuid.Parse(strings.TrimSpace(c.DefaultTagID))
	if err != nil || !tagSet[id] {
		return uuid.Nil
	}
	return id
}

// convertModelItem validates raw model output into the DetectedItem contract:
// hallucinated tag IDs are dropped, the default tag is appended, custom-field
// values are matched against the template and coerced by type, and dates must
// parse as YYYY-MM-DD.
func convertModelItem(mi modelItem, tagSet map[uuid.UUID]bool, defaultTag uuid.UUID, customFields []repo.TemplateField) DetectedItem {
	item := DetectedItem{
		Name:          mi.Name,
		Quantity:      mi.Quantity,
		Description:   mi.Description,
		Manufacturer:  mi.Manufacturer,
		ModelNumber:   mi.ModelNumber,
		SerialNumber:  mi.SerialNumber,
		PurchasePrice: mi.PurchasePrice,
		PurchaseFrom:  mi.PurchaseFrom,
		PurchaseDate:  validDate(mi.PurchaseDate),
		Notes:         mi.Notes,
		TagIDs:        []uuid.UUID{},
		Fields:        []repo.EntityFieldData{},
	}

	seen := map[uuid.UUID]bool{}
	for _, raw := range mi.TagIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil || !tagSet[id] || seen[id] {
			continue
		}
		seen[id] = true
		item.TagIDs = append(item.TagIDs, id)
	}
	if defaultTag != uuid.Nil && !seen[defaultTag] {
		item.TagIDs = append(item.TagIDs, defaultTag)
	}

	for _, tf := range customFields {
		value, ok := mi.CustomFields[tf.Name]
		if !ok {
			continue
		}
		if data, ok := coerceFieldValue(tf, value); ok {
			item.Fields = append(item.Fields, data)
		}
	}
	return item
}

// coerceFieldValue converts one raw custom-field value into typed
// EntityFieldData. Zero values ("", 0, false, unparseable dates) report
// ok=false — the model saying "nothing determinable" is not a suggestion.
func coerceFieldValue(tf repo.TemplateField, value any) (repo.EntityFieldData, bool) {
	data := repo.EntityFieldData{Type: tf.Type, Name: tf.Name}
	switch tf.Type {
	case "number":
		n, ok := value.(float64)
		if !ok || n == 0 {
			return data, false
		}
		data.NumberValue = int(n)
	case "boolean":
		b, ok := value.(bool)
		if !ok || !b {
			return data, false
		}
		data.BooleanValue = true
	case "time":
		s, ok := value.(string)
		if !ok {
			return data, false
		}
		date := validDate(s)
		if date == "" {
			return data, false
		}
		t, _ := time.Parse("2006-01-02", date)
		data.TimeValue = t
	default: // text
		s, ok := value.(string)
		if !ok || emptyish(s) {
			return data, false
		}
		data.TextValue = strings.TrimSpace(s)
	}
	return data, true
}

// validDate returns the trimmed value when it parses as YYYY-MM-DD, else "".
func validDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ""
	}
	return s
}

// findDuplicate looks up an existing entity with the same serial number.
// Lookup failures degrade to "no duplicate" rather than failing the analysis.
func (s *Service) findDuplicate(ctx context.Context, gid uuid.UUID, serial string) *DuplicateMatch {
	matches, err := s.repos.Entities.GetBySerialNumber(ctx, gid, serial)
	if err != nil || len(matches) == 0 {
		return nil
	}
	return &DuplicateMatch{
		ID:           matches[0].ID,
		Name:         matches[0].Name,
		SerialNumber: matches[0].SerialNumber,
	}
}
