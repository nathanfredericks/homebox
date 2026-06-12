// Package llm is a minimal client for OpenAI-compatible chat-completion
// endpoints (OpenAI, OpenRouter, Ollama, LM Studio, vLLM, ...). It only
// supports the one shape Homebox needs: a vision-capable chat call returning
// schema-constrained JSON.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// requestTimeout bounds a single completion call. Vision requests over slow
// providers can take a while; this is intentionally generous.
const requestTimeout = 120 * time.Second

// ErrNotConfigured is returned when the client is missing its endpoint or model.
var ErrNotConfigured = errors.New("llm: endpoint not configured")

// Client talks to one OpenAI-compatible /v1 endpoint.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// NewClient builds a client; an empty base URL or model yields a client whose
// calls fail with ErrNotConfigured.
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		HTTP:    &http.Client{Timeout: requestTimeout},
	}
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role string `json:"role"`
	// Content is a plain string for system messages and an array of content
	// parts for user messages (text + images).
	Content any `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ChatJSON sends a system prompt plus user content parts and decodes the
// model's JSON reply into out. The response is constrained with a strict
// json_schema response_format; providers that reject that format get one
// retry with the looser json_object mode and the schema inlined into the
// system prompt.
func (c *Client) ChatJSON(ctx context.Context, system string, user []ContentPart, schemaName string, schema json.RawMessage, out any) error {
	if c.BaseURL == "" || c.Model == "" {
		return ErrNotConfigured
	}

	strict, err := json.Marshal(map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   schemaName,
			"strict": true,
			"schema": schema,
		},
	})
	if err != nil {
		return fmt.Errorf("llm: marshaling response format: %w", err)
	}

	content, status, err := c.complete(ctx, system, user, strict)
	if err != nil && status >= 400 && status < 500 {
		// Provider likely doesn't support json_schema; fall back to
		// json_object with the schema described in the prompt.
		fallbackSystem := system + "\n\nRespond with a single JSON object matching this JSON Schema exactly:\n" + string(schema)
		content, _, err = c.complete(ctx, fallbackSystem, user, json.RawMessage(`{"type":"json_object"}`))
	}
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(stripFences(content)), out); err != nil {
		return fmt.Errorf("llm: model returned invalid JSON: %w", err)
	}
	return nil
}

// complete performs one chat-completion round trip and returns the assistant
// message content. The HTTP status is returned so callers can distinguish
// 4xx (request shape rejected) from transport errors.
func (c *Client) complete(ctx context.Context, system string, user []ContentPart, responseFormat json.RawMessage) (string, int, error) {
	body, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		ResponseFormat: responseFormat,
	})
	if err != nil {
		return "", 0, fmt.Errorf("llm: marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("llm: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("llm: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("llm: reading response: %w", err)
	}

	var parsed chatResponse
	_ = json.Unmarshal(raw, &parsed)

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(raw))
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		if len(msg) > 500 {
			msg = msg[:500]
		}
		return "", resp.StatusCode, fmt.Errorf("llm: provider returned %d: %s", resp.StatusCode, msg)
	}

	if len(parsed.Choices) == 0 {
		return "", resp.StatusCode, errors.New("llm: provider returned no choices")
	}
	return parsed.Choices[0].Message.Content, resp.StatusCode, nil
}

// stripFences removes a markdown code fence wrapper some models emit even in
// JSON mode.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
