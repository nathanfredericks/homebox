package ai

import "encoding/json"

// detectedItemSchema describes one DetectedItem. Strict structured output
// requires every property listed in "required".
const detectedItemSchema = `{
	"type": "object",
	"properties": {
		"name": {"type": "string"},
		"quantity": {"type": "number"},
		"description": {"type": "string"},
		"manufacturer": {"type": "string"},
		"modelNumber": {"type": "string"},
		"serialNumber": {"type": "string"},
		"purchasePrice": {"type": "number"},
		"purchaseFrom": {"type": "string"},
		"notes": {"type": "string"}
	},
	"required": ["name", "quantity", "description", "manufacturer", "modelNumber", "serialNumber", "purchasePrice", "purchaseFrom", "notes"],
	"additionalProperties": false
}`

var (
	analyzeSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"items": {"type": "array", "items": ` + detectedItemSchema + `}
	},
	"required": ["items"],
	"additionalProperties": false
}`)

	suggestSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"name": {"type": "string"},
		"description": {"type": "string"},
		"manufacturer": {"type": "string"},
		"modelNumber": {"type": "string"},
		"serialNumber": {"type": "string"},
		"purchasePrice": {"type": "string"},
		"purchaseFrom": {"type": "string"},
		"notes": {"type": "string"}
	},
	"required": ["name", "description", "manufacturer", "modelNumber", "serialNumber", "purchasePrice", "purchaseFrom", "notes"],
	"additionalProperties": false
}`)
)
