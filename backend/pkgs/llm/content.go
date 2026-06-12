package llm

import "encoding/base64"

// ContentPart is one element of a multimodal user message.
type ContentPart map[string]any

// Text builds a text content part.
func Text(s string) ContentPart {
	return ContentPart{"type": "text", "text": s}
}

// ImageJPEG builds an image content part from raw JPEG bytes as a data URL.
func ImageJPEG(b []byte) ContentPart {
	return ContentPart{
		"type": "image_url",
		"image_url": map[string]any{
			"url": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b),
		},
	}
}
