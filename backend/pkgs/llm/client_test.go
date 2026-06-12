package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testSchema = json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`)

func TestChatJSONSendsStrictSchemaAndAuth(t *testing.T) {
	var got chatRequest
	var auth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"name\":\"Drill\"}"}}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/v1/", "sk-test", "test-model")
	var out struct {
		Name string `json:"name"`
	}
	if err := c.ChatJSON(context.Background(), "sys", []ContentPart{Text("hi"), ImageJPEG([]byte{0xFF, 0xD8})}, "item", testSchema, &out); err != nil {
		t.Fatalf("ChatJSON: %v", err)
	}

	if out.Name != "Drill" {
		t.Errorf("decoded name: got %q", out.Name)
	}
	if auth != "Bearer sk-test" {
		t.Errorf("auth header: got %q", auth)
	}
	if got.Model != "test-model" {
		t.Errorf("model: got %q", got.Model)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Role != "user" {
		t.Fatalf("messages: got %+v", got.Messages)
	}
	parts, ok := got.Messages[1].Content.([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("user content: got %T %+v", got.Messages[1].Content, got.Messages[1].Content)
	}
	img := parts[1].(map[string]any)
	if img["type"] != "image_url" {
		t.Errorf("second part type: got %v", img["type"])
	}

	var rf map[string]any
	if err := json.Unmarshal(got.ResponseFormat, &rf); err != nil {
		t.Fatalf("response_format: %v", err)
	}
	if rf["type"] != "json_schema" {
		t.Errorf("response_format type: got %v", rf["type"])
	}
}

func TestChatJSONFallsBackToJSONObject(t *testing.T) {
	var formats []string
	var lastSystem string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		var rf map[string]any
		_ = json.Unmarshal(req.ResponseFormat, &rf)
		formats = append(formats, rf["type"].(string))
		lastSystem, _ = req.Messages[0].Content.(string)

		if rf["type"] == "json_schema" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"response_format not supported"}}`))
			return
		}
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"name\\\":\\\"Saw\\\"}\\n```\"}}]}"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "test-model")
	var out struct {
		Name string `json:"name"`
	}
	if err := c.ChatJSON(context.Background(), "sys", []ContentPart{Text("hi")}, "item", testSchema, &out); err != nil {
		t.Fatalf("ChatJSON: %v", err)
	}

	if out.Name != "Saw" {
		t.Errorf("decoded name (fence-stripped): got %q", out.Name)
	}
	if len(formats) != 2 || formats[0] != "json_schema" || formats[1] != "json_object" {
		t.Errorf("formats tried: got %v, want [json_schema json_object]", formats)
	}
	if lastSystem == "sys" {
		t.Error("fallback system prompt should embed the schema")
	}
}

func TestChatJSONServerErrorIsNotRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "test-model")
	var out any
	if err := c.ChatJSON(context.Background(), "sys", []ContentPart{Text("hi")}, "item", testSchema, &out); err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls: got %d, want 1 (5xx must not trigger format fallback)", calls)
	}
}

func TestChatJSONNotConfigured(t *testing.T) {
	c := NewClient("", "", "")
	var out any
	if err := c.ChatJSON(context.Background(), "sys", nil, "item", testSchema, &out); err != ErrNotConfigured {
		t.Errorf("got %v, want ErrNotConfigured", err)
	}
}
