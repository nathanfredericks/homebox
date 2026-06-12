package ai

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/pkgs/llm"
)

// AnalyzeImages runs item detection over capture photos and annotates results
// with duplicate matches by serial number. A non-empty opts.Feedback re-runs
// the analysis as a correction round over opts.PriorItems.
func (s *Service) AnalyzeImages(ctx context.Context, gid uuid.UUID, images [][]byte, opts AnalyzeOptions) ([]AnalyzedItem, error) {
	client, extra, err := s.client()
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("ai: no images supplied")
	}
	if len(images) > MaxImages {
		images = images[:MaxImages]
	}

	parts := make([]llm.ContentPart, 0, len(images)+1)
	parts = append(parts, llm.Text("Identify the inventory items in these photos."))
	for _, raw := range images {
		prepared, err := prepareImage(raw)
		if err != nil {
			return nil, err
		}
		parts = append(parts, llm.ImageJPEG(prepared))
	}

	var result struct {
		Items []DetectedItem `json:"items"`
	}
	if err := client.ChatJSON(ctx, buildAnalyzeSystem(opts, extra), parts, "detected_items", analyzeSchema, &result); err != nil {
		return nil, err
	}

	out := make([]AnalyzedItem, 0, len(result.Items))
	for _, item := range result.Items {
		analyzed := AnalyzedItem{DetectedItem: item}
		if !emptyish(item.SerialNumber) {
			analyzed.Duplicate = s.findDuplicate(ctx, gid, item.SerialNumber)
		}
		out = append(out, analyzed)
	}
	return out, nil
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
