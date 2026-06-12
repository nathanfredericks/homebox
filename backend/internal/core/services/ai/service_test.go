package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

func newTestService(t *testing.T, llmURL string) (*Service, *repo.AllRepos, repo.Group) {
	t.Helper()

	client, err := ent.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared&_fk=1&_time_format=sqlite")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	bus := eventbus.New()
	repos := repo.New(client, bus, config.Storage{PrefixPath: "/", ConnString: "mem://"}, "mem://{{ .Topic }}",
		repo.StaticThumbnail(config.Thumbnail{}))

	group, err := repos.Groups.GroupCreate(context.Background(), "ai-test-group")
	if err != nil {
		t.Fatalf("creating group: %v", err)
	}

	svc := NewService(repos, func() config.AIConf {
		return config.AIConf{Enabled: true, BaseURL: llmURL, Model: "test-model"}
	})
	return svc, repos, group
}

// testPNG renders a small valid image for prepareImage to chew on.
func testPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatalf("encoding test png: %v", err)
	}
	return buf.Bytes()
}

func llmStub(t *testing.T, content string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": content}}},
		})
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAnalyzeImagesParsesAndFlagsDuplicates(t *testing.T) {
	srv := llmStub(t, `{"items":[
		{"name":"DeWalt Drill","quantity":1,"description":"Cordless drill","manufacturer":"DeWalt","modelNumber":"DCD771","serialNumber":"SN-123","purchasePrice":0,"purchaseFrom":"","notes":""},
		{"name":"Hammer","quantity":2,"description":"Claw hammer","manufacturer":"","modelNumber":"","serialNumber":"","purchasePrice":0,"purchaseFrom":"","notes":""}
	]}`)

	svc, repos, group := newTestService(t, srv.URL)
	ctx := context.Background()

	// Seed an existing item with the same serial (different case).
	existing, err := repos.Entities.Create(ctx, group.ID, repo.EntityCreate{
		Name:         "Old Drill",
		SerialNumber: "sn-123",
	})
	if err != nil {
		t.Fatalf("seeding entity: %v", err)
	}

	items, err := svc.AnalyzeImages(ctx, group.ID, [][]byte{testPNG(t)}, AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeImages: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("items: got %d, want 2", len(items))
	}
	if items[0].Name != "DeWalt Drill" || items[0].Quantity != 1 {
		t.Errorf("first item: got %+v", items[0].DetectedItem)
	}
	if items[0].Duplicate == nil {
		t.Fatal("first item: expected duplicate match by serial")
	}
	if items[0].Duplicate.ID != existing.ID || items[0].Duplicate.Name != "Old Drill" {
		t.Errorf("duplicate: got %+v", items[0].Duplicate)
	}
	if items[1].Duplicate != nil {
		t.Errorf("second item (no serial): unexpected duplicate %+v", items[1].Duplicate)
	}
}

func TestAnalyzeImagesValidatesTagsFieldsAndDates(t *testing.T) {
	svcPlaceholder, repos, group := newTestService(t, "")
	_ = svcPlaceholder
	ctx := context.Background()

	tag, err := repos.Tags.Create(ctx, group.ID, repo.TagCreate{Name: "Tools"})
	if err != nil {
		t.Fatalf("creating tag: %v", err)
	}
	defaultTag, err := repos.Tags.Create(ctx, group.ID, repo.TagCreate{Name: "AI Imported"})
	if err != nil {
		t.Fatalf("creating default tag: %v", err)
	}

	// The model returns one valid tag, one hallucinated tag, a custom field
	// from the template, a hallucinated custom field, and a junk date.
	srv := llmStub(t, `{"items":[{
		"name":"DeWalt Drill","quantity":1,"description":"","manufacturer":"","modelNumber":"","serialNumber":"",
		"purchasePrice":0,"purchaseFrom":"","purchaseDate":"not-a-date","notes":"",
		"tagIds":["`+tag.ID.String()+`","11111111-2222-3333-4444-555555555555","garbage"],
		"customFields":{"Voltage":18,"Imaginary":"x"}
	}]}`)

	svc := NewService(repos, func() config.AIConf {
		c := config.AIConf{Enabled: true, BaseURL: srv.URL, Model: "test-model", DefaultTagID: defaultTag.ID.String()}
		c.Fields.Tags.Enabled = true
		c.Fields.CustomFields.Enabled = true
		return c
	})

	// Seed an entity type with a default template defining "Voltage".
	tpl, err := repos.EntityTemplates.Create(ctx, group.ID, repo.EntityTemplateCreate{
		Name:   "Tool Template",
		Fields: []repo.TemplateField{{Type: "number", Name: "Voltage"}},
	})
	if err != nil {
		t.Fatalf("creating template: %v", err)
	}
	et, err := repos.EntityTypes.Create(ctx, group.ID, repo.EntityTypeCreate{Name: "Tool"})
	if err != nil {
		t.Fatalf("creating entity type: %v", err)
	}
	if _, err := repos.EntityTypes.Update(ctx, group.ID, repo.EntityTypeUpdate{
		ID: et.ID, Name: "Tool", DefaultTemplateID: &tpl.ID,
	}); err != nil {
		t.Fatalf("setting default template: %v", err)
	}

	items, err := svc.AnalyzeImages(ctx, group.ID, [][]byte{testPNG(t)}, AnalyzeOptions{EntityTypeID: et.ID})
	if err != nil {
		t.Fatalf("AnalyzeImages: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items: got %d, want 1", len(items))
	}

	got := items[0].DetectedItem
	if len(got.TagIDs) != 2 || got.TagIDs[0] != tag.ID || got.TagIDs[1] != defaultTag.ID {
		t.Errorf("tagIds: got %v, want [%s %s] (valid tag + default, hallucinations dropped)", got.TagIDs, tag.ID, defaultTag.ID)
	}
	if len(got.Fields) != 1 || got.Fields[0].Name != "Voltage" || got.Fields[0].NumberValue != 18 {
		t.Errorf("fields: got %+v, want only Voltage=18", got.Fields)
	}
	if got.PurchaseDate != "" {
		t.Errorf("purchaseDate: junk date must blank, got %q", got.PurchaseDate)
	}
}

func TestBuildSchemasRespectFieldToggles(t *testing.T) {
	var fields config.AIFieldConfs
	fields.Name.Enabled = true
	fields.Quantity.Enabled = true
	fields.SerialNumber.Enabled = false
	fields.CustomFields.Enabled = true

	schema := buildAnalyzeSchema(fields, true, []repo.TemplateField{
		{Type: "number", Name: "Voltage"},
		{Type: "text", Name: ""},        // unusable: empty name
		{Type: "text", Name: "voltage"}, // duplicate after case fold
	})

	var parsed struct {
		Properties struct {
			Items struct {
				Items struct {
					Properties map[string]json.RawMessage `json:"properties"`
					Required   []string                   `json:"required"`
				} `json:"items"`
			} `json:"items"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("unmarshaling schema: %v", err)
	}

	props := parsed.Properties.Items.Items.Properties
	if _, ok := props["serialNumber"]; ok {
		t.Error("disabled serialNumber must be absent from schema")
	}
	if _, ok := props["name"]; !ok {
		t.Error("name must always be present")
	}
	if _, ok := props["tagIds"]; !ok {
		t.Error("tagIds expected when tags are enabled and present")
	}
	if len(parsed.Properties.Items.Items.Required) != len(props) {
		t.Errorf("strict mode requires every property in required: %d props, %d required",
			len(props), len(parsed.Properties.Items.Items.Required))
	}

	var custom struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(props["customFields"], &custom); err != nil {
		t.Fatalf("customFields schema: %v", err)
	}
	if len(custom.Properties) != 1 {
		t.Errorf("custom fields: got %d properties, want 1 (empty + case-duplicate dropped)", len(custom.Properties))
	}
}

func TestTagAndCustomFieldSuggestions(t *testing.T) {
	toolsID := uuid.New()
	ownedID := uuid.New()
	groupTags := []repo.TagSummary{{ID: toolsID, Name: "Tools"}, {ID: ownedID, Name: "Owned"}}
	itemTags := []repo.TagSummary{{ID: ownedID, Name: "Owned"}}

	tags := tagSuggestions(groupTags, itemTags, []string{
		toolsID.String(),                       // new -> suggested
		ownedID.String(),                       // already on item -> dropped
		toolsID.String(),                       // duplicate -> dropped
		"11111111-2222-3333-4444-555555555555", // unknown -> dropped
		"garbage",                              // not a uuid -> dropped
	})
	if len(tags) != 1 || tags[0].ID != toolsID || tags[0].Name != "Tools" {
		t.Errorf("tag suggestions: got %+v, want only Tools", tags)
	}

	template := []repo.TemplateField{
		{Type: "number", Name: "Voltage"},
		{Type: "text", Name: "Color"},
		{Type: "boolean", Name: "Cordless"},
	}
	current := []repo.EntityFieldData{
		{Type: "text", Name: "Color", TextValue: "Yellow"},
	}
	proposed := map[string]any{
		"Voltage":  float64(18),
		"Color":    "Black", // current non-empty, no overwrite -> dropped
		"Cordless": true,
		"Ghost":    "x", // not in template -> dropped
	}

	got := customFieldSuggestions(template, current, proposed, false)
	want := map[string]string{"Voltage": "18", "Cordless": "true"}
	if len(got) != len(want) {
		t.Fatalf("custom suggestions: got %+v, want %v", got, want)
	}
	for _, s := range got {
		if want[s.Name] != s.Suggested {
			t.Errorf("field %s: got %q, want %q", s.Name, s.Suggested, want[s.Name])
		}
	}

	// With overwrite, the filled Color field becomes suggestible.
	got = customFieldSuggestions(template, current, proposed, true)
	found := false
	for _, s := range got {
		if s.Name == "Color" && s.Suggested == "Black" && s.Current == "Yellow" {
			found = true
		}
	}
	if !found {
		t.Errorf("overwrite: expected Color suggestion, got %+v", got)
	}
}

func TestAnalyzeImagesDisabled(t *testing.T) {
	svc, _, group := newTestService(t, "")
	svc.conf = func() config.AIConf { return config.AIConf{} }

	if _, err := svc.AnalyzeImages(context.Background(), group.ID, [][]byte{testPNG(t)}, AnalyzeOptions{}); err != ErrDisabled {
		t.Errorf("got %v, want ErrDisabled", err)
	}
}

func TestFilterSuggestions(t *testing.T) {
	current := map[string]string{
		"name":          "Old Name",
		"description":   "",
		"manufacturer":  "DeWalt",
		"modelNumber":   "",
		"serialNumber":  "",
		"purchasePrice": "",
		"purchaseFrom":  "",
		"notes":         "",
	}
	suggested := map[string]string{
		"name":          "New Name",
		"description":   "A cordless drill",
		"manufacturer":  "DeWalt",
		"modelNumber":   "unknown",
		"serialNumber":  "SN-9",
		"purchasePrice": "not-a-number",
		"purchaseFrom":  "Home Depot",
		"notes":         "",
	}

	got := filterSuggestions(current, suggested, false)

	want := map[string]string{
		"description":  "A cordless drill",
		"serialNumber": "SN-9",
		"purchaseFrom": "Home Depot",
	}
	if len(got) != len(want) {
		t.Fatalf("suggestions: got %+v, want fields %v", got, want)
	}
	for _, s := range got {
		if want[s.Field] != s.Suggested {
			t.Errorf("field %s: got %q, want %q", s.Field, s.Suggested, want[s.Field])
		}
	}

	// With overwrite, filled fields become suggestible; equal values and
	// junk values stay excluded.
	got = filterSuggestions(current, suggested, true)
	fields := map[string]bool{}
	for _, s := range got {
		fields[s.Field] = true
	}
	if !fields["name"] {
		t.Error("overwrite: expected name suggestion")
	}
	if fields["manufacturer"] {
		t.Error("overwrite: manufacturer equals current and must be excluded")
	}
	if fields["modelNumber"] {
		t.Error("'unknown' must be treated as empty")
	}
	if fields["purchasePrice"] {
		t.Error("unparseable price must be excluded")
	}
}
