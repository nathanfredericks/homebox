package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Prompts are fixed in code; the only runtime customization is the
// extraInstructions admin setting appended to every system prompt.

const analyzePrompt = `You are an inventory assistant for a home inventory system. Analyze the supplied photos and identify every distinct physical item.

Rules:
- One entry per distinct item. Identical items get one entry with the correct quantity.
- name: concise title-case name, "Brand Model Item" when identifiable (e.g. "DeWalt DCD771 Drill").
- description: what the item is, visible features and specs. Plain text, max 500 characters.
- manufacturer: only brands/logos actually visible. Empty string if unknown.
- modelNumber / serialNumber: only text actually readable on labels or stickers. Empty string if not visible. Never guess.
- purchasePrice: only from a visible price tag or receipt, as a number. 0 if not visible.
- purchaseFrom: only from visible packaging or receipts. Empty string otherwise.
- notes: visible damage, missing parts, or new/sealed condition. Empty string otherwise.
- Ignore the background, fixtures, and surfaces items are resting on.`

const singleItemPrompt = `All photos show the SAME single item from different angles. Return exactly one entry combining everything visible across the photos.`

const suggestPrompt = `You are an inventory assistant. The photos all show one inventory item. Fill in the item's catalog fields from what is visible.

Rules:
- name: concise title-case name, "Brand Model Item" when identifiable.
- description: what the item is, visible features and specs. Plain text, max 500 characters.
- manufacturer: only brands/logos actually visible. Empty string if unknown.
- modelNumber / serialNumber: only text actually readable on labels or stickers. Empty string if not visible. Never guess.
- purchasePrice: only from a visible price tag or receipt, as a plain decimal number string (e.g. "129.99"). Empty string if not visible.
- purchaseFrom: only from visible packaging or receipts. Empty string otherwise.
- notes: visible damage, missing parts, or new/sealed condition. Empty string otherwise.
- Return an empty string for any field you cannot determine from the photos.`

// buildAnalyzeSystem assembles the detection prompt, optionally with the
// single-item hint, a correction round, and the instance extra instructions.
func buildAnalyzeSystem(opts AnalyzeOptions, extraInstructions string) string {
	var b strings.Builder
	b.WriteString(analyzePrompt)

	if opts.SingleItem {
		b.WriteString("\n\n")
		b.WriteString(singleItemPrompt)
	}

	if opts.Feedback != "" {
		prior, _ := json.Marshal(opts.PriorItems)
		fmt.Fprintf(&b, "\n\nYou previously analyzed these photos and returned:\n%s\n\nThe user corrected you: %q\nRe-analyze the photos with this correction applied and return the full corrected item list.", prior, opts.Feedback)
	}

	if extraInstructions != "" {
		b.WriteString("\n\nAdditional instructions from the administrator:\n")
		b.WriteString(extraInstructions)
	}
	return b.String()
}

func buildSuggestSystem(extraInstructions string) string {
	if extraInstructions == "" {
		return suggestPrompt
	}
	return suggestPrompt + "\n\nAdditional instructions from the administrator:\n" + extraInstructions
}
