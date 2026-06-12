package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

// Prompt structure is fixed in code; admins customize behavior through the
// per-field instructions, output language, extra instructions and the
// per-field enable toggles in the AI settings section.

// maxPromptTags caps how many group tags are offered to the model so huge tag
// trees don't blow up the prompt.
const maxPromptTags = 150

// fieldOrder fixes the order field rules appear in prompts and schemas.
var fieldOrder = []string{
	"name", "quantity", "description", "manufacturer", "modelNumber",
	"serialNumber", "purchasePrice", "purchaseFrom", "purchaseDate", "notes",
}

// defaultFieldRules are the built-in per-field prompt rules. An admin
// instruction override (AIFieldConf.Instruction) replaces the rule verbatim.
// The admin settings UI shows these as placeholders (duplicated in the
// frontend's en.json admin.settings.ai.fields.*_default keys — keep in sync).
var defaultFieldRules = map[string]string{
	"name":          `concise title-case name, "Brand Model Item" when identifiable (e.g. "DeWalt DCD771 Drill").`,
	"quantity":      `how many identical units this entry covers, as a number. Use 1 when unsure.`,
	"description":   `what the item is, visible features and specs. Plain text, max 500 characters.`,
	"manufacturer":  `only brands/logos actually visible. Empty string if unknown.`,
	"modelNumber":   `only text actually readable on labels or stickers. Empty string if not visible. Never guess.`,
	"serialNumber":  `only text actually readable on labels or stickers. Empty string if not visible. Never guess.`,
	"purchaseFrom":  `only from visible packaging or receipts. Empty string otherwise.`,
	"purchaseDate":  `only from a visible receipt or dated label, formatted YYYY-MM-DD. Empty string if not visible.`,
	"notes":         `visible damage, missing parts, or new/sealed condition. Empty string otherwise.`,
	"tags":          `assign the ids of listed tags that clearly apply to the item. Empty array when none apply.`,
	"customFields":  `fill each custom field only when its value is determinable from the photos.`,
	"purchasePrice": `only from a visible price tag or receipt. Not visible means no value.`,
}

// purchasePrice carries a mode-specific output format because the analyze
// schema types it as a number and the suggest schema as a string.
const (
	priceFormatAnalyze = ` Return a number; 0 if not visible.`
	priceFormatSuggest = ` Return a plain decimal number string (e.g. "129.99"); empty string if not visible.`
)

// fieldEnabled reports whether a field participates in prompts and schemas.
// name is always on: an item without a name is unusable.
func fieldEnabled(c config.AIFieldConfs, key string) bool {
	if key == "name" {
		return true
	}
	return fieldConf(c, key).Enabled
}

// fieldConf maps a JSON field key to its admin configuration.
func fieldConf(c config.AIFieldConfs, key string) config.AIFieldConf {
	switch key {
	case "name":
		return c.Name
	case "quantity":
		return c.Quantity
	case "description":
		return c.Description
	case "manufacturer":
		return c.Manufacturer
	case "modelNumber":
		return c.ModelNumber
	case "serialNumber":
		return c.SerialNumber
	case "purchasePrice":
		return c.PurchasePrice
	case "purchaseFrom":
		return c.PurchaseFrom
	case "purchaseDate":
		return c.PurchaseDate
	case "notes":
		return c.Notes
	case "tags":
		return c.Tags
	case "customFields":
		return c.CustomFields
	}
	return config.AIFieldConf{}
}

// fieldRule returns the effective prompt rule for one field.
func fieldRule(c config.AIFieldConfs, key, priceFormat string) string {
	if instr := strings.TrimSpace(fieldConf(c, key).Instruction); instr != "" {
		return instr
	}
	rule := defaultFieldRules[key]
	if key == "purchasePrice" {
		rule += priceFormat
	}
	return rule
}

// buildFieldRules emits the "- field: rule" block for all enabled fields.
func buildFieldRules(c config.AIFieldConfs, priceFormat string) string {
	var b strings.Builder
	for _, key := range fieldOrder {
		if !fieldEnabled(c, key) {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s\n", key, fieldRule(c, key, priceFormat))
	}
	return strings.TrimRight(b.String(), "\n")
}

// buildLanguageInstruction asks for output in the configured language. An
// empty or "English" setting emits nothing — prompts are English already.
func buildLanguageInstruction(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" || strings.EqualFold(lang, "english") {
		return ""
	}
	return fmt.Sprintf("Write all names, descriptions and notes in %s. Keep JSON keys, tag ids and serial/model numbers exactly as given.", lang)
}

// buildTagSection offers the group's tags for assignment.
func buildTagSection(c config.AIFieldConfs, tags []repo.TagSummary) string {
	if !fieldEnabled(c, "tags") || len(tags) == 0 {
		return ""
	}
	if len(tags) > maxPromptTags {
		tags = tags[:maxPromptTags]
	}
	var b strings.Builder
	b.WriteString("Available tags — ")
	b.WriteString(fieldRule(c, "tags", ""))
	b.WriteString("\n")
	for _, tag := range tags {
		fmt.Fprintf(&b, "- %s (id: %s)\n", tag.Name, tag.ID)
	}
	return strings.TrimRight(b.String(), "\n")
}

// buildCustomFieldSection describes the template's custom fields and the
// per-type value format.
func buildCustomFieldSection(c config.AIFieldConfs, fields []repo.TemplateField) string {
	if !fieldEnabled(c, "customFields") || len(fields) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Custom fields — ")
	b.WriteString(fieldRule(c, "customFields", ""))
	b.WriteString(` Use the zero value ("", 0, false) when not determinable.`)
	b.WriteString("\n")
	for _, f := range fields {
		if strings.TrimSpace(f.Name) == "" {
			continue
		}
		var format string
		switch f.Type {
		case "number":
			format = "integer"
		case "boolean":
			format = "true or false"
		case "time":
			format = `date string formatted YYYY-MM-DD, "" if unknown`
		default:
			format = `short text, "" if unknown`
		}
		fmt.Fprintf(&b, "- %q: %s\n", f.Name, format)
	}
	return strings.TrimRight(b.String(), "\n")
}

const analyzeIntro = `You are an inventory assistant for a home inventory system. Analyze the supplied photos and identify every distinct physical item.

Rules:
- One entry per distinct item. Identical items get one entry with the correct quantity.
- Ignore the background, fixtures, and surfaces items are resting on.`

const singleItemPrompt = `All photos show the SAME single item from different angles. Return exactly one entry combining everything visible across the photos.`

// buildAnalyzeSystem assembles the detection prompt from the intro, the
// optional single-item hint, the per-field rules, tag/custom-field sections,
// an optional correction round, and the instance extra instructions.
func buildAnalyzeSystem(opts AnalyzeOptions, c config.AIConf, tags []repo.TagSummary, customFields []repo.TemplateField) string {
	sections := []string{analyzeIntro}

	if opts.SingleItem {
		sections = append(sections, singleItemPrompt)
	}
	if lang := buildLanguageInstruction(c.OutputLanguage); lang != "" {
		sections = append(sections, lang)
	}

	sections = append(sections, "Field rules:\n"+buildFieldRules(c.Fields, priceFormatAnalyze))

	if s := buildCustomFieldSection(c.Fields, customFields); s != "" {
		sections = append(sections, s)
	}
	if s := buildTagSection(c.Fields, tags); s != "" {
		sections = append(sections, s)
	}

	if opts.Feedback != "" {
		prior, _ := json.Marshal(opts.PriorItems)
		sections = append(sections, fmt.Sprintf(
			"You previously analyzed these photos and returned:\n%s\n\nThe user corrected you: %q\nRe-analyze the photos with this correction applied and return the full corrected item list.",
			prior, opts.Feedback))
	}

	if c.ExtraInstructions != "" {
		sections = append(sections, "Additional instructions from the administrator:\n"+c.ExtraInstructions)
	}
	return strings.Join(sections, "\n\n")
}

// buildAnalyzeUserText is the text part accompanying the photos. A user hint
// is quoted so the model treats it as context rather than instructions to
// echo back.
func buildAnalyzeUserText(hint string) string {
	text := "Identify the inventory items in these photos."
	if hint = strings.TrimSpace(hint); hint != "" {
		text += fmt.Sprintf("\n\nContext from the user: %q\nUse it to interpret the photos, and copy any stated price, store, brand or identifiers into the matching fields.", hint)
	}
	return text
}

const suggestIntro = `You are an inventory assistant. The photos all show one inventory item. Fill in the item's catalog fields from what is visible.

Rules:
- Return an empty string for any field you cannot determine from the photos.`

// buildSuggestSystem assembles the field-suggestion prompt; currentItem is
// the item's existing values, included so suggestions improve rather than
// echo them.
func buildSuggestSystem(c config.AIConf, tags []repo.TagSummary, customFields []repo.TemplateField, currentItem string) string {
	sections := []string{suggestIntro}

	if lang := buildLanguageInstruction(c.OutputLanguage); lang != "" {
		sections = append(sections, lang)
	}

	sections = append(sections, "Field rules:\n"+buildFieldRules(c.Fields, priceFormatSuggest))

	if s := buildCustomFieldSection(c.Fields, customFields); s != "" {
		sections = append(sections, s)
	}
	if s := buildTagSection(c.Fields, tags); s != "" {
		sections = append(sections, s)
	}

	if currentItem != "" {
		sections = append(sections, "The item's current catalog values are:\n"+currentItem+
			"\nOnly propose values that are better or fill gaps; repeat a current value when you cannot improve it.")
	}

	if c.ExtraInstructions != "" {
		sections = append(sections, "Additional instructions from the administrator:\n"+c.ExtraInstructions)
	}
	return strings.Join(sections, "\n\n")
}
